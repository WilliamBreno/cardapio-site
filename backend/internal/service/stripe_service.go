package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/stripe/stripe-go/v86"
	"gorm.io/gorm"
)

// custoRealMercadoPagoPercentual é o custo real médio do Mercado Pago
// (Pix, ~0,99% do GMV processado) — usado como base do "lucro líquido"
// da comissão de plataforma pra calcular a comissão do afiliado (Fase
// 7.5, ver docs/drenux-planos-comissoes-definido.md § 4). Número de
// negócio fixo, não uma conta em tempo real contra a taxa efetiva por
// método de pagamento (mesma simplificação já usada na comissão de
// plataforma em si, ver calcularComissaoEscalonada).
const custoRealMercadoPagoPercentual = 0.99

// ProporcaoComissaoAfiliadoPadrao é o valor sugerido no cadastro de um
// afiliado novo — 45% do LUCRO LÍQUIDO da comissão de plataforma (Fase
// 7.5), o padrão usado igual pra todo mundo antes da Fase 5.5 permitir
// negociar por afiliado (ver domain.Afiliado.ComissaoPercentual). Depois
// do cadastro, quem manda é o percentual salvo em cada Afiliado, não essa
// constante. Antes da Fase 7.5 era 37,6% sobre a comissão BRUTA — mudou
// de base (bruto → líquido) e de percentual juntos, ver
// calcularComissaoAfiliado.
const ProporcaoComissaoAfiliadoPadrao = 0.45

// calcularComissaoAfiliado é a fórmula única de comissão de afiliado,
// usada tanto pelo repasse automático via Stripe Transfer quanto pelo
// registro manual do Mercado Pago (ver RepasseAfiliadoService, Fase 5.5
// do roadmap). Desde a Fase 7.5, aplica proporcaoAfiliado sobre o LUCRO
// LÍQUIDO desse pedido — a comissão que a Drenux cobra da loja
// (calcularComissaoEscalonada, Fase 7.2) menos o custo real do Mercado
// Pago sobre o mesmo valor (custoRealMercadoPagoPercentual) — nunca sobre
// a comissão bruta. Isso garante que a Drenux nunca opera no negativo
// numa loja indicada, em qualquer volume: o afiliado nunca leva uma fatia
// maior do que a margem que sobra depois do custo do processador.
// proporcaoAfiliado vem de domain.Afiliado.ComissaoPercentual — cada
// afiliado pode ter a própria, não é mais um valor fixo global. gmvAntes
// é o GMV do mês da loja antes desse pedido (ver
// calcularComissaoEscalonada).
func calcularComissaoAfiliado(pedidoTotal float64, plano string, gmvAntes, proporcaoAfiliado float64) float64 {
	comissaoDrenux := calcularComissaoEscalonada(plano, gmvAntes, pedidoTotal)
	custoProcessador := pedidoTotal * custoRealMercadoPagoPercentual / 100
	lucroLiquido := math.Max(0, comissaoDrenux-custoProcessador)
	return lucroLiquido * proporcaoAfiliado
}

// valoresMensalidadePlano define o preço mensal (em reais) de cada plano
// pago. Start e Basic não entram aqui — nenhum dos dois tem mensalidade.
var valoresMensalidadePlano = map[string]float64{
	"pro":   99.0,
	"scale": 349.0,
}

// ordemPlano define a hierarquia de planos, usada em dois lugares: (1)
// MudarPlano, pra decidir se uma troca é upgrade (imediato) ou downgrade
// (agendado); (2) validação de entrada — qualquer plano fora deste mapa é
// inválido.
var ordemPlano = map[string]int{"start": 0, "basic": 1, "pro": 2, "scale": 3}

// temMensalidade diz se um plano tem cobrança recorrente (Pro/Scale) — só
// esses passam por checkout/assinatura Stripe. Start e Basic são os dois
// planos "grátis" (Basic tem comissão por pedido, mas nenhuma mensalidade).
func temMensalidade(plano string) bool {
	_, ok := valoresMensalidadePlano[plano]
	return ok
}

type StripeService struct {
	client             *stripe.Client
	secretKey          string
	webhookSecret      string
	db                 *gorm.DB
	lojaRepo           *repository.LojaRepository
	pedidoRepo         *repository.PedidoRepository
	solicitacaoRepo    *repository.SolicitacaoEntregaRepository
	afiliadoRepo       *repository.AfiliadoRepository
	assinaturaRepo     *repository.AssinaturaPendenteRepository
	notificationSender notification.NotificationSender
	emailSender        *notification.EmailSender
	frontendURL        string
	posPagamento       *PosPagamentoService
}

func NewStripeService(secretKey, webhookSecret string, db *gorm.DB, notificationSender notification.NotificationSender, emailSender *notification.EmailSender, frontendURL string, posPagamento *PosPagamentoService) *StripeService {
	return &StripeService{
		client:             stripe.NewClient(secretKey),
		secretKey:          secretKey,
		webhookSecret:      webhookSecret,
		db:                 db,
		lojaRepo:           repository.NewLojaRepository(db),
		pedidoRepo:         repository.NewPedidoRepository(db),
		solicitacaoRepo:    repository.NewSolicitacaoEntregaRepository(db),
		afiliadoRepo:       repository.NewAfiliadoRepository(db),
		assinaturaRepo:     repository.NewAssinaturaPendenteRepository(db),
		notificationSender: notificationSender,
		emailSender:        emailSender,
		frontendURL:        frontendURL,
		posPagamento:       posPagamento,
	}
}

// IniciarOnboarding garante que a loja tem uma conta Stripe Connect tipo
// Express — cria uma na primeira vez, reaproveita se já existir — e
// devolve um link de onboarding de uso único.
func (s *StripeService) IniciarOnboarding(ctx context.Context, lojaID uint, returnURL, refreshURL string) (string, error) {
	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return "", errors.New("loja não encontrada")
	}

	accountID := loja.StripeAccountID
	if accountID == "" {
		accountParams := &stripe.AccountCreateParams{
			Type:    stripe.String(string(stripe.AccountTypeExpress)),
			Country: stripe.String("BR"),
		}
		account, err := s.client.V1Accounts.Create(ctx, accountParams)
		if err != nil {
			return "", fmt.Errorf("criando conta Stripe: %w", err)
		}
		accountID = account.ID

		if err := s.lojaRepo.AtualizarStripeAccountID(lojaID, accountID); err != nil {
			return "", fmt.Errorf("salvando conta Stripe: %w", err)
		}
	}

	linkParams := &stripe.AccountLinkCreateParams{
		Account:    stripe.String(accountID),
		Type:       stripe.String("account_onboarding"),
		ReturnURL:  stripe.String(returnURL),
		RefreshURL: stripe.String(refreshURL),
	}
	link, err := s.client.V1AccountLinks.Create(ctx, linkParams)
	if err != nil {
		return "", fmt.Errorf("gerando link de onboarding: %w", err)
	}

	return link.URL, nil
}

func (s *StripeService) StatusOnboarding(lojaID uint) (bool, error) {
	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return false, errors.New("loja não encontrada")
	}
	return loja.StripeAccountID != "", nil
}

// CriarCheckout monta uma sessão de pagamento Stripe Checkout pra um
// pedido específico, direcionada pra conta Connect da loja.
func (s *StripeService) CriarCheckout(ctx context.Context, pedidoID uint, frontendURL string) (string, error) {
	pedido, err := s.pedidoRepo.BuscarPorID(pedidoID)
	if err != nil {
		return "", errors.New("pedido não encontrado")
	}
	if pedido.Status != domain.StatusAguardandoPagamento {
		return "", errors.New("esse pedido já foi pago ou cancelado")
	}

	loja, err := s.lojaRepo.BuscarPorID(pedido.LojaID)
	if err != nil {
		return "", errors.New("loja não encontrada")
	}
	if loja.StripeAccountID == "" {
		return "", errors.New("essa loja ainda não conectou uma conta de pagamento")
	}

	successURL := fmt.Sprintf("%s/%s?pago=1", frontendURL, loja.Slug)
	cancelURL := fmt.Sprintf("%s/%s", frontendURL, loja.Slug)

	lineItems := make([]*stripe.CheckoutSessionCreateLineItemParams, 0, len(pedido.Itens))
	for _, item := range pedido.Itens {
		lineItems = append(lineItems, &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(int64(item.Quantidade)),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String("brl"),
				UnitAmount: stripe.Int64(int64(math.Round(item.PrecoUnit * 100))),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String(item.ProdutoNome),
				},
			},
		})
	}

	// Comissão da plataforma é escalonada por faixa de GMV mensal da loja
	// (calcularComissaoEscalonada, Fase 7.2) — mesma fórmula usada pelo
	// Mercado Pago. Não incide sobre frete — mesma regra de lá (ver
	// MercadoPagoService.CriarCheckout, baseComissao). NOTA: essa rota
	// (checkout de pedido via Stripe) está sem uso desde a Fase 5.2 —
	// nenhuma rota chama CriarCheckout pra pedido hoje, o checkout migrou
	// pro Mercado Pago — mantido corrigido só por consistência, caso essa
	// rota volte a ser registrada algum dia.
	baseComissao := pedido.Total - pedido.TaxaEntrega
	gmvAntes, err := s.pedidoRepo.SomarGMVMesAtual(loja.ID)
	if err != nil {
		return "", fmt.Errorf("calculando GMV do mês da loja %d: %w", loja.ID, err)
	}
	taxaPlataforma := int64(math.Round(calcularComissaoEscalonada(loja.Plano, gmvAntes, baseComissao) * 100))

	params := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:  lineItems,
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentIntentData: &stripe.CheckoutSessionCreatePaymentIntentDataParams{
			ApplicationFeeAmount: stripe.Int64(taxaPlataforma),
			TransferData: &stripe.CheckoutSessionCreatePaymentIntentDataTransferDataParams{
				Destination: stripe.String(loja.StripeAccountID),
			},
		},
		Metadata: map[string]string{
			"pedido_id": strconv.FormatUint(uint64(pedido.ID), 10),
		},
	}

	session, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("criando sessão de checkout: %w", err)
	}

	if err := s.pedidoRepo.AtualizarStripeSessionID(pedido.ID, session.ID); err != nil {
		fmt.Printf("aviso: não foi possível salvar stripe_session_id do pedido %d: %v\n", pedido.ID, err)
	}

	return session.URL, nil
}

// CriarCheckoutFrete monta uma sessão de pagamento Stripe Checkout pro
// frete de uma SolicitacaoEntrega.
func (s *StripeService) CriarCheckoutFrete(ctx context.Context, solicitacaoID uint, frontendURL string) (string, error) {
	solicitacao, err := s.solicitacaoRepo.BuscarPorID(solicitacaoID)
	if err != nil {
		return "", errors.New("solicitação de entrega não encontrada")
	}
	if solicitacao.Status != domain.StatusSolicitacaoAguardandoPagamento {
		return "", errors.New("essa solicitação já foi paga ou cancelada")
	}

	loja, err := s.lojaRepo.BuscarPorID(solicitacao.LojaID)
	if err != nil {
		return "", errors.New("loja não encontrada")
	}
	if loja.StripeAccountID == "" {
		return "", errors.New("essa loja ainda não conectou uma conta de pagamento")
	}

	successURL := fmt.Sprintf("%s/%s?frete_pago=1", frontendURL, loja.Slug)
	cancelURL := fmt.Sprintf("%s/%s", frontendURL, loja.Slug)

	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String("brl"),
					UnitAmount: stripe.Int64(int64(math.Round(solicitacao.ValorFrete * 100))),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name: stripe.String("Frete — entrega de itens guardados"),
					},
				},
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentIntentData: &stripe.CheckoutSessionCreatePaymentIntentDataParams{
			TransferData: &stripe.CheckoutSessionCreatePaymentIntentDataTransferDataParams{
				Destination: stripe.String(loja.StripeAccountID),
			},
		},
		Metadata: map[string]string{
			"solicitacao_id": strconv.FormatUint(uint64(solicitacao.ID), 10),
		},
	}

	session, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("criando sessão de checkout do frete: %w", err)
	}

	if err := s.solicitacaoRepo.AtualizarStripeSessionID(solicitacao.ID, session.ID); err != nil {
		fmt.Printf("aviso: não foi possível salvar stripe_session_id da solicitação %d: %v\n", solicitacao.ID, err)
	}

	return session.URL, nil
}

// obterOuCriarPriceAssinatura acha (ou cria, na primeira vez) o Price
// recorrente de um plano pago, usando lookup_key como identificador
// estável — evita duplicar Product/Price a cada assinatura nova.
//
// A busca é feita via chamada HTTP direta à API da Stripe (em vez do
// iterador do SDK) — mais simples e previsível que acompanhar a API de
// listagem genérica do client tipado.
func (s *StripeService) obterOuCriarPriceAssinatura(ctx context.Context, plano string) (string, error) {
	lookupKey := fmt.Sprintf("drenux_%s_mensal", plano)

	priceID, err := s.buscarPricePorLookupKey(ctx, lookupKey)
	if err != nil {
		return "", fmt.Errorf("buscando price existente: %w", err)
	}
	if priceID != "" {
		return priceID, nil
	}

	valorMensal, ok := valoresMensalidadePlano[plano]
	if !ok {
		return "", fmt.Errorf("plano %q não tem mensalidade configurada", plano)
	}

	nomeProduto := fmt.Sprintf("Drenux %s%s", strings.ToUpper(plano[:1]), plano[1:])
	product, err := s.client.V1Products.Create(ctx, &stripe.ProductCreateParams{
		Name: stripe.String(nomeProduto),
	})
	if err != nil {
		return "", fmt.Errorf("criando produto Stripe pro plano %s: %w", plano, err)
	}

	price, err := s.client.V1Prices.Create(ctx, &stripe.PriceCreateParams{
		Currency:   stripe.String("brl"),
		UnitAmount: stripe.Int64(int64(math.Round(valorMensal * 100))),
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval: stripe.String("month"),
		},
		Product:   stripe.String(product.ID),
		LookupKey: stripe.String(lookupKey),
	})
	if err != nil {
		return "", fmt.Errorf("criando price Stripe pro plano %s: %w", plano, err)
	}

	return price.ID, nil
}

// buscarPricePorLookupKey consulta a API da Stripe diretamente via HTTP,
// pedindo só o Price com aquele lookup_key. Devolve "" (sem erro) se
// nenhum Price ainda existir com essa chave.
func (s *StripeService) buscarPricePorLookupKey(ctx context.Context, lookupKey string) (string, error) {
	url := fmt.Sprintf("https://api.stripe.com/v1/prices?lookup_keys[]=%s", lookupKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var resultado struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&resultado); err != nil {
		return "", fmt.Errorf("lendo resposta da Stripe: %w", err)
	}

	if len(resultado.Data) == 0 {
		return "", nil
	}
	return resultado.Data[0].ID, nil
}

// CriarCheckoutAssinatura monta o Checkout Session de assinatura pro
// plano Pro/Scale — é a conta PRINCIPAL da Stripe (não Connect), já que
// é a plataforma cobrando o lojista, não uma loja cobrando cliente.
func (s *StripeService) CriarCheckoutAssinatura(ctx context.Context, plano string) (string, error) {
	if plano != "pro" && plano != "scale" {
		return "", fmt.Errorf("plano inválido: %s", plano)
	}

	priceID, err := s.obterOuCriarPriceAssinatura(ctx, plano)
	if err != nil {
		return "", err
	}

	successURL := fmt.Sprintf("%s/cadastro/finalizar?session_id={CHECKOUT_SESSION_ID}", s.frontendURL)
	cancelURL := fmt.Sprintf("%s/", s.frontendURL)

	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			"tipo":  "assinatura_plano",
			"plano": plano,
		},
	}

	session, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("criando sessão de assinatura: %w", err)
	}

	return session.URL, nil
}

// BuscarAssinaturaPendentePorToken é usado pela tela de "finalizar
// cadastro" pra confirmar que o token é válido e ainda não foi usado.
func (s *StripeService) BuscarAssinaturaPendentePorToken(token string) (*domain.AssinaturaPendente, error) {
	return s.assinaturaRepo.BuscarPorToken(token)
}

// BuscarAssinaturaPendentePorSessionID é usado no redirecionamento
// direto da Stripe (?session_id=...) — o frontend chama isso em loop
// curto até o webhook terminar de processar e o registro aparecer.
func (s *StripeService) BuscarAssinaturaPendentePorSessionID(sessionID string) (*domain.AssinaturaPendente, error) {
	return s.assinaturaRepo.BuscarPorSessionID(sessionID)
}

// MudarPlanoResultado informa o frontend se a troca já foi aplicada
// (upgrade/troca entre pagos) ou se precisa redirecionar pro Checkout
// (Start → Pro/Scale, quando ainda não existe assinatura).
type MudarPlanoResultado struct {
	CheckoutURL string
	Imediato    bool
}

// MudarPlano decide o caminho certo conforme a direção da troca:
//   - Start → Pro/Scale: cria uma assinatura nova via Checkout (não tem
//     cartão salvo ainda, precisa da Stripe coletar).
//   - Pro ↔ Scale: já tem assinatura ativa, só troca o Price — imediato,
//     com proration automático da própria Stripe.
//   - Qualquer downgrade: agenda pro fim do ciclo atual, sem mexer na
//     assinatura agora — aplicado depois pelo webhook de renovação.
func (s *StripeService) MudarPlano(ctx context.Context, lojaID uint, novoPlano string) (*MudarPlanoResultado, error) {
	if _, ok := ordemPlano[novoPlano]; !ok {
		return nil, fmt.Errorf("plano inválido: %s", novoPlano)
	}

	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}

	if loja.PlanoVitalicio {
		return nil, errors.New("essa conta tem plano vitalício — não pode ser alterada")
	}

	if novoPlano == loja.Plano && loja.PlanoAgendado == nil {
		return nil, errors.New("essa loja já está nesse plano")
	}

	// Troca entre planos sem mensalidade (Start <-> Basic): aplica na
	// hora, não existe cobrança nem assinatura envolvida nos dois lados.
	if !temMensalidade(loja.Plano) && !temMensalidade(novoPlano) {
		if err := s.lojaRepo.AtualizarPlano(loja.ID, novoPlano, "", ""); err != nil {
			return nil, err
		}
		return &MudarPlanoResultado{Imediato: true}, nil
	}

	atual, novo := ordemPlano[loja.Plano], ordemPlano[novoPlano]

	// Upgrade ou troca entre planos pagos: imediato
	if novo > atual {
		if loja.StripeSubscriptionID == "" {
			// Sem assinatura ativa ainda (vindo do Start) — precisa do
			// Checkout da Stripe pra coletar o cartão.
			url, err := s.criarCheckoutMudancaPlano(ctx, loja.ID, novoPlano)
			if err != nil {
				return nil, err
			}
			return &MudarPlanoResultado{CheckoutURL: url, Imediato: false}, nil
		}

		// Já tem assinatura — só troca o Price, sem passar pelo Checkout
		if err := s.atualizarPriceDaAssinatura(ctx, loja.StripeSubscriptionID, novoPlano); err != nil {
			return nil, err
		}
		if err := s.lojaRepo.AtualizarPlano(loja.ID, novoPlano, "", ""); err != nil {
			return nil, err
		}
		return &MudarPlanoResultado{Imediato: true}, nil
	}

	// Downgrade: agenda pro fim do ciclo, não mexe em nada agora
	if err := s.lojaRepo.AtualizarPlanoAgendado(loja.ID, &novoPlano); err != nil {
		return nil, err
	}
	return &MudarPlanoResultado{Imediato: false}, nil
}

// CancelarMudancaAgendada desfaz um downgrade agendado — a loja
// continua no plano atual normalmente, sem nenhuma mudança futura.
func (s *StripeService) CancelarMudancaAgendada(lojaID uint) error {
	return s.lojaRepo.AtualizarPlanoAgendado(lojaID, nil)
}

// criarCheckoutMudancaPlano monta o Checkout de assinatura pra uma loja
// que JÁ EXISTE (diferente de CriarCheckoutAssinatura, que é pro
// cadastro de loja nova) — por isso os metadados levam loja_id direto,
// sem precisar do fluxo de "finalizar cadastro".
func (s *StripeService) criarCheckoutMudancaPlano(ctx context.Context, lojaID uint, plano string) (string, error) {
	priceID, err := s.obterOuCriarPriceAssinatura(ctx, plano)
	if err != nil {
		return "", err
	}

	successURL := fmt.Sprintf("%s/admin/configuracoes?plano_atualizado=1", s.frontendURL)
	cancelURL := fmt.Sprintf("%s/admin/configuracoes", s.frontendURL)

	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Metadata: map[string]string{
			"tipo":    "mudanca_plano",
			"loja_id": strconv.FormatUint(uint64(lojaID), 10),
			"plano":   plano,
		},
	}

	session, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("criando sessão de mudança de plano: %w", err)
	}

	return session.URL, nil
}

// atualizarPriceDaAssinatura troca o Price de uma assinatura já ativa
// (Pro ↔ Scale) sem passar pelo Checkout — a Stripe calcula sozinha a
// proporção a cobrar/creditar pelo resto do ciclo atual.
func (s *StripeService) atualizarPriceDaAssinatura(ctx context.Context, subscriptionID, novoPlano string) error {
	novoPriceID, err := s.obterOuCriarPriceAssinatura(ctx, novoPlano)
	if err != nil {
		return err
	}

	sub, err := s.client.V1Subscriptions.Retrieve(ctx, subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("buscando assinatura atual: %w", err)
	}
	if len(sub.Items.Data) == 0 {
		return errors.New("assinatura sem itens — estado inesperado")
	}
	itemID := sub.Items.Data[0].ID

	_, err = s.client.V1Subscriptions.Update(ctx, subscriptionID, &stripe.SubscriptionUpdateParams{
		Items: []*stripe.SubscriptionUpdateItemParams{
			{ID: stripe.String(itemID), Price: stripe.String(novoPriceID)},
		},
	})
	if err != nil {
		return fmt.Errorf("atualizando price da assinatura: %w", err)
	}

	return nil
}

// processarMudancaPlanoConfirmada aplica a troca imediatamente quando o
// Checkout de uma loja JÁ EXISTENTE (Start → Pro/Scale) é confirmado.
func (s *StripeService) processarMudancaPlanoConfirmada(session *stripe.CheckoutSession) error {
	lojaIDStr := session.Metadata["loja_id"]
	plano := session.Metadata["plano"]

	lojaID, err := strconv.ParseUint(lojaIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("loja_id inválido nos metadados: %w", err)
	}

	customerID := ""
	if session.Customer != nil {
		customerID = session.Customer.ID
	}
	subscriptionID := ""
	if session.Subscription != nil {
		subscriptionID = session.Subscription.ID
	}

	return s.lojaRepo.AtualizarPlano(uint(lojaID), plano, customerID, subscriptionID)
}

// processarRenovacaoAssinatura roda a cada renovação mensal bem-sucedida
// (evento invoice.payment_succeeded) e aplica um downgrade agendado, se
// existir — é o único lugar onde PlanoAgendado é consumido.
func (s *StripeService) processarRenovacaoAssinatura(payload []byte) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(payload, &invoice); err != nil {
		return fmt.Errorf("lendo dados da invoice: %w", err)
	}

	// Desde a versão "Basil" da API, invoice.subscription foi
	// descontinuado — o caminho certo agora é invoice.parent.
	// subscription_details.subscription (só existe quando parent.type
	// == "subscription_details").
	if invoice.Parent == nil || invoice.Parent.SubscriptionDetails == nil || invoice.Parent.SubscriptionDetails.Subscription == nil {
		return nil // fatura avulsa, não é de assinatura — ignora
	}
	subscriptionID := invoice.Parent.SubscriptionDetails.Subscription.ID

	loja, err := s.lojaRepo.BuscarPorStripeSubscriptionID(subscriptionID)
	if err != nil {
		return nil // renovação de assinatura que não é de loja nossa (não deveria acontecer, mas não é erro fatal)
	}

	if loja.PlanoAgendado == nil {
		return nil // nada agendado, renovação normal
	}

	novoPlano := *loja.PlanoAgendado

	if !temMensalidade(novoPlano) {
		// Downgrade pra Start ou Basic — nenhum dos dois tem mensalidade,
		// então a assinatura Stripe é cancelada de vez nos dois casos; só
		// muda o plano final gravado na loja.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := s.client.V1Subscriptions.Cancel(ctx, loja.StripeSubscriptionID, nil); err != nil {
			log.Printf("aviso: falha ao cancelar assinatura da loja %d no downgrade agendado: %v", loja.ID, err)
		}
		return s.lojaRepo.LimparAssinatura(loja.ID, novoPlano)
	}

	// Downgrade pra outro plano pago (ex: Scale → Pro)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.atualizarPriceDaAssinatura(ctx, loja.StripeSubscriptionID, novoPlano); err != nil {
		log.Printf("aviso: falha ao aplicar downgrade agendado da loja %d: %v", loja.ID, err)
		return err
	}
	return s.lojaRepo.AtualizarPlano(loja.ID, novoPlano, "", "")
}

// ProcessarWebhook valida a assinatura do evento e trata os três casos
// possíveis: pedido normal pago, frete de itens guardados pago, ou
// assinatura de plano (Pro/Scale) confirmada.
func (s *StripeService) ProcessarWebhook(payload []byte, signature string) error {
	event, err := stripe.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return fmt.Errorf("assinatura do webhook inválida: %w", err)
	}

	// Renovação de assinatura — é aqui que downgrades agendados são
	// aplicados, uma vez por ciclo de cobrança.
	if event.Type == "invoice.payment_succeeded" {
		return s.processarRenovacaoAssinatura(event.Data.Raw)
	}

	if event.Type != "checkout.session.completed" {
		return nil
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("lendo dados do evento: %w", err)
	}

	// Caso 1: frete de itens guardados
	if solicitacaoIDStr, ok := session.Metadata["solicitacao_id"]; ok {
		solicitacaoID, err := strconv.ParseUint(solicitacaoIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("solicitacao_id inválido nos metadados: %w", err)
		}
		if err := s.solicitacaoRepo.AtualizarStatus(uint(solicitacaoID), domain.StatusSolicitacaoPaga); err != nil {
			return fmt.Errorf("atualizando status da solicitação %d: %w", solicitacaoID, err)
		}
		go s.notificarFretePago(uint(solicitacaoID))
		return nil
	}

	// Caso 2: assinatura de plano Pro/Scale confirmada (loja NOVA, ainda
	// não cadastrada — vem do fluxo de cadastro)
	if tipo, ok := session.Metadata["tipo"]; ok && tipo == "assinatura_plano" {
		return s.processarAssinaturaConfirmada(&session)
	}

	// Caso 3: mudança de plano de uma loja JÁ EXISTENTE (Start → Pro/Scale
	// pedido de dentro do painel admin)
	if tipo, ok := session.Metadata["tipo"]; ok && tipo == "mudanca_plano" {
		return s.processarMudancaPlanoConfirmada(&session)
	}

	// Caso 4: pedido normal
	pedidoIDStr, ok := session.Metadata["pedido_id"]
	if !ok {
		return errors.New("evento sem pedido_id, solicitacao_id nem tipo de assinatura nos metadados")
	}

	pedidoID, err := strconv.ParseUint(pedidoIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("pedido_id inválido nos metadados: %w", err)
	}

	if err := s.pedidoRepo.AtualizarStatus(uint(pedidoID), domain.StatusPago); err != nil {
		return fmt.Errorf("atualizando status do pedido %d: %w", pedidoID, err)
	}

	go s.processarPosPagamento(uint(pedidoID))

	return nil
}

// processarAssinaturaConfirmada cria o registro de AssinaturaPendente e
// dispara o email com o link de "finalizar cadastro" — sem prazo de
// expiração, já que o cliente pagou de verdade e o link precisa
// continuar válido até ele completar o cadastro, mesmo dias depois.
func (s *StripeService) processarAssinaturaConfirmada(session *stripe.CheckoutSession) error {
	plano := session.Metadata["plano"]

	email := ""
	if session.CustomerDetails != nil {
		email = session.CustomerDetails.Email
	}
	if email == "" {
		return errors.New("sessão de assinatura sem email do cliente")
	}

	customerID := ""
	if session.Customer != nil {
		customerID = session.Customer.ID
	}
	subscriptionID := ""
	if session.Subscription != nil {
		subscriptionID = session.Subscription.ID
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("gerando token da assinatura: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	assinatura := domain.AssinaturaPendente{
		Email:                email,
		Plano:                plano,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: subscriptionID,
		StripeSessionID:      session.ID,
		Token:                token,
	}
	if err := s.assinaturaRepo.Criar(&assinatura); err != nil {
		return fmt.Errorf("salvando assinatura pendente: %w", err)
	}

	if s.emailSender == nil {
		log.Printf("aviso: email não configurado — assinatura pendente %d criada mas email não enviado. Token: %s", assinatura.ID, token)
		return nil
	}

	link := fmt.Sprintf("%s/cadastro/finalizar?token=%s", s.frontendURL, token)
	if err := s.emailSender.EnviarAssinaturaConfirmada(email, plano, link); err != nil {
		log.Printf("falha ao enviar email de assinatura confirmada: %v", err)
	}

	return nil
}

func (s *StripeService) notificarFretePago(solicitacaoID uint) {
	if s.notificationSender == nil {
		log.Printf("WhatsApp não conectado — frete da solicitação %d foi pago mas a notificação foi pulada", solicitacaoID)
		return
	}

	solicitacao, err := s.solicitacaoRepo.BuscarPorID(solicitacaoID)
	if err != nil {
		log.Printf("não foi possível recarregar solicitação %d pra notificar: %v", solicitacaoID, err)
		return
	}

	loja, err := s.lojaRepo.BuscarPorID(solicitacao.LojaID)
	if err != nil {
		log.Printf("não foi possível carregar loja da solicitação %d pra notificar: %v", solicitacaoID, err)
		return
	}
	if loja.WhatsappNumero == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	texto := fmt.Sprintf(
		"📦 Frete pago — %s\n\nO cliente %s pagou o frete pra receber os itens guardados. Endereço: %s",
		loja.Nome, solicitacao.ClienteNome, solicitacao.EnderecoEntrega,
	)
	if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, texto); err != nil {
		log.Printf("falha ao notificar admin do frete pago (solicitação %d): %v", solicitacaoID, err)
	}

	// Reforço — a solicitação já fica marcada com PesoPendente (visível na
	// lista do admin), mas manda mais essa mensagem separada pra não
	// depender só do lojista abrir a tela pra perceber que o frete
	// cobrado pode estar errado.
	if solicitacao.PesoPendente {
		ctxAviso, cancelAviso := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelAviso()
		nomes := nomesItensSemPeso(solicitacao.Itens)
		aviso := fmt.Sprintf(
			"⚠️ Peso pendente — %s\n\nA entrega #%d de %s ficou fora da sua região e foi cobrada com o frete estimado, mas tem produto(s) sem peso cadastrado: %s.\n\nEsse frete pode estar sub-cobrado — confere e ajusta manualmente se precisar, e já aproveita pra cadastrar o peso desses produtos.",
			loja.Nome, solicitacao.ID, solicitacao.ClienteNome, strings.Join(nomes, ", "),
		)
		if err := s.notificationSender.EnviarTextoAdmin(ctxAviso, loja.WhatsappNumero, aviso); err != nil {
			log.Printf("falha ao enviar aviso de peso pendente da solicitação %d: %v", solicitacaoID, err)
		}
	}
}

func (s *StripeService) processarPosPagamento(pedidoID uint) {
	pedido, err := s.pedidoRepo.BuscarPorID(pedidoID)
	if err != nil {
		log.Printf("não foi possível recarregar pedido %d pós-pagamento: %v", pedidoID, err)
		return
	}

	loja, err := s.lojaRepo.BuscarPorID(pedido.LojaID)
	if err != nil {
		log.Printf("não foi possível carregar loja do pedido %d pós-pagamento: %v", pedidoID, err)
		return
	}

	if loja.AfiliadoID != nil {
		s.transferirComissaoAfiliado(pedido, loja)
	}

	// Estoque e notificações são compartilhados com o Mercado Pago — ver
	// PosPagamentoService.
	s.posPagamento.ProcessarPedidoPago(pedidoID)
}

func (s *StripeService) transferirComissaoAfiliado(pedido *domain.Pedido, loja *domain.Loja) {
	if loja.AfiliadoID == nil {
		return
	}

	afiliado, err := s.afiliadoRepo.BuscarPorID(*loja.AfiliadoID)
	if err != nil {
		log.Printf("aviso: afiliado %d não encontrado pro pedido %d: %v", *loja.AfiliadoID, pedido.ID, err)
		return
	}
	if afiliado.StripeAccountID == "" {
		log.Printf("aviso: afiliado %d ainda não conectou conta Stripe — repasse do pedido %d pulado por enquanto", afiliado.ID, pedido.ID)
		return
	}

	// Comissão não incide sobre frete — mesma regra do marketplace_fee do
	// Mercado Pago. NOTA: rota sem uso desde a Fase 5.2 (ver comentário em
	// CriarCheckout acima) — corrigido só por consistência.
	gmvAntes, err := s.pedidoRepo.SomarGMVMesAtual(loja.ID)
	if err != nil {
		log.Printf("aviso: não foi possível calcular GMV do mês da loja %d pro repasse de afiliado do pedido %d: %v", loja.ID, pedido.ID, err)
	}
	valorComissao := calcularComissaoAfiliado(pedido.Total-pedido.TaxaEntrega, loja.Plano, gmvAntes, afiliado.ComissaoPercentual)
	valorCentavos := int64(math.Round(valorComissao * 100))
	if valorCentavos <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	params := &stripe.TransferCreateParams{
		Amount:      stripe.Int64(valorCentavos),
		Currency:    stripe.String(string(stripe.CurrencyBRL)),
		Destination: stripe.String(afiliado.StripeAccountID),
	}
	transfer, err := s.client.V1Transfers.Create(ctx, params)
	if err != nil {
		log.Printf("falha ao repassar comissão de afiliado pro pedido %d: %v", pedido.ID, err)
		return
	}

	if err := s.pedidoRepo.AtualizarComissaoAfiliado(pedido.ID, valorComissao, transfer.ID); err != nil {
		log.Printf("aviso: não foi possível salvar registro de comissão do pedido %d: %v", pedido.ID, err)
	}
}

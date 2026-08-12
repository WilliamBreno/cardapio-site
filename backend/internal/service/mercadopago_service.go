package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// MercadoPagoService implementa a Fase 5 do roadmap (ver
// docs/plano-melhorias-drenux.md): cada Loja conecta a própria conta do
// Mercado Pago via OAuth (modelo "split 1:1", self-service — diferente do
// modelo 1:N, que precisa de contato comercial) e os pedidos são cobrados
// direto na conta dela, com a Drenux retendo uma fatia via
// marketplace_fee. Repasse de comissão de afiliado (Fase 5.5) ainda não
// está implementado aqui — ver PosPagamentoService e o aviso em
// processarPosPagamento mais abaixo.
type MercadoPagoService struct {
	clientID      string
	clientSecret  string
	webhookSecret string
	jwtSecret     string
	apiPublicURL  string
	frontendURL   string
	httpClient    *http.Client

	lojaRepo       *repository.LojaRepository
	pedidoRepo     *repository.PedidoRepository
	posPagamento   *PosPagamentoService
	repasseService *RepasseAfiliadoService

	// notificationSender avisa o dono por WhatsApp quando um cliente tenta
	// pagar e a loja ainda não conectou o Mercado Pago (ver
	// avisarPagamentoNaoConfigurado) — nil-safe, igual ao resto do
	// projeto: se o WhatsApp não estiver conectado, só pula o aviso.
	notificationSender notification.NotificationSender
}

func NewMercadoPagoService(clientID, clientSecret, webhookSecret, jwtSecret, apiPublicURL, frontendURL string, db *gorm.DB, posPagamento *PosPagamentoService, repasseService *RepasseAfiliadoService, notificationSender notification.NotificationSender) *MercadoPagoService {
	return &MercadoPagoService{
		clientID:           clientID,
		clientSecret:       clientSecret,
		webhookSecret:      webhookSecret,
		jwtSecret:          jwtSecret,
		apiPublicURL:       apiPublicURL,
		frontendURL:        frontendURL,
		httpClient:         &http.Client{Timeout: 20 * time.Second},
		lojaRepo:           repository.NewLojaRepository(db),
		pedidoRepo:         repository.NewPedidoRepository(db),
		posPagamento:       posPagamento,
		repasseService:     repasseService,
		notificationSender: notificationSender,
	}
}

func (s *MercadoPagoService) redirectURI() string {
	return s.apiPublicURL + "/admin/mercadopago/callback"
}

// mercadoPagoStateClaims carrega o loja_id no "state" do OAuth — o
// callback do Mercado Pago não manda nenhum header de autenticação nossa
// (é um redirect de navegador vindo direto do Mercado Pago), então
// precisamos de alguma forma de saber com segurança de qual loja é aquele
// código. Um JWT curto assinado com o mesmo segredo usado no login
// resolve isso sem precisar guardar estado em sessão/banco.
type mercadoPagoStateClaims struct {
	LojaID uint `json:"loja_id"`
	jwt.RegisteredClaims
}

// IniciarOnboarding monta a URL de autorização OAuth do Mercado Pago pra
// essa loja — o dono é redirecionado pra lá, loga na própria conta MP
// dele e autoriza a Drenux.
func (s *MercadoPagoService) IniciarOnboarding(lojaID uint) (string, error) {
	if s.clientID == "" {
		return "", errors.New("integração com o Mercado Pago não está configurada (MERCADOPAGO_CLIENT_ID ausente)")
	}

	claims := mercadoPagoStateClaims{
		LojaID: lojaID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	state, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("gerando state do onboarding: %w", err)
	}

	query := url.Values{}
	query.Set("client_id", s.clientID)
	query.Set("response_type", "code")
	query.Set("platform_id", "mp")
	query.Set("state", state)
	query.Set("redirect_uri", s.redirectURI())

	return "https://auth.mercadopago.com.br/authorization?" + query.Encode(), nil
}

// StatusOnboarding diz se a loja já conectou uma conta do Mercado Pago.
func (s *MercadoPagoService) StatusOnboarding(lojaID uint) (bool, error) {
	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return false, errors.New("loja não encontrada")
	}
	return loja.MercadoPagoUserID != "", nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	UserID       int64  `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
	PublicKey    string `json:"public_key"`
	Error        string `json:"error"`
	ErrorMessage string `json:"message"`
}

// ProcessarCallback valida o state, troca o code pelo access_token e
// salva a conexão na loja. Devolve o ID da loja pra o handler saber pra
// onde redirecionar de volta.
func (s *MercadoPagoService) ProcessarCallback(ctx context.Context, code, state string) (uint, error) {
	var claims mercadoPagoStateClaims
	_, err := jwt.ParseWithClaims(state, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil || claims.LojaID == 0 {
		return 0, errors.New("state inválido ou expirado — tenta conectar de novo")
	}

	tok, err := s.trocarToken(ctx, map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": s.redirectURI(),
	})
	if err != nil {
		return 0, err
	}

	expiraEm := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	userIDStr := strconv.FormatInt(tok.UserID, 10)
	if err := s.lojaRepo.AtualizarMercadoPago(claims.LojaID, tok.AccessToken, tok.RefreshToken, userIDStr, expiraEm); err != nil {
		return 0, fmt.Errorf("salvando conexão com o Mercado Pago: %w", err)
	}

	return claims.LojaID, nil
}

// RenovarToken troca o refresh_token de uma loja específica por um par
// novo de access/refresh token — o Mercado Pago rotaciona o refresh_token
// a cada troca, então o valor anterior deixa de funcionar e precisa ser
// substituído também.
func (s *MercadoPagoService) RenovarToken(ctx context.Context, loja *domain.Loja) error {
	if loja.MercadoPagoRefreshToken == "" {
		return errors.New("loja sem refresh_token do Mercado Pago salvo")
	}

	tok, err := s.trocarToken(ctx, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": loja.MercadoPagoRefreshToken,
	})
	if err != nil {
		return err
	}

	expiraEm := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	userIDStr := strconv.FormatInt(tok.UserID, 10)
	if userIDStr == "0" {
		// Algumas respostas de refresh não repetem o user_id — mantém o
		// que a loja já tinha salvo em vez de sobrescrever com vazio.
		userIDStr = loja.MercadoPagoUserID
	}
	return s.lojaRepo.AtualizarMercadoPago(loja.ID, tok.AccessToken, tok.RefreshToken, userIDStr, expiraEm)
}

// RenovarTokensExpirando roda periodicamente (ver rota protegida por
// CronSecret em main.go, mesmo padrão do /relatorio/semanal) e renova o
// token de toda loja que vence nos próximos 15 dias — a antecedência
// evita que uma renovação atrasada (cron fora do ar por um dia, por
// exemplo) derrube o recebimento de pagamento de alguma loja.
func (s *MercadoPagoService) RenovarTokensExpirando(ctx context.Context) (renovadas int, erros []string) {
	limite := time.Now().Add(15 * 24 * time.Hour)
	lojas, err := s.lojaRepo.ListarComMercadoPagoExpirandoAte(limite)
	if err != nil {
		return 0, []string{fmt.Sprintf("erro listando lojas: %v", err)}
	}

	for i := range lojas {
		loja := &lojas[i]
		if err := s.RenovarToken(ctx, loja); err != nil {
			erros = append(erros, fmt.Sprintf("loja %d: %v", loja.ID, err))
			log.Printf("falha ao renovar token Mercado Pago da loja %d: %v", loja.ID, err)
			continue
		}
		renovadas++
	}
	return renovadas, erros
}

func (s *MercadoPagoService) trocarToken(ctx context.Context, extra map[string]string) (*tokenResponse, error) {
	corpo := map[string]string{
		"client_id":     s.clientID,
		"client_secret": s.clientSecret,
	}
	for k, v := range extra {
		corpo[k] = v
	}

	payload, err := json.Marshal(corpo)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mercadopago.com/oauth/token", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando /oauth/token do Mercado Pago: %w", err)
	}
	defer resp.Body.Close()

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("lendo resposta do Mercado Pago: %w", err)
	}

	if resp.StatusCode >= 300 {
		msg := tok.ErrorMessage
		if msg == "" {
			msg = tok.Error
		}
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Mercado Pago recusou a troca de token: %s", msg)
	}

	return &tok, nil
}

// faixaComissao é um degrau da comissão escalonada: a fatia do GMV mensal
// entre o teto da faixa anterior e `ateGMV` paga `percentual`. ateGMV = 0
// marca a última faixa (sem teto).
type faixaComissao struct {
	ateGMV     float64
	percentual float64
}

// faixasComissaoPorPlano implementa a comissão escalonada por faixa de GMV
// mensal (ver docs/plano-melhorias-drenux.md, Fase 7.2, e
// docs/drenux-planos-comissoes-definido.md § 2) — cada faixa se aplica só
// à fatia do GMV do mês que cai dentro dela, igual uma tabela progressiva.
// Scale é flat (uma faixa só, sem teto). Start não entra aqui: não tem
// split de pagamento, logo não gera comissão nenhuma.
var faixasComissaoPorPlano = map[string][]faixaComissao{
	"basic": {{5000, 2.4}, {20000, 1.5}, {0, 1.3}},
	"pro":   {{5000, 1.8}, {20000, 1.2}, {0, 1.05}},
	"scale": {{0, 1.1}},
}

// calcularComissaoEscalonada devolve a comissão da Drenux sobre um pedido,
// considerando o GMV já acumulado nesse mês (gmvAntes, sem esse pedido)
// pra saber em qual faixa (ou faixas, se o valor do pedido cruzar um teto)
// ele cai. Substitui o percentual fixo por plano usado até a Fase 7.
func calcularComissaoEscalonada(plano string, gmvAntes, valorPedido float64) float64 {
	faixas, ok := faixasComissaoPorPlano[plano]
	if !ok || valorPedido <= 0 {
		return 0
	}

	comissao := 0.0
	restante := valorPedido
	pisoFaixa := 0.0
	for _, faixa := range faixas {
		tetoFaixa := faixa.ateGMV
		if tetoFaixa == 0 {
			tetoFaixa = math.MaxFloat64
		}
		gmvJaNaFaixa := math.Max(0, math.Min(gmvAntes, tetoFaixa)-pisoFaixa)
		disponivelNaFaixa := math.Max(0, (tetoFaixa-pisoFaixa)-gmvJaNaFaixa)
		fatiaNestaFaixa := math.Min(restante, disponivelNaFaixa)

		comissao += fatiaNestaFaixa * faixa.percentual / 100
		restante -= fatiaNestaFaixa
		pisoFaixa = tetoFaixa

		if restante <= 0 {
			break
		}
	}

	return math.Round(comissao*100) / 100
}

// CriarCheckout monta uma preference de pagamento no Mercado Pago pra um
// pedido, usando o access_token da PRÓPRIA loja (não um token da
// plataforma) — é isso que faz o dinheiro cair direto na conta dela, com
// a Drenux retendo a comissão via marketplace_fee.
func (s *MercadoPagoService) CriarCheckout(ctx context.Context, pedidoID uint) (string, error) {
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
	if loja.MercadoPagoAccessToken == "" {
		s.avisarPagamentoNaoConfigurado(loja)
		return "", errors.New("essa loja ainda não conectou uma conta de pagamento")
	}

	successURL := fmt.Sprintf("%s/%s?pago=1", s.frontendURL, loja.Slug)
	outrasURL := fmt.Sprintf("%s/%s", s.frontendURL, loja.Slug)

	itens := make([]map[string]interface{}, 0, len(pedido.Itens))
	totalCobrado := 0.0
	for _, item := range pedido.Itens {
		precoUnit := math.Round(item.PrecoUnit*100) / 100
		totalCobrado += precoUnit * float64(item.Quantidade)
		itens = append(itens, map[string]interface{}{
			"title":       item.ProdutoNome,
			"quantity":    item.Quantidade,
			"unit_price":  precoUnit,
			"currency_id": "BRL",
		})
	}
	if pedido.TaxaEntrega > 0 {
		freteArredondado := math.Round(pedido.TaxaEntrega*100) / 100
		totalCobrado += freteArredondado
		itens = append(itens, map[string]interface{}{
			"title":       "Taxa de entrega",
			"quantity":    1,
			"unit_price":  freteArredondado,
			"currency_id": "BRL",
		})
	}
	totalCobrado = math.Round(totalCobrado*100) / 100

	// Comissão não incide sobre a taxa de entrega — pedido.Total já inclui
	// o frete (ver PedidoService.CriarPorSlug), então a base de cálculo
	// aqui é só o subtotal dos itens.
	baseComissao := pedido.Total - pedido.TaxaEntrega

	gmvAntes, err := s.pedidoRepo.SomarGMVMesAtual(loja.ID)
	if err != nil {
		return "", fmt.Errorf("calculando GMV do mês da loja %d: %w", loja.ID, err)
	}
	marketplaceFee := calcularComissaoEscalonada(loja.Plano, gmvAntes, baseComissao)

	// Salvaguarda: o Mercado Pago rejeita (invalid_marketplace_fee) se a
	// comissão for maior que o total da preference. Não deveria acontecer
	// com as faixas atuais (todas bem abaixo de 100%), mas mantém a Drenux
	// no máximo com o total do pedido como comissão em vez de travar o
	// checkout, caso as faixas mudem no futuro.
	if marketplaceFee > totalCobrado {
		marketplaceFee = totalCobrado
	}

	corpo := map[string]interface{}{
		"items":              itens,
		"marketplace_fee":    marketplaceFee,
		"external_reference": strconv.FormatUint(uint64(pedido.ID), 10),
		"notification_url":   s.apiPublicURL + "/webhooks/mercadopago",
		"back_urls": map[string]string{
			"success": successURL,
			"failure": outrasURL,
			"pending": outrasURL,
		},
		"auto_return": "approved",
	}

	preference, err := s.chamarComTokenDaLoja(ctx, loja.MercadoPagoAccessToken, http.MethodPost, "https://api.mercadopago.com/checkout/preferences", corpo)
	if err != nil {
		return "", err
	}

	// O Access Token devolvido pelo OAuth já indica o ambiente pelo prefixo
	// ("TEST-..." pra Seller Test User, "APP_USR-..." pra conta de
	// produção) — se a loja estiver conectada com uma conta de teste,
	// precisamos do sandbox_init_point, não do init_point de produção. Os
	// dois ambientes não se misturam: vendedor de produção com link de
	// sandbox falha, e vendedor de teste com link de produção é bloqueado
	// pelo próprio Mercado Pago ("uma das partes é de teste"). Não dá pra
	// remover essa checagem achando redundante — os dois campos vêm
	// preenchidos na mesma resposta e o Mercado Pago não escolhe por nós.
	campoLink := "init_point"
	ambiente := "produção"
	if strings.HasPrefix(loja.MercadoPagoAccessToken, "TEST-") {
		campoLink = "sandbox_init_point"
		ambiente = "sandbox"
	}
	log.Printf("pedido %d: preferência Mercado Pago criada (ambiente=%s, marketplace_fee=%.2f)", pedido.ID, ambiente, marketplaceFee)

	initPoint, _ := preference[campoLink].(string)
	if initPoint == "" {
		return "", errors.New("Mercado Pago não devolveu o link de pagamento")
	}
	if preferenceID, ok := preference["id"].(string); ok {
		if err := s.pedidoRepo.AtualizarMercadoPagoPreferenceID(pedido.ID, preferenceID); err != nil {
			log.Printf("aviso: não foi possível salvar mercado_pago_preference_id do pedido %d: %v", pedido.ID, err)
		}
	}

	return initPoint, nil
}

// avisoPagamentoNaoConfiguradoIntervalo evita mandar o aviso de novo a
// cada tentativa de checkout enquanto a loja não resolve — se o dono
// receber a primeira mensagem e demorar pra conectar, não faz sentido
// tomar uma mensagem por cliente que tentar pagar nesse meio tempo.
const avisoPagamentoNaoConfiguradoIntervalo = 12 * time.Hour

// avisarPagamentoNaoConfigurado dispara um WhatsApp pro dono quando um
// cliente esbarra no checkout porque a loja ainda não conectou o Mercado
// Pago — sem isso, o lojista só ia descobrir que está perdendo venda se
// um cliente reclamasse diretamente. Roda em goroutine (não atrasa o erro
// devolvido pro cliente) e é silenciosa em qualquer falha — um aviso
// perdido não pode derrubar o fluxo de checkout.
func (s *MercadoPagoService) avisarPagamentoNaoConfigurado(loja *domain.Loja) {
	if s.notificationSender == nil || loja.WhatsappNumero == "" {
		return
	}
	if loja.AvisoPagamentoNaoConfiguradoEm != nil && time.Since(*loja.AvisoPagamentoNaoConfiguradoEm) < avisoPagamentoNaoConfiguradoIntervalo {
		return
	}

	agora := time.Now()
	if err := s.lojaRepo.AtualizarAvisoPagamentoNaoConfigurado(loja.ID, agora); err != nil {
		log.Printf("aviso: não foi possível registrar o aviso de pagamento não configurado da loja %d: %v", loja.ID, err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		texto := fmt.Sprintf(
			"⚠️ Pagamento não configurado — %s\n\nUm cliente tentou finalizar um pedido mas você ainda não conectou sua conta do Mercado Pago. Enquanto isso não for feito, ninguém consegue pagar e você está perdendo venda.\n\nConecta agora em Configurações no painel da Drenux.",
			loja.Nome,
		)
		if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, texto); err != nil {
			log.Printf("falha ao enviar aviso de pagamento não configurado da loja %d: %v", loja.ID, err)
		}
	}()
}

func (s *MercadoPagoService) chamarComTokenDaLoja(ctx context.Context, accessToken, metodo, url string, corpo map[string]interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(corpo)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, metodo, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando API do Mercado Pago: %w", err)
	}
	defer resp.Body.Close()

	corpoResposta, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var resultado map[string]interface{}
	if err := json.Unmarshal(corpoResposta, &resultado); err != nil {
		return nil, fmt.Errorf("lendo resposta do Mercado Pago: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mercado Pago recusou a requisição (status %d): %s", resp.StatusCode, string(corpoResposta))
	}

	return resultado, nil
}

// obterTokenAplicacao pega um access_token de aplicação (grant_type
// client_credentials) — usado pra buscar detalhes de um pagamento no
// webhook, antes de sabermos a qual loja ele pertence (não dá pra usar o
// access_token de uma loja específica nesse momento, já que é isso que
// ainda estamos descobrindo).
func (s *MercadoPagoService) obterTokenAplicacao(ctx context.Context) (string, error) {
	tok, err := s.trocarToken(ctx, map[string]string{"grant_type": "client_credentials"})
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// ValidarAssinaturaWebhook confere o header x-signature enviado pelo
// Mercado Pago — sem isso, qualquer um que descobrisse a URL do webhook
// poderia forjar "pagamento aprovado" pra qualquer pedido. Segue o
// algoritmo documentado pelo Mercado Pago: HMAC-SHA256 sobre o manifest
// "id:{data_id};request-id:{x_request_id};ts:{ts};" usando o webhook
// secret configurado no painel do Mercado Pago.
//
// IMPORTANTE: isso ainda não foi validado contra uma notificação real do
// Mercado Pago (sandbox) — testar de ponta a ponta antes de confiar em
// produção (ver aviso no docs/plano-melhorias-drenux.md).
func (s *MercadoPagoService) ValidarAssinaturaWebhook(signatureHeader, requestID, dataID string) error {
	if s.webhookSecret == "" {
		return errors.New("MERCADOPAGO_WEBHOOK_SECRET não configurada")
	}
	if signatureHeader == "" {
		return errors.New("notificação sem header x-signature")
	}

	var ts, v1 string
	for _, parte := range strings.Split(signatureHeader, ",") {
		partes := strings.SplitN(strings.TrimSpace(parte), "=", 2)
		if len(partes) != 2 {
			continue
		}
		switch partes[0] {
		case "ts":
			ts = partes[1]
		case "v1":
			v1 = partes[1]
		}
	}
	if ts == "" || v1 == "" {
		return errors.New("header x-signature em formato inesperado")
	}

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", strings.ToLower(dataID), requestID, ts)

	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(manifest))
	esperado := hex.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(esperado), []byte(v1)) != 1 {
		return errors.New("assinatura do webhook não confere")
	}
	return nil
}

// ProcessarNotificacaoPagamento busca os detalhes do pagamento no
// Mercado Pago e, se estiver aprovado, marca o pedido correspondente
// como pago e dispara o pós-processamento (estoque + notificações — ver
// PosPagamentoService). Repasse de comissão de afiliado ainda não existe
// pra esse processador (Fase 5.5 em aberto).
// ProcessarMerchantOrder trata uma notificação "topic=merchant_order" —
// hoje o tipo de notificação que o Mercado Pago realmente manda pra essa
// integração (Checkout Pro via preferência), diferente do que a
// documentação de webhooks "type=payment" sozinha sugere. Uma
// merchant_order agrupa uma ou mais tentativas de pagamento contra a
// mesma preferência; busca os pagamentos de dentro dela e delega cada um
// pro mesmo processamento de sempre (ProcessarNotificacaoPagamento), que
// já é idempotente (pedido.Status == pago vira no-op).
func (s *MercadoPagoService) ProcessarMerchantOrder(ctx context.Context, merchantOrderID string) error {
	tokenApp, err := s.obterTokenAplicacao(ctx)
	if err != nil {
		return fmt.Errorf("obtendo token de aplicação: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.mercadopago.com/merchant_orders/"+merchantOrderID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tokenApp)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("buscando merchant_order %s: %w", merchantOrderID, err)
	}
	defer resp.Body.Close()

	var order struct {
		ExternalReference string `json:"external_reference"`
		Payments          []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"payments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return fmt.Errorf("lendo dados da merchant_order %s: %w", merchantOrderID, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Mercado Pago recusou consulta da merchant_order %s (status %d)", merchantOrderID, resp.StatusCode)
	}

	log.Printf("merchant_order %s: external_reference=%q, %d pagamento(s) associado(s)", merchantOrderID, order.ExternalReference, len(order.Payments))

	if len(order.Payments) == 0 {
		return nil // preferência ainda sem nenhuma tentativa de pagamento — nada a fazer
	}

	var ultimoErro error
	for _, p := range order.Payments {
		paymentID := strconv.FormatInt(p.ID, 10)
		if err := s.ProcessarNotificacaoPagamento(ctx, paymentID); err != nil {
			log.Printf("erro processando pagamento %s (via merchant_order %s): %v", paymentID, merchantOrderID, err)
			ultimoErro = err
		}
	}
	return ultimoErro
}

func (s *MercadoPagoService) ProcessarNotificacaoPagamento(ctx context.Context, paymentID string) error {
	tokenApp, err := s.obterTokenAplicacao(ctx)
	if err != nil {
		return fmt.Errorf("obtendo token de aplicação: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.mercadopago.com/v1/payments/"+paymentID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tokenApp)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("buscando pagamento %s: %w", paymentID, err)
	}
	defer resp.Body.Close()

	var pagamento struct {
		Status            string `json:"status"`
		ExternalReference string `json:"external_reference"`
		CollectorID       int64  `json:"collector_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pagamento); err != nil {
		return fmt.Errorf("lendo dados do pagamento %s: %w", paymentID, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Mercado Pago recusou consulta do pagamento %s (status %d)", paymentID, resp.StatusCode)
	}

	log.Printf("pagamento %s: status=%q external_reference=%q collector_id=%d", paymentID, pagamento.Status, pagamento.ExternalReference, pagamento.CollectorID)

	if pagamento.Status != "approved" {
		return nil // ainda não aprovado (pending, rejected, etc.) — nada a fazer agora
	}
	if pagamento.ExternalReference == "" {
		return errors.New("pagamento aprovado sem external_reference — não dá pra saber qual pedido é")
	}

	pedidoID, err := strconv.ParseUint(pagamento.ExternalReference, 10, 64)
	if err != nil {
		return fmt.Errorf("external_reference inválido (%q): %w", pagamento.ExternalReference, err)
	}

	pedido, err := s.pedidoRepo.BuscarPorID(uint(pedidoID))
	if err != nil {
		return fmt.Errorf("pedido %d do webhook não encontrado: %w", pedidoID, err)
	}

	// Confere que o pagamento realmente veio da conta MP conectada à
	// mesma loja do pedido — defesa contra um external_reference forjado
	// apontando pro pedido de outra loja.
	loja, err := s.lojaRepo.BuscarPorID(pedido.LojaID)
	if err != nil {
		return fmt.Errorf("loja do pedido %d não encontrada: %w", pedidoID, err)
	}
	collectorIDStr := strconv.FormatInt(pagamento.CollectorID, 10)
	if loja.MercadoPagoUserID != "" && loja.MercadoPagoUserID != collectorIDStr {
		return fmt.Errorf("pagamento %s: collector_id %s não bate com a loja do pedido %d (esperado %s)", paymentID, collectorIDStr, pedidoID, loja.MercadoPagoUserID)
	}

	if pedido.Status == domain.StatusPago {
		log.Printf("pedido %d já estava marcado como pago — notificação repetida do Mercado Pago, ignorando", pedidoID)
		return nil // já processado — webhook pode repetir a mesma notificação
	}

	// GMV do mês tem que ser lido ANTES de marcar esse pedido como pago —
	// SomarGMVMesAtual só soma pedidos com status='pago', então rodar essa
	// consulta depois do AtualizarStatus abaixo contaria esse próprio
	// pedido como "GMV de antes dele", desalinhando a faixa de comissão do
	// afiliado (calcularComissaoEscalonada, mesma base do marketplace_fee).
	gmvAntes, err := s.pedidoRepo.SomarGMVMesAtual(loja.ID)
	if err != nil {
		log.Printf("aviso: não foi possível calcular GMV do mês da loja %d pro repasse de afiliado: %v", loja.ID, err)
	}

	if err := s.pedidoRepo.AtualizarStatus(uint(pedidoID), domain.StatusPago); err != nil {
		return fmt.Errorf("atualizando status do pedido %d: %w", pedidoID, err)
	}
	log.Printf("pedido %d marcado como pago via Mercado Pago (payment_id=%s)", pedidoID, paymentID)

	// O split do Mercado Pago hoje é só 1:1 (Loja + Drenux) — não dá pra
	// repassar a comissão do afiliado na mesma transação como a Stripe
	// fazia via Transfer. Decisão fechada na Fase 5.5 (ver
	// docs/plano-melhorias-drenux.md): registrar o valor devido e repassar
	// via Pix manual depois, controlado pela tela /drenux/afiliados.
	if loja.AfiliadoID != nil {
		// Mesma regra do marketplace_fee: comissão não incide sobre frete
		// (ver CriarCheckout, baseComissao). O percentual em si é o do
		// próprio afiliado (RegistrarPendente busca e aplica), não um
		// valor fixo global.
		baseComissao := pedido.Total - pedido.TaxaEntrega
		if err := s.repasseService.RegistrarPendente(*loja.AfiliadoID, pedido.ID, loja.ID, baseComissao, loja.Plano, gmvAntes); err != nil {
			log.Printf("aviso: não foi possível registrar repasse de afiliado do pedido %d: %v", pedidoID, err)
		} else {
			log.Printf("pedido %d: repasse registrado como pendente pro afiliado %d", pedidoID, *loja.AfiliadoID)
		}

		// Bônus de ativação (Fase 7.5) — checado em todo pedido pago, mas só
		// gera lançamento uma vez por loja (ExisteBonusAtivacao) e só depois
		// de atingir o mínimo de pedidos dentro do prazo. Erro aqui não deve
		// derrubar o processamento do pagamento em si.
		if err := s.repasseService.VerificarBonusAtivacao(loja); err != nil {
			log.Printf("aviso: não foi possível verificar bônus de ativação da loja %d: %v", loja.ID, err)
		}
	}

	log.Printf("pedido %d: disparando pós-pagamento (estoque + notificação)", pedidoID)
	go s.posPagamento.ProcessarPedidoPago(uint(pedidoID))

	return nil
}

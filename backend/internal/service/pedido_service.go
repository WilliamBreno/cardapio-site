package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"
	_ "time/tzdata"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type ItemPedidoInput struct {
	ProdutoID  uint
	VariacaoID *uint
	Quantidade int

	// SugestaoProdutoID marca que esse item foi adicionado através da
	// Sugestão Inteligente do carrinho (Fase 6) — o desconto configurado
	// só é aplicado se, ao validar, o produto de origem daquela sugestão
	// realmente estiver no pedido (item avulso ou componente de combo).
	// Se não bater, o item entra pelo preço cheio, sem erro — proteção
	// contra alguém forjar esse campo pra ganhar desconto indevido.
	SugestaoProdutoID *uint
}

// ComboItemPedidoInput é a variação escolhida (se houver) pra um item
// específico do combo — referenciado pelo ComboItemID (não ProdutoID,
// pra não dar ambiguidade se o mesmo produto aparecer duas vezes no
// combo).
type ComboItemPedidoInput struct {
	ComboItemID uint
	VariacaoID  *uint
}

type ComboPedidoInput struct {
	ComboID    uint
	Quantidade int
	Itens      []ComboItemPedidoInput
}

type PedidoInput struct {
	ClienteNome        string
	ClienteTelefone    string
	DataRetirada       time.Time
	ModoEntrega        string
	EnderecoEntrega    string
	EnderecoEntregaGeo EnderecoEstruturado
	CupomCodigo        string
	Itens              []ItemPedidoInput
	Combos             []ComboPedidoInput
}

type PedidoService struct {
	db                 *gorm.DB
	lojaRepo           *repository.LojaRepository
	pedidoRepo         *repository.PedidoRepository
	cupomRepo          *repository.CupomRepository
	comboRepo          *repository.ComboRepository
	sugestaoRepo       *repository.SugestaoProdutoRepository
	distanciaService   *DistanciaService
	notificationSender notification.NotificationSender
}

func NewPedidoService(db *gorm.DB, distanciaService *DistanciaService, notificationSender notification.NotificationSender) *PedidoService {
	return &PedidoService{
		db:                 db,
		lojaRepo:           repository.NewLojaRepository(db),
		pedidoRepo:         repository.NewPedidoRepository(db),
		cupomRepo:          repository.NewCupomRepository(db),
		comboRepo:          repository.NewComboRepository(db),
		sugestaoRepo:       repository.NewSugestaoProdutoRepository(db),
		distanciaService:   distanciaService,
		notificationSender: notificationSender,
	}
}

// gerarCodigoConfirmacao devolve um código numérico de 4 dígitos
// (0000-9999, com zero à esquerda quando necessário), usado pra
// confirmar entrega — fácil do cliente ler em voz alta e do entregador
// digitar. crypto/rand em vez de math/rand porque, apesar de não ser um
// segredo forte (só 10 mil combinações), ainda é o mecanismo que evita
// marcar entrega sem confirmação nenhuma, então não custa usar a fonte
// de aleatoriedade correta.
func gerarCodigoConfirmacao() string {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		// Praticamente nunca acontece (fonte de aleatoriedade do SO
		// indisponível) — cai num código fixo em vez de travar a
		// criação do pedido inteiro por causa disso.
		return "0000"
	}
	return fmt.Sprintf("%04d", n.Int64())
}

// gerarTokenEntregador devolve um token de 32 caracteres hexadecimais
// (16 bytes de crypto/rand) — diferente do CodigoConfirmacao (4 dígitos,
// pensado pra ser lido/digitado por humano), esse token vai numa URL
// pública compartilhada com quem for entregar, e a única coisa que
// impede qualquer um de abrir a tela de gerenciar entrega de um pedido
// alheio é ele não ser adivinhável — por isso precisa de espaço de
// busca grande de verdade, não só "difícil de adivinhar de cabeça".
func gerarTokenEntregador() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Praticamente nunca acontece (mesma situação de
		// gerarCodigoConfirmacao) — cai num token fixo em vez de travar a
		// criação do pedido; na pior hipótese, esse pedido específico fica
		// sem link de entregador utilizável, não é vazamento de outro
		// pedido (a chance de colisão real é desprezível pro handler, que
		// também confirma loja_id).
		return "0000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

func (s *PedidoService) CriarPorSlug(slug string, input PedidoInput) (*domain.Pedido, error) {
	loja, err := s.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}

	if len(input.Itens) == 0 && len(input.Combos) == 0 {
		return nil, errors.New("o pedido precisa ter pelo menos um item")
	}

	// Valida modo de entrega
	modoEntrega := domain.ModoEntregaRetirada
	switch input.ModoEntrega {
	case string(domain.ModoEntregaEntrega):
		if !loja.AceitaEntrega {
			return nil, errors.New("essa loja não aceita entrega em domicílio")
		}
		if input.EnderecoEntrega == "" {
			return nil, errors.New("endereço de entrega é obrigatório")
		}
		modoEntrega = domain.ModoEntregaEntrega

	case string(domain.ModoEntregaGuardar):
		if !loja.AceitaGuardarEntregar {
			return nil, errors.New("essa loja não aceita guardar itens pra entregar depois")
		}
		modoEntrega = domain.ModoEntregaGuardar

	default:
		if !loja.AceitaRetirada {
			return nil, errors.New("essa loja não aceita retirada — só entrega em domicílio")
		}
	}

	// Combo não tem TipoProduto/PesoGramas por componente (ver
	// domain.PedidoComboItem) — estender o fluxo de "guardar e entregar
	// depois" pra combo exigiria levar esses campos até lá também, então
	// por ora um combo só pode ser comprado pra retirada/entrega imediata.
	if modoEntrega == domain.ModoEntregaGuardar && len(input.Combos) > 0 {
		return nil, errors.New("combos não podem ser guardados pra entrega depois — peça os produtos avulsos pra isso")
	}

	// Validações da loja antes de aceitar o pedido
	if err := validarLojaAberta(loja); err != nil {
		return nil, err
	}
	if err := s.verificarLimiteStart(loja); err != nil {
		return nil, err
	}
	// Pedidos "guardar" não têm data de retirada — os itens ficam
	// guardados por tempo indeterminado até o cliente pedir a entrega.
	if modoEntrega != domain.ModoEntregaGuardar {
		if err := validarDataRetirada(input.DataRetirada, loja); err != nil {
			return nil, err
		}
	}

	// Calcula a taxa de entrega ANTES da transação — se for "por_km" e a
	// geocodificação falhar, queremos rejeitar o pedido com uma mensagem
	// clara, sem nem chegar a mexer no banco. Esse valor é a fonte de
	// verdade cobrada de verdade; qualquer cotação mostrada antes no
	// carrinho do cliente foi só uma prévia, nunca confiável sozinha.
	var taxaEntrega float64
	if modoEntrega == domain.ModoEntregaEntrega {
		switch loja.TaxaEntregaTipo {
		case "fixa":
			taxaEntrega = loja.TaxaEntregaValor

		case "por_km":
			if loja.Latitude == 0 && loja.Longitude == 0 {
				return nil, errors.New("essa loja ainda não configurou o endereço de origem pra calcular o frete")
			}
			if s.distanciaService == nil {
				return nil, errors.New("cálculo de frete indisponível no momento")
			}
			destino, err := s.distanciaService.GeocodificarEstruturado(input.EnderecoEntregaGeo)
			if err != nil {
				return nil, errors.New("não conseguimos localizar o endereço de entrega informado — confere se está completo")
			}
			origem := Coordenada{Latitude: loja.Latitude, Longitude: loja.Longitude}
			distancia := s.distanciaService.DistanciaKm(origem, *destino)
			taxaEntrega = CalcularTaxaPorKm(distancia, loja.TaxaEntregaBase, loja.TaxaEntregaPorKm)

		case "combinado":
			taxaEntrega = 0 // combinado fora do sistema, não entra no total cobrado agora
		}
	}

	pedido := domain.Pedido{
		LojaID:            loja.ID,
		ClienteNome:       input.ClienteNome,
		ClienteTelefone:   input.ClienteTelefone,
		DataRetirada:      input.DataRetirada,
		Status:            domain.StatusAguardandoPagamento,
		ModoEntrega:       modoEntrega,
		EnderecoEntrega:   input.EnderecoEntrega,
		TaxaEntrega:       taxaEntrega,
		CodigoConfirmacao: gerarCodigoConfirmacao(),
		TokenEntregador:   gerarTokenEntregador(),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		produtoRepo := repository.NewProdutoRepository(tx)
		pedidoRepo := repository.NewPedidoRepository(tx)
		variacaoRepo := repository.NewVariacaoRepository(tx)
		comboRepo := repository.NewComboRepository(tx)
		sugestaoRepo := repository.NewSugestaoProdutoRepository(tx)

		// Pré-carrega os combos pedidos (validando dono/disponibilidade já
		// aqui) pra montar o conjunto de produtos presentes no pedido ANTES
		// de processar os itens avulsos — a Sugestão Inteligente (abaixo)
		// considera componente de combo como "já no carrinho" também.
		combosMap := make(map[uint]*domain.Combo, len(input.Combos))
		produtosNoCarrinho := make(map[uint]bool)
		for _, comboInput := range input.Combos {
			if comboInput.Quantidade <= 0 {
				return fmt.Errorf("quantidade inválida pro combo %d", comboInput.ComboID)
			}
			combo, err := comboRepo.BuscarPorID(comboInput.ComboID)
			if err != nil {
				return fmt.Errorf("combo %d não encontrado", comboInput.ComboID)
			}
			if combo.LojaID != loja.ID {
				return fmt.Errorf("combo %d não pertence a essa loja", comboInput.ComboID)
			}
			if !combo.Disponivel {
				return fmt.Errorf("combo %q está indisponível no momento", combo.Nome)
			}
			combosMap[comboInput.ComboID] = combo
			for _, comboItem := range combo.Itens {
				produtosNoCarrinho[comboItem.ProdutoID] = true
			}
		}
		for _, itemInput := range input.Itens {
			produtosNoCarrinho[itemInput.ProdutoID] = true
		}

		var total float64
		itens := make([]domain.ItemPedido, 0, len(input.Itens))

		for _, itemInput := range input.Itens {
			if itemInput.Quantidade <= 0 {
				return fmt.Errorf("quantidade inválida pro produto %d", itemInput.ProdutoID)
			}

			produto, err := produtoRepo.BuscarPorID(itemInput.ProdutoID)
			if err != nil {
				return fmt.Errorf("produto %d não encontrado", itemInput.ProdutoID)
			}
			if produto.LojaID != loja.ID {
				return fmt.Errorf("produto %d não pertence a essa loja", itemInput.ProdutoID)
			}
			if !produto.Disponivel {
				return fmt.Errorf("produto %q está indisponível no momento", produto.Nome)
			}
			if modoEntrega == domain.ModoEntregaGuardar && produto.TipoProduto != domain.TipoProdutoMercadoria {
				return fmt.Errorf("produto %q é alimentício e não pode ser guardado pra entrega posterior", produto.Nome)
			}

			precoUnit := produto.Preco
			variacaoNome := ""

			// Escolher uma variação é opcional pro cliente, mesmo quando o
			// produto tem variações cadastradas — sem variação escolhida,
			// usa o preço/estoque base do próprio produto (mesmo caminho
			// de quem nunca teve variação nenhuma).
			if itemInput.VariacaoID != nil {
				variacao, err := variacaoRepo.BuscarPorID(*itemInput.VariacaoID)
				if err != nil || variacao.ProdutoID != produto.ID {
					return fmt.Errorf("variação inválida pro produto %q", produto.Nome)
				}
				if !variacao.Disponivel {
					return fmt.Errorf("variação %q do produto %q está indisponível", variacao.Nome, produto.Nome)
				}
				if variacao.EstoqueAtual != nil && *variacao.EstoqueAtual < itemInput.Quantidade {
					if *variacao.EstoqueAtual == 0 {
						return fmt.Errorf("variação %q do produto %q está esgotada", variacao.Nome, produto.Nome)
					}
					return fmt.Errorf("variação %q tem apenas %d unidade(s) disponível(is)", variacao.Nome, *variacao.EstoqueAtual)
				}
				if variacao.ModoPreco == domain.ModoPrecoAbsoluto {
					precoUnit = variacao.PrecoAdicional
				} else {
					precoUnit += variacao.PrecoAdicional
				}
				variacaoNome = variacao.Nome
			} else {
				if produto.EstoqueAtual != nil && *produto.EstoqueAtual < itemInput.Quantidade {
					if *produto.EstoqueAtual == 0 {
						return fmt.Errorf("produto %q está esgotado", produto.Nome)
					}
					return fmt.Errorf("produto %q tem apenas %d unidade(s) disponível(is)", produto.Nome, *produto.EstoqueAtual)
				}
			}

			// Sugestão Inteligente: só aplica o desconto do vínculo se ele
			// existir de verdade, pertencer a essa loja, apontar pra esse
			// produto como sugerido, e o produto de origem realmente estar
			// no pedido (avulso ou componente de combo) — sem essas três
			// checagens, dava pra forjar o campo e ganhar desconto indevido.
			// Se não bater, ignora silenciosamente e cobra o preço cheio,
			// sem travar o pedido por isso.
			var sugestaoAplicadaID *uint
			if itemInput.SugestaoProdutoID != nil {
				sugestao, err := sugestaoRepo.BuscarPorID(*itemInput.SugestaoProdutoID)
				if err == nil &&
					sugestao.LojaID == loja.ID &&
					sugestao.ProdutoSugeridoID == produto.ID &&
					produtosNoCarrinho[sugestao.ProdutoOrigemID] {
					precoUnit = sugestao.PrecoComDesconto(precoUnit)
					sugestaoAplicadaID = itemInput.SugestaoProdutoID
				}
			}

			subtotal := precoUnit * float64(itemInput.Quantidade)
			total += subtotal

			pesoGramas := 0
			if produto.PesoGramas != nil {
				pesoGramas = *produto.PesoGramas
			}

			itens = append(itens, domain.ItemPedido{
				ProdutoID:         produto.ID,
				ProdutoNome:       produto.Nome,
				Quantidade:        itemInput.Quantidade,
				PrecoUnit:         precoUnit,
				VariacaoID:        itemInput.VariacaoID,
				VariacaoNome:      variacaoNome,
				TipoProduto:       produto.TipoProduto,
				PesoGramas:        pesoGramas,
				SugestaoProdutoID: sugestaoAplicadaID,
			})
		}

		// Combos — pacote fixo, com o preço final definido diretamente pelo
		// lojista (não recalculado a partir da soma dos itens). O cliente
		// escolhe a variação de cada componente igual comprando avulso; o
		// estoque só é CHECADO aqui, nunca decrementado na criação do
		// pedido — isso só acontece depois que o pagamento é confirmado
		// (ver PosPagamentoService), mesmo padrão já usado pros itens
		// avulsos acima.
		combos := make([]domain.PedidoCombo, 0, len(input.Combos))
		for _, comboInput := range input.Combos {
			combo := combosMap[comboInput.ComboID]

			selecoes := make(map[uint]ComboItemPedidoInput, len(comboInput.Itens))
			for _, selecao := range comboInput.Itens {
				selecoes[selecao.ComboItemID] = selecao
			}

			pedidoComboItens := make([]domain.PedidoComboItem, 0, len(combo.Itens))
			for _, comboItem := range combo.Itens {
				produto, err := produtoRepo.BuscarPorID(comboItem.ProdutoID)
				if err != nil {
					return fmt.Errorf("produto do combo %q não encontrado", combo.Nome)
				}
				if !produto.Disponivel {
					return fmt.Errorf("produto %q do combo %q está indisponível no momento", produto.Nome, combo.Nome)
				}

				qtdNecessaria := comboItem.Quantidade * comboInput.Quantidade
				var variacaoID *uint
				variacaoNome := ""

				if selecao, ok := selecoes[comboItem.ID]; ok && selecao.VariacaoID != nil {
					variacao, err := variacaoRepo.BuscarPorID(*selecao.VariacaoID)
					if err != nil || variacao.ProdutoID != produto.ID {
						return fmt.Errorf("variação inválida pro produto %q do combo %q", produto.Nome, combo.Nome)
					}
					if !variacao.Disponivel {
						return fmt.Errorf("variação %q do produto %q está indisponível", variacao.Nome, produto.Nome)
					}
					if variacao.EstoqueAtual != nil && *variacao.EstoqueAtual < qtdNecessaria {
						return fmt.Errorf("variação %q do produto %q não tem estoque suficiente pro combo %q", variacao.Nome, produto.Nome, combo.Nome)
					}
					variacaoID = selecao.VariacaoID
					variacaoNome = variacao.Nome
				} else if produto.EstoqueAtual != nil && *produto.EstoqueAtual < qtdNecessaria {
					return fmt.Errorf("produto %q não tem estoque suficiente pro combo %q", produto.Nome, combo.Nome)
				}

				pedidoComboItens = append(pedidoComboItens, domain.PedidoComboItem{
					ProdutoID:    produto.ID,
					ProdutoNome:  produto.Nome,
					VariacaoID:   variacaoID,
					VariacaoNome: variacaoNome,
					Quantidade:   comboItem.Quantidade,
				})
			}

			total += combo.Preco * float64(comboInput.Quantidade)

			combos = append(combos, domain.PedidoCombo{
				ComboID:    combo.ID,
				Nome:       combo.Nome,
				FotoURL:    combo.FotoURL,
				Preco:      combo.Preco,
				Quantidade: comboInput.Quantidade,
				Itens:      pedidoComboItens,
			})
		}

		pedido.Total = total + taxaEntrega

		// Valida valor mínimo de pedido (sobre o subtotal, sem taxa de entrega)
		if loja.ValorMinimoPedido > 0 && total < loja.ValorMinimoPedido {
			return fmt.Errorf(
				"pedido mínimo de R$ %.2f — seu carrinho está com R$ %.2f",
				loja.ValorMinimoPedido, total,
			)
		}

		// Aplica cupom de desconto, se informado
		if input.CupomCodigo != "" {
			cupomRepo := repository.NewCupomRepository(tx)
			cupom, err := cupomRepo.BuscarPorCodigo(input.CupomCodigo, loja.ID)
			if err != nil {
				return errors.New("cupom não encontrado")
			}
			if !cupom.Ativo {
				return errors.New("esse cupom não está mais ativo")
			}
			if cupom.Validade != nil && time.Now().After(*cupom.Validade) {
				return errors.New("esse cupom expirou")
			}
			if cupom.UsoMaximo != nil && cupom.UsoAtual >= *cupom.UsoMaximo {
				return errors.New("esse cupom atingiu o limite de usos")
			}
			if cupom.ValorMinimoPedido > 0 && total < cupom.ValorMinimoPedido {
				return fmt.Errorf("pedido mínimo de R$ %.2f pra usar esse cupom", cupom.ValorMinimoPedido)
			}

			var desconto float64
			if cupom.Tipo == domain.TipoCupomPercentual {
				desconto = total * cupom.Valor / 100
			} else {
				desconto = cupom.Valor
			}
			if desconto > total {
				desconto = total
			}

			pedido.CupomCodigo = cupom.Codigo
			pedido.Desconto = desconto
			pedido.Total -= desconto
			if pedido.Total < 0 {
				pedido.Total = 0
			}

			if err := cupomRepo.IncrementarUso(cupom.ID); err != nil {
				return fmt.Errorf("erro ao registrar uso do cupom: %w", err)
			}
		}

		pedido.Itens = itens
		pedido.Combos = combos

		// Aviso preventivo: só faz sentido em modo "guardar" — é o único
		// modo em que a entrega pode acabar fora da região da loja depois
		// (ver SolicitacaoEntrega.PesoPendente, o aviso definitivo). Não
		// bloqueia o pedido, só sinaliza pro lojista completar o peso.
		if modoEntrega == domain.ModoEntregaGuardar {
			pedido.PesoPendente = pesoPendenteEmItens(itens)
		}

		if err := pedidoRepo.Criar(&pedido); err != nil {
			return fmt.Errorf("criando pedido: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Geocodifica o destino em segundo plano, fora da transação — só pra
	// modo "entrega" (retirada/guardar não têm entregador). Roda numa
	// goroutine pra não atrasar o checkout do cliente com a espera do
	// Nominatim (rate-limit de ~1,1s por chamada, ver DistanciaService) —
	// mesmo padrão já usado nesse arquivo pra notificação de WhatsApp
	// (PedidoHandler.notificarSaiuParaEntrega). Diferente do cálculo de
	// frete "por_km" (que já geocodifica, mas só nesse tipo de taxa e sem
	// persistir o resultado), aqui a geocodificação roda pra qualquer
	// tipo de taxa de entrega — o mapa do entregador precisa do pino de
	// destino independente de como o frete é cobrado.
	if modoEntrega == domain.ModoEntregaEntrega {
		s.geocodificarDestinoEmSegundoPlano(pedido.ID, input.EnderecoEntregaGeo)
	}

	return &pedido, nil
}

// geocodificarDestinoEmSegundoPlano busca a coordenada do endereço de
// entrega e grava em Pedido.DestinoLatitude/DestinoLongitude. Falha
// silenciosa (só loga) — a tela do entregador cai pro endereço em texto
// se a geocodificação não terminar ou não achar o endereço, igual já
// acontece com falha de notificação por WhatsApp em outros pontos do
// sistema.
func (s *PedidoService) geocodificarDestinoEmSegundoPlano(pedidoID uint, endereco EnderecoEstruturado) {
	if s.distanciaService == nil {
		return
	}
	go func() {
		destino, err := s.distanciaService.GeocodificarEstruturado(endereco)
		if err != nil {
			log.Printf("aviso: não foi possível geocodificar o destino do pedido %d: %v", pedidoID, err)
			return
		}
		if err := s.pedidoRepo.AtualizarDestinoGeo(pedidoID, destino.Latitude, destino.Longitude); err != nil {
			log.Printf("aviso: não foi possível salvar a coordenada de destino do pedido %d: %v", pedidoID, err)
		}
	}()
}

// PreencherDestinoGeoFaltantes é uma migração de dado, chamada uma vez
// no boot da API (main.go) — achada em produção em 28/08/2026 junto com
// o bug do token vazio: todo pedido criado antes da geocodificação em
// segundo plano existir (ver geocodificarDestinoEmSegundoPlano) fica
// sem DestinoLatitude/DestinoLongitude pra sempre, então a tela do
// entregador mostra "localização não disponível" em vez do pino, e o
// botão "Iniciar corrida" cai pro fallback de endereço em texto livre —
// bem menos confiável que coordenada exata (foi o que causou o
// "endereço totalmente errado" relatado: geocodificação de texto livre
// erra mais que a estruturada, ver comentário de tentativasEstruturadas
// em distancia_service.go).
//
// Diferente de PreencherCodigosDeConfirmacaoFaltantes/
// PreencherTokensEntregadorFaltantes (grava um valor local, instantâneo,
// sem depender de rede), essa migração chama o Nominatim uma vez por
// pedido, com o rate-limit de ~1,1s já embutido em DistanciaService —
// roda em goroutine, não trava o boot da API esperando isso terminar.
func (s *PedidoService) PreencherDestinoGeoFaltantes() {
	if s.distanciaService == nil {
		return
	}
	go func() {
		pedidos, err := s.pedidoRepo.ListarEntregaSemDestinoGeo()
		if err != nil {
			log.Printf("aviso: não foi possível listar pedidos sem coordenada de destino: %v", err)
			return
		}
		for _, pedido := range pedidos {
			destino, err := s.distanciaService.GeocodificarTextoLivre(pedido.EnderecoEntrega)
			if err != nil {
				log.Printf("aviso: não foi possível geocodificar o destino do pedido antigo %d: %v", pedido.ID, err)
				continue
			}
			if err := s.pedidoRepo.AtualizarDestinoGeo(pedido.ID, destino.Latitude, destino.Longitude); err != nil {
				log.Printf("aviso: não foi possível salvar a coordenada de destino do pedido antigo %d: %v", pedido.ID, err)
			}
		}
	}()
}

func (s *PedidoService) ListarPorLoja(lojaID uint) ([]domain.Pedido, error) {
	return s.pedidoRepo.ListarPorLoja(lojaID)
}

// validarLojaAberta verifica se a loja está aceitando pedidos agora —
// checa pausa manual, horário de funcionamento e margem de fechamento.
func validarLojaAberta(loja *domain.Loja) error {
	if loja.Pausado {
		msg := "loja temporariamente fechada"
		if loja.MensagemPausa != "" {
			msg = loja.MensagemPausa
		}
		return errors.New(msg)
	}

	if loja.HorarioAbertura == "" || loja.HorarioFechamento == "" {
		return nil
	}

	fusoBrasil, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fusoBrasil = time.UTC
	}

	agora := time.Now().In(fusoBrasil)
	agoraStr := agora.Format("15:04")

	fechamento := loja.HorarioFechamento
	if loja.MargemFechamentoMinutos > 0 {
		t, err := time.Parse("15:04", loja.HorarioFechamento)
		if err == nil {
			t = t.Add(-time.Duration(loja.MargemFechamentoMinutos) * time.Minute)
			fechamento = t.Format("15:04")
		}
	}

	if agoraStr < loja.HorarioAbertura || agoraStr >= fechamento {
		return fmt.Errorf("loja fechada — funcionamos das %s às %s", loja.HorarioAbertura, loja.HorarioFechamento)
	}

	return nil
}

// verificarLimiteStart aplica o limite de pedidos/mês do plano Start (Fase
// 7.3): não bloqueia nada por conta própria (isso é papel da rotina
// agendada, ver LojaService.VerificarLimiteStart, que também avisa o dono
// por WhatsApp antes de bloquear de vez) — só (1) recusa o pedido se a
// loja já estiver marcada como bloqueada nesse mês, e (2) dispara o aviso
// inicial de "passou de 30" no exato pedido que estoura a cota, sem
// segurar esse pedido (ele é aceito normalmente, só o aviso é disparado).
func (s *PedidoService) verificarLimiteStart(loja *domain.Loja) error {
	if loja.Plano != "start" {
		return nil
	}

	inicioMes := inicioMesCalendarioService()

	if loja.PedidosBloqueadosDesde != nil && !loja.PedidosBloqueadosDesde.Before(inicioMes) {
		return errors.New("essa loja está temporariamente indisponível pra novos pedidos — tenta de novo mais tarde")
	}

	pedidosNoMes, err := s.pedidoRepo.ContarPedidosMesAtual(loja.ID)
	if err != nil {
		log.Printf("aviso: não foi possível contar pedidos do mês da loja %d pro limite do Start: %v", loja.ID, err)
		return nil // erro de leitura não deve travar o pedido do cliente
	}

	jaAvisadoEsseMes := loja.AvisoLimitePedidosEm != nil && !loja.AvisoLimitePedidosEm.Before(inicioMes)
	if pedidosNoMes+1 > domain.LimitePedidosStart && !jaAvisadoEsseMes {
		s.avisarLimitePedidos(loja)
	}

	return nil
}

// avisarLimitePedidos manda o aviso (painel + WhatsApp) de que a loja
// Start passou de 30 pedidos no mês — tom de sugestão, não de punição
// (pedidos continuam sendo aceitos normalmente até o bloqueio, 3 dias
// depois, ver VerificarLimiteStart).
func (s *PedidoService) avisarLimitePedidos(loja *domain.Loja) {
	if err := s.lojaRepo.AtualizarAvisoLimitePedidos(loja.ID, time.Now()); err != nil {
		log.Printf("aviso: não foi possível registrar o aviso de limite de pedidos da loja %d: %v", loja.ID, err)
		return
	}

	if s.notificationSender == nil || loja.WhatsappNumero == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		texto := fmt.Sprintf(
			"🎉 Sua loja já passou de %d pedidos este mês — parabéns pelo movimento!\n\nO plano Start tem esse limite. Ative o Basic (sem mensalidade) pra continuar recebendo pedidos sem parar. Você ainda tem alguns dias antes do limite pausar novos pedidos.\n\nAtiva em Meu Plano, no painel da Drenux.",
			domain.LimitePedidosStart,
		)
		if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, texto); err != nil {
			log.Printf("falha ao enviar aviso de limite de pedidos da loja %d: %v", loja.ID, err)
		}
	}()
}

// inicioMesCalendarioService é a mesma conta de inicioMesCalendario (repository),
// duplicada aqui porque o pacote repository não exporta ela e não vale a
// pena criar um pacote compartilhado só pra 4 linhas.
func inicioMesCalendarioService() time.Time {
	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	agora := time.Now().In(fusoBrasil)
	return time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, fusoBrasil)
}

// validarDataRetirada aplica as regras do modo de pedido da loja.
func validarDataRetirada(dataRetirada time.Time, loja *domain.Loja) error {
	fusoBrasil, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fusoBrasil = time.UTC
	}

	agora := time.Now().In(fusoBrasil)

	if loja.ModoPedido == domain.ModoPedidoImediato {
		if dataRetirada.Before(agora.Add(-1 * time.Minute)) {
			return errors.New("data de retirada não pode ser no passado")
		}
		return nil
	}

	minimoHoras := loja.AntecedenciaMinimaHoras
	if minimoHoras <= 0 {
		minimoHoras = 1
	}
	minimo := agora.Add(time.Duration(minimoHoras) * time.Hour)

	if dataRetirada.Before(minimo) {
		return fmt.Errorf("essa loja exige pelo menos %d hora(s) de antecedência pra fazer um pedido", minimoHoras)
	}

	return nil
}

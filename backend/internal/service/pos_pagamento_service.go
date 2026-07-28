package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// PosPagamentoService concentra o que precisa acontecer depois que um
// pedido é confirmado como pago, independente de qual processador
// (Stripe, Mercado Pago) recebeu o dinheiro: descontar estoque, avisar o
// dono quando bate o alerta configurado, e notificar cliente/dono via
// WhatsApp. Extraído do que antes vivia só dentro do StripeService pra
// não duplicar essa lógica quando o Mercado Pago entrou (ver
// docs/plano-melhorias-drenux.md, Fase 5) — repasse de comissão de
// afiliado NÃO está aqui porque hoje só existe via Stripe Transfer,
// específico de cada processador (ver MercadoPagoService.processarPosPagamento).
type PosPagamentoService struct {
	db                 *gorm.DB
	pedidoRepo         *repository.PedidoRepository
	lojaRepo           *repository.LojaRepository
	notificationSender notification.NotificationSender
}

func NewPosPagamentoService(db *gorm.DB, notificationSender notification.NotificationSender) *PosPagamentoService {
	return &PosPagamentoService{
		db:                 db,
		pedidoRepo:         repository.NewPedidoRepository(db),
		lojaRepo:           repository.NewLojaRepository(db),
		notificationSender: notificationSender,
	}
}

// ProcessarPedidoPago desconta o estoque de cada item do pedido (por
// variação, quando houver, senão do produto) e notifica cliente/dono.
// Não devolve erro — é sempre chamado a partir de uma goroutine própria
// do processador de pagamento, então falha aqui só é logada.
func (s *PosPagamentoService) ProcessarPedidoPago(pedidoID uint) {
	log.Printf("pós-pagamento do pedido %d: iniciando (estoque + notificação)", pedidoID)

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

	produtoRepo := repository.NewProdutoRepository(s.db)

	for _, item := range pedido.Itens {
		if alerta := s.descontarEstoque(produtoRepo, item.ProdutoID, item.VariacaoID, item.ProdutoNome, item.Quantidade); alerta != nil {
			s.notificarAlertaEstoque(pedido, loja, alerta.nome, alerta.restante)
		}
	}

	// Combo: cada componente desconta estoque igual um item avulso — a
	// quantidade real subtraída é a do componente dentro do combo
	// multiplicada por quantos combos foram pedidos (ver
	// domain.PedidoComboItem.Quantidade, que guarda só a quantidade "por
	// combo", não a total do pedido).
	for _, combo := range pedido.Combos {
		for _, item := range combo.Itens {
			qtd := item.Quantidade * combo.Quantidade
			if alerta := s.descontarEstoque(produtoRepo, item.ProdutoID, item.VariacaoID, item.ProdutoNome, qtd); alerta != nil {
				s.notificarAlertaEstoque(pedido, loja, alerta.nome, alerta.restante)
			}
		}
	}

	s.notificarPagamento(pedido, loja.Nome, loja.WhatsappNumero)

	if pedido.PesoPendente && s.notificationSender != nil && loja.WhatsappNumero != "" {
		nomes := nomesItensSemPeso(pedido.Itens)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		aviso := fmt.Sprintf(
			"⚠️ Peso pendente — %s\n\nO pedido #%d (guardar e entregar depois) tem produto(s) sem peso cadastrado: %s.\n\nCadastre o peso desses produtos antes de uma entrega fora da sua região — sem isso o frete estimado pode sair errado.",
			loja.Nome, pedido.ID, strings.Join(nomes, ", "),
		)
		if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, aviso); err != nil {
			log.Printf("falha ao enviar aviso de peso pendente do pedido %d: %v", pedido.ID, err)
		}
		cancel()
	}

	log.Printf("pós-pagamento do pedido %d: concluído", pedidoID)
}

// itemEmAlerta carrega o suficiente pra montar o aviso de estoque baixo —
// devolvido por descontarEstoque só quando o alerta configurado foi
// atingido, senão nil.
type itemEmAlerta struct {
	nome     string
	restante int
}

// descontarEstoque subtrai a quantidade do produto ou, se houver
// variação, da variação — mesma função usada tanto pra item avulso quanto
// pra componente de combo (ver chamadas em ProcessarPedidoPago). Erros de
// subtração só são logados (nunca interrompem o pós-pagamento inteiro por
// causa de um item).
func (s *PosPagamentoService) descontarEstoque(produtoRepo *repository.ProdutoRepository, produtoID uint, variacaoID *uint, produtoNome string, quantidade int) *itemEmAlerta {
	if variacaoID != nil {
		variacaoRepo := repository.NewVariacaoRepository(s.db)
		restante, err := variacaoRepo.SubtrairEstoque(*variacaoID, quantidade)
		if err != nil {
			log.Printf("erro ao subtrair estoque da variação %d: %v", *variacaoID, err)
			return nil
		}
		if restante < 0 {
			return nil
		}
		v, emAlerta := variacaoRepo.BuscarEstoqueAlerta(*variacaoID)
		if !emAlerta {
			return nil
		}
		return &itemEmAlerta{nome: fmt.Sprintf("%s (%s)", produtoNome, v.Nome), restante: restante}
	}

	restante, err := produtoRepo.SubtrairEstoque(produtoID, quantidade)
	if err != nil {
		log.Printf("erro ao subtrair estoque do produto %d: %v", produtoID, err)
		return nil
	}
	if restante < 0 {
		return nil
	}
	if _, emAlerta := produtoRepo.BuscarEstoqueAlerta(produtoID); !emAlerta {
		return nil
	}
	return &itemEmAlerta{nome: produtoNome, restante: restante}
}

func (s *PosPagamentoService) notificarAlertaEstoque(pedido *domain.Pedido, loja *domain.Loja, nomeItem string, restante int) {
	if s.notificationSender == nil || loja.WhatsappNumero == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	aviso := fmt.Sprintf("⚠️ Alerta de estoque — %s\n\nO produto *%s* chegou a %d unidade(s) restante(s).", loja.Nome, nomeItem, restante)
	if restante == 0 {
		aviso = fmt.Sprintf("⚠️ Estoque esgotado — %s\n\nO produto *%s* acabou e foi marcado como indisponível automaticamente.", loja.Nome, nomeItem)
	}
	if err := s.notificationSender.EnviarNotificacaoAdmin(ctx, pedido, aviso, loja.WhatsappNumero); err != nil {
		log.Printf("falha ao enviar alerta de estoque: %v", err)
	}
}

func (s *PosPagamentoService) notificarPagamento(pedido *domain.Pedido, lojaNome, whatsappNumero string) {
	if s.notificationSender == nil {
		log.Printf("WhatsApp não conectado — pedido %d foi pago mas a notificação foi pulada", pedido.ID)
		return
	}
	log.Printf("pedido %d: disparando notificação WhatsApp pro cliente e pro dono da loja", pedido.ID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.notificationSender.EnviarConfirmacaoPedido(ctx, pedido, lojaNome); err != nil {
			log.Printf("falha ao notificar cliente do pedido %d: %v", pedido.ID, err)
		}
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.notificationSender.EnviarNotificacaoAdmin(ctx, pedido, lojaNome, whatsappNumero); err != nil {
			log.Printf("falha ao notificar admin do pedido %d: %v", pedido.ID, err)
		}
	}()
}

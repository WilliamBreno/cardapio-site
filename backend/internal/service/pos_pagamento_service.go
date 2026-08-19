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
	db                     *gorm.DB
	pedidoRepo             *repository.PedidoRepository
	lojaRepo               *repository.LojaRepository
	usuarioRepo            *repository.UsuarioRepository
	movimentacaoRepo       *repository.MovimentacaoEstoqueRepository
	fichaTecnicaRepo       *repository.FichaTecnicaRepository
	insumoRepo             *repository.InsumoRepository
	movimentacaoInsumoRepo *repository.MovimentacaoInsumoRepository
	notificationSender     notification.NotificationSender
	emailSender            *notification.EmailSender
}

func NewPosPagamentoService(db *gorm.DB, notificationSender notification.NotificationSender, emailSender *notification.EmailSender) *PosPagamentoService {
	return &PosPagamentoService{
		db:                     db,
		pedidoRepo:             repository.NewPedidoRepository(db),
		lojaRepo:               repository.NewLojaRepository(db),
		usuarioRepo:            repository.NewUsuarioRepository(db),
		movimentacaoRepo:       repository.NewMovimentacaoEstoqueRepository(db),
		fichaTecnicaRepo:       repository.NewFichaTecnicaRepository(db),
		insumoRepo:             repository.NewInsumoRepository(db),
		movimentacaoInsumoRepo: repository.NewMovimentacaoInsumoRepository(db),
		notificationSender:     notificationSender,
		emailSender:            emailSender,
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

	// Fase 10.2: todo pedido que vira pago entra automaticamente na
	// primeira etapa do fluxo de preparo — só se ainda não tiver etapa
	// nenhuma, pra esse pós-pagamento continuar seguro mesmo se rodar
	// mais de uma vez pro mesmo pedido (mesmo espírito idempotente do
	// resto desta função).
	if pedido.StatusEntrega == "" {
		if err := s.pedidoRepo.AtualizarStatusEntrega(pedido.ID, "a_preparar"); err != nil {
			log.Printf("aviso: não foi possível marcar pedido %d como 'a_preparar': %v", pedido.ID, err)
		} else {
			pedido.StatusEntrega = "a_preparar"
		}
	}

	loja, err := s.lojaRepo.BuscarPorID(pedido.LojaID)
	if err != nil {
		log.Printf("não foi possível carregar loja do pedido %d pós-pagamento: %v", pedidoID, err)
		return
	}

	produtoRepo := repository.NewProdutoRepository(s.db)

	for _, item := range pedido.Itens {
		if s.descontarInsumosSeFichaTecnica(loja, pedido.ID, item.ProdutoID, item.Quantidade) {
			continue
		}
		if alerta := s.descontarEstoque(produtoRepo, loja.ID, pedido.ID, item.ProdutoID, item.VariacaoID, item.ProdutoNome, item.Quantidade); alerta != nil {
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
			if s.descontarInsumosSeFichaTecnica(loja, pedido.ID, item.ProdutoID, qtd) {
				continue
			}
			if alerta := s.descontarEstoque(produtoRepo, loja.ID, pedido.ID, item.ProdutoID, item.VariacaoID, item.ProdutoNome, qtd); alerta != nil {
				s.notificarAlertaEstoque(pedido, loja, alerta.nome, alerta.restante)
			}
		}
	}

	s.notificarPagamento(pedido, loja)

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
// causa de um item). Registra a movimentação (Fase 8) sempre que o item
// tinha controle de estoque ativo — pra loja de qualquer plano, mesmo
// que a leitura desse histórico seja exclusiva do Scale.
func (s *PosPagamentoService) descontarEstoque(produtoRepo *repository.ProdutoRepository, lojaID, pedidoID, produtoID uint, variacaoID *uint, produtoNome string, quantidade int) *itemEmAlerta {
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
		s.registrarMovimentoVenda(lojaID, produtoID, variacaoID, pedidoID, quantidade, restante)
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
	s.registrarMovimentoVenda(lojaID, produtoID, nil, pedidoID, quantidade, restante)
	if _, emAlerta := produtoRepo.BuscarEstoqueAlerta(produtoID); !emAlerta {
		return nil
	}
	return &itemEmAlerta{nome: produtoNome, restante: restante}
}

// descontarInsumosSeFichaTecnica desconta os insumos da ficha técnica de
// um produto (Fase 9.1), multiplicando a quantidade de cada insumo pela
// quantidade vendida do produto. Devolve true se o produto tinha ficha
// técnica (mesmo que algum insumo não tenha controle de estoque ativo) —
// nesse caso o chamador NÃO deve cair pro desconto de estoque simples do
// produto/variação (Fase 8), os dois caminhos são mutuamente exclusivos
// por produto. Devolve false se o produto não tem ficha técnica, caso em
// que o chamador segue pro caminho antigo, sem mudança nenhuma.
//
// Escopo v1: a ficha técnica é do produto, não da variação — a
// quantidade de cada insumo não muda conforme a variação escolhida (ver
// domain.FichaTecnicaItem).
func (s *PosPagamentoService) descontarInsumosSeFichaTecnica(loja *domain.Loja, pedidoID, produtoID uint, quantidadeVendida int) bool {
	temFicha, err := s.fichaTecnicaRepo.ExisteFichaTecnica(produtoID)
	if err != nil {
		log.Printf("erro ao verificar ficha técnica do produto %d: %v", produtoID, err)
		return false
	}
	if !temFicha {
		return false
	}

	itens, err := s.fichaTecnicaRepo.BuscarPorProduto(produtoID)
	if err != nil {
		log.Printf("erro ao carregar ficha técnica do produto %d: %v", produtoID, err)
		return true
	}

	for _, item := range itens {
		quantidadeConsumida := item.Quantidade * float64(quantidadeVendida)
		restante, err := s.insumoRepo.SubtrairEstoque(item.InsumoID, quantidadeConsumida)
		if err != nil {
			log.Printf("erro ao subtrair estoque do insumo %d: %v", item.InsumoID, err)
			continue
		}
		if restante < 0 {
			continue // insumo sem controle de estoque ativo, ignora
		}

		s.registrarMovimentoInsumoVenda(loja.ID, item.InsumoID, pedidoID, quantidadeConsumida, restante)

		insumo, emAlerta := s.insumoRepo.BuscarEstoqueAlerta(item.InsumoID)
		if emAlerta {
			s.notificarAlertaInsumo(loja, insumo.Nome, restante)
		}
	}
	return true
}

func (s *PosPagamentoService) registrarMovimentoInsumoVenda(lojaID, insumoID, pedidoID uint, quantidade, estoqueResultante float64) {
	mov := domain.MovimentacaoInsumo{
		LojaID:            lojaID,
		InsumoID:          insumoID,
		Tipo:              domain.MovimentoEstoqueVenda,
		Quantidade:        -quantidade,
		EstoqueResultante: estoqueResultante,
		PedidoID:          &pedidoID,
	}
	if err := s.movimentacaoInsumoRepo.Criar(&mov); err != nil {
		log.Printf("aviso: não foi possível registrar movimentação de venda do insumo %d (pedido %d): %v", insumoID, pedidoID, err)
	}
}

func (s *PosPagamentoService) notificarAlertaInsumo(loja *domain.Loja, nomeInsumo string, restante float64) {
	if s.notificationSender == nil || loja.WhatsappNumero == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	aviso := fmt.Sprintf("⚠️ Alerta de estoque — %s\n\nO insumo *%s* chegou a %.2f restante(s).", loja.Nome, nomeInsumo, restante)
	if restante == 0 {
		aviso = fmt.Sprintf("⚠️ Estoque esgotado — %s\n\nO insumo *%s* acabou. Produtos que dependem dele podem ficar sem poder ser preparados até repor.", loja.Nome, nomeInsumo)
	}
	if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, aviso); err != nil {
		log.Printf("falha ao enviar alerta de estoque de insumo: %v", err)
	}
}

// registrarMovimentoVenda grava a linha de histórico (Fase 8) de um
// desconto de estoque por venda — quantidade sempre negativa (delta),
// diferente do restante devolvido por SubtrairEstoque (que já é o valor
// absoluto final).
func (s *PosPagamentoService) registrarMovimentoVenda(lojaID, produtoID uint, variacaoID *uint, pedidoID uint, quantidade, estoqueResultante int) {
	mov := domain.MovimentacaoEstoque{
		LojaID:            lojaID,
		ProdutoID:         produtoID,
		VariacaoID:        variacaoID,
		Tipo:              domain.MovimentoEstoqueVenda,
		Quantidade:        -quantidade,
		EstoqueResultante: estoqueResultante,
		PedidoID:          &pedidoID,
	}
	if err := s.movimentacaoRepo.Criar(&mov); err != nil {
		log.Printf("aviso: não foi possível registrar movimentação de venda do produto %d (pedido %d): %v", produtoID, pedidoID, err)
	}
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
	// Correção: chamava EnviarNotificacaoAdmin (que ignora esse texto e
	// monta a mensagem padrão de "novo pedido pago", usando `aviso` no
	// lugar do nome da loja por engano) — EnviarTextoAdmin é a função
	// certa pra mensagem de texto livre, é literalmente pra isso que ela
	// existe (ver o comentário dela em sender.go).
	if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, aviso); err != nil {
		log.Printf("falha ao enviar alerta de estoque: %v", err)
	}
}

// notificarPagamento dispara os avisos de pagamento confirmado: pro
// cliente e pro dono da loja. O aviso pro CLIENTE é o "aviso automático
// de status" da matriz de planos (Fase 7.4) — Start não tem, então é
// pulado nesse caso. O aviso pro DONO ("Pedidos via WhatsApp" na matriz)
// muda de canal por plano desde a Fase 8.5, a pedido do William: Start
// não tem NENHUMA interação via WhatsApp incluída — o dono recebe por
// email em vez disso. Basic/Pro/Scale continuam por WhatsApp, sem
// mudança nenhuma nesses planos.
func (s *PosPagamentoService) notificarPagamento(pedido *domain.Pedido, loja *domain.Loja) {
	log.Printf("pedido %d: disparando notificações de pagamento", pedido.ID)

	if loja.Plano != "start" && s.notificationSender != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := s.notificationSender.EnviarConfirmacaoPedido(ctx, pedido, loja.Nome); err != nil {
				log.Printf("falha ao notificar cliente do pedido %d: %v", pedido.ID, err)
			}
		}()
	}

	if loja.Plano == "start" {
		s.notificarNovoPedidoPorEmail(pedido, loja)
		return
	}

	if s.notificationSender == nil {
		log.Printf("WhatsApp não conectado — pedido %d foi pago mas o aviso ao dono foi pulado", pedido.ID)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.notificationSender.EnviarNotificacaoAdmin(ctx, pedido, loja.Nome, loja.WhatsappNumero); err != nil {
			log.Printf("falha ao notificar admin do pedido %d: %v", pedido.ID, err)
		}
	}()
}

// notificarNovoPedidoPorEmail é o aviso de novo pedido pro dono de uma
// loja Start — substitui o WhatsApp (Fase 8.5). Busca o email a partir
// de Loja.UsuarioID porque o email de login mora em Usuario, não em Loja.
func (s *PosPagamentoService) notificarNovoPedidoPorEmail(pedido *domain.Pedido, loja *domain.Loja) {
	if s.emailSender == nil {
		log.Printf("aviso: Resend não configurado — pedido %d (loja Start) foi pago mas o aviso por email foi pulado", pedido.ID)
		return
	}
	usuario, err := s.usuarioRepo.BuscarPorID(loja.UsuarioID)
	if err != nil {
		log.Printf("aviso: não foi possível carregar o dono da loja %d pra avisar do pedido %d por email: %v", loja.ID, pedido.ID, err)
		return
	}
	go func() {
		if err := s.emailSender.EnviarNovoPedido(usuario.Email, loja.Nome, pedido); err != nil {
			log.Printf("falha ao enviar email de novo pedido (loja %d, pedido %d): %v", loja.ID, pedido.ID, err)
		}
	}()
}

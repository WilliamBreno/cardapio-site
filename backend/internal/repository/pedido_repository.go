package repository

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type PedidoRepository struct {
	db *gorm.DB
}

func NewPedidoRepository(db *gorm.DB) *PedidoRepository {
	return &PedidoRepository{db: db}
}

// Criar salva o pedido. Como Pedido.Itens vem preenchido, o GORM cria os
// ItemPedido junto automaticamente (associação has-many), na mesma
// operação — não precisa de um Criar separado pros itens.
func (r *PedidoRepository) Criar(pedido *domain.Pedido) error {
	return r.db.Create(pedido).Error
}

// ListarPorLoja devolve os pedidos de uma loja, mais recentes primeiro,
// com os itens de cada um já carregados.
func (r *PedidoRepository) ListarPorLoja(lojaID uint) ([]domain.Pedido, error) {
	var pedidos []domain.Pedido
	if err := r.db.Where("loja_id = ?", lojaID).Preload("Itens").Preload("Combos.Itens").Order("id desc").Find(&pedidos).Error; err != nil {
		return nil, err
	}
	return pedidos, nil
}

// ListarPorTelefone retorna os últimos pedidos pagos de um cliente
// específico nessa loja. Usado pelo histórico público do cliente.
func (r *PedidoRepository) ListarPorTelefone(lojaID uint, telefone string, limite int) ([]domain.Pedido, error) {
	var pedidos []domain.Pedido
	err := r.db.
		Where("loja_id = ? AND cliente_telefone = ? AND status = ?",
			lojaID, telefone, domain.StatusPago).
		Preload("Itens").
		Preload("Combos.Itens").
		Order("id desc").
		Limit(limite).
		Find(&pedidos).Error
	return pedidos, err
}

func (r *PedidoRepository) BuscarPorID(id uint) (*domain.Pedido, error) {
	var pedido domain.Pedido
	if err := r.db.Preload("Itens").Preload("Combos.Itens").First(&pedido, id).Error; err != nil {
		return nil, err
	}
	return &pedido, nil
}

// BuscarPorIDETelefone é usado pelo rastreamento público — funciona como
// uma "senha simples": só quem sabe o número de telefone usado no pedido
// consegue ver a localização de entrega, sem precisar de login.
func (r *PedidoRepository) BuscarPorIDETelefone(id uint, telefone string) (*domain.Pedido, error) {
	var pedido domain.Pedido
	if err := r.db.Where("id = ? AND cliente_telefone = ?", id, telefone).
		Preload("Itens").First(&pedido).Error; err != nil {
		return nil, err
	}
	return &pedido, nil
}

func (r *PedidoRepository) AtualizarStatus(pedidoID uint, status domain.StatusPedido) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Update("status", status).Error
}

func (r *PedidoRepository) AtualizarStripeSessionID(pedidoID uint, sessionID string) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Update("stripe_session_id", sessionID).Error
}

func (r *PedidoRepository) AtualizarMercadoPagoPreferenceID(pedidoID uint, preferenceID string) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Update("mercado_pago_preference_id", preferenceID).Error
}

// AtualizarStatusEntrega muda o progresso da entrega ("saiu_para_entrega"
// ou "entregue"). Chamado pelo dono/motoboy no painel admin.
func (r *PedidoRepository) AtualizarStatusEntrega(pedidoID uint, statusEntrega string) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Update("status_entrega", statusEntrega).Error
}

// AtualizarLocalizacaoEntregador grava a posição mais recente de quem
// está entregando. Chamado periodicamente pelo navegador de quem
// compartilha a localização, enquanto a entrega está em andamento.
func (r *PedidoRepository) AtualizarLocalizacaoEntregador(pedidoID uint, latitude, longitude float64) error {
	agora := time.Now()
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Updates(map[string]interface{}{
		"entregador_latitude":      latitude,
		"entregador_longitude":     longitude,
		"entregador_atualizado_em": agora,
	}).Error
}

// AtualizarComissaoAfiliado registra quanto foi repassado ao afiliado
// nesse pedido e o ID da Transfer no Stripe (útil pra auditoria e pra
// nunca repassar em duplicidade se o webhook disparar mais de uma vez).
func (r *PedidoRepository) AtualizarComissaoAfiliado(pedidoID uint, comissao float64, transferID string) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Updates(map[string]interface{}{
		"comissao_afiliado":    comissao,
		"afiliado_transfer_id": transferID,
	}).Error
}

// ResumoSemana agrega os pedidos pagos de uma loja em um período.
type ResumoSemana struct {
	TotalPedidos  int
	Faturamento   float64
	ProdutoTop    string
	QuantidadeTop int
}

// BuscarResumoSemana retorna os dados agregados de pedidos pagos
// num intervalo de datas, pra montar o relatório semanal.
func (r *PedidoRepository) BuscarResumoSemana(lojaID uint, inicio, fim interface{}) (*ResumoSemana, error) {
	var pedidos []domain.Pedido
	if err := r.db.
		Where("loja_id = ? AND status = ? AND updated_at BETWEEN ? AND ?",
			lojaID, domain.StatusPago, inicio, fim).
		Preload("Itens").
		Find(&pedidos).Error; err != nil {
		return nil, err
	}

	resumo := &ResumoSemana{}
	resumo.TotalPedidos = len(pedidos)

	contagem := map[string]int{}
	for _, pedido := range pedidos {
		resumo.Faturamento += pedido.Total
		for _, item := range pedido.Itens {
			nome := item.ProdutoNome
			if item.VariacaoNome != "" {
				nome += " (" + item.VariacaoNome + ")"
			}
			contagem[nome] += item.Quantidade
		}
	}

	for nome, qtd := range contagem {
		if qtd > resumo.QuantidadeTop {
			resumo.QuantidadeTop = qtd
			resumo.ProdutoTop = nome
		}
	}

	return resumo, nil
}

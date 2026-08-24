package repository

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

// inicioMesCalendario devolve a meia-noite do dia 1º do mês corrente, no
// fuso de Brasília — limite usado por tudo que precisa resetar "todo dia
// 1º" (GMV escalonado da Fase 7.2, limite de pedidos do Start da Fase
// 7.3), diferente de DashboardData.TotalMes, que é uma janela rolante de
// 30 dias.
func inicioMesCalendario() time.Time {
	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	agora := time.Now().In(fusoBrasil)
	return time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, fusoBrasil)
}

// SomarGMVMesAtual soma o valor pago pela loja no mês corrente, sem taxa
// de entrega — mesma base usada pra calcular comissão (ver
// calcularComissaoEscalonada). Usado pra saber em qual faixa de GMV um
// novo pedido cai (Fase 7.2).
func (r *PedidoRepository) SomarGMVMesAtual(lojaID uint) (float64, error) {
	var total float64
	err := r.db.Raw(`SELECT COALESCE(SUM(total - taxa_entrega), 0) FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND updated_at >= ?`,
		lojaID, inicioMesCalendario()).Scan(&total).Error
	return total, err
}

// ContarPedidosMesAtual conta os pedidos (exceto cancelados) que a loja
// recebeu no mês corrente — usado pelo limite de 30 pedidos/mês do plano
// Start (Fase 7.3). Conta pela data do pedido (created_at), não de
// pagamento: o Start não tem pagamento integrado, então "pedido recebido"
// é o que importa aqui, não "pedido pago".
func (r *PedidoRepository) ContarPedidosMesAtual(lojaID uint) (int, error) {
	var total int64
	err := r.db.Model(&domain.Pedido{}).
		Where("loja_id = ? AND status != ? AND created_at >= ?", lojaID, domain.StatusCancelado, inicioMesCalendario()).
		Count(&total).Error
	return int(total), err
}

// ContarPedidosDesde conta os pedidos (exceto cancelados) que a loja
// recebeu a partir de uma data qualquer — usado pelo bônus de ativação de
// afiliado (Fase 7.5), que mede pedidos desde o cadastro da loja, não do
// mês corrente (diferente de ContarPedidosMesAtual).
func (r *PedidoRepository) ContarPedidosDesde(lojaID uint, desde time.Time) (int, error) {
	var total int64
	err := r.db.Model(&domain.Pedido{}).
		Where("loja_id = ? AND status != ? AND created_at >= ?", lojaID, domain.StatusCancelado, desde).
		Count(&total).Error
	return int(total), err
}

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

// AtualizarFormaPagamento grava o payment_type_id devolvido pelo Mercado
// Pago (Fase 10.6) — vazio não sobrescreve nada (chamado só quando o
// webhook realmente devolveu essa informação).
func (r *PedidoRepository) AtualizarFormaPagamento(pedidoID uint, forma string) error {
	return r.db.Model(&domain.Pedido{}).Where("id = ?", pedidoID).Update("forma_pagamento", forma).Error
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

// PreencherCodigosDeConfirmacaoFaltantes é uma migração de dado, chamada
// uma vez no boot da API (main.go) — pedido criado antes do campo
// CodigoConfirmacao existir fica com ele vazio, o que o travaria pra
// sempre de ser marcado "entregue" (o handler agora exige um código
// não-vazio que bata com o do pedido). Preenche com um valor aleatório
// qualquer — não precisa da mesma robustez de crypto/rand usada na
// geração de pedido novo (PedidoService.gerarCodigoConfirmacao), é só
// destravar pedido antigo.
func (r *PedidoRepository) PreencherCodigosDeConfirmacaoFaltantes() error {
	return r.db.Exec(`
		UPDATE pedidos SET codigo_confirmacao = lpad(floor(random() * 10000)::text, 4, '0')
		WHERE codigo_confirmacao = '' OR codigo_confirmacao IS NULL
	`).Error
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

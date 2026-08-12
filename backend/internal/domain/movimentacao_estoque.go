package domain

import "time"

// TipoMovimentoEstoque distingue a origem de uma mudança de estoque.
type TipoMovimentoEstoque string

const (
	// MovimentoEstoqueVenda é gravado automaticamente pelo pós-pagamento
	// (PosPagamentoService.descontarEstoque) — quantidade sempre negativa.
	MovimentoEstoqueVenda TipoMovimentoEstoque = "venda"
	// MovimentoEstoqueReposicao é um incremento manual do dono (ex:
	// "chegou mercadoria nova") — quantidade sempre positiva.
	MovimentoEstoqueReposicao TipoMovimentoEstoque = "reposicao"
	// MovimentoEstoqueAjuste é uma correção manual pra um valor absoluto
	// (ex: contagem física bateu diferente) — quantidade é a diferença
	// (pode ser positiva ou negativa) entre o valor antigo e o novo.
	MovimentoEstoqueAjuste TipoMovimentoEstoque = "ajuste"
)

// MovimentacaoEstoque registra toda mudança de estoque de um produto (ou
// variação) — venda automática, reposição ou ajuste manual (Fase 8). A
// gravação acontece pra loja de QUALQUER plano, porque o desconto de
// estoque na venda já existe hoje independente de plano — só a LEITURA
// desse histórico é que fica restrita ao plano Scale (EstoqueHandler).
type MovimentacaoEstoque struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	LojaID uint `gorm:"not null;index" json:"loja_id"`

	ProdutoID uint `gorm:"not null;index" json:"produto_id"`
	// VariacaoID é nulo quando o movimento é do estoque geral do produto
	// (produto sem variação, ou variação sem estoque próprio).
	VariacaoID *uint `gorm:"index" json:"variacao_id"`

	Tipo TipoMovimentoEstoque `gorm:"size:20;not null" json:"tipo"`
	// Quantidade é o delta aplicado (negativo em venda, positivo em
	// reposição, qualquer sinal em ajuste) — nunca o valor absoluto.
	Quantidade        int    `gorm:"not null" json:"quantidade"`
	EstoqueResultante int    `gorm:"not null" json:"estoque_resultante"`
	Motivo            string `gorm:"size:200" json:"motivo"`

	// PedidoID só é preenchido quando Tipo == venda — referencia o pedido
	// que originou o desconto.
	PedidoID *uint `gorm:"index" json:"pedido_id"`

	CreatedAt time.Time `json:"created_at"`
}

func (MovimentacaoEstoque) TableName() string {
	return "movimentacoes_estoque"
}

package domain

import "time"

// MovimentacaoInsumo registra toda mudança de estoque de um insumo —
// consumo automático (quando um produto com ficha técnica é vendido),
// reposição ou ajuste manual (Fase 9.1). Mesmo espírito de
// MovimentacaoEstoque (Fase 8), em tabela própria porque Insumo é uma
// entidade diferente de Produto/VariacaoProduto — reaproveita
// TipoMovimentoEstoque (venda/reposicao/ajuste) porque a semântica dos
// três tipos é idêntica, só muda o que está sendo movimentado.
type MovimentacaoInsumo struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	LojaID uint `gorm:"not null;index" json:"loja_id"`

	InsumoID uint `gorm:"not null;index" json:"insumo_id"`

	Tipo TipoMovimentoEstoque `gorm:"size:20;not null" json:"tipo"`
	// Quantidade é o delta aplicado (negativo em consumo por venda,
	// positivo em reposição, qualquer sinal em ajuste) — sempre em
	// UnidadeUso do insumo, nunca o valor absoluto.
	Quantidade        float64 `gorm:"not null" json:"quantidade"`
	EstoqueResultante float64 `gorm:"not null" json:"estoque_resultante"`
	Motivo            string  `gorm:"size:200" json:"motivo"`

	// PedidoID só é preenchido quando Tipo == venda — referencia o pedido
	// que originou o consumo.
	PedidoID *uint `gorm:"index" json:"pedido_id"`

	CreatedAt time.Time `json:"created_at"`
}

func (MovimentacaoInsumo) TableName() string {
	return "movimentacoes_insumo"
}

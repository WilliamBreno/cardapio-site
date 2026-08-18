package domain

// FichaTecnicaItem é um insumo componente da ficha técnica de um produto
// (Fase 9.1, plano Scale) — ex: "Burguer Bacon = 100g carne + 1 pão + 30g
// queijo" são 3 FichaTecnicaItem. Quantidade é sempre expressa na
// UnidadeUso do insumo (ver Insumo.UnidadeUso) — venda desse produto
// desconta o insumo, não o produto pronto (ver
// PosPagamentoService.descontarInsumosSeFichaTecnica).
//
// Escopo v1: a ficha técnica é do PRODUTO, não da variação — todo pedido
// desse produto consome a mesma receita, independente da variação
// escolhida (ex: "Burguer Bacon P" e "G" descontam os mesmos insumos, na
// mesma quantidade). Ficha técnica por variação fica pra uma fase futura,
// se necessário — não foi pedido agora.
type FichaTecnicaItem struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	ProdutoID  uint    `gorm:"not null;index" json:"produto_id"`
	InsumoID   uint    `gorm:"not null;index" json:"insumo_id"`
	Insumo     Insumo  `gorm:"foreignKey:InsumoID" json:"insumo,omitempty"`
	Quantidade float64 `gorm:"not null" json:"quantidade"`
}

func (FichaTecnicaItem) TableName() string {
	return "ficha_tecnica_itens"
}

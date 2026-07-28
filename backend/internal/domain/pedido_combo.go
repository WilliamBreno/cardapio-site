package domain

// PedidoCombo é a "foto" de um Combo no momento da compra — guarda nome,
// foto e preço copiados do Combo original, igual ItemPedido já faz com
// Produto. Se o lojista editar ou apagar o Combo depois, pedidos antigos
// não mudam. ComboID fica só de referência (sem FK de verdade, mesmo
// padrão de ItemPedido.ProdutoID) — pode apontar pra um combo que não
// existe mais.
type PedidoCombo struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	PedidoID   uint    `gorm:"not null;index" json:"pedido_id"`
	ComboID    uint    `gorm:"not null" json:"combo_id"`
	Nome       string  `gorm:"size:100;not null" json:"nome"`
	FotoURL    string  `gorm:"size:500" json:"foto_url"`
	Preco      float64 `gorm:"not null" json:"preco"`
	Quantidade int     `gorm:"not null;default:1" json:"quantidade"`

	Itens []PedidoComboItem `gorm:"foreignKey:PedidoComboID;constraint:OnDelete:CASCADE" json:"itens"`
}

func (PedidoCombo) TableName() string {
	return "pedido_combos"
}

// PedidoComboItem é o snapshot de um produto componente do combo — nome
// e variação escolhida copiados no momento da compra, mesma lógica de
// ItemPedido. Quantidade aqui é por UNIDADE do combo (ex: 1 batata por
// combo) — pra saber quanto descontar de estoque de verdade, multiplica
// por PedidoCombo.Quantidade (quantos combos foram comprados).
type PedidoComboItem struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	PedidoComboID uint   `gorm:"not null;index" json:"pedido_combo_id"`
	ProdutoID     uint   `gorm:"not null" json:"produto_id"`
	ProdutoNome   string `gorm:"size:100;not null" json:"produto_nome"`
	VariacaoID    *uint  `gorm:"default:null" json:"variacao_id"`
	VariacaoNome  string `gorm:"size:50" json:"variacao_nome"`
	Quantidade    int    `gorm:"not null" json:"quantidade"`
}

func (PedidoComboItem) TableName() string {
	return "pedido_combo_itens"
}

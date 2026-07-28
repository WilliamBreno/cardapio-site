package domain

import "time"

// Combo é um pacote fixo de produtos vendido por um preço único, definido
// diretamente pelo lojista (não calculado por desconto sobre a soma dos
// itens) — Fase 6 do roadmap. Uma única entidade serve tanto pra loja
// alimentícia ("Combo X-Tudo") quanto mercadoria ("Kit Presente"): o nome
// é livre, não existe distinção de segmento aqui.
//
// V1 propositalmente simples: pacote fixo (sem "escolha 1 de 3" dentro do
// combo). O cliente escolhe variação de cada item componente (se houver)
// igual escolheria comprando o produto avulso — ver ComboItem.
type Combo struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	LojaID     uint    `gorm:"not null;index" json:"loja_id"`
	Nome       string  `gorm:"size:100;not null" json:"nome"`
	Descricao  string  `gorm:"type:text" json:"descricao"`
	FotoURL    string  `gorm:"size:500" json:"foto_url"`
	Preco      float64 `gorm:"not null" json:"preco"`
	Disponivel bool    `gorm:"default:true" json:"disponivel"`
	Ordem      int     `gorm:"default:0" json:"ordem"`

	Itens []ComboItem `gorm:"foreignKey:ComboID;constraint:OnDelete:CASCADE" json:"itens,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Combo) TableName() string {
	return "combos"
}

// ComboItem é um produto componente do combo, com a quantidade incluída
// no pacote — ex: "1x X-Burger + 1x Batata + 1x Refri" são 3 ComboItem.
// Não guarda variação escolhida aqui (isso só existe no momento da
// compra, quando o cliente monta o combo no carrinho — ver
// PedidoComboItem, o snapshot que guarda a variação escolhida de fato).
type ComboItem struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	ComboID    uint    `gorm:"not null;index" json:"combo_id"`
	ProdutoID  uint    `gorm:"not null" json:"produto_id"`
	Produto    Produto `gorm:"foreignKey:ProdutoID" json:"produto,omitempty"`
	Quantidade int     `gorm:"not null;default:1" json:"quantidade"`
}

func (ComboItem) TableName() string {
	return "combo_itens"
}

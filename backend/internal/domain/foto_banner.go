package domain

// FotoBanner é uma das fotos do carrossel de banner do cardápio público
// (redesign de 24/08/2026) — substitui o antigo Loja.BannerURL (foto
// única) por uma lista, exibida em rotação no topo do cardápio. Mesmo
// padrão de FotoProduto: Ordem decide a sequência de exibição.
type FotoBanner struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	LojaID uint   `gorm:"not null;index" json:"loja_id"`
	URL    string `gorm:"size:255;not null" json:"url"`
	Ordem  int    `gorm:"default:0" json:"ordem"`
}

func (FotoBanner) TableName() string { return "fotos_banner" }

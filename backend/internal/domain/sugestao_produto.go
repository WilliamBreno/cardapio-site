package domain

import "time"

type TipoDesconto string

const (
	TipoDescontoPercentual TipoDesconto = "percentual"
	TipoDescontoFixo       TipoDesconto = "fixo"
)

// SugestaoProduto é um vínculo manual "quem compra X, sugerir Y" — Fase 6
// (upsell estilo totem de fastfood). Não é algoritmo automático de
// "frequentemente comprados juntos": o lojista escolhe os dois lados do
// vínculo na tela de configuração. O desconto (opcional) só vale quando
// o produto sugerido é adicionado através dessa sugestão especificamente
// — comprando ele avulso, o preço normal continua valendo (ver
// PedidoService.CriarPorSlug, onde o desconto só é aplicado se o item do
// pedido referenciar essa sugestão).
type SugestaoProduto struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	LojaID uint `gorm:"not null;index" json:"loja_id"`

	ProdutoOrigemID uint `gorm:"not null;uniqueIndex:idx_sugestao_origem_sugerido" json:"produto_origem_id"`

	ProdutoSugeridoID uint    `gorm:"not null;uniqueIndex:idx_sugestao_origem_sugerido" json:"produto_sugerido_id"`
	ProdutoSugerido   Produto `gorm:"foreignKey:ProdutoSugeridoID" json:"produto_sugerido,omitempty"`

	// TipoDesconto/ValorDesconto ficam os dois nil quando não há
	// desconto — a sugestão só destaca o produto, sem mexer no preço.
	TipoDesconto  *TipoDesconto `gorm:"size:20" json:"tipo_desconto"`
	ValorDesconto *float64      `json:"valor_desconto"`

	CreatedAt time.Time `json:"created_at"`
}

func (SugestaoProduto) TableName() string {
	return "sugestoes_produto"
}

// PrecoComDesconto aplica o desconto (se houver) sobre um preço base —
// usado tanto na prévia de sugestões do carrinho quanto na validação do
// checkout, sempre com a mesma fórmula.
func (s *SugestaoProduto) PrecoComDesconto(precoBase float64) float64 {
	if s.TipoDesconto == nil || s.ValorDesconto == nil {
		return precoBase
	}
	var final float64
	switch *s.TipoDesconto {
	case TipoDescontoPercentual:
		final = precoBase * (1 - *s.ValorDesconto/100)
	case TipoDescontoFixo:
		final = precoBase - *s.ValorDesconto
	default:
		final = precoBase
	}
	if final < 0 {
		final = 0
	}
	return final
}

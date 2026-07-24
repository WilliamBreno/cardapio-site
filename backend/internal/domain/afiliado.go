package domain

import "time"

// Afiliado indica uma loja pro Drenux via um link com código único
// (?ref=CODIGO no cadastro) e recebe uma comissão automática (3,01% do
// total de cada pedido pago) por tempo indeterminado, enquanto a loja
// indicada continuar ativa. Contas de afiliado são criadas manualmente
// (não há autocadastro), mas o próprio afiliado gerencia sua conta
// Stripe Connect e acompanha os ganhos pelo painel dele.
type Afiliado struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	Nome            string `gorm:"size:100;not null" json:"nome"`
	Email           string `gorm:"size:150;not null;unique" json:"email"`
	SenhaHash       string `gorm:"size:255;not null" json:"-"`
	Codigo          string `gorm:"size:30;not null;unique" json:"codigo"`
	StripeAccountID string `gorm:"size:100" json:"-"`

	// ComissaoPercentual é a fração da taxa de plataforma que ESSE
	// afiliado recebe de comissão — 0.376 (37,6%) é o valor padrão
	// histórico, usado igual pra todo mundo antes da Fase 5.5 permitir
	// negociar por afiliado. Definido no cadastro (ver
	// AfiliadoService.CriarAfiliado), usado tanto no repasse automático
	// via Stripe Transfer quanto no registro manual do Mercado Pago.
	ComissaoPercentual float64 `gorm:"not null;default:0.376" json:"comissao_percentual"`

	ResetToken       *string    `json:"-"`
	ResetTokenExpira *time.Time `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Afiliado) TableName() string {
	return "afiliados"
}

package domain

import "time"

type StatusRepasse string

const (
	StatusRepassePendente StatusRepasse = "pendente"
	StatusRepassePago     StatusRepasse = "pago"
)

// RepasseAfiliado registra o valor devido a um afiliado por um pedido
// específico pago via Mercado Pago. Existe porque o split do Mercado Pago
// hoje é só 1:1 (Loja + Drenux) — o modelo 1:N que permitiria repassar
// direto pro afiliado na mesma transação exige contato comercial com o
// Mercado Pago, sem prazo garantido (ver docs/plano-melhorias-drenux.md,
// Fase 5.5, decisão fechada em 24/07/2026: controle manual em vez de
// esperar o 1:N). O repasse em si (Pix) continua manual, fora do sistema
// — isso aqui é só o registro de quanto é devido e se já foi pago.
//
// Pedidos pagos via Stripe continuam usando o repasse automático via
// Stripe Transfer (ver StripeService.transferirComissaoAfiliado) — não
// geram RepasseAfiliado, já são resolvidos na hora.
type RepasseAfiliado struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	AfiliadoID uint `gorm:"not null;index" json:"afiliado_id"`

	// PedidoID é único: um pedido só pode gerar um lançamento — trava
	// contra duplicar caso a notificação do Mercado Pago (merchant_order)
	// chegue mais de uma vez pro mesmo pagamento.
	PedidoID uint `gorm:"not null;uniqueIndex" json:"pedido_id"`

	LojaID uint `gorm:"not null;index" json:"loja_id"`
	Loja   Loja `gorm:"foreignKey:LojaID" json:"loja,omitempty"`

	Valor  float64       `gorm:"not null" json:"valor"`
	Status StatusRepasse `gorm:"size:20;not null;default:pendente;index" json:"status"`
	PagoEm *time.Time    `json:"pago_em"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RepasseAfiliado) TableName() string {
	return "repasses_afiliado"
}

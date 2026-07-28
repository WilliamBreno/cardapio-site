package domain

import "time"

// AssinaturaPendente representa uma assinatura de mensalidade (Pro/Scale)
// aguardando o cliente completar o cadastro da loja (nome, senha, nome da
// loja). Não tem prazo de expiração de propósito — depois que o pagamento
// confirma, o link pra finalizar continua válido até ser usado, mesmo que
// isso demore dias.
//
// Desde a Fase 6 (Mercado Pago), o registro é criado ANTES do pagamento
// (com Token já gerado, pra poder montar a URL de checkout com
// external_reference=Token) e só fica utilizável depois que o webhook
// confirma — ver Confirmada. Os campos Stripe* seguem aqui só porque o
// fluxo antigo (dormant, não chamado mais) ainda referencia esse struct;
// não usar pra novo código.
type AssinaturaPendente struct {
	ID                   uint   `gorm:"primaryKey" json:"id"`
	Email                string `gorm:"size:150;not null" json:"email"`
	Plano                string `gorm:"size:20;not null" json:"plano"`
	StripeCustomerID     string `gorm:"size:100" json:"-"`
	StripeSubscriptionID string `gorm:"size:100" json:"-"`
	Token                string `gorm:"size:100;not null;unique" json:"-"`
	Usado                bool   `gorm:"default:false" json:"-"`

	// MercadoPagoPreapprovalID e Confirmada só são preenchidos quando o
	// webhook de assinatura confirma o pagamento (ver
	// MercadoPagoAssinaturaService.ProcessarWebhook) — antes disso, o
	// registro existe (criado no momento do checkout) mas não pode ser
	// usado pra finalizar cadastro ainda.
	MercadoPagoPreapprovalID string `gorm:"size:100;index" json:"-"`
	Confirmada               bool   `gorm:"default:false" json:"-"`

	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	StripeSessionID string    `gorm:"size:100;index" json:"-"`
}

func (AssinaturaPendente) TableName() string {
	return "assinaturas_pendentes"
}

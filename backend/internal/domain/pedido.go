package domain

import "time"

type StatusPedido string

const (
	StatusAguardandoPagamento StatusPedido = "aguardando_pagamento"
	StatusPago                StatusPedido = "pago"
	StatusCancelado           StatusPedido = "cancelado"
)

type ModoEntrega string

const (
	ModoEntregaRetirada ModoEntrega = "retirada"
	ModoEntregaEntrega  ModoEntrega = "entrega"

	// ModoEntregaGuardar: cliente paga os itens agora mas não os recebe
	// ainda — a loja guarda por tempo indeterminado. Só disponível pra
	// produtos do tipo "mercadoria" (ver TipoProduto). O cliente volta
	// depois pra escolher o que quer receber e paga só o frete nesse
	// momento (ver domain/solicitacao_entrega.go).
	ModoEntregaGuardar ModoEntrega = "guardar"
)

// Pedido representa um pedido feito por um cliente final numa loja.
type Pedido struct {
	ID              uint         `gorm:"primaryKey" json:"id"`
	LojaID          uint         `gorm:"not null;index" json:"loja_id"`
	ClienteNome     string       `gorm:"size:100;not null" json:"cliente_nome"`
	ClienteTelefone string       `gorm:"size:20;not null" json:"cliente_telefone"`
	DataRetirada    time.Time    `gorm:"not null" json:"data_retirada"`
	Status          StatusPedido `gorm:"size:30;not null;default:aguardando_pagamento" json:"status"`
	Total           float64      `gorm:"not null" json:"total"`

	// Modo de recebimento — preenchido pelo cliente no momento do pedido.
	ModoEntrega     ModoEntrega `gorm:"size:20;default:'retirada'" json:"modo_entrega"`
	EnderecoEntrega string      `gorm:"size:300" json:"endereco_entrega"`

	// PesoPendente é um aviso preventivo: true quando o pedido é modo
	// "guardar" e tem item mercadoria sem peso cadastrado — não trava a
	// compra, só sinaliza pro lojista que vai precisar completar o peso
	// antes de uma entrega interestadual de verdade (ver
	// SolicitacaoEntrega.PesoPendente, que é o aviso definitivo, calculado
	// só quando o frete realmente precisar do peso e ele faltar).
	PesoPendente bool `gorm:"default:false" json:"peso_pendente"`

	// Cupom aplicado — snapshot do código no momento do pedido.
	// Guardamos o código (não o ID) porque se o cupom for deletado depois,
	// o histórico do pedido ainda faz sentido.
	CupomCodigo string  `gorm:"size:30" json:"cupom_codigo"`
	Desconto    float64 `gorm:"default:0" json:"desconto"`

	StripeSessionID string `gorm:"size:255" json:"-"`

	// MercadoPagoPreferenceID identifica a "preference" criada no checkout
	// de pedido via Mercado Pago (ver Fase 5) — equivalente ao
	// StripeSessionID acima, só que pro novo processador.
	MercadoPagoPreferenceID string `gorm:"size:255" json:"-"`

	// Comissão repassada automaticamente pro afiliado que indicou a loja
	// (se houver), via Stripe Transfer. AfiliadoTransferID guarda o ID
	// da Transfer no Stripe — serve de trava contra repasse em
	// duplicidade se o webhook de pagamento disparar mais de uma vez.
	ComissaoAfiliado   float64      `gorm:"default:0" json:"-"`
	AfiliadoTransferID string       `gorm:"size:100" json:"-"`
	Itens              []ItemPedido `gorm:"foreignKey:PedidoID" json:"itens"`

	// Combos (Fase 6) — snapshot dos combos comprados nesse pedido, à
	// parte de Itens (que continua só produto avulso). Um pedido pode
	// misturar os dois: produtos soltos + combos, na mesma compra.
	Combos                 []PedidoCombo `gorm:"foreignKey:PedidoID;constraint:OnDelete:CASCADE" json:"combos,omitempty"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
	TaxaEntrega            float64       `gorm:"default:0" json:"taxa_entrega"`
	StatusEntrega          string        `gorm:"size:30;default:''" json:"status_entrega"`
	EntregadorLatitude     float64       `gorm:"default:0" json:"entregador_latitude"`
	EntregadorLongitude    float64       `gorm:"default:0" json:"entregador_longitude"`
	EntregadorAtualizadoEm *time.Time    `json:"entregador_atualizado_em"`
}

func (Pedido) TableName() string {
	return "pedidos"
}

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

	// FormaPagamento (Fase 10.6) é o payment_type_id devolvido pelo
	// Mercado Pago (ex: "pix", "credit_card", "debit_card", "ticket") —
	// capturado só a partir dessa fase, então pedido pago antes fica com
	// esse campo vazio pra sempre (não dá pra reconstruir
	// retroativamente sem reconsultar a API do Mercado Pago).
	FormaPagamento string `gorm:"size:30" json:"forma_pagamento"`

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
	Combos        []PedidoCombo `gorm:"foreignKey:PedidoID;constraint:OnDelete:CASCADE" json:"combos,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	TaxaEntrega   float64       `gorm:"default:0" json:"taxa_entrega"`
	StatusEntrega string        `gorm:"size:30;default:''" json:"status_entrega"`
	// CodigoConfirmacao (24/08/2026) — código de 4 dígitos gerado na
	// criação do pedido, mostrado pro cliente na própria tela de
	// rastreamento. O entregador precisa digitar esse código pra marcar o
	// pedido como "entregue" (ver PedidoHandler.AtualizarStatusEntrega) —
	// fecha o ciclo de confirmar que a entrega aconteceu de verdade, em
	// vez de só um clique sem nenhuma checagem. Só é exigido pra pedido
	// modo "entrega"; retirada/guardar não passam por essa checagem (não
	// tem entregador).
	CodigoConfirmacao      string     `gorm:"size:10" json:"codigo_confirmacao"`
	EntregadorLatitude     float64    `gorm:"default:0" json:"entregador_latitude"`
	EntregadorLongitude    float64    `gorm:"default:0" json:"entregador_longitude"`
	EntregadorAtualizadoEm *time.Time `json:"entregador_atualizado_em"`

	// TokenEntregador (26/08/2026) — token de acesso gerado na criação do
	// pedido, usado pelo link público "Gerar link" (ver PedidoHandler,
	// grupo de rotas /lojas/:slug/pedidos/:id/entregador). Funciona como
	// senha simples pra quem for entregar acessar a tela de gerenciar a
	// entrega SEM precisar da conta de dono da loja — mesmo espírito do
	// telefone no link de rastreamento do cliente, só que aqui é um
	// segredo de verdade (não algo que o entregador já saiba de cor),
	// porque essa tela também aceita atualizar status/localização.
	TokenEntregador string `gorm:"size:40" json:"token_entregador"`

	// DestinoLatitude/DestinoLongitude (26/08/2026) — coordenada do
	// endereço de entrega, geocodificada em segundo plano na criação do
	// pedido (ver PedidoService.geocodificarDestinoEmSegundoPlano).
	// Diferente do cálculo de frete "por_km" (que já geocodifica, mas só
	// nesse tipo de taxa e sem persistir o resultado), esses campos
	// existem pra sempre mostrar o pino de destino no mapa do entregador,
	// não importa o tipo de taxa de entrega da loja. Ficam em 0,0
	// enquanto a geocodificação (assíncrona) ainda não terminou, ou se
	// falhar — nesse caso a tela do entregador cai pro endereço em texto.
	DestinoLatitude  float64 `gorm:"default:0" json:"destino_latitude"`
	DestinoLongitude float64 `gorm:"default:0" json:"destino_longitude"`
}

func (Pedido) TableName() string {
	return "pedidos"
}

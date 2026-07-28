package domain

// ItemPedido guarda o nome e o preço do produto no momento da compra
// (uma "foto" daquele instante) — se o dono mudar o preço ou o nome do
// produto depois, o histórico do pedido não muda. É o registro do que
// foi cobrado de verdade, não um espelho do estado atual do produto.
//
// De propósito não existe um campo Produto (struct aninhado) aqui — só
// ProdutoID como referência solta, sem chave estrangeira de verdade no
// banco. Assim, mesmo que o produto seja excluído no futuro, o pedido
// continua íntegro e legível.
type ItemPedido struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	PedidoID    uint    `gorm:"not null;index" json:"pedido_id"`
	ProdutoID   uint    `gorm:"not null" json:"produto_id"`
	ProdutoNome string  `gorm:"size:100;not null" json:"produto_nome"`
	Quantidade  int     `gorm:"not null" json:"quantidade"`
	PrecoUnit   float64 `gorm:"not null" json:"preco_unit"`

	// Variação escolhida (snapshot do momento da compra, igual ao nome
	// do produto — se a variação mudar depois, o histórico não muda).
	VariacaoID   *uint  `gorm:"default:null" json:"variacao_id"`
	VariacaoNome string `gorm:"size:50" json:"variacao_nome"`

	// TipoProduto e PesoGramas são snapshot do produto no momento da
	// compra (mesma lógica de ProdutoNome) — usados depois pra calcular
	// o frete de itens guardados sem depender do produto ainda existir
	// ou não ter mudado de peso/tipo nesse meio tempo.
	TipoProduto TipoProduto `gorm:"size:20;default:'alimenticio'" json:"tipo_produto"`
	PesoGramas  int         `gorm:"default:0" json:"peso_gramas"`

	// SolicitacaoEntregaID marca se esse item (guardado) já foi
	// reivindicado por um pedido de entrega — nil = ainda disponível
	// pro cliente escolher depois.
	SolicitacaoEntregaID *uint `gorm:"default:null;index" json:"solicitacao_entrega_id"`

	// SugestaoProdutoID marca se esse item foi adicionado através da
	// Sugestão Inteligente do carrinho (Fase 6) — nil = item normal. Só
	// auditoria/analytics; o desconto já foi aplicado direto em
	// PrecoUnit no momento da compra (ver PedidoService.CriarPorSlug),
	// então não depende desse campo continuar existindo/válido depois.
	SugestaoProdutoID *uint `gorm:"default:null" json:"sugestao_produto_id"`
}

func (ItemPedido) TableName() string {
	return "itens_pedido"
}

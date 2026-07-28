package repository

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type LojaRepository struct {
	db *gorm.DB
}

func NewLojaRepository(db *gorm.DB) *LojaRepository {
	return &LojaRepository{db: db}
}

func (r *LojaRepository) Criar(loja *domain.Loja) error {
	return r.db.Create(loja).Error
}

func (r *LojaRepository) BuscarPorUsuarioID(usuarioID uint) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("usuario_id = ?", usuarioID).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

func (r *LojaRepository) BuscarPorID(id uint) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.First(&loja, id).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

// BuscarPorStripeSubscriptionID é usado pelo webhook de renovação de
// assinatura, pra achar qual loja pertence a uma subscription da Stripe.
func (r *LojaRepository) BuscarPorStripeSubscriptionID(subscriptionID string) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("stripe_subscription_id = ?", subscriptionID).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

func (r *LojaRepository) AtualizarStripeAccountID(lojaID uint, stripeAccountID string) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("stripe_account_id", stripeAccountID).Error
}

// AtualizarMercadoPago salva os dados da conexão OAuth da loja com o
// Mercado Pago — chamado tanto na conexão inicial quanto na renovação de
// token (que troca access/refresh token e empurra a expiração pra frente).
func (r *LojaRepository) AtualizarMercadoPago(lojaID uint, accessToken, refreshToken, userID string, expiraEm time.Time) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"mercado_pago_access_token":    accessToken,
		"mercado_pago_refresh_token":   refreshToken,
		"mercado_pago_user_id":         userID,
		"mercado_pago_token_expira_em": expiraEm,
	}).Error
}

// BuscarPorMercadoPagoUserID é usado pelo webhook do Mercado Pago pra
// achar de qual loja é um pagamento — a notificação identifica o
// vendedor pelo "collector_id" (aqui salvo como MercadoPagoUserID), não
// por um ID nosso.
func (r *LojaRepository) BuscarPorMercadoPagoUserID(userID string) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("mercado_pago_user_id = ?", userID).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

// ListarComMercadoPagoExpirandoAte devolve as lojas conectadas ao Mercado
// Pago cujo token expira até o instante informado — usado pela rotina de
// renovação automática (Fase 5.4) pra renovar antes do vencimento, não
// depois.
func (r *LojaRepository) ListarComMercadoPagoExpirandoAte(limite time.Time) ([]domain.Loja, error) {
	var lojas []domain.Loja
	if err := r.db.Where("mercado_pago_user_id != ? AND mercado_pago_token_expira_em <= ?", "", limite).Find(&lojas).Error; err != nil {
		return nil, err
	}
	return lojas, nil
}

// AtualizarPlano aplica uma troca de plano imediatamente (upgrade ou
// troca entre planos pagos) — usado tanto na confirmação do checkout de
// nova assinatura quanto na troca direta de Price numa assinatura já
// existente.
func (r *LojaRepository) AtualizarPlano(lojaID uint, plano, stripeCustomerID, stripeSubscriptionID string) error {
	updates := map[string]interface{}{
		"plano":          plano,
		"plano_agendado": nil,
	}
	if stripeCustomerID != "" {
		updates["stripe_customer_id"] = stripeCustomerID
	}
	if stripeSubscriptionID != "" {
		updates["stripe_subscription_id"] = stripeSubscriptionID
	}
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(updates).Error
}

// AtualizarPlanoAgendado marca (ou limpa, se nil) um downgrade pendente
// pro fim do ciclo de cobrança atual.
func (r *LojaRepository) AtualizarPlanoAgendado(lojaID uint, planoAgendado *string) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("plano_agendado", planoAgendado).Error
}

// LimparAssinatura remove os dados da assinatura Stripe da loja —
// usado quando um downgrade agendado pro Start é aplicado (cancela a
// assinatura de vez).
func (r *LojaRepository) LimparAssinatura(lojaID uint) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"plano":                  "start",
		"plano_agendado":         nil,
		"stripe_subscription_id": "",
	}).Error
}

// ConfiguracoesLoja agrupa todos os campos editáveis pelo dono no painel.
type ConfiguracoesLoja struct {
	WhatsappNumero           string
	LogoURL                  string
	ModoPedido               string
	AntecedenciaMinimaHoras  int
	HorarioAbertura          string
	HorarioFechamento        string
	MargemFechamentoMinutos  int
	Pausado                  bool
	MensagemPausa            string
	AceitaRetirada           bool
	AceitaEntrega            bool
	TaxaEntregaTipo          string
	TaxaEntregaValor         float64
	TaxaEntregaBase          float64
	TaxaEntregaPorKm         float64
	ValorMinimoPedido        float64
	Tema                     string
	AceitaGuardarEntregar    bool
	SegmentoPrincipal        string
	SugestaoInteligenteAtiva bool

	Endereco  string
	Latitude  float64
	Longitude float64
	Cidade    string
	Estado    string
}

func (r *LojaRepository) AtualizarConfiguracoes(lojaID uint, cfg ConfiguracoesLoja) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"whatsapp_numero":            cfg.WhatsappNumero,
		"logo_url":                   cfg.LogoURL,
		"modo_pedido":                cfg.ModoPedido,
		"antecedencia_minima_horas":  cfg.AntecedenciaMinimaHoras,
		"horario_abertura":           cfg.HorarioAbertura,
		"horario_fechamento":         cfg.HorarioFechamento,
		"margem_fechamento_minutos":  cfg.MargemFechamentoMinutos,
		"pausado":                    cfg.Pausado,
		"mensagem_pausa":             cfg.MensagemPausa,
		"aceita_retirada":            cfg.AceitaRetirada,
		"aceita_entrega":             cfg.AceitaEntrega,
		"taxa_entrega_tipo":          cfg.TaxaEntregaTipo,
		"taxa_entrega_valor":         cfg.TaxaEntregaValor,
		"taxa_entrega_base":          cfg.TaxaEntregaBase,
		"taxa_entrega_por_km":        cfg.TaxaEntregaPorKm,
		"valor_minimo_pedido":        cfg.ValorMinimoPedido,
		"tema":                       cfg.Tema,
		"aceita_guardar_entregar":    cfg.AceitaGuardarEntregar,
		"segmento_principal":         cfg.SegmentoPrincipal,
		"sugestao_inteligente_ativa": cfg.SugestaoInteligenteAtiva,
		"endereco":                   cfg.Endereco,
		"latitude":                   cfg.Latitude,
		"longitude":                  cfg.Longitude,
		"cidade":                     cfg.Cidade,
		"estado":                     cfg.Estado,
	}).Error
}

func (r *LojaRepository) BuscarPorSlug(slug string) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("slug = ?", slug).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

func (r *LojaRepository) ListarComWhatsapp() ([]domain.Loja, error) {
	var lojas []domain.Loja
	if err := r.db.Where("whatsapp_numero != ''").Find(&lojas).Error; err != nil {
		return nil, err
	}
	return lojas, nil
}

func (r *LojaRepository) SlugExiste(slug string) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.Loja{}).Where("slug = ?", slug).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

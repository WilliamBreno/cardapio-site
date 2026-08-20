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

// AtualizarMercadoPagoContaTeste grava se a conexão da loja com o Mercado
// Pago é uma Test User (ver domain.Loja.MercadoPagoContaTeste) — chamado só
// no momento da conexão inicial (ProcessarCallback), não em toda renovação
// de token.
func (r *LojaRepository) AtualizarMercadoPagoContaTeste(lojaID uint, contaTeste bool) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("mercado_pago_conta_teste", contaTeste).Error
}

// AtualizarAvisoPagamentoNaoConfigurado registra quando o dono foi
// avisado por último de que a loja ainda não tem Mercado Pago conectado
// (ver MercadoPagoService.CriarCheckout).
func (r *LojaRepository) AtualizarAvisoPagamentoNaoConfigurado(lojaID uint, quando time.Time) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("aviso_pagamento_nao_configurado_em", quando).Error
}

// AtualizarAvisoLimitePedidos registra quando o dono de uma loja Start foi
// avisado de que passou os 30 pedidos do mês (Fase 7.3, ver
// PedidoService.CriarPorSlug).
func (r *LojaRepository) AtualizarAvisoLimitePedidos(lojaID uint, quando time.Time) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("aviso_limite_pedidos_em", quando).Error
}

// AtualizarBloqueioLimitePedidos marca que o bloqueio de novos pedidos
// (Fase 7.3) passou a valer pra essa loja — setado pela rotina agendada
// (ver LojaService.VerificarLimiteStart), não no momento do pedido.
func (r *LojaRepository) AtualizarBloqueioLimitePedidos(lojaID uint, quando time.Time) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Update("pedidos_bloqueados_desde", quando).Error
}

// ListarStartParaBloquearLimite devolve lojas Start avisadas do limite de
// pedidos há pelo menos 3 dias corridos, ainda dentro do mês em que o
// aviso foi mandado (senão a cota já resetou sozinha) e ainda não
// bloqueadas — usado pela rotina agendada da Fase 7.3.
func (r *LojaRepository) ListarStartParaBloquearLimite(agora time.Time) ([]domain.Loja, error) {
	inicioMes := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, agora.Location())
	limiteAviso := agora.Add(-domain.LimiteToleranciaBloqueioPedidos)

	var lojas []domain.Loja
	err := r.db.Where(
		"plano = ? AND aviso_limite_pedidos_em IS NOT NULL AND aviso_limite_pedidos_em >= ? AND aviso_limite_pedidos_em <= ? AND pedidos_bloqueados_desde IS NULL",
		"start", inicioMes, limiteAviso,
	).Find(&lojas).Error
	return lojas, err
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

// BuscarPorMercadoPagoPreapprovalIDPlano é usado pelo webhook de
// assinaturas (Fase 6 Parte 3) pra achar de qual loja é um evento de
// renovação/cancelamento da assinatura de PLANO — o preapproval_id não
// muda durante a vida da assinatura, então dá pra usar como chave estável
// depois da confirmação inicial (que usa AssinaturaPendente, antes da
// loja existir).
func (r *LojaRepository) BuscarPorMercadoPagoPreapprovalIDPlano(preapprovalID string) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("mercado_pago_preapproval_id_plano = ?", preapprovalID).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

// AtualizarAssinaturaPlano aplica uma assinatura de plano confirmada
// (Pro/Scale) — usado tanto na finalização do cadastro quanto, futuramente,
// numa troca de plano de loja já existente.
func (r *LojaRepository) AtualizarAssinaturaPlano(lojaID uint, plano, preapprovalID string) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"plano":                             plano,
		"mercado_pago_preapproval_id_plano": preapprovalID,
	}).Error
}

// CancelarAssinaturaPlano volta a loja pro Start — chamado quando o
// webhook de assinatura reporta a assinatura de plano cancelada ou pausada
// (pagamento recorrente falhou).
func (r *LojaRepository) CancelarAssinaturaPlano(lojaID uint) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"plano":                             "start",
		"mercado_pago_preapproval_id_plano": "",
	}).Error
}

// BuscarPorSugestaoInteligenteMercadoPagoPreapprovalID é o equivalente
// acima, só que pra assinatura da Sugestão Inteligente (independente da
// assinatura de plano).
func (r *LojaRepository) BuscarPorSugestaoInteligenteMercadoPagoPreapprovalID(preapprovalID string) (*domain.Loja, error) {
	var loja domain.Loja
	if err := r.db.Where("sugestao_inteligente_mercado_pago_preapproval_id = ?", preapprovalID).First(&loja).Error; err != nil {
		return nil, err
	}
	return &loja, nil
}

// AtivarSugestaoInteligente marca a loja como tendo contratado o recurso
// completo — chamado pelo webhook quando o preapproval é aprovado.
func (r *LojaRepository) AtivarSugestaoInteligente(lojaID uint, preapprovalID string, contratadaEm time.Time) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"sugestao_inteligente_contratada":                  true,
		"sugestao_inteligente_contratada_em":               contratadaEm,
		"sugestao_inteligente_mercado_pago_preapproval_id": preapprovalID,
	}).Error
}

// DesativarSugestaoInteligente volta a loja pro limite de 1 vínculo
// grátis — chamado tanto pelo botão "Cancelar assinatura" quanto pelo
// webhook (cancelamento ou pagamento recorrente falhou). Não apaga
// ContratadaEm (histórico de quando contratou da primeira vez) nem os
// vínculos já cadastrados — só os que passam do limite ficam
// inativos/ocultos (ver SugestaoProdutoService).
func (r *LojaRepository) DesativarSugestaoInteligente(lojaID uint) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"sugestao_inteligente_contratada":                  false,
		"sugestao_inteligente_mercado_pago_preapproval_id": "",
	}).Error
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

// LimparAssinatura remove os dados da assinatura Stripe da loja — usado
// quando um downgrade agendado pra um plano sem mensalidade (Start ou
// Basic) é aplicado (cancela a assinatura de vez).
func (r *LojaRepository) LimparAssinatura(lojaID uint, plano string) error {
	return r.db.Model(&domain.Loja{}).Where("id = ?", lojaID).Updates(map[string]interface{}{
		"plano":                  plano,
		"plano_agendado":         nil,
		"stripe_subscription_id": "",
	}).Error
}

// ConfiguracoesLoja agrupa todos os campos editáveis pelo dono no painel.
type ConfiguracoesLoja struct {
	WhatsappNumero           string
	LogoURL                  string
	BannerURL                string
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
		"banner_url":                 cfg.BannerURL,
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

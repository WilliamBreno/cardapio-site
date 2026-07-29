package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MercadoPagoHandler struct {
	mercadoPagoService *service.MercadoPagoService
	assinaturaService  *service.MercadoPagoAssinaturaService
	frontendURL        string
	cronSecret         string
}

func NewMercadoPagoHandler(mercadoPagoService *service.MercadoPagoService, assinaturaService *service.MercadoPagoAssinaturaService, frontendURL, cronSecret string) *MercadoPagoHandler {
	return &MercadoPagoHandler{
		mercadoPagoService: mercadoPagoService,
		assinaturaService:  assinaturaService,
		frontendURL:        frontendURL,
		cronSecret:         cronSecret,
	}
}

// IniciarOnboarding atende GET /admin/mercadopago/onboarding — protegida.
// Devolve a URL de autorização OAuth do Mercado Pago; o frontend
// redireciona o dono da loja pra lá.
func (h *MercadoPagoHandler) IniciarOnboarding(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	url, err := h.mercadoPagoService.IniciarOnboarding(lojaID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// Status atende GET /admin/mercadopago/status — protegida.
func (h *MercadoPagoHandler) Status(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	conectado, err := h.mercadoPagoService.StatusOnboarding(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mercadopago_conectado": conectado})
}

// Callback atende GET /admin/mercadopago/callback — rota pública (fora do
// grupo /admin autenticado): é o próprio Mercado Pago que redireciona o
// navegador do dono pra cá depois da autorização, sem nenhum header
// nosso. A identidade da loja vem do "state" assinado (ver
// MercadoPagoService.IniciarOnboarding), não de um token de sessão.
func (h *MercadoPagoHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	destino := h.frontendURL + "/admin/configuracoes"

	if erroOAuth := c.Query("error"); erroOAuth != "" {
		c.Redirect(http.StatusFound, destino+"?mercadopago_erro=1")
		return
	}
	if code == "" || state == "" {
		c.Redirect(http.StatusFound, destino+"?mercadopago_erro=1")
		return
	}

	if _, err := h.mercadoPagoService.ProcessarCallback(c.Request.Context(), code, state); err != nil {
		log.Printf("erro processando callback do Mercado Pago: %v", err)
		c.Redirect(http.StatusFound, destino+"?mercadopago_erro=1")
		return
	}

	c.Redirect(http.StatusFound, destino+"?mercadopago_conectado=1")
}

type checkoutMercadoPagoParams struct {
	ID uint `uri:"id" binding:"required"`
}

// Checkout atende POST /pedidos/:id/checkout — rota pública, no lugar do
// StripeHandler.Checkout que atendia essa rota antes (ver Fase 5.2 do
// roadmap: só a chamada foi trocada, o código da Stripe continua no
// repositório, só não é mais chamado por essa rota).
func (h *MercadoPagoHandler) Checkout(c *gin.Context) {
	var params checkoutMercadoPagoParams
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	url, err := h.mercadoPagoService.CriarCheckout(c.Request.Context(), params.ID)
	if err != nil {
		log.Printf("erro criando checkout Mercado Pago do pedido %d: %v", params.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	log.Printf("checkout Mercado Pago criado pro pedido %d: %s", params.ID, url)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// Webhook atende POST /webhooks/mercadopago — chamado pelo próprio
// Mercado Pago. O ID do recurso vem via query string (?data.id=... ou
// ?id=..., dependendo do formato da notificação — v1 "topic/id" e v2
// "type/data.id" convivem na API do Mercado Pago).
//
// IMPORTANTE (achado em teste real em 24/07/2026): pra essa integração
// (Checkout Pro via preferência), o Mercado Pago manda majoritariamente
// notificações "topic=merchant_order" — não "type=payment". Uma
// merchant_order não é o pagamento em si, é o "pedido" que agrupa uma ou
// mais tentativas de pagamento; o(s) ID(s) de pagamento de verdade vêm
// dentro da resposta de GET /merchant_orders/:id (ver
// MercadoPagoService.ProcessarMerchantOrder). Ignorar "merchant_order"
// (como o código fazia antes) significava nunca processar pagamento
// nenhum de verdade — não é opcional tratar os dois tipos.
//
// IMPORTANTE 2 (achado configurando o painel real em 28/07/2026): a
// aplicação do Mercado Pago só aceita UMA URL de notificação por
// ambiente — não dá pra ter uma URL separada pra eventos de assinatura
// (Fase 6 Parte 3, "Planos e assinaturas"). Por isso esse mesmo endpoint
// também trata notificações de preapproval (tipo ainda não confirmado
// contra um evento real — o código aceita qualquer tipo contendo
// "subscription" ou "preapproval", e loga o tipo exato recebido pra
// facilitar o ajuste se o nome vier diferente do esperado, mesma lição
// do merchant_order acima). A assinatura (x-signature) é validada com o
// MESMO secret do checkout de pedido, já que é a mesma URL/aplicação.
func (h *MercadoPagoHandler) Webhook(c *gin.Context) {
	tipo := c.Query("type")
	if tipo == "" {
		tipo = c.Query("topic")
	}

	dataID := c.Query("data.id")
	if dataID == "" {
		dataID = c.Query("id")
	}
	if dataID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "notificação sem id"})
		return
	}

	log.Printf("webhook Mercado Pago recebido — type/topic=%q id=%s", tipo, dataID)

	ehAssinatura := strings.Contains(tipo, "subscription") || strings.Contains(tipo, "preapproval")
	if tipo != "payment" && tipo != "merchant_order" && !ehAssinatura {
		log.Printf("webhook Mercado Pago ignorado — type/topic=%q ainda não tratado (id=%s)", tipo, dataID)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	signature := c.GetHeader("x-signature")
	requestID := c.GetHeader("x-request-id")
	if err := h.mercadoPagoService.ValidarAssinaturaWebhook(signature, requestID, dataID); err != nil {
		log.Printf("erro validando assinatura do webhook Mercado Pago (id=%s): %v", dataID, err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": "notificação inválida"})
		return
	}
	log.Printf("assinatura do webhook Mercado Pago validada (id=%s)", dataID)

	var err error
	switch {
	case tipo == "merchant_order":
		err = h.mercadoPagoService.ProcessarMerchantOrder(c.Request.Context(), dataID)
	case ehAssinatura:
		err = h.assinaturaService.ProcessarWebhook(c.Request.Context(), dataID)
	default:
		err = h.mercadoPagoService.ProcessarNotificacaoPagamento(c.Request.Context(), dataID)
	}
	if err != nil {
		log.Printf("erro processando notificação %s (id=%s) do Mercado Pago: %v", tipo, dataID, err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": "não foi possível processar a notificação"})
		return
	}

	log.Printf("notificação %s (id=%s) do Mercado Pago processada com sucesso", tipo, dataID)
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// RenovarTokens atende POST /mercadopago/renovar-tokens — chamado por um
// cron externo (mesmo padrão de /relatorio/semanal), protegido pelo
// mesmo header X-Cron-Secret.
func (h *MercadoPagoHandler) RenovarTokens(c *gin.Context) {
	if h.cronSecret != "" && c.GetHeader("X-Cron-Secret") != h.cronSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "não autorizado"})
		return
	}

	renovadas, erros := h.mercadoPagoService.RenovarTokensExpirando(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"renovadas": renovadas,
		"erros":     erros,
	})
}

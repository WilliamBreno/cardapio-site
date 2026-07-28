package handler

import (
	"log"
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// AssinaturaHandler cobre a Fase 6 Parte 3: cobrança recorrente
// consolidada (plano da loja + Sugestão Inteligente) via Mercado Pago
// Subscriptions, na própria conta da Drenux.
type AssinaturaHandler struct {
	assinaturaService *service.MercadoPagoAssinaturaService
}

func NewAssinaturaHandler(assinaturaService *service.MercadoPagoAssinaturaService) *AssinaturaHandler {
	return &AssinaturaHandler{assinaturaService: assinaturaService}
}

// AssinarSugestaoInteligente atende POST /admin/sugestao-inteligente/assinatura
// — protegida. Devolve a URL do checkout de assinatura do Mercado Pago
// pro frontend redirecionar.
func (h *AssinaturaHandler) AssinarSugestaoInteligente(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	url, err := h.assinaturaService.CriarCheckoutSugestaoInteligente(c.Request.Context(), lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// CancelarSugestaoInteligente atende DELETE /admin/sugestao-inteligente/assinatura
// — protegida. Cancela o preapproval no Mercado Pago e desativa o
// recurso na loja.
func (h *AssinaturaHandler) CancelarSugestaoInteligente(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	if err := h.assinaturaService.CancelarSugestaoInteligente(c.Request.Context(), lojaID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

// Webhook atende POST /webhooks/mercadopago/assinaturas — endpoint novo e
// separado do webhook de pagamento de pedido já existente
// (/webhooks/mercadopago), de propósito: evita mexer em algo que já
// funciona, e usa um secret de validação próprio (ver
// MercadoPagoAssinaturaService.ValidarAssinaturaWebhook). O Mercado Pago
// manda type=subscription_preapproval com o ID do preapproval em
// data.id.
func (h *AssinaturaHandler) Webhook(c *gin.Context) {
	tipo := c.Query("type")
	dataID := c.Query("data.id")
	if dataID == "" {
		dataID = c.Query("id")
	}
	if dataID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "notificação sem id"})
		return
	}

	log.Printf("webhook Mercado Pago (assinaturas) recebido — type=%q id=%s", tipo, dataID)

	if tipo != "" && tipo != "subscription_preapproval" {
		log.Printf("webhook Mercado Pago (assinaturas) ignorado — type=%q ainda não tratado (id=%s)", tipo, dataID)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	signature := c.GetHeader("x-signature")
	requestID := c.GetHeader("x-request-id")
	if err := h.assinaturaService.ValidarAssinaturaWebhook(signature, requestID, dataID); err != nil {
		log.Printf("erro validando assinatura do webhook Mercado Pago (assinaturas, id=%s): %v", dataID, err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": "notificação inválida"})
		return
	}

	if err := h.assinaturaService.ProcessarWebhook(c.Request.Context(), dataID); err != nil {
		log.Printf("erro processando notificação de assinatura (id=%s) do Mercado Pago: %v", dataID, err)
		c.JSON(http.StatusBadRequest, gin.H{"erro": "não foi possível processar a notificação"})
		return
	}

	log.Printf("notificação de assinatura (id=%s) do Mercado Pago processada com sucesso", dataID)
	c.JSON(http.StatusOK, gin.H{"received": true})
}

package handler

import (
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

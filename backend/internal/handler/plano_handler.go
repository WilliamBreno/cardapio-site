package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type PlanoHandler struct {
	stripeService                *service.StripeService
	mercadoPagoAssinaturaService *service.MercadoPagoAssinaturaService
}

func NewPlanoHandler(stripeService *service.StripeService, mercadoPagoAssinaturaService *service.MercadoPagoAssinaturaService) *PlanoHandler {
	return &PlanoHandler{stripeService: stripeService, mercadoPagoAssinaturaService: mercadoPagoAssinaturaService}
}

type checkoutAssinaturaRequest struct {
	Plano string `json:"plano" binding:"required,oneof=pro scale"`
}

// CriarCheckout atende POST /planos/checkout — rota pública, chamada
// quando o cliente clica em "Escolher Pro/Scale" na página de planos.
// Migrado da Stripe pro Mercado Pago (Fase 6 Parte 3, consolidando com a
// assinatura da Sugestão Inteligente no mesmo mecanismo) — o código da
// Stripe pra isso continua no repositório (stripeService.CriarCheckoutAssinatura),
// só parou de ser chamado por aqui, mesmo padrão já usado na migração do
// checkout de pedido (Fase 5.2).
func (h *PlanoHandler) CriarCheckout(c *gin.Context) {
	var req checkoutAssinaturaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	url, err := h.mercadoPagoAssinaturaService.CriarCheckoutPlano(c.Request.Context(), req.Plano)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// VerificarToken atende GET /planos/verificar-token?token=XXX — usado
// pela tela "/cadastro/finalizar" pra confirmar que o pagamento já foi
// feito e pré-preencher email/plano antes do cliente completar o
// cadastro. Como a confirmação do Mercado Pago é assíncrona (webhook), o
// registro pode existir mas ainda não estar confirmado — devolve 202
// nesse caso pra o frontend saber que precisa tentar de novo em vez de
// mostrar erro (ver FinalizarCadastro.tsx).
func (h *PlanoHandler) VerificarToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "token não informado"})
		return
	}

	assinatura, err := h.mercadoPagoAssinaturaService.VerificarToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "link inválido ou já utilizado"})
		return
	}
	if !assinatura.Confirmada {
		c.JSON(http.StatusAccepted, gin.H{"pendente": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email": assinatura.Email,
		"plano": assinatura.Plano,
		"token": assinatura.Token,
	})
}

type mudarPlanoRequest struct {
	Plano string `json:"plano" binding:"required,oneof=start pro scale"`
}

// VerificarSessao atende GET /planos/verificar-sessao?session_id=XXX —
// usado no redirecionamento direto da Stripe, logo após o pagamento.
// Pode retornar 404 nos primeiros segundos (webhook ainda processando)
// — o frontend tenta de novo por um tempo curto antes de desistir.
func (h *PlanoHandler) VerificarSessao(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "session_id não informado"})
		return
	}

	assinatura, err := h.stripeService.BuscarAssinaturaPendentePorSessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ainda processando o pagamento, tenta de novo em instantes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email": assinatura.Email,
		"plano": assinatura.Plano,
		"token": assinatura.Token,
	})
}

// MudarPlano atende POST /admin/plano/mudar — protegida, chamada pelo
// dono da loja na tela "Meu Plano".
func (h *PlanoHandler) MudarPlano(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	var req mudarPlanoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	resultado, err := h.stripeService.MudarPlano(c.Request.Context(), lojaID, req.Plano)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": resultado.CheckoutURL,
		"imediato":     resultado.Imediato,
	})
}

// CancelarMudancaAgendada atende DELETE /admin/plano/agendamento —
// desfaz um downgrade agendado, mantendo a loja no plano atual.
func (h *PlanoHandler) CancelarMudancaAgendada(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	if err := h.stripeService.CancelarMudancaAgendada(lojaID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

package handler

import (
	"fmt"
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AfiliadoHandler struct {
	afiliadoService *service.AfiliadoService
	repasseService  *service.RepasseAfiliadoService
	frontendURL     string
}

func NewAfiliadoHandler(afiliadoService *service.AfiliadoService, repasseService *service.RepasseAfiliadoService, frontendURL string) *AfiliadoHandler {
	return &AfiliadoHandler{afiliadoService: afiliadoService, repasseService: repasseService, frontendURL: frontendURL}
}

type afiliadoLoginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required"`
}

// Login atende POST /afiliados/login — rota pública, separada do login
// de dono de loja.
func (h *AfiliadoHandler) Login(c *gin.Context) {
	var req afiliadoLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	token, err := h.afiliadoService.Login(req.Email, req.Senha)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Dashboard atende GET /afiliado/dashboard — protegida pelo token do
// próprio afiliado (claim "afiliado_id").
func (h *AfiliadoHandler) Dashboard(c *gin.Context) {
	afiliadoID := c.GetUint("afiliado_id")
	dados, err := h.afiliadoService.Dashboard(afiliadoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dados)
}

// Extrato atende GET /afiliado/repasses — protegida pelo token do
// próprio afiliado. Mostra o histórico de comissões de pedidos pagos via
// Mercado Pago (repasse manual, ver Fase 5.5) e quanto ainda está
// pendente — separado do "total_ganho" do Dashboard, que soma tudo
// (Stripe + Mercado Pago) já efetivamente repassado.
func (h *AfiliadoHandler) Extrato(c *gin.Context) {
	afiliadoID := c.GetUint("afiliado_id")
	extrato, err := h.repasseService.ExtratoAfiliado(afiliadoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, extrato)
}

// IniciarOnboarding atende POST /afiliado/stripe/onboarding.
func (h *AfiliadoHandler) IniciarOnboarding(c *gin.Context) {
	afiliadoID := c.GetUint("afiliado_id")
	returnURL := fmt.Sprintf("%s/afiliado/dashboard", h.frontendURL)
	link, err := h.afiliadoService.IniciarOnboarding(c.Request.Context(), afiliadoID, returnURL, returnURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": link})
}

type afiliadoEsqueciSenhaRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EsqueciSenha atende POST /afiliados/esqueci-senha — rota pública.
func (h *AfiliadoHandler) EsqueciSenha(c *gin.Context) {
	var req afiliadoEsqueciSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	_ = h.afiliadoService.EsqueciSenha(req.Email)
	c.JSON(http.StatusOK, gin.H{"mensagem": "se o email existir, você receberá um link de redefinição"})
}

type afiliadoRedefinirSenhaRequest struct {
	Token string `json:"token" binding:"required"`
	Senha string `json:"senha" binding:"required,min=6"`
}

// RedefinirSenha atende POST /afiliados/redefinir-senha — rota pública.
func (h *AfiliadoHandler) RedefinirSenha(c *gin.Context) {
	var req afiliadoRedefinirSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	if err := h.afiliadoService.RedefinirSenha(req.Token, req.Senha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mensagem": "senha redefinida com sucesso"})
}

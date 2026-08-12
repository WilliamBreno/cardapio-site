package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type cadastroRequest struct {
	Nome              string `json:"nome" binding:"required"`
	Email             string `json:"email" binding:"required,email"`
	Senha             string `json:"senha" binding:"required,min=6"`
	NomeLoja          string `json:"nome_loja" binding:"required"`
	SegmentoPrincipal string `json:"segmento_principal" binding:"required,oneof=alimenticio mercadoria"`
	CodigoAfiliado    string `json:"codigo_afiliado"`
	TokenAssinatura   string `json:"token_assinatura"`
	// Plano opcional: "start" (padrão, se vazio) ou "basic" — os dois são
	// planos sem mensalidade, então dá pra aplicar direto no cadastro, sem
	// checkout. Pro/Scale só chegam aqui via TokenAssinatura (pagamento já
	// confirmado antes do cadastro).
	Plano string `json:"plano" binding:"omitempty,oneof=start basic"`
}

func (h *AuthHandler) Cadastrar(c *gin.Context) {
	var req cadastroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	token, err := h.authService.Cadastrar(service.CadastroInput{
		Nome:              req.Nome,
		Email:             req.Email,
		Senha:             req.Senha,
		NomeLoja:          req.NomeLoja,
		SegmentoPrincipal: req.SegmentoPrincipal,
		CodigoAfiliado:    req.CodigoAfiliado,
		TokenAssinatura:   req.TokenAssinatura,
		Plano:             req.Plano,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

type loginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	token, err := h.authService.Login(req.Email, req.Senha)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

type esqueciSenhaRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) EsqueciSenha(c *gin.Context) {
	var req esqueciSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	// Sempre responde sucesso, mesmo se o email não existir — a lógica de
	// não vazar informação já está no service.
	_ = h.authService.EsqueciSenha(req.Email)
	c.JSON(http.StatusOK, gin.H{"mensagem": "se o email existir, você receberá um link de redefinição"})
}

type redefinirSenhaRequest struct {
	Token string `json:"token" binding:"required"`
	Senha string `json:"senha" binding:"required,min=6"`
}

func (h *AuthHandler) RedefinirSenha(c *gin.Context) {
	var req redefinirSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.authService.RedefinirSenha(req.Token, req.Senha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensagem": "senha redefinida com sucesso"})
}

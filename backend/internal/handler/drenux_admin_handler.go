package handler

import (
	"net/http"
	"strconv"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// DrenuxAdminHandler atende as rotas internas /drenux/* — controle manual
// de repasse de comissão de afiliado (Fase 5.5 do roadmap). Protegido por
// middleware.DrenuxAdminRequired, não por um login próprio (ver decisão
// registrada em docs/plano-melhorias-drenux.md).
type DrenuxAdminHandler struct {
	repasseService  *service.RepasseAfiliadoService
	afiliadoService *service.AfiliadoService
}

func NewDrenuxAdminHandler(repasseService *service.RepasseAfiliadoService, afiliadoService *service.AfiliadoService) *DrenuxAdminHandler {
	return &DrenuxAdminHandler{repasseService: repasseService, afiliadoService: afiliadoService}
}

type criarAfiliadoRequest struct {
	Nome  string `json:"nome" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required,min=6"`
	// ComissaoPercentual é a fração da taxa de plataforma que ESSE
	// afiliado recebe (ex: 0.376 = 37,6%) — definida no cadastro, negociada
	// por afiliado (ver domain.Afiliado.ComissaoPercentual, Fase 5.5).
	ComissaoPercentual float64 `json:"comissao_percentual" binding:"required,gt=0,lte=1"`
}

// CriarAfiliado atende POST /drenux/afiliados — não existe autocadastro
// de afiliado (ver domain.Afiliado), então essa é a única forma de criar
// uma conta hoje, restrita a quem tem o secret de /drenux/*.
func (h *DrenuxAdminHandler) CriarAfiliado(c *gin.Context) {
	var req criarAfiliadoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	afiliado, err := h.afiliadoService.CriarAfiliado(req.Nome, req.Email, req.Senha, req.ComissaoPercentual)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                  afiliado.ID,
		"nome":                afiliado.Nome,
		"email":               afiliado.Email,
		"codigo":              afiliado.Codigo,
		"comissao_percentual": afiliado.ComissaoPercentual,
	})
}

// ListarAfiliados atende GET /drenux/afiliados — visão geral: TODOS os
// afiliados cadastrados, mesmo sem nenhum lançamento ainda, com o total
// pago e pendente de cada um.
func (h *DrenuxAdminHandler) ListarAfiliados(c *gin.Context) {
	afiliados, err := h.repasseService.ListarTodosComTotais()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, afiliados)
}

// DetalheAfiliado atende GET /drenux/afiliados/:id/repasses — extrato
// completo (pendente + pago) de um afiliado específico.
func (h *DrenuxAdminHandler) DetalheAfiliado(c *gin.Context) {
	afiliadoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	repasses, err := h.repasseService.DetalheAfiliado(uint(afiliadoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, repasses)
}

type marcarComoPagoRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// MarcarComoPago atende POST /drenux/repasses/marcar-pago — chamado
// depois do repasse via Pix ser feito manualmente, fora do sistema. Só
// registra a confirmação, não movimenta dinheiro nenhum.
func (h *DrenuxAdminHandler) MarcarComoPago(c *gin.Context) {
	var req marcarComoPagoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	total, err := h.repasseService.MarcarComoPago(req.IDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"marcados": total})
}

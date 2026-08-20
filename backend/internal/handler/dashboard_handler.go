package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ─── Dashboard ────────────────────────────────────────────────────────────────

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

func (h *DashboardHandler) Dados(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	data, err := h.dashboardService.BuscarDados(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Periodo atende GET /admin/dashboard/periodo?tipo=dia|semana|mes&data=AAAA-MM-DD
// (Fase 10.5) — resumo de um período exato escolhido pelo dono, usado pra
// montar o relatório enviado via WhatsApp (link wa.me, montado no
// frontend). Backend só devolve o resumo, não envia nada.
func (h *DashboardHandler) Periodo(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	tipo := c.Query("tipo")
	if tipo != "dia" && tipo != "semana" && tipo != "mes" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "tipo precisa ser dia, semana ou mes"})
		return
	}

	data, err := time.Parse("2006-01-02", c.Query("data"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "data inválida, use o formato AAAA-MM-DD"})
		return
	}

	resumo, err := h.dashboardService.BuscarResumoPeriodo(lojaID, tipo, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resumo)
}

// ─── Fotos de Produto ─────────────────────────────────────────────────────────

type FotoHandler struct {
	fotoRepo    *repository.FotoRepository
	produtoRepo *repository.ProdutoRepository
}

func NewFotoHandler(db *gorm.DB) *FotoHandler {
	return &FotoHandler{
		fotoRepo:    repository.NewFotoRepository(db),
		produtoRepo: repository.NewProdutoRepository(db),
	}
}

func (h *FotoHandler) validarDono(lojaID, produtoID uint) bool {
	produto, err := h.produtoRepo.BuscarPorID(produtoID)
	return err == nil && produto.LojaID == lojaID
}

func (h *FotoHandler) Adicionar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	produtoID, err := strconv.ParseUint(c.Param("produtoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "produto_id inválido"})
		return
	}
	if !h.validarDono(lojaID, uint(produtoID)) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "produto não encontrado"})
		return
	}

	var req struct {
		URL   string `json:"url" binding:"required"`
		Ordem int    `json:"ordem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	foto := domain.FotoProduto{
		ProdutoID: uint(produtoID),
		URL:       req.URL,
		Ordem:     req.Ordem,
	}
	if err := h.fotoRepo.Adicionar(&foto); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, foto)
}

// Reordenar atende PUT /admin/fotos/:produtoId/reordenar — recebe a lista
// de IDs na nova ordem desejada e aplica; a primeira da lista vira a
// "principal" (menor Ordem), consistente com o critério já usado em
// ListarPorProduto (Order("ordem, id")).
func (h *FotoHandler) Reordenar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	produtoID, err := strconv.ParseUint(c.Param("produtoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "produto_id inválido"})
		return
	}
	if !h.validarDono(lojaID, uint(produtoID)) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "produto não encontrado"})
		return
	}

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.fotoRepo.ReordenarTodas(uint(produtoID), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FotoHandler) Deletar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	produtoID, err := strconv.ParseUint(c.Param("produtoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "produto_id inválido"})
		return
	}
	fotoID, err := strconv.ParseUint(c.Param("fotoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "foto_id inválido"})
		return
	}
	if !h.validarDono(lojaID, uint(produtoID)) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "produto não encontrado"})
		return
	}

	if err := h.fotoRepo.Deletar(uint(fotoID), uint(produtoID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

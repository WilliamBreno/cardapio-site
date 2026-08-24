package handler

import (
	"net/http"
	"strconv"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FotoBannerHandler gerencia o carrossel de fotos do banner do cardápio
// público (redesign de 24/08/2026, substitui o antigo Loja.BannerURL
// único). Sempre escopado pelo loja_id do próprio token — diferente de
// FotoHandler (fotos de produto), não precisa validar dono de um recurso
// à parte, porque o banner já pertence direto à loja autenticada.
type FotoBannerHandler struct {
	repo *repository.FotoBannerRepository
}

func NewFotoBannerHandler(db *gorm.DB) *FotoBannerHandler {
	return &FotoBannerHandler{repo: repository.NewFotoBannerRepository(db)}
}

// Listar atende GET /admin/banners.
func (h *FotoBannerHandler) Listar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	fotos, err := h.repo.ListarPorLoja(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fotos)
}

// Adicionar atende POST /admin/banners.
func (h *FotoBannerHandler) Adicionar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	var req struct {
		URL   string `json:"url" binding:"required"`
		Ordem int    `json:"ordem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	foto := domain.FotoBanner{LojaID: lojaID, URL: req.URL, Ordem: req.Ordem}
	if err := h.repo.Adicionar(&foto); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, foto)
}

// Reordenar atende PUT /admin/banners/reordenar — recebe a lista de IDs
// na nova ordem desejada e aplica, mesmo padrão de FotoHandler.Reordenar.
func (h *FotoBannerHandler) Reordenar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.repo.ReordenarTodas(lojaID, req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Deletar atende DELETE /admin/banners/:fotoId.
func (h *FotoBannerHandler) Deletar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	fotoID, err := strconv.ParseUint(c.Param("fotoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "foto_id inválido"})
		return
	}

	if err := h.repo.Deletar(uint(fotoID), lojaID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

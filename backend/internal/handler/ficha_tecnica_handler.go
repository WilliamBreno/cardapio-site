package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// FichaTecnicaHandler implementa a ficha técnica de produto + CMV
// automático (Fase 9.1 do roadmap) — exclusivo do plano Scale, mesmo gate
// de InsumoHandler.
type FichaTecnicaHandler struct {
	fichaTecnicaService *service.FichaTecnicaService
	lojaRepo            *repository.LojaRepository
}

func NewFichaTecnicaHandler(fichaTecnicaService *service.FichaTecnicaService, lojaRepo *repository.LojaRepository) *FichaTecnicaHandler {
	return &FichaTecnicaHandler{fichaTecnicaService: fichaTecnicaService, lojaRepo: lojaRepo}
}

func (h *FichaTecnicaHandler) carregarLojaScale(c *gin.Context) (*domain.Loja, bool) {
	lojaID := c.GetUint("loja_id")
	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return nil, false
	}
	if !controleEstoqueCompletoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "ficha técnica e insumos são um recurso do plano Scale"})
		return nil, false
	}
	return loja, true
}

// Buscar atende GET /admin/produtos/:id/ficha-tecnica
func (h *FichaTecnicaHandler) Buscar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	produtoID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	ficha, err := h.fichaTecnicaService.Buscar(loja.ID, produtoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ficha)
}

type fichaTecnicaItemRequest struct {
	InsumoID   uint    `json:"insumo_id" binding:"required"`
	Quantidade float64 `json:"quantidade" binding:"required,gt=0"`
}

type fichaTecnicaRequest struct {
	Itens []fichaTecnicaItemRequest `json:"itens"`
}

// Salvar atende PUT /admin/produtos/:id/ficha-tecnica — substitui a
// lista de itens por completo (lista vazia remove a ficha técnica do
// produto, ele volta a usar o estoque simples de sempre).
func (h *FichaTecnicaHandler) Salvar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	produtoID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	var req fichaTecnicaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	itens := make([]service.FichaTecnicaItemInput, len(req.Itens))
	for i, item := range req.Itens {
		itens[i] = service.FichaTecnicaItemInput{InsumoID: item.InsumoID, Quantidade: item.Quantidade}
	}

	ficha, err := h.fichaTecnicaService.Salvar(loja.ID, produtoID, itens)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ficha)
}

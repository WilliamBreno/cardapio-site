package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// InsumoHandler implementa o CRUD de insumo (Fase 9.1 do roadmap, ver
// docs/plano-melhorias-drenux.md) — exclusivo do plano Scale, mesmo gate
// de controleEstoqueCompletoDisponivel já usado em EstoqueHandler (nível
// 2 do controle de estoque, Fase 8).
type InsumoHandler struct {
	insumoService *service.InsumoService
	lojaRepo      *repository.LojaRepository
}

func NewInsumoHandler(insumoService *service.InsumoService, lojaRepo *repository.LojaRepository) *InsumoHandler {
	return &InsumoHandler{insumoService: insumoService, lojaRepo: lojaRepo}
}

func (h *InsumoHandler) carregarLojaScale(c *gin.Context) (*domain.Loja, bool) {
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

// Listar atende GET /admin/insumos
func (h *InsumoHandler) Listar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	insumos, err := h.insumoService.Listar(loja.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, insumos)
}

type insumoRequest struct {
	Nome               string   `json:"nome" binding:"required"`
	UnidadeCompra      string   `json:"unidade_compra" binding:"required"`
	UnidadeUso         string   `json:"unidade_uso" binding:"required"`
	FatorConversao     float64  `json:"fator_conversao" binding:"required,gt=0"`
	CustoUnidadeCompra float64  `json:"custo_unidade_compra"`
	EstoqueAtual       *float64 `json:"estoque_atual"`
	EstoqueAlerta      *float64 `json:"estoque_alerta"`
}

func (req insumoRequest) paraInput() service.InsumoInput {
	return service.InsumoInput{
		Nome:               req.Nome,
		UnidadeCompra:      req.UnidadeCompra,
		UnidadeUso:         req.UnidadeUso,
		FatorConversao:     req.FatorConversao,
		CustoUnidadeCompra: req.CustoUnidadeCompra,
		EstoqueAtual:       req.EstoqueAtual,
		EstoqueAlerta:      req.EstoqueAlerta,
	}
}

// Criar atende POST /admin/insumos
func (h *InsumoHandler) Criar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	var req insumoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	insumo, err := h.insumoService.Criar(loja.ID, req.paraInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, insumo)
}

// Atualizar atende PUT /admin/insumos/:id
func (h *InsumoHandler) Atualizar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	insumoID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	var req insumoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	insumo, err := h.insumoService.Atualizar(loja.ID, insumoID, req.paraInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, insumo)
}

// Deletar atende DELETE /admin/insumos/:id
func (h *InsumoHandler) Deletar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	insumoID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	if err := h.insumoService.Deletar(loja.ID, insumoID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

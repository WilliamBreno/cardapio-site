package handler

import (
	"net/http"
	"strconv"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// EstoqueHandler implementa a Fase 8 do roadmap (ver
// docs/plano-melhorias-drenux.md): controle de estoque em dois níveis —
// relatório (Pro e Scale, só leitura) e controle completo com reposição
// guiada + histórico de movimentação (Scale, exclusivo).
type EstoqueHandler struct {
	estoqueService *service.EstoqueService
	lojaRepo       *repository.LojaRepository
}

func NewEstoqueHandler(estoqueService *service.EstoqueService, lojaRepo *repository.LojaRepository) *EstoqueHandler {
	return &EstoqueHandler{estoqueService: estoqueService, lojaRepo: lojaRepo}
}

// relatorioEstoqueDisponivel diz se o plano da loja inclui ao menos o
// relatório de estoque (nível 1, Fase 8) — Pro e Scale.
func relatorioEstoqueDisponivel(plano string) bool {
	return plano == "pro" || plano == "scale"
}

// controleEstoqueCompletoDisponivel diz se o plano inclui reposição
// guiada e histórico de movimentação (nível 2, Fase 8) — Scale.
func controleEstoqueCompletoDisponivel(plano string) bool {
	return plano == "scale"
}

func (h *EstoqueHandler) carregarLoja(c *gin.Context) (*domain.Loja, bool) {
	lojaID := c.GetUint("loja_id")
	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return nil, false
	}
	return loja, true
}

// Relatorio atende GET /admin/estoque — nível 1 (Pro e Scale).
func (h *EstoqueHandler) Relatorio(c *gin.Context) {
	loja, ok := h.carregarLoja(c)
	if !ok {
		return
	}
	if !relatorioEstoqueDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "relatório de estoque disponível a partir do plano Pro"})
		return
	}

	itens, err := h.estoqueService.Relatorio(loja.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, itens)
}

// Movimentacoes atende GET /admin/estoque/movimentacoes?produto_id=... —
// nível 2 (Scale).
func (h *EstoqueHandler) Movimentacoes(c *gin.Context) {
	loja, ok := h.carregarLoja(c)
	if !ok {
		return
	}
	if !controleEstoqueCompletoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "histórico de movimentação disponível no plano Scale"})
		return
	}

	var produtoID uint
	if v := c.Query("produto_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			produtoID = uint(id)
		}
	}

	movimentacoes, err := h.estoqueService.Movimentacoes(loja.ID, produtoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, movimentacoes)
}

type reporEstoqueRequest struct {
	ProdutoID  uint   `json:"produto_id" binding:"required"`
	VariacaoID *uint  `json:"variacao_id"`
	Quantidade int    `json:"quantidade" binding:"required"`
	Motivo     string `json:"motivo"`
}

// Repor atende POST /admin/estoque/repor — nível 2 (Scale).
func (h *EstoqueHandler) Repor(c *gin.Context) {
	loja, ok := h.carregarLoja(c)
	if !ok {
		return
	}
	if !controleEstoqueCompletoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "reposição de estoque disponível no plano Scale"})
		return
	}

	var req reporEstoqueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	novoValor, err := h.estoqueService.Repor(loja.ID, service.ReporInput{
		ProdutoID:  req.ProdutoID,
		VariacaoID: req.VariacaoID,
		Quantidade: req.Quantidade,
		Motivo:     req.Motivo,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"estoque_atual": novoValor})
}

type ajustarEstoqueRequest struct {
	ProdutoID  uint   `json:"produto_id" binding:"required"`
	VariacaoID *uint  `json:"variacao_id"`
	NovoValor  int    `json:"novo_valor"`
	Motivo     string `json:"motivo" binding:"required"`
}

// Ajustar atende POST /admin/estoque/ajustar — nível 2 (Scale).
func (h *EstoqueHandler) Ajustar(c *gin.Context) {
	loja, ok := h.carregarLoja(c)
	if !ok {
		return
	}
	if !controleEstoqueCompletoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "ajuste de estoque disponível no plano Scale"})
		return
	}

	var req ajustarEstoqueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	novoValor, err := h.estoqueService.Ajustar(loja.ID, service.AjustarInput{
		ProdutoID:  req.ProdutoID,
		VariacaoID: req.VariacaoID,
		NovoValor:  req.NovoValor,
		Motivo:     req.Motivo,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"estoque_atual": novoValor})
}

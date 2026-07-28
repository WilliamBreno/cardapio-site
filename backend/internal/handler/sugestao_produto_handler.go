package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type SugestaoProdutoHandler struct {
	sugestaoService *service.SugestaoProdutoService
	lojaRepo        *repository.LojaRepository
}

func NewSugestaoProdutoHandler(sugestaoService *service.SugestaoProdutoService, lojaRepo *repository.LojaRepository) *SugestaoProdutoHandler {
	return &SugestaoProdutoHandler{sugestaoService: sugestaoService, lojaRepo: lojaRepo}
}

// Listar atende GET /admin/sugestoes-produto
func (h *SugestaoProdutoHandler) Listar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	sugestoes, err := h.sugestaoService.Listar(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sugestoes)
}

type sugestaoProdutoRequest struct {
	ProdutoOrigemID   uint    `json:"produto_origem_id" binding:"required"`
	ProdutoSugeridoID uint    `json:"produto_sugerido_id" binding:"required"`
	TipoDesconto      string  `json:"tipo_desconto"`
	ValorDesconto     float64 `json:"valor_desconto"`
}

// Criar atende POST /admin/sugestoes-produto
func (h *SugestaoProdutoHandler) Criar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	var req sugestaoProdutoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	sugestao, err := h.sugestaoService.Criar(lojaID, service.SugestaoProdutoInput{
		ProdutoOrigemID:   req.ProdutoOrigemID,
		ProdutoSugeridoID: req.ProdutoSugeridoID,
		TipoDesconto:      req.TipoDesconto,
		ValorDesconto:     req.ValorDesconto,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sugestao)
}

// Deletar atende DELETE /admin/sugestoes-produto/:id
func (h *SugestaoProdutoHandler) Deletar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	sugestaoID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	if err := h.sugestaoService.Deletar(lojaID, sugestaoID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// SugestoesCarrinho atende GET /lojas/:slug/sugestoes-carrinho?produtos=1,2,3
// — rota pública, chamada na revisão do carrinho antes do cliente
// finalizar. Devolve lista vazia (não erro) se a loja não tiver a
// Sugestão Inteligente contratada/ativa — assim o frontend não precisa
// checar o flag antes de chamar, só decide se mostra a seção com base no
// que voltou.
func (h *SugestaoProdutoHandler) SugestoesCarrinho(c *gin.Context) {
	slug := c.Param("slug")

	loja, err := h.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}

	if !loja.SugestaoInteligenteAtiva {
		c.JSON(http.StatusOK, []service.SugestaoCarrinhoItem{})
		return
	}

	produtosParam := strings.TrimSpace(c.Query("produtos"))
	if produtosParam == "" {
		c.JSON(http.StatusOK, []service.SugestaoCarrinhoItem{})
		return
	}

	partes := strings.Split(produtosParam, ",")
	produtoIDs := make([]uint, 0, len(partes))
	for _, parte := range partes {
		id, err := strconv.ParseUint(strings.TrimSpace(parte), 10, 64)
		if err != nil {
			continue
		}
		produtoIDs = append(produtoIDs, uint(id))
	}

	sugestoes, err := h.sugestaoService.MontarSugestoesCarrinho(loja.ID, produtoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sugestoes)
}

package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ComboHandler struct {
	comboService *service.ComboService
}

func NewComboHandler(comboService *service.ComboService) *ComboHandler {
	return &ComboHandler{comboService: comboService}
}

// Listar atende GET /admin/combos — mostra todo combo da loja, disponível
// ou não (o dono precisa ver tudo pra poder reativar).
func (h *ComboHandler) Listar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	combos, err := h.comboService.Listar(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, combos)
}

type comboItemRequest struct {
	ProdutoID  uint `json:"produto_id" binding:"required"`
	Quantidade int  `json:"quantidade" binding:"required,gt=0"`
}

type comboRequest struct {
	Nome       string             `json:"nome" binding:"required"`
	Descricao  string             `json:"descricao"`
	FotoURL    string             `json:"foto_url"`
	Preco      float64            `json:"preco" binding:"required,gt=0"`
	Disponivel bool               `json:"disponivel"`
	Ordem      int                `json:"ordem"`
	Itens      []comboItemRequest `json:"itens" binding:"required,min=1,dive"`
}

func (req comboRequest) paraInput() service.ComboInput {
	itens := make([]service.ComboItemInput, len(req.Itens))
	for i, item := range req.Itens {
		itens[i] = service.ComboItemInput{
			ProdutoID:  item.ProdutoID,
			Quantidade: item.Quantidade,
		}
	}
	return service.ComboInput{
		Nome:       req.Nome,
		Descricao:  req.Descricao,
		FotoURL:    req.FotoURL,
		Preco:      req.Preco,
		Disponivel: req.Disponivel,
		Ordem:      req.Ordem,
		Itens:      itens,
	}
}

// Criar atende POST /admin/combos
func (h *ComboHandler) Criar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	var req comboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	combo, err := h.comboService.Criar(lojaID, req.paraInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, combo)
}

// Atualizar atende PUT /admin/combos/:id
func (h *ComboHandler) Atualizar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	comboID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	var req comboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	combo, err := h.comboService.Atualizar(lojaID, comboID, req.paraInput())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, combo)
}

// Deletar atende DELETE /admin/combos/:id
func (h *ComboHandler) Deletar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	comboID, err := parseIDParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	if err := h.comboService.Deletar(lojaID, comboID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

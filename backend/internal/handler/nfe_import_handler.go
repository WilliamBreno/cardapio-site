package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// NFeImportHandler implementa a importação de insumos via XML de NF-e
// (Fase 9.2 do roadmap) — exclusivo do plano Scale, mesmo gate de
// InsumoHandler/FichaTecnicaHandler. O XML chega como texto no corpo do
// JSON (o navegador lê o arquivo com File.text() antes de mandar) — não
// precisa de upload multipart, esse backend não tem esse mecanismo em
// lugar nenhum e o XML é sempre texto puro.
type NFeImportHandler struct {
	nfeImportService *service.NFeImportService
	lojaRepo         *repository.LojaRepository
}

func NewNFeImportHandler(nfeImportService *service.NFeImportService, lojaRepo *repository.LojaRepository) *NFeImportHandler {
	return &NFeImportHandler{nfeImportService: nfeImportService, lojaRepo: lojaRepo}
}

func (h *NFeImportHandler) carregarLojaScale(c *gin.Context) (*domain.Loja, bool) {
	lojaID := c.GetUint("loja_id")
	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return nil, false
	}
	if !controleEstoqueCompletoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "importação de NF-e é um recurso do plano Scale"})
		return nil, false
	}
	return loja, true
}

type previewNFeRequest struct {
	XML string `json:"xml" binding:"required"`
}

// Preview atende POST /admin/insumos/importar-nfe/preview — só leitura,
// nenhum insumo/estoque é alterado nessa etapa.
func (h *NFeImportHandler) Preview(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	var req previewNFeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	preview, err := h.nfeImportService.Preview(loja.ID, req.XML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

type confirmarItemNFeRequest struct {
	Acao           string  `json:"acao" binding:"required,oneof=vincular criar ignorar"`
	InsumoID       *uint   `json:"insumo_id"`
	Nome           string  `json:"nome"`
	UnidadeCompra  string  `json:"unidade_compra"`
	UnidadeUso     string  `json:"unidade_uso"`
	FatorConversao float64 `json:"fator_conversao"`
	Quantidade     float64 `json:"quantidade"`
	ValorUnitario  float64 `json:"valor_unitario"`
}

type confirmarNFeRequest struct {
	NumeroNota string                    `json:"numero_nota"`
	Itens      []confirmarItemNFeRequest `json:"itens" binding:"required,min=1,dive"`
}

// Confirmar atende POST /admin/insumos/importar-nfe/confirmar — aplica
// o que o admin decidiu na tela de conferência.
func (h *NFeImportHandler) Confirmar(c *gin.Context) {
	loja, ok := h.carregarLojaScale(c)
	if !ok {
		return
	}

	var req confirmarNFeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	itens := make([]service.ConfirmarItemNFeInput, len(req.Itens))
	for i, item := range req.Itens {
		itens[i] = service.ConfirmarItemNFeInput{
			Acao:           item.Acao,
			InsumoID:       item.InsumoID,
			Nome:           item.Nome,
			UnidadeCompra:  item.UnidadeCompra,
			UnidadeUso:     item.UnidadeUso,
			FatorConversao: item.FatorConversao,
			Quantidade:     item.Quantidade,
			ValorUnitario:  item.ValorUnitario,
		}
	}

	insumos, err := h.nfeImportService.Confirmar(loja.ID, req.NumeroNota, itens)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, insumos)
}

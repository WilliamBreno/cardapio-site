package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// FreteHandler expõe o endpoint público de cotação de frete por
// quilometragem. Depende do LojaService (pra achar a loja pelo slug e ler
// suas coordenadas/configurações de taxa) e do DistanciaService (pra
// geocodificar o endereço do cliente e calcular a distância).
type FreteHandler struct {
	lojaService      *service.LojaService
	distanciaService *service.DistanciaService
}

func NewFreteHandler(lojaService *service.LojaService, distanciaService *service.DistanciaService) *FreteHandler {
	return &FreteHandler{
		lojaService:      lojaService,
		distanciaService: distanciaService,
	}
}

type cotarFreteRequest struct {
	Endereco    string `json:"endereco" binding:"required"`
	Rua         string `json:"rua"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Cidade      string `json:"cidade" binding:"required"`
	Estado      string `json:"estado" binding:"required"`
	CEP         string `json:"cep"`
}

// Cotar recebe o endereço digitado pelo cliente no carrinho e devolve a
// distância até a loja e o valor calculado do frete, sem criar nenhum
// pedido — é só uma prévia. O valor final cobrado de verdade é recalculado
// no backend na hora de criar o pedido, pra não confiar em nada vindo do
// navegador do cliente.
func (h *FreteHandler) Cotar(c *gin.Context) {
	slug := c.Param("slug")
	loja, err := h.lojaService.BuscarPorSlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}

	if loja.TaxaEntregaTipo != "por_km" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "essa loja não usa cálculo automático de frete"})
		return
	}
	if loja.Latitude == 0 && loja.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "loja ainda não configurou o endereço de origem"})
		return
	}

	var req cotarFreteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	destino, err := h.distanciaService.GeocodificarEstruturado(service.EnderecoEstruturado{
		Rua:         req.Rua,
		Numero:      req.Numero,
		Complemento: req.Complemento,
		Bairro:      req.Bairro,
		Cidade:      req.Cidade,
		Estado:      req.Estado,
		CEP:         req.CEP,
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "não conseguimos localizar esse endereço"})
		return
	}

	origem := service.Coordenada{Latitude: loja.Latitude, Longitude: loja.Longitude}
	distancia := h.distanciaService.DistanciaKm(origem, *destino)
	valorFrete := service.CalcularTaxaPorKm(distancia, loja.TaxaEntregaBase, loja.TaxaEntregaPorKm)

	c.JSON(http.StatusOK, gin.H{
		"distancia_km": distancia,
		"valor_frete":  valorFrete,
	})
}

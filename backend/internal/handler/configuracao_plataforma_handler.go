package handler

import (
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ConfiguracaoPlataformaHandler struct {
	configuracaoService *service.ConfiguracaoPlataformaService
}

func NewConfiguracaoPlataformaHandler(configuracaoService *service.ConfiguracaoPlataformaService) *ConfiguracaoPlataformaHandler {
	return &ConfiguracaoPlataformaHandler{configuracaoService: configuracaoService}
}

// Buscar atende GET /admin/configuracao-plataforma — hoje só devolve o
// preço mensal da Sugestão Inteligente, pra tela de contratação do
// lojista mostrar o valor sem ele nunca ficar hardcoded no frontend (ver
// docs/plano-melhorias-drenux.md, Fase 6, Parte 3).
func (h *ConfiguracaoPlataformaHandler) Buscar(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sugestao_inteligente_preco_mensal": h.configuracaoService.PrecoSugestaoInteligente(),
	})
}

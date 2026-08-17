package handler

import (
	"net/http"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CatalogoHandler struct {
	catalogoService *service.CatalogoService
	lojaRepo        *repository.LojaRepository
	pedidoRepo      *repository.PedidoRepository
}

func NewCatalogoHandler(catalogoService *service.CatalogoService, db *gorm.DB) *CatalogoHandler {
	return &CatalogoHandler{
		catalogoService: catalogoService,
		lojaRepo:        repository.NewLojaRepository(db),
		pedidoRepo:      repository.NewPedidoRepository(db),
	}
}

// BuscarCardapio atende GET /lojas/:slug — rota pública, sem autenticação.
func (h *CatalogoHandler) BuscarCardapio(c *gin.Context) {
	slug := c.Param("slug")

	cardapio, err := h.catalogoService.BuscarCardapioPorSlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"loja": gin.H{
			"nome":                       cardapio.Loja.Nome,
			"slug":                       cardapio.Loja.Slug,
			"logo_url":                   cardapio.Loja.LogoURL,
			"modo_pedido":                cardapio.Loja.ModoPedido,
			"antecedencia_minima_horas":  cardapio.Loja.AntecedenciaMinimaHoras,
			"horario_abertura":           cardapio.Loja.HorarioAbertura,
			"horario_fechamento":         cardapio.Loja.HorarioFechamento,
			"margem_fechamento_minutos":  cardapio.Loja.MargemFechamentoMinutos,
			"pausado":                    cardapio.Loja.Pausado,
			"mensagem_pausa":             cardapio.Loja.MensagemPausa,
			"aceita_retirada":            cardapio.Loja.AceitaRetirada,
			"aceita_entrega":             cardapio.Loja.AceitaEntrega,
			"aceita_guardar_entregar":    cardapio.Loja.AceitaGuardarEntregar,
			"segmento_principal":         cardapio.Loja.SegmentoPrincipal,
			"sugestao_inteligente_ativa": cardapio.Loja.SugestaoInteligenteAtiva,
			"taxa_entrega_tipo":          cardapio.Loja.TaxaEntregaTipo,
			"taxa_entrega_valor":         cardapio.Loja.TaxaEntregaValor,
			"valor_minimo_pedido":        cardapio.Loja.ValorMinimoPedido,
			"tema":                       cardapio.Loja.Tema,
			// mostrar_selo_drenux: marca "Feito com Drenux" é removível a
			// partir do Basic (pedido do William, 12/08/2026 — antes só Pro/
			// Scale removiam) — computado aqui, não no frontend, pra manter
			// a regra de plano num lugar só.
			"mostrar_selo_drenux": cardapio.Loja.Plano == "start",
			// avisos_whatsapp_cliente: diz se o CLIENTE recebe aviso
			// automático por WhatsApp (confirmação de pagamento, saiu pra
			// entrega) — Start não tem (ver PosPagamentoService.
			// notificarPagamento e PedidoHandler.notificarSaiuParaEntrega,
			// os dois pulam o cliente quando o plano é Start). Usado pra
			// não prometer WhatsApp que nunca chega na mensagem de "pedido
			// confirmado" do cardápio público.
			"avisos_whatsapp_cliente": cardapio.Loja.Plano != "start",
		},
		"categorias":    cardapio.Categorias,
		"subcategorias": cardapio.Subcategorias,
		"grupos_cor":    cardapio.GruposCor,
		"produtos":      cardapio.Produtos,
		"combos":        cardapio.Combos,
	})
}

// BuscarHistorico atende GET /lojas/:slug/historico?telefone=5579... — público.
// Retorna os últimos pedidos pagos de um cliente nessa loja, identificado
// pelo número de WhatsApp. Sem conta, sem senha — o número é o identificador.
func (h *CatalogoHandler) BuscarHistorico(c *gin.Context) {
	slug := c.Param("slug")
	telefone := strings.TrimSpace(c.Query("telefone"))

	if telefone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o telefone"})
		return
	}
	// Normaliza: garante que começa com 55
	if !strings.HasPrefix(telefone, "55") {
		telefone = "55" + telefone
	}
	telefone = strings.ReplaceAll(telefone, " ", "")

	loja, err := h.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}

	pedidos, err := h.pedidoRepo.ListarPorTelefone(loja.ID, telefone, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pedidos)

}

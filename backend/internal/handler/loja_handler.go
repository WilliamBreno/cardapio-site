package handler

import (
	"log"
	"net/http"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type LojaHandler struct {
	lojaService      *service.LojaService
	distanciaService *service.DistanciaService
}

func NewLojaHandler(lojaService *service.LojaService, distanciaService *service.DistanciaService) *LojaHandler {
	return &LojaHandler{
		lojaService:      lojaService,
		distanciaService: distanciaService,
	}
}

// lojaResponse agrega o domain.Loja com campos computados que não vêm do
// banco — hoje só PodeEditarSegmento, que diz pro frontend se mostra o
// seletor de segmento como editável ou travado (ver
// LojaService.PodeEditarSegmento).
type lojaResponse struct {
	domain.Loja
	PodeEditarSegmento bool `json:"pode_editar_segmento"`
}

// Buscar atende GET /admin/loja
func (h *LojaHandler) Buscar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	usuarioID := c.GetUint("usuario_id")

	loja, err := h.lojaService.Buscar(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}

	c.JSON(http.StatusOK, lojaResponse{
		Loja:               *loja,
		PodeEditarSegmento: h.lojaService.PodeEditarSegmento(usuarioID),
	})
}

type configuracoesRequest struct {
	WhatsappNumero           string  `json:"whatsapp_numero" binding:"required"`
	LogoURL                  string  `json:"logo_url"`
	ModoPedido               string  `json:"modo_pedido"`
	AntecedenciaMinimaHoras  int     `json:"antecedencia_minima_horas"`
	HorarioAbertura          string  `json:"horario_abertura"`
	HorarioFechamento        string  `json:"horario_fechamento"`
	MargemFechamentoMinutos  int     `json:"margem_fechamento_minutos"`
	Pausado                  bool    `json:"pausado"`
	MensagemPausa            string  `json:"mensagem_pausa"`
	AceitaRetirada           bool    `json:"aceita_retirada"`
	AceitaEntrega            bool    `json:"aceita_entrega"`
	TaxaEntregaTipo          string  `json:"taxa_entrega_tipo"`
	TaxaEntregaValor         float64 `json:"taxa_entrega_valor"`
	TaxaEntregaBase          float64 `json:"taxa_entrega_base"`
	TaxaEntregaPorKm         float64 `json:"taxa_entrega_por_km"`
	ValorMinimoPedido        float64 `json:"valor_minimo_pedido"`
	Tema                     string  `json:"tema"`
	Endereco                 string  `json:"endereco"`
	AceitaGuardarEntregar    bool    `json:"aceita_guardar_entregar"`
	SegmentoPrincipal        string  `json:"segmento_principal" binding:"required,oneof=alimenticio mercadoria"`
	SugestaoInteligenteAtiva bool    `json:"sugestao_inteligente_ativa"`
}

// AtualizarConfiguracoes atende PUT /admin/loja
func (h *LojaHandler) AtualizarConfiguracoes(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	usuarioID := c.GetUint("usuario_id")

	var req configuracoesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	modo := req.ModoPedido
	if modo == "" {
		modo = "imediato"
	}

	// Geocodifica o endereço sempre que informado — não só quando o modo
	// de taxa é "por_km". O cálculo de frete de itens guardados (fluxo
	// "guardar e entregar depois") também depende de latitude/longitude/
	// cidade/estado da loja, independente de como a entrega imediata é
	// cobrada. Se a geocodificação falhar, não travamos o salvamento das
	// outras configurações — só avisamos no log e deixa os campos como
	// estavam (endpoints que dependem disso rejeitam com mensagem clara).
	var latitude, longitude float64
	var cidade, estado string
	if req.Endereco != "" {
		geo, err := h.distanciaService.GeocodificarDetalhado(req.Endereco)
		if err != nil {
			log.Printf("aviso: não foi possível geocodificar endereço da loja %d: %v", lojaID, err)
		} else {
			latitude = geo.Latitude
			longitude = geo.Longitude
			cidade = geo.Cidade
			estado = geo.Estado
		}
	}

	cfg := repository.ConfiguracoesLoja{
		WhatsappNumero:           req.WhatsappNumero,
		LogoURL:                  req.LogoURL,
		ModoPedido:               modo,
		AntecedenciaMinimaHoras:  req.AntecedenciaMinimaHoras,
		HorarioAbertura:          req.HorarioAbertura,
		HorarioFechamento:        req.HorarioFechamento,
		MargemFechamentoMinutos:  req.MargemFechamentoMinutos,
		Pausado:                  req.Pausado,
		MensagemPausa:            req.MensagemPausa,
		AceitaRetirada:           req.AceitaRetirada,
		AceitaEntrega:            req.AceitaEntrega,
		TaxaEntregaTipo:          req.TaxaEntregaTipo,
		TaxaEntregaValor:         req.TaxaEntregaValor,
		TaxaEntregaBase:          req.TaxaEntregaBase,
		TaxaEntregaPorKm:         req.TaxaEntregaPorKm,
		ValorMinimoPedido:        req.ValorMinimoPedido,
		Tema:                     req.Tema,
		AceitaGuardarEntregar:    req.AceitaGuardarEntregar,
		SegmentoPrincipal:        req.SegmentoPrincipal,
		SugestaoInteligenteAtiva: req.SugestaoInteligenteAtiva,
		Endereco:                 req.Endereco,
		Latitude:                 latitude,
		Longitude:                longitude,
		Cidade:                   cidade,
		Estado:                   estado,
	}

	if err := h.lojaService.AtualizarConfiguracoes(lojaID, usuarioID, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

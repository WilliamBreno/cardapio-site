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
	cronSecret       string
}

func NewLojaHandler(lojaService *service.LojaService, distanciaService *service.DistanciaService, cronSecret string) *LojaHandler {
	return &LojaHandler{
		lojaService:      lojaService,
		distanciaService: distanciaService,
		cronSecret:       cronSecret,
	}
}

// lojaResponse agrega o domain.Loja com campos computados que não vêm
// direto do banco: PodeEditarSegmento (se mostra o seletor de segmento
// como editável ou travado, ver LojaService.PodeEditarSegmento) e o
// contador/bloqueio de pedidos do plano Start (Fase 7.3, ver
// LojaService.LimitePedidosStart) — sempre presentes, mesmo pra lojas que
// não são Start (o painel decide se mostra com base no plano).
type lojaResponse struct {
	domain.Loja
	PodeEditarSegmento   bool `json:"pode_editar_segmento"`
	PedidosMesAtual      int  `json:"pedidos_mes_atual"`
	LimiteStartBloqueado bool `json:"limite_start_bloqueado"`
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

	pedidosNoMes, bloqueada, err := h.lojaService.LimitePedidosStart(loja)
	if err != nil {
		log.Printf("aviso: não foi possível calcular o limite de pedidos do Start da loja %d: %v", loja.ID, err)
	}

	c.JSON(http.StatusOK, lojaResponse{
		Loja:                 *loja,
		PodeEditarSegmento:   h.lojaService.PodeEditarSegmento(usuarioID),
		PedidosMesAtual:      pedidosNoMes,
		LimiteStartBloqueado: bloqueada,
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
	EnderecoRua              string  `json:"endereco_rua"`
	EnderecoNumero           string  `json:"endereco_numero"`
	EnderecoComplemento      string  `json:"endereco_complemento"`
	EnderecoBairro           string  `json:"endereco_bairro"`
	EnderecoCidade           string  `json:"endereco_cidade"`
	EnderecoEstado           string  `json:"endereco_estado"`
	EnderecoCEP              string  `json:"endereco_cep"`
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
	// outras configurações — só avisamos no log e MANTÉM as coordenadas
	// que a loja já tinha (buscadas antes de qualquer alteração): sobrescrever
	// com zero aqui apagaria silenciosamente o endereço de origem sempre que
	// o lojista salvasse qualquer outra configuração durante uma falha
	// passageira de geocodificação, quebrando o cálculo de frete pra todos os
	// pedidos seguintes até ele notar e resalvar o endereço.
	lojaAtual, err := h.lojaService.Buscar(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}
	latitude, longitude := lojaAtual.Latitude, lojaAtual.Longitude
	cidade, estado := lojaAtual.Cidade, lojaAtual.Estado
	if req.Endereco != "" && req.Endereco != lojaAtual.Endereco {
		geo, err := h.distanciaService.GeocodificarEstruturadoDetalhado(service.EnderecoEstruturado{
			Rua:         req.EnderecoRua,
			Numero:      req.EnderecoNumero,
			Complemento: req.EnderecoComplemento,
			Bairro:      req.EnderecoBairro,
			Cidade:      req.EnderecoCidade,
			Estado:      req.EnderecoEstado,
			CEP:         req.EnderecoCEP,
		})
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

// VerificarLimiteStart atende POST /drenux/lojas/verificar-limite-start —
// rota pública protegida por X-Cron-Secret (mesmo padrão de
// MercadoPagoHandler.RenovarTokens), pensada pra ser chamada por um cron
// externo (cron-job.org) uma vez por dia. Aplica o bloqueio de pedidos das
// lojas Start que passaram os 30 pedidos do mês há mais de 3 dias e ainda
// não migraram pro Basic (Fase 7.3, ver LojaService.VerificarLimiteStart).
func (h *LojaHandler) VerificarLimiteStart(c *gin.Context) {
	if h.cronSecret != "" && c.GetHeader("X-Cron-Secret") != h.cronSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "não autorizado"})
		return
	}

	bloqueadas, erros := h.lojaService.VerificarLimiteStart(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"bloqueadas": bloqueadas,
		"erros":      erros,
	})
}

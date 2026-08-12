package handler

import (
	"net/http"
	"strconv"

	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/gin-gonic/gin"
)

// SolicitacaoHandler cuida do rastreamento/entrega de uma SolicitacaoEntrega
// já paga — mesmo padrão de PedidoHandler (Fase 2), só que aplicado à
// entrega de itens que já estavam guardados em vez de um pedido normal.
type SolicitacaoHandler struct {
	solicitacaoRepo *repository.SolicitacaoEntregaRepository
	lojaRepo        *repository.LojaRepository
}

func NewSolicitacaoHandler(solicitacaoRepo *repository.SolicitacaoEntregaRepository, lojaRepo *repository.LojaRepository) *SolicitacaoHandler {
	return &SolicitacaoHandler{solicitacaoRepo: solicitacaoRepo, lojaRepo: lojaRepo}
}

// Listar atende GET /admin/solicitacoes — protegida, mostra as
// solicitações de entrega pagas e aguardando envio da loja do token.
func (h *SolicitacaoHandler) Listar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	solicitacoes, err := h.solicitacaoRepo.ListarPagasPorLoja(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, solicitacoes)
}

// AtualizarStatusEntrega atende PUT /admin/solicitacoes/:id/status-entrega.
func (h *SolicitacaoHandler) AtualizarStatusEntrega(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	solicitacao, err := h.solicitacaoRepo.BuscarPorID(uint(id))
	if err != nil || solicitacao.LojaID != lojaID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "solicitação não encontrada"})
		return
	}

	var req statusEntregaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.solicitacaoRepo.AtualizarStatusEntrega(uint(id), req.StatusEntrega); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

// AtualizarLocalizacao atende POST /admin/solicitacoes/:id/localizacao.
// Recusa se a loja não estiver num plano com rastreamento em tempo real
// (Fase 7.4, mesma regra de PedidoHandler.AtualizarLocalizacao).
func (h *SolicitacaoHandler) AtualizarLocalizacao(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	solicitacao, err := h.solicitacaoRepo.BuscarPorID(uint(id))
	if err != nil || solicitacao.LojaID != lojaID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "solicitação não encontrada"})
		return
	}

	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}
	if !rastreamentoDisponivel(loja.Plano) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "rastreamento em tempo real disponível a partir do plano Pro"})
		return
	}

	var req localizacaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.solicitacaoRepo.AtualizarLocalizacaoEntregador(uint(id), req.Latitude, req.Longitude); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

// Rastrear atende GET /lojas/:slug/solicitacoes/:id/rastrear?telefone=...
// Rota pública — telefone como "senha simples", mesmo padrão do
// rastreamento de pedidos normais.
func (h *SolicitacaoHandler) Rastrear(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	telefone := c.Query("telefone")
	if telefone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o telefone usado na solicitação"})
		return
	}

	solicitacao, err := h.solicitacaoRepo.BuscarPorIDETelefone(uint(id), telefone)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "solicitação não encontrada pra esse telefone"})
		return
	}

	loja, err := h.lojaRepo.BuscarPorID(solicitacao.LojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}
	disponivel := rastreamentoDisponivel(loja.Plano)

	resposta := rastrearResponse{
		StatusEntrega: solicitacao.StatusEntrega,
		Disponivel:    disponivel,
	}
	if disponivel {
		resposta.EntregadorLatitude = solicitacao.EntregadorLatitude
		resposta.EntregadorLongitude = solicitacao.EntregadorLongitude
		resposta.EntregadorAtualizadoEm = solicitacao.EntregadorAtualizadoEm
	}
	c.JSON(http.StatusOK, resposta)
}

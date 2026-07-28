package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type PedidoHandler struct {
	pedidoService      *service.PedidoService
	pedidoRepo         *repository.PedidoRepository
	lojaRepo           *repository.LojaRepository
	notificationSender notification.NotificationSender
	frontendURL        string
}

func NewPedidoHandler(
	pedidoService *service.PedidoService,
	pedidoRepo *repository.PedidoRepository,
	lojaRepo *repository.LojaRepository,
	notificationSender notification.NotificationSender,
	frontendURL string,
) *PedidoHandler {
	return &PedidoHandler{
		pedidoService:      pedidoService,
		pedidoRepo:         pedidoRepo,
		lojaRepo:           lojaRepo,
		notificationSender: notificationSender,
		frontendURL:        frontendURL,
	}
}

type itemPedidoRequest struct {
	ProdutoID         uint  `json:"produto_id" binding:"required"`
	VariacaoID        *uint `json:"variacao_id"`
	Quantidade        int   `json:"quantidade" binding:"required,gt=0"`
	SugestaoProdutoID *uint `json:"sugestao_produto_id"`
}

type comboItemPedidoRequest struct {
	ComboItemID uint  `json:"combo_item_id" binding:"required"`
	VariacaoID  *uint `json:"variacao_id"`
}

type comboPedidoRequest struct {
	ComboID    uint                     `json:"combo_id" binding:"required"`
	Quantidade int                      `json:"quantidade" binding:"required,gt=0"`
	Itens      []comboItemPedidoRequest `json:"itens"`
}

type pedidoRequest struct {
	ClienteNome     string               `json:"cliente_nome" binding:"required"`
	ClienteTelefone string               `json:"cliente_telefone" binding:"required"`
	DataRetirada    time.Time            `json:"data_retirada" binding:"required"`
	ModoEntrega     string               `json:"modo_entrega"`
	EnderecoEntrega string               `json:"endereco_entrega"`
	CupomCodigo     string               `json:"cupom_codigo"`
	Itens           []itemPedidoRequest  `json:"itens" binding:"dive"`
	Combos          []comboPedidoRequest `json:"combos" binding:"dive"`
}

// Criar atende POST /lojas/:slug/pedidos — rota pública. O cliente final
// não precisa de login pra fazer um pedido.
func (h *PedidoHandler) Criar(c *gin.Context) {
	slug := c.Param("slug")

	var req pedidoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	itensInput := make([]service.ItemPedidoInput, len(req.Itens))
	for i, item := range req.Itens {
		itensInput[i] = service.ItemPedidoInput{
			ProdutoID:         item.ProdutoID,
			VariacaoID:        item.VariacaoID,
			Quantidade:        item.Quantidade,
			SugestaoProdutoID: item.SugestaoProdutoID,
		}
	}

	combosInput := make([]service.ComboPedidoInput, len(req.Combos))
	for i, combo := range req.Combos {
		itensCombo := make([]service.ComboItemPedidoInput, len(combo.Itens))
		for j, item := range combo.Itens {
			itensCombo[j] = service.ComboItemPedidoInput{
				ComboItemID: item.ComboItemID,
				VariacaoID:  item.VariacaoID,
			}
		}
		combosInput[i] = service.ComboPedidoInput{
			ComboID:    combo.ComboID,
			Quantidade: combo.Quantidade,
			Itens:      itensCombo,
		}
	}

	pedido, err := h.pedidoService.CriarPorSlug(slug, service.PedidoInput{
		ClienteNome:     req.ClienteNome,
		ClienteTelefone: req.ClienteTelefone,
		DataRetirada:    req.DataRetirada,
		ModoEntrega:     req.ModoEntrega,
		EnderecoEntrega: req.EnderecoEntrega,
		CupomCodigo:     req.CupomCodigo,
		Itens:           itensInput,
		Combos:          combosInput,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pedido)
}

// Listar atende GET /admin/pedidos — protegida, mostra os pedidos da
// loja do token.
func (h *PedidoHandler) Listar(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	pedidos, err := h.pedidoService.ListarPorLoja(lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pedidos)
}

type statusEntregaRequest struct {
	StatusEntrega string `json:"status_entrega" binding:"required,oneof=saiu_para_entrega entregue"`
}

// AtualizarStatusEntrega atende PUT /admin/pedidos/:id/status-entrega.
// Marca o pedido como "saiu para entrega" ou "entregue". Confirma que o
// pedido pertence à loja do token antes de deixar alterar. Quando o
// status vira "saiu_para_entrega", dispara o aviso de WhatsApp com o
// link de rastreamento em segundo plano, sem atrasar a resposta.
func (h *PedidoHandler) AtualizarStatusEntrega(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	pedidoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	pedido, err := h.pedidoRepo.BuscarPorID(uint(pedidoID))
	if err != nil || pedido.LojaID != lojaID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
		return
	}

	var req statusEntregaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.pedidoRepo.AtualizarStatusEntrega(uint(pedidoID), req.StatusEntrega); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	if req.StatusEntrega == "saiu_para_entrega" {
		go h.notificarSaiuParaEntrega(pedido.ID, lojaID)
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

// notificarSaiuParaEntrega busca os dados atualizados e dispara a
// mensagem de WhatsApp pro cliente. Roda em goroutine separada — se o
// WhatsApp não estiver conectado ou a mensagem falhar, isso não deve
// travar nem reverter a marcação de "saiu para entrega".
func (h *PedidoHandler) notificarSaiuParaEntrega(pedidoID, lojaID uint) {
	if h.notificationSender == nil {
		return
	}

	pedido, err := h.pedidoRepo.BuscarPorID(pedidoID)
	if err != nil {
		log.Printf("aviso: não foi possível recarregar pedido %d pra notificar saída pra entrega: %v", pedidoID, err)
		return
	}

	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		log.Printf("aviso: não foi possível carregar loja %d pra notificar saída pra entrega: %v", lojaID, err)
		return
	}

	link := fmt.Sprintf("%s/%s/pedido/%d/rastrear?telefone=%s", h.frontendURL, loja.Slug, pedido.ID, pedido.ClienteTelefone)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := h.notificationSender.EnviarSaiuParaEntrega(ctx, pedido, loja.Nome, link); err != nil {
		log.Printf("falha ao notificar saída pra entrega do pedido %d: %v", pedido.ID, err)
	}
}

type localizacaoRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// AtualizarLocalizacao atende POST /admin/pedidos/:id/localizacao.
// Chamado periodicamente (a cada ~25s) pelo navegador de quem está
// entregando, enquanto a página de compartilhamento estiver aberta.
func (h *PedidoHandler) AtualizarLocalizacao(c *gin.Context) {
	lojaID := c.GetUint("loja_id")

	pedidoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	pedido, err := h.pedidoRepo.BuscarPorID(uint(pedidoID))
	if err != nil || pedido.LojaID != lojaID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
		return
	}

	var req localizacaoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if err := h.pedidoRepo.AtualizarLocalizacaoEntregador(uint(pedidoID), req.Latitude, req.Longitude); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

type rastrearResponse struct {
	StatusEntrega          string     `json:"status_entrega"`
	EntregadorLatitude     float64    `json:"entregador_latitude"`
	EntregadorLongitude    float64    `json:"entregador_longitude"`
	EntregadorAtualizadoEm *time.Time `json:"entregador_atualizado_em"`
}

// Rastrear atende GET /lojas/:slug/pedidos/:id/rastrear?telefone=...
// Rota pública — usa o telefone do cliente como "senha simples", mesmo
// padrão já usado no histórico de pedidos. Sem o telefone certo, não dá
// pra ver a localização de outro pedido só sabendo o ID.
func (h *PedidoHandler) Rastrear(c *gin.Context) {
	pedidoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}

	telefone := c.Query("telefone")
	if telefone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o telefone usado no pedido"})
		return
	}

	pedido, err := h.pedidoRepo.BuscarPorIDETelefone(uint(pedidoID), telefone)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado pra esse telefone"})
		return
	}

	c.JSON(http.StatusOK, rastrearResponse{
		StatusEntrega:          pedido.StatusEntrega,
		EntregadorLatitude:     pedido.EntregadorLatitude,
		EntregadorLongitude:    pedido.EntregadorLongitude,
		EntregadorAtualizadoEm: pedido.EntregadorAtualizadoEm,
	})
}

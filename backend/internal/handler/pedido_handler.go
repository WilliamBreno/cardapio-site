package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
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
	ClienteNome         string               `json:"cliente_nome" binding:"required"`
	ClienteTelefone     string               `json:"cliente_telefone" binding:"required"`
	DataRetirada        time.Time            `json:"data_retirada" binding:"required"`
	ModoEntrega         string               `json:"modo_entrega"`
	EnderecoEntrega     string               `json:"endereco_entrega"`
	EnderecoRua         string               `json:"endereco_rua"`
	EnderecoNumero      string               `json:"endereco_numero"`
	EnderecoComplemento string               `json:"endereco_complemento"`
	EnderecoBairro      string               `json:"endereco_bairro"`
	EnderecoCidade      string               `json:"endereco_cidade"`
	EnderecoEstado      string               `json:"endereco_estado"`
	EnderecoCEP         string               `json:"endereco_cep"`
	CupomCodigo         string               `json:"cupom_codigo"`
	Itens               []itemPedidoRequest  `json:"itens" binding:"dive"`
	Combos              []comboPedidoRequest `json:"combos" binding:"dive"`
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
		EnderecoEntregaGeo: service.EnderecoEstruturado{
			Rua:         req.EnderecoRua,
			Numero:      req.EnderecoNumero,
			Complemento: req.EnderecoComplemento,
			Bairro:      req.EnderecoBairro,
			Cidade:      req.EnderecoCidade,
			Estado:      req.EnderecoEstado,
			CEP:         req.EnderecoCEP,
		},
		CupomCodigo: req.CupomCodigo,
		Itens:       itensInput,
		Combos:      combosInput,
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

// HistoricoCliente atende GET /admin/clientes/:telefone/pedidos (Fase
// 10.6) — histórico de pedidos pagos de um cliente específico dessa
// loja. Reaproveita PedidoRepository.ListarPorTelefone (mesma consulta
// já usada pelo "Meus pedidos" público, catalogo_handler.go), só com um
// limite maior — aqui é o dono vendo o histórico completo, não o
// cliente numa tela compacta.
func (h *PedidoHandler) HistoricoCliente(c *gin.Context) {
	lojaID := c.GetUint("loja_id")
	telefone := c.Param("telefone")

	pedidos, err := h.pedidoRepo.ListarPorTelefone(lojaID, telefone, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pedidos)
}

// statusEntregaRequest aceita as 4 etapas do fluxo de preparo/entrega
// (Fase 10.2): a_preparar → preparando → saiu_para_entrega → entregue.
// Mesmo endpoint de sempre, só o oneof ficou mais largo — antes só
// aceitava as duas últimas etapas.
type statusEntregaRequest struct {
	StatusEntrega string `json:"status_entrega" binding:"required,oneof=a_preparar preparando saiu_para_entrega entregue"`
	// CodigoConfirmacao só é exigido (e checado) quando StatusEntrega ==
	// "entregue" e o pedido é modo "entrega" — ver AtualizarStatusEntrega.
	CodigoConfirmacao string `json:"codigo_confirmacao"`
}

// AtualizarStatusEntrega atende PUT /admin/pedidos/:id/status-entrega.
// Marca a etapa de preparo/entrega do pedido. Confirma que o pedido
// pertence à loja do token antes de deixar alterar. Quando a etapa vira
// "saiu_para_entrega", dispara o aviso de WhatsApp com o link de
// rastreamento em segundo plano, sem atrasar a resposta — mesmo
// comportamento de antes, só que agora essa é uma de 4 etapas possíveis
// em vez de uma de duas.
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

	// Confirmação por código (24/08/2026): só pra pedido modo "entrega"
	// indo pra "entregue" — o entregador precisa digitar o código que o
	// cliente já vê na própria tela de rastreamento, fechando o ciclo de
	// que a entrega realmente aconteceu (em vez de um clique sem nenhuma
	// checagem). Retirada/guardar não têm entregador, não passam por
	// aqui — continuam marcando "entregue" livremente.
	if req.StatusEntrega == "entregue" && pedido.ModoEntrega == domain.ModoEntregaEntrega {
		if req.CodigoConfirmacao == "" || req.CodigoConfirmacao != pedido.CodigoConfirmacao {
			c.JSON(http.StatusBadRequest, gin.H{"erro": "código de confirmação incorreto — peça o código pro cliente"})
			return
		}
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
// travar nem reverter a marcação de "saiu para entrega". Gates de plano
// (Fase 7.4): Start não tem avisos automáticos de status nenhum (não
// manda mensagem); Basic tem o aviso mas sem link de rastreamento (não
// tem rastreamento em tempo real); Pro/Scale mandam com o link.
func (h *PedidoHandler) notificarSaiuParaEntrega(pedidoID, lojaID uint) {
	if h.notificationSender == nil {
		return
	}

	loja, err := h.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		log.Printf("aviso: não foi possível carregar loja %d pra notificar saída pra entrega: %v", lojaID, err)
		return
	}
	if loja.Plano == "start" {
		return
	}

	pedido, err := h.pedidoRepo.BuscarPorID(pedidoID)
	if err != nil {
		log.Printf("aviso: não foi possível recarregar pedido %d pra notificar saída pra entrega: %v", pedidoID, err)
		return
	}

	var link string
	if loja.Plano == "pro" || loja.Plano == "scale" {
		link = fmt.Sprintf("%s/%s/pedido/%d/rastrear?telefone=%s", h.frontendURL, loja.Slug, pedido.ID, pedido.ClienteTelefone)
	}

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

// rastreamentoDisponivel diz se o plano de uma loja inclui rastreamento
// de entrega em tempo real (Fase 7.4: Pro e Scale têm, Start e Basic não).
// Compartilhado entre PedidoHandler e SolicitacaoHandler — os dois
// fluxos têm exatamente a mesma regra.
func rastreamentoDisponivel(plano string) bool {
	return plano == "pro" || plano == "scale"
}

// AtualizarLocalizacao atende POST /admin/pedidos/:id/localizacao.
// Chamado periodicamente (a cada ~25s) pelo navegador de quem está
// entregando, enquanto a página de compartilhamento estiver aberta.
// Recusa se a loja não estiver num plano com rastreamento em tempo real
// (Fase 7.4).
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
	// Disponivel diz se o plano da loja inclui rastreamento em tempo real
	// (Fase 7.4) — false não é erro, é o frontend sabendo pra mostrar um
	// aviso em vez do mapa (as coordenadas acima vêm zeradas nesse caso).
	Disponivel bool `json:"disponivel"`
	// CodigoConfirmacao (24/08/2026) — sempre incluído, independente do
	// plano (Disponivel só afeta o mapa ao vivo, não esse mecanismo de
	// confirmação): o cliente mostra esse código pro entregador na
	// entrega, que digita ele pra marcar o pedido como "entregue" (ver
	// AtualizarStatusEntrega).
	CodigoConfirmacao string `json:"codigo_confirmacao"`
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

	loja, err := h.lojaRepo.BuscarPorID(pedido.LojaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
		return
	}
	disponivel := rastreamentoDisponivel(loja.Plano)

	resposta := rastrearResponse{
		StatusEntrega:     pedido.StatusEntrega,
		Disponivel:        disponivel,
		CodigoConfirmacao: pedido.CodigoConfirmacao,
	}
	if disponivel {
		resposta.EntregadorLatitude = pedido.EntregadorLatitude
		resposta.EntregadorLongitude = pedido.EntregadorLongitude
		resposta.EntregadorAtualizadoEm = pedido.EntregadorAtualizadoEm
	}
	c.JSON(http.StatusOK, resposta)
}

// buscarPedidoPorToken resolve loja (pelo slug) + pedido (pelo ID e pelo
// TokenEntregador) pras três rotas públicas do entregador abaixo —
// mesmo padrão de validação em todas: 404 genérico tanto pra loja
// inexistente quanto pra token errado/pedido de outra loja, sem
// distinguir o motivo (não dar pista pra quem estiver tentando adivinhar
// um token).
func (h *PedidoHandler) buscarPedidoPorToken(c *gin.Context) (*domain.Pedido, *domain.Loja, bool) {
	slug := c.Param("slug")
	pedidoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return nil, nil, false
	}

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "link inválido"})
		return nil, nil, false
	}

	loja, err := h.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
		return nil, nil, false
	}

	pedido, err := h.pedidoRepo.BuscarPorIDEToken(uint(pedidoID), token)
	if err != nil || pedido.LojaID != loja.ID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "pedido não encontrado"})
		return nil, nil, false
	}

	return pedido, loja, true
}

type entregadorResponse struct {
	StatusEntrega    string  `json:"status_entrega"`
	ClienteNome      string  `json:"cliente_nome"`
	EnderecoEntrega  string  `json:"endereco_entrega"`
	DestinoLatitude  float64 `json:"destino_latitude"`
	DestinoLongitude float64 `json:"destino_longitude"`
	// Disponivel diz se o plano da loja permite compartilhar localização
	// em tempo real (Fase 7.4: Pro/Scale) — false não impede a tela de
	// funcionar, só esconde o botão de compartilhar GPS (a confirmação
	// por código continua funcionando em qualquer plano).
	Disponivel bool `json:"disponivel"`
}

// BuscarParaEntregador atende GET
// /lojas/:slug/pedidos/:id/entregador?token=... — rota pública, sem
// login. É o que a tela do link "Gerar link" (Pedidos.tsx admin) carrega
// pra montar o mapa com o pino de destino e mostrar o endereço.
func (h *PedidoHandler) BuscarParaEntregador(c *gin.Context) {
	pedido, loja, ok := h.buscarPedidoPorToken(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, entregadorResponse{
		StatusEntrega:    pedido.StatusEntrega,
		ClienteNome:      pedido.ClienteNome,
		EnderecoEntrega:  pedido.EnderecoEntrega,
		DestinoLatitude:  pedido.DestinoLatitude,
		DestinoLongitude: pedido.DestinoLongitude,
		Disponivel:       rastreamentoDisponivel(loja.Plano),
	})
}

// AtualizarLocalizacaoEntregador atende POST
// /lojas/:slug/pedidos/:id/entregador/localizacao?token=... — equivalente
// público de AtualizarLocalizacao, autenticado pelo token do link em vez
// do JWT do dono (o entregador não tem login nesse sistema).
func (h *PedidoHandler) AtualizarLocalizacaoEntregador(c *gin.Context) {
	pedido, loja, ok := h.buscarPedidoPorToken(c)
	if !ok {
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

	if err := h.pedidoRepo.AtualizarLocalizacaoEntregador(pedido.ID, req.Latitude, req.Longitude); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

type statusEntregadorRequest struct {
	// oneof deliberadamente mais estreito que statusEntregaRequest (que
	// aceita as 4 etapas) — o link do entregador só faz sentido pras duas
	// últimas etapas; "a_preparar"/"preparando" são passos de cozinha,
	// não deveriam ser alcançáveis por esse link mesmo que vazado.
	StatusEntrega     string `json:"status_entrega" binding:"required,oneof=saiu_para_entrega entregue"`
	CodigoConfirmacao string `json:"codigo_confirmacao"`
}

// AtualizarStatusEntregador atende PUT
// /lojas/:slug/pedidos/:id/entregador/status?token=... — equivalente
// público de AtualizarStatusEntrega, mesma checagem de código de
// confirmação pra "entregue" num pedido modo "entrega".
func (h *PedidoHandler) AtualizarStatusEntregador(c *gin.Context) {
	pedido, loja, ok := h.buscarPedidoPorToken(c)
	if !ok {
		return
	}

	var req statusEntregadorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	if req.StatusEntrega == "entregue" {
		if req.CodigoConfirmacao == "" || req.CodigoConfirmacao != pedido.CodigoConfirmacao {
			c.JSON(http.StatusBadRequest, gin.H{"erro": "código de confirmação incorreto — peça o código pro cliente"})
			return
		}
	}

	if err := h.pedidoRepo.AtualizarStatusEntrega(pedido.ID, req.StatusEntrega); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": err.Error()})
		return
	}

	if req.StatusEntrega == "saiu_para_entrega" {
		go h.notificarSaiuParaEntrega(pedido.ID, loja.ID)
	}

	c.JSON(http.StatusOK, gin.H{"sucesso": true})
}

package handler

import (
	"net/http"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// GuardadosHandler expõe as rotas públicas do fluxo "guardar e entregar
// depois": listar o que um cliente tem guardado, cotar o frete de uma
// entrega desses itens, e efetivar a solicitação de entrega. Nenhuma
// rota aqui exige login — o telefone do cliente é o identificador, mesmo
// padrão já usado no histórico e no rastreamento de pedidos.
type GuardadosHandler struct {
	guardadosService *service.GuardadosService
}

func NewGuardadosHandler(guardadosService *service.GuardadosService) *GuardadosHandler {
	return &GuardadosHandler{guardadosService: guardadosService}
}

// normalizarTelefone garante que o telefone sempre começa com 55 (DDI do
// Brasil) — mesma normalização já aplicada no histórico público.
func normalizarTelefone(telefone string) string {
	telefone = strings.ReplaceAll(strings.TrimSpace(telefone), " ", "")
	if telefone != "" && !strings.HasPrefix(telefone, "55") {
		telefone = "55" + telefone
	}
	return telefone
}

// Listar atende GET /lojas/:slug/guardados?telefone=...
func (h *GuardadosHandler) Listar(c *gin.Context) {
	slug := c.Param("slug")
	telefone := normalizarTelefone(c.Query("telefone"))
	if telefone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe o telefone"})
		return
	}

	itens, err := h.guardadosService.ListarPorSlugETelefone(slug, telefone)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": err.Error()})
		return
	}
	c.JSON(http.StatusOK, itens)
}

type cotarFreteGuardadosRequest struct {
	Telefone    string `json:"telefone" binding:"required"`
	Endereco    string `json:"endereco" binding:"required"`
	Rua         string `json:"rua"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Cidade      string `json:"cidade" binding:"required"`
	Estado      string `json:"estado" binding:"required"`
	CEP         string `json:"cep"`
	ItemIDs     []uint `json:"item_ids" binding:"required,min=1"`
}

// CotarFrete atende POST /lojas/:slug/guardados/cotar-frete — prévia, não
// cria nada. O valor final é sempre recalculado no servidor na hora de
// efetivar a solicitação de entrega.
func (h *GuardadosHandler) CotarFrete(c *gin.Context) {
	slug := c.Param("slug")

	var req cotarFreteGuardadosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	cotacao, err := h.guardadosService.CotarFrete(slug, normalizarTelefone(req.Telefone), service.EnderecoEstruturado{
		Rua:         req.Rua,
		Numero:      req.Numero,
		Complemento: req.Complemento,
		Bairro:      req.Bairro,
		Cidade:      req.Cidade,
		Estado:      req.Estado,
		CEP:         req.CEP,
	}, req.ItemIDs)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"distancia_km": cotacao.DistanciaKm,
		"mesma_regiao": cotacao.MesmaRegiao,
		"valor_frete":  cotacao.ValorFrete,
	})
}

type solicitarEntregaRequest struct {
	ClienteNome     string `json:"cliente_nome" binding:"required"`
	ClienteTelefone string `json:"cliente_telefone" binding:"required"`
	Endereco        string `json:"endereco" binding:"required"`
	Rua             string `json:"rua"`
	Numero          string `json:"numero"`
	Complemento     string `json:"complemento"`
	Bairro          string `json:"bairro"`
	Cidade          string `json:"cidade" binding:"required"`
	Estado          string `json:"estado" binding:"required"`
	CEP             string `json:"cep"`
	ItemIDs         []uint `json:"item_ids" binding:"required,min=1"`
}

// SolicitarEntrega atende POST /lojas/:slug/guardados/solicitar-entrega —
// reivindica os itens selecionados e cria a SolicitacaoEntrega aguardando
// pagamento do frete.
func (h *GuardadosHandler) SolicitarEntrega(c *gin.Context) {
	slug := c.Param("slug")

	var req solicitarEntregaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	solicitacao, err := h.guardadosService.SolicitarEntrega(slug, service.SolicitarEntregaInput{
		ClienteNome:     req.ClienteNome,
		ClienteTelefone: normalizarTelefone(req.ClienteTelefone),
		Endereco:        req.Endereco,
		EnderecoGeo: service.EnderecoEstruturado{
			Rua:         req.Rua,
			Numero:      req.Numero,
			Complemento: req.Complemento,
			Bairro:      req.Bairro,
			Cidade:      req.Cidade,
			Estado:      req.Estado,
			CEP:         req.CEP,
		},
		ItemIDs: req.ItemIDs,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, solicitacao)
}

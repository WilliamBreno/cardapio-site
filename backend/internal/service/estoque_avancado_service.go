package service

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// EstoqueAvancadoService implementa a Fase 9.3 do roadmap (ver
// docs/plano-melhorias-drenux.md): lista de compras automática e
// relatórios avançados de estoque — exclusivo do plano Scale, mesmo
// gate de controleEstoqueCompletoDisponivel já usado pra reposição/
// ajuste/histórico (Fase 8, nível 2).
type EstoqueAvancadoService struct {
	db          *gorm.DB
	produtoRepo *repository.ProdutoRepository
}

func NewEstoqueAvancadoService(db *gorm.DB) *EstoqueAvancadoService {
	return &EstoqueAvancadoService{
		db:          db,
		produtoRepo: repository.NewProdutoRepository(db),
	}
}

// --- 9.3, parte 1: lista de compras ---

// ItemListaCompras é um insumo abaixo do próprio alerta configurado —
// só entram insumos com EstoqueAlerta definido (sem isso não tem como
// saber o que é "pouco" pra esse insumo específico). Deficit é só até o
// alerta (não até um "estoque ideal" maior — não foi pedido inventar
// essa margem), convertido também pra unidade de compra porque é nela
// que o lojista vai comprar de verdade.
type ItemListaCompras struct {
	InsumoID             uint     `json:"insumo_id"`
	Nome                 string   `json:"nome"`
	EstoqueAtual         float64  `json:"estoque_atual"`
	EstoqueAlerta        float64  `json:"estoque_alerta"`
	UnidadeUso           string   `json:"unidade_uso"`
	UnidadeCompra        string   `json:"unidade_compra"`
	DeficitUnidadeUso    float64  `json:"deficit_unidade_uso"`
	DeficitUnidadeCompra float64  `json:"deficit_unidade_compra"`
	ProdutosAfetados     []string `json:"produtos_afetados"`
}

// ListaDeCompras cruza os insumos abaixo do alerta com a ficha técnica
// (Fase 9.1) pra mostrar não só "o que comprar", mas "o que fica sem
// poder ser feito" se não comprar.
func (s *EstoqueAvancadoService) ListaDeCompras(lojaID uint) ([]ItemListaCompras, error) {
	var insumos []domain.Insumo
	if err := s.db.Where("loja_id = ? AND estoque_atual IS NOT NULL AND estoque_alerta IS NOT NULL AND estoque_atual <= estoque_alerta", lojaID).
		Order("nome").Find(&insumos).Error; err != nil {
		return nil, err
	}

	itens := make([]ItemListaCompras, 0, len(insumos))
	for _, insumo := range insumos {
		var produtosAfetados []string
		s.db.Raw(`
			SELECT DISTINCT p.nome FROM ficha_tecnica_itens fti
			JOIN produtos p ON p.id = fti.produto_id
			WHERE fti.insumo_id = ?
			ORDER BY p.nome
		`, insumo.ID).Scan(&produtosAfetados)

		deficit := *insumo.EstoqueAlerta - *insumo.EstoqueAtual
		if deficit < 0 {
			deficit = 0
		}
		deficitCompra := 0.0
		if insumo.FatorConversao > 0 {
			deficitCompra = deficit / insumo.FatorConversao
		}

		itens = append(itens, ItemListaCompras{
			InsumoID:             insumo.ID,
			Nome:                 insumo.Nome,
			EstoqueAtual:         *insumo.EstoqueAtual,
			EstoqueAlerta:        *insumo.EstoqueAlerta,
			UnidadeUso:           insumo.UnidadeUso,
			UnidadeCompra:        insumo.UnidadeCompra,
			DeficitUnidadeUso:    deficit,
			DeficitUnidadeCompra: deficitCompra,
			ProdutosAfetados:     produtosAfetados,
		})
	}
	return itens, nil
}

// --- 9.3, parte 2: relatórios avançados ---

type ItemProdutoParado struct {
	ProdutoID   uint   `json:"produto_id"`
	ProdutoNome string `json:"produto_nome"`
}

type ItemGiroEstoque struct {
	ProdutoID    uint   `json:"produto_id"`
	ProdutoNome  string `json:"produto_nome"`
	VariacaoNome string `json:"variacao_nome"`
	EstoqueAtual int    `json:"estoque_atual"`
	Vendido30d   int    `json:"vendido_30d"`
	// Giro é quantas vezes o estoque atual "virou" nos últimos 30 dias
	// (vendido ÷ estoque atual) — quanto maior, mais rápido o item sai.
	// 0 quando o estoque atual é 0 (não dá pra calcular razão sobre
	// zero; o item aparece com giro 0 em vez de sumir do relatório).
	Giro float64 `json:"giro"`
}

type ItemInsumoMaisSai struct {
	InsumoID uint   `json:"insumo_id"`
	Nome     string `json:"nome"`
	// gorm:"column:..." explícito porque o nome derivado automaticamente
	// de "Consumido30d" (GORM tenta converter pra snake_case sozinho) não
	// bate com o alias `consumido_30d` da query — sem o tag, esse campo
	// fica sempre 0, mesmo com a query devolvendo o valor certo.
	Consumido30d float64 `json:"consumido_30d" gorm:"column:consumido_30d"`
	UnidadeUso   string  `json:"unidade_uso"`
}

type RelatoriosAvancados struct {
	// ProdutosParados: disponíveis no cardápio mas sem nenhuma venda nos
	// últimos 30 dias.
	ProdutosParados []ItemProdutoParado `json:"produtos_parados"`
	// GiroEstoque: só itens com controle de estoque ativo (mesmo
	// universo do relatório de estoque simples, Fase 8).
	GiroEstoque []ItemGiroEstoque `json:"giro_estoque"`
	// ValorParadoEstoque soma só o que dá pra calcular com dado real: o
	// estoque de INSUMO ao custo cadastrado. Produto/variação não entra
	// aqui — Produto não tem campo de custo (só preço de venda), e só
	// teria custo conhecido através da ficha técnica dos produtos que o
	// usam, o que não é o mesmo conceito de "quanto vale o estoque
	// parado desse produto pronto". Documentado explicitamente em vez de
	// inventar um custo estimado.
	ValorParadoEstoque    float64             `json:"valor_parado_estoque"`
	ValorParadoObservacao string              `json:"valor_parado_observacao"`
	InsumosQueMaisSaem    []ItemInsumoMaisSai `json:"insumos_que_mais_saem"`
}

// RelatoriosAvancados monta os 4 relatórios da Fase 9.3 de uma vez só —
// mesmo espírito de bundling de DashboardService.BuscarDados.
func (s *EstoqueAvancadoService) RelatoriosAvancados(lojaID uint) (*RelatoriosAvancados, error) {
	resultado := &RelatoriosAvancados{
		ValorParadoObservacao: "Considera só o estoque de insumos (custo conhecido). Produtos prontos sem ficha técnica não têm custo cadastrado no sistema, só preço de venda — não entram nessa soma.",
	}

	if err := s.db.Raw(`
		SELECT p.id AS produto_id, p.nome AS produto_nome
		FROM produtos p
		WHERE p.loja_id = ? AND p.disponivel = true
		AND NOT EXISTS (
			SELECT 1 FROM itens_pedido ip
			JOIN pedidos pe ON pe.id = ip.pedido_id
			WHERE ip.produto_id = p.id AND pe.status = 'pago' AND pe.updated_at >= NOW() - INTERVAL '30 days'
		)
		ORDER BY p.nome
	`, lojaID).Scan(&resultado.ProdutosParados).Error; err != nil {
		return nil, err
	}

	giro, err := s.giroEstoque(lojaID)
	if err != nil {
		return nil, err
	}
	resultado.GiroEstoque = giro

	var valorInsumos float64
	s.db.Raw(`
		SELECT COALESCE(SUM(estoque_atual * (custo_unidade_compra / NULLIF(fator_conversao, 0))), 0)
		FROM insumos WHERE loja_id = ? AND estoque_atual IS NOT NULL
	`, lojaID).Scan(&valorInsumos)
	resultado.ValorParadoEstoque = valorInsumos

	var insumosMaisSaem []ItemInsumoMaisSai
	s.db.Raw(`
		SELECT i.id AS insumo_id, i.nome, i.unidade_uso, SUM(ABS(m.quantidade)) AS consumido_30d
		FROM movimentacoes_insumo m
		JOIN insumos i ON i.id = m.insumo_id
		WHERE m.loja_id = ? AND m.tipo = 'venda' AND m.created_at >= NOW() - INTERVAL '30 days'
		GROUP BY i.id, i.nome, i.unidade_uso
		ORDER BY consumido_30d DESC
		LIMIT 10
	`, lojaID).Scan(&insumosMaisSaem)
	resultado.InsumosQueMaisSaem = insumosMaisSaem

	return resultado, nil
}

// giroEstoque reaproveita a mesma lista de itens com estoque controlado
// do relatório simples (Fase 8) — evita reescrever a lógica de "produto
// sem variação vs. variação com estoque próprio" numa query nova.
func (s *EstoqueAvancadoService) giroEstoque(lojaID uint) ([]ItemGiroEstoque, error) {
	produtos, err := s.produtoRepo.ListarPorLoja(lojaID, false)
	if err != nil {
		return nil, err
	}

	var resultado []ItemGiroEstoque
	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	desde := time.Now().In(fusoBrasil).AddDate(0, 0, -30)

	for _, p := range produtos {
		if len(p.Variacoes) == 0 {
			if p.EstoqueAtual == nil {
				continue
			}
			vendido := s.vendidoDesde(p.ID, nil, desde)
			resultado = append(resultado, montarItemGiro(p.ID, p.Nome, "", *p.EstoqueAtual, vendido))
			continue
		}
		for _, v := range p.Variacoes {
			if v.EstoqueAtual == nil {
				continue
			}
			variacaoID := v.ID
			vendido := s.vendidoDesde(p.ID, &variacaoID, desde)
			resultado = append(resultado, montarItemGiro(p.ID, p.Nome, v.Nome, *v.EstoqueAtual, vendido))
		}
	}
	return resultado, nil
}

// vendidoDesde soma a quantidade vendida de um produto (ou variação
// específica) desde uma data. Duas queries separadas em vez de uma só
// parametrizada com `variacaoID` podendo ser nil — passar um *uint nil
// como bind param de `?` num Raw() não vira SQL NULL de forma confiável
// (testado ao vivo: sempre devolvia 0, mesmo com vendas reais no
// período), então é mais seguro decidir a query em Go.
func (s *EstoqueAvancadoService) vendidoDesde(produtoID uint, variacaoID *uint, desde time.Time) int {
	var total int
	if variacaoID == nil {
		s.db.Raw(`
			SELECT COALESCE(SUM(ip.quantidade), 0) FROM itens_pedido ip
			JOIN pedidos pe ON pe.id = ip.pedido_id
			WHERE ip.produto_id = ? AND ip.variacao_id IS NULL AND pe.status = 'pago' AND pe.updated_at >= ?
		`, produtoID, desde).Scan(&total)
	} else {
		s.db.Raw(`
			SELECT COALESCE(SUM(ip.quantidade), 0) FROM itens_pedido ip
			JOIN pedidos pe ON pe.id = ip.pedido_id
			WHERE ip.produto_id = ? AND ip.variacao_id = ? AND pe.status = 'pago' AND pe.updated_at >= ?
		`, produtoID, *variacaoID, desde).Scan(&total)
	}
	return total
}

func montarItemGiro(produtoID uint, produtoNome, variacaoNome string, estoqueAtual, vendido30d int) ItemGiroEstoque {
	giro := 0.0
	if estoqueAtual > 0 {
		giro = float64(vendido30d) / float64(estoqueAtual)
	}
	return ItemGiroEstoque{
		ProdutoID:    produtoID,
		ProdutoNome:  produtoNome,
		VariacaoNome: variacaoNome,
		EstoqueAtual: estoqueAtual,
		Vendido30d:   vendido30d,
		Giro:         giro,
	}
}

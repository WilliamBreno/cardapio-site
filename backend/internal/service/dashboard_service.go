package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ReceitaDia struct {
	Data  string  `json:"data"`
	Total float64 `json:"total"`
}

type ReceitaSemana struct {
	Semana string  `json:"semana"`
	Total  float64 `json:"total"`
}

type TopProduto struct {
	Nome       string `json:"nome"`
	Quantidade int    `json:"quantidade"`
}

// ClienteRanking é uma linha do ranking de clientes (Fase 10.4) — não
// existe entidade Cliente própria no sistema, então o agrupamento é por
// ClienteTelefone (mais estável que o nome, que pode variar grafia entre
// pedidos da mesma pessoa). ClienteNome é o mais recente usado por esse
// telefone, pra não mostrar uma grafia antiga/errada.
type ClienteRanking struct {
	ClienteNome     string  `json:"cliente_nome"`
	ClienteTelefone string  `json:"cliente_telefone"`
	TotalPedidos    int     `json:"total_pedidos"`
	ValorTotal      float64 `json:"valor_total"`
}

// Contagem é uma linha genérica de "quantas vezes cada valor apareceu"
// (Fase 10.6) — usada tanto pra tipo de entrega quanto forma de
// pagamento. Chave fica com o valor cru (ex: "entrega", "pix") — o
// frontend já traduz esses valores pra rótulo amigável em outras telas
// (ver Pedidos.tsx), então repete o mesmo padrão aqui em vez de duplicar
// a tradução no Go.
type Contagem struct {
	Chave string `json:"chave"`
	Total int    `json:"total"`
}

type DashboardData struct {
	TotalSemana           float64          `json:"total_semana"`
	TotalMes              float64          `json:"total_mes"`
	PedidosSemana         int              `json:"pedidos_semana"`
	Receita7Dias          []ReceitaDia     `json:"receita_7_dias"`
	Receita4Semanas       []ReceitaSemana  `json:"receita_4_semanas"`
	TopProdutos           []TopProduto     `json:"top_produtos"`
	TopClientesPorPedidos []ClienteRanking `json:"top_clientes_por_pedidos"`
	TopClientesPorValor   []ClienteRanking `json:"top_clientes_por_valor"`
	TiposEntrega          []Contagem       `json:"tipos_entrega"`
	FormasPagamento       []Contagem       `json:"formas_pagamento"`
}

// PeriodoResumo (Fase 10.5) é o resumo de um período exato escolhido pelo
// dono (dia/semana/mês), pra montar o texto do relatório enviado via
// WhatsApp — diferente das janelas fixas de DashboardData (7 dias/4
// semanas/30 dias), que não dão pra escolher a data.
type PeriodoResumo struct {
	Tipo        string       `json:"tipo"`
	Inicio      string       `json:"inicio"` // AAAA-MM-DD, primeiro dia do período
	Fim         string       `json:"fim"`    // AAAA-MM-DD, último dia do período (inclusive)
	Total       float64      `json:"total"`
	NumPedidos  int          `json:"num_pedidos"`
	TicketMedio float64      `json:"ticket_medio"`
	TopProdutos []TopProduto `json:"top_produtos"`
}

type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

func (s *DashboardService) BuscarDados(lojaID uint) (*DashboardData, error) {
	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	agora := time.Now().In(fusoBrasil)

	inicioSemana := agora.AddDate(0, 0, -7)
	inicioMes := agora.AddDate(0, -1, 0)

	data := &DashboardData{}

	// Total da semana
	s.db.Raw(`SELECT COALESCE(SUM(total), 0) FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND updated_at >= ?`,
		lojaID, inicioSemana).Scan(&data.TotalSemana)

	// Total do mês
	s.db.Raw(`SELECT COALESCE(SUM(total), 0) FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND updated_at >= ?`,
		lojaID, inicioMes).Scan(&data.TotalMes)

	// Pedidos da semana
	s.db.Raw(`SELECT COUNT(*) FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND updated_at >= ?`,
		lojaID, inicioSemana).Scan(&data.PedidosSemana)

	// Receita por dia — últimos 7 dias, preenchendo dias sem pedido com 0
	var dias []ReceitaDia
	s.db.Raw(`
		SELECT
			TO_CHAR(d.dia, 'DD/MM') as data,
			COALESCE(SUM(p.total), 0) as total
		FROM generate_series(
			DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Sao_Paulo' - INTERVAL '6 days'),
			DATE_TRUNC('day', NOW() AT TIME ZONE 'America/Sao_Paulo'),
			'1 day'::interval
		) d(dia)
		LEFT JOIN pedidos p ON
			DATE_TRUNC('day', p.updated_at AT TIME ZONE 'America/Sao_Paulo') = d.dia
			AND p.loja_id = ?
			AND p.status = 'pago'
		GROUP BY d.dia
		ORDER BY d.dia
	`, lojaID).Scan(&dias)
	data.Receita7Dias = dias

	// Receita por semana — últimas 4 semanas
	var semanas []ReceitaSemana
	s.db.Raw(`
		SELECT
			TO_CHAR(d.semana, 'DD/MM') as semana,
			COALESCE(SUM(p.total), 0) as total
		FROM generate_series(
			DATE_TRUNC('week', NOW() AT TIME ZONE 'America/Sao_Paulo' - INTERVAL '3 weeks'),
			DATE_TRUNC('week', NOW() AT TIME ZONE 'America/Sao_Paulo'),
			'1 week'::interval
		) d(semana)
		LEFT JOIN pedidos p ON
			DATE_TRUNC('week', p.updated_at AT TIME ZONE 'America/Sao_Paulo') = d.semana
			AND p.loja_id = ?
			AND p.status = 'pago'
		GROUP BY d.semana
		ORDER BY d.semana
	`, lojaID).Scan(&semanas)
	data.Receita4Semanas = semanas

	// Top 5 produtos mais vendidos (últimos 30 dias)
	var topProdutos []TopProduto
	s.db.Raw(`
		SELECT
			ip.produto_nome as nome,
			SUM(ip.quantidade) as quantidade
		FROM itens_pedido ip
		JOIN pedidos p ON p.id = ip.pedido_id
		WHERE p.loja_id = ? AND p.status = 'pago'
		AND p.updated_at >= NOW() - INTERVAL '30 days'
		GROUP BY ip.produto_nome
		ORDER BY quantidade DESC
		LIMIT 5
	`, lojaID).Scan(&topProdutos)
	data.TopProdutos = topProdutos

	// Top 5 clientes por quantidade de pedidos e top 5 por valor total
	// gasto (Fase 10.4) — mesmo universo de sempre (só pedidos pagos),
	// sem janela de tempo (é sobre o histórico inteiro da loja, não só
	// os últimos 30 dias como TopProdutos).
	var topClientesPorPedidos []ClienteRanking
	s.db.Raw(`
		WITH nomes_recentes AS (
			SELECT DISTINCT ON (cliente_telefone) cliente_telefone, cliente_nome
			FROM pedidos
			WHERE loja_id = ? AND status = 'pago'
			ORDER BY cliente_telefone, updated_at DESC
		)
		SELECT p.cliente_telefone, nr.cliente_nome, COUNT(*) AS total_pedidos, SUM(p.total) AS valor_total
		FROM pedidos p
		JOIN nomes_recentes nr ON nr.cliente_telefone = p.cliente_telefone
		WHERE p.loja_id = ? AND p.status = 'pago'
		GROUP BY p.cliente_telefone, nr.cliente_nome
		ORDER BY total_pedidos DESC
		LIMIT 5
	`, lojaID, lojaID).Scan(&topClientesPorPedidos)
	data.TopClientesPorPedidos = topClientesPorPedidos

	var topClientesPorValor []ClienteRanking
	s.db.Raw(`
		WITH nomes_recentes AS (
			SELECT DISTINCT ON (cliente_telefone) cliente_telefone, cliente_nome
			FROM pedidos
			WHERE loja_id = ? AND status = 'pago'
			ORDER BY cliente_telefone, updated_at DESC
		)
		SELECT p.cliente_telefone, nr.cliente_nome, COUNT(*) AS total_pedidos, SUM(p.total) AS valor_total
		FROM pedidos p
		JOIN nomes_recentes nr ON nr.cliente_telefone = p.cliente_telefone
		WHERE p.loja_id = ? AND p.status = 'pago'
		GROUP BY p.cliente_telefone, nr.cliente_nome
		ORDER BY valor_total DESC
		LIMIT 5
	`, lojaID, lojaID).Scan(&topClientesPorValor)
	data.TopClientesPorValor = topClientesPorValor

	// Tipo de entrega mais usado (Fase 10.6) — sem janela de tempo, mesmo
	// espírito do ranking de clientes acima.
	var tiposEntrega []Contagem
	s.db.Raw(`
		SELECT modo_entrega AS chave, COUNT(*) AS total
		FROM pedidos
		WHERE loja_id = ? AND status = 'pago'
		GROUP BY modo_entrega
		ORDER BY total DESC
	`, lojaID).Scan(&tiposEntrega)
	data.TiposEntrega = tiposEntrega

	// Forma de pagamento mais usada (Fase 10.6) — só conta pedido que já
	// tem esse dado capturado (forma_pagamento != ''). Pedido pago antes
	// dessa fase fica de fora da conta, não entra como uma categoria
	// "vazia" enganosa.
	var formasPagamento []Contagem
	s.db.Raw(`
		SELECT forma_pagamento AS chave, COUNT(*) AS total
		FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND forma_pagamento != ''
		GROUP BY forma_pagamento
		ORDER BY total DESC
	`, lojaID).Scan(&formasPagamento)
	data.FormasPagamento = formasPagamento

	return data, nil
}

// BuscarResumoPeriodo (Fase 10.5) calcula o resumo de um período exato
// (dia/semana/mês) que contém `data`, no fuso de Brasília. "semana" segue
// segunda a domingo (mesmo critério do DATE_TRUNC('week', ...) do
// Postgres, já usado em Receita4Semanas acima). Diferente do resto desse
// serviço, essa consulta não é uma janela fixa relativa a "agora" — o dono
// escolhe a data de referência no frontend.
func (s *DashboardService) BuscarResumoPeriodo(lojaID uint, tipo string, data time.Time) (*PeriodoResumo, error) {
	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	// `data` chega de time.Parse("2006-01-02", ...), ou seja, meia-noite em
	// UTC — .In(fusoBrasil) deslocaria isso pra 21h do dia ANTERIOR (UTC-3),
	// trocando Year/Month/Day silenciosamente. Em vez de converter o
	// instante, extrai o ano/mês/dia crus (que já são exatamente os dígitos
	// da string recebida) e monta a data diretamente no fuso de Brasília.
	data = time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, fusoBrasil)

	var inicio, fimExclusivo time.Time
	switch tipo {
	case "dia":
		inicio = time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, fusoBrasil)
		fimExclusivo = inicio.AddDate(0, 0, 1)
	case "semana":
		diaBase := time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, fusoBrasil)
		offsetSegunda := (int(diaBase.Weekday()) + 6) % 7 // segunda=0 ... domingo=6
		inicio = diaBase.AddDate(0, 0, -offsetSegunda)
		fimExclusivo = inicio.AddDate(0, 0, 7)
	case "mes":
		inicio = time.Date(data.Year(), data.Month(), 1, 0, 0, 0, 0, fusoBrasil)
		fimExclusivo = inicio.AddDate(0, 1, 0)
	default:
		return nil, fmt.Errorf("tipo de período inválido: %s", tipo)
	}

	resumo := &PeriodoResumo{
		Tipo:   tipo,
		Inicio: inicio.Format("2006-01-02"),
		Fim:    fimExclusivo.AddDate(0, 0, -1).Format("2006-01-02"),
	}

	var linha struct {
		Total      float64
		NumPedidos int
	}
	if err := s.db.Raw(`
		SELECT COALESCE(SUM(total), 0) AS total, COUNT(*) AS num_pedidos
		FROM pedidos
		WHERE loja_id = ? AND status = 'pago' AND updated_at >= ? AND updated_at < ?
	`, lojaID, inicio, fimExclusivo).Scan(&linha).Error; err != nil {
		return nil, err
	}
	resumo.Total = linha.Total
	resumo.NumPedidos = linha.NumPedidos
	if linha.NumPedidos > 0 {
		resumo.TicketMedio = linha.Total / float64(linha.NumPedidos)
	}

	var topProdutos []TopProduto
	s.db.Raw(`
		SELECT ip.produto_nome as nome, SUM(ip.quantidade) as quantidade
		FROM itens_pedido ip
		JOIN pedidos p ON p.id = ip.pedido_id
		WHERE p.loja_id = ? AND p.status = 'pago' AND p.updated_at >= ? AND p.updated_at < ?
		GROUP BY ip.produto_nome
		ORDER BY quantidade DESC
		LIMIT 3
	`, lojaID, inicio, fimExclusivo).Scan(&topProdutos)
	resumo.TopProdutos = topProdutos

	return resumo, nil
}

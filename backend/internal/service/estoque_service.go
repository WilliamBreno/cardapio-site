package service

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// ItemEstoque é uma linha do relatório de estoque (Fase 8, nível 1 —
// Pro e Scale) — um produto sem variação, ou uma variação específica,
// sempre que tiver controle de estoque ativo (EstoqueAtual != nil).
type ItemEstoque struct {
	ProdutoID     uint   `json:"produto_id"`
	ProdutoNome   string `json:"produto_nome"`
	VariacaoID    *uint  `json:"variacao_id"`
	VariacaoNome  string `json:"variacao_nome"`
	EstoqueAtual  int    `json:"estoque_atual"`
	EstoqueAlerta *int   `json:"estoque_alerta"`
	// Critico é true quando zerado ou no limite/abaixo do alerta
	// configurado — usado tanto pra ordenar o relatório quanto pro
	// frontend destacar visualmente.
	Critico bool `json:"critico"`
}

type EstoqueService struct {
	produtoRepo      *repository.ProdutoRepository
	variacaoRepo     *repository.VariacaoRepository
	movimentacaoRepo *repository.MovimentacaoEstoqueRepository
}

func NewEstoqueService(db *gorm.DB) *EstoqueService {
	return &EstoqueService{
		produtoRepo:      repository.NewProdutoRepository(db),
		variacaoRepo:     repository.NewVariacaoRepository(db),
		movimentacaoRepo: repository.NewMovimentacaoEstoqueRepository(db),
	}
}

// Relatorio monta a lista de produtos/variações com estoque controlado
// de uma loja, com os mais críticos primeiro (zerado → abaixo do alerta
// → resto) — Fase 8, nível 1 (Pro e Scale). Só entram itens que o dono
// realmente ativou controle de estoque (EstoqueAtual != nil); produto
// "ilimitado" não aparece aqui, mesmo espírito de ContarPorLoja não
// considerar o que está fora do que se pretende medir.
func (s *EstoqueService) Relatorio(lojaID uint) ([]ItemEstoque, error) {
	produtos, err := s.produtoRepo.ListarPorLoja(lojaID, false)
	if err != nil {
		return nil, err
	}

	itens := make([]ItemEstoque, 0, len(produtos))
	for _, p := range produtos {
		if len(p.Variacoes) == 0 {
			if p.EstoqueAtual == nil {
				continue
			}
			itens = append(itens, montarItemEstoque(p.ID, p.Nome, nil, "", *p.EstoqueAtual, p.EstoqueAlerta))
			continue
		}
		for _, v := range p.Variacoes {
			if v.EstoqueAtual == nil {
				continue
			}
			variacaoID := v.ID
			itens = append(itens, montarItemEstoque(p.ID, p.Nome, &variacaoID, v.Nome, *v.EstoqueAtual, v.EstoqueAlerta))
		}
	}

	sort.SliceStable(itens, func(i, j int) bool {
		return criticidade(itens[i]) < criticidade(itens[j])
	})
	return itens, nil
}

func montarItemEstoque(produtoID uint, produtoNome string, variacaoID *uint, variacaoNome string, estoqueAtual int, estoqueAlerta *int) ItemEstoque {
	return ItemEstoque{
		ProdutoID:     produtoID,
		ProdutoNome:   produtoNome,
		VariacaoID:    variacaoID,
		VariacaoNome:  variacaoNome,
		EstoqueAtual:  estoqueAtual,
		EstoqueAlerta: estoqueAlerta,
		Critico:       estoqueAtual == 0 || (estoqueAlerta != nil && estoqueAtual <= *estoqueAlerta),
	}
}

// criticidade dá a ordem de exibição do relatório: zerado primeiro,
// depois crítico (no limite do alerta), depois o resto.
func criticidade(item ItemEstoque) int {
	switch {
	case item.EstoqueAtual == 0:
		return 0
	case item.Critico:
		return 1
	default:
		return 2
	}
}

// Movimentacoes devolve o histórico de movimentação de estoque da loja —
// Fase 8, nível 2 (Scale). produtoID == 0 não filtra por produto.
func (s *EstoqueService) Movimentacoes(lojaID, produtoID uint) ([]domain.MovimentacaoEstoque, error) {
	return s.movimentacaoRepo.ListarPorLoja(lojaID, produtoID)
}

// ReporInput é um incremento de estoque (delta positivo) — "chegou mais
// mercadoria". VariacaoID nulo mexe no estoque geral do produto.
type ReporInput struct {
	ProdutoID  uint
	VariacaoID *uint
	Quantidade int
	Motivo     string
}

// AjustarInput corrige o estoque pra um valor absoluto (ex: contagem
// física bateu diferente) — motivo é obrigatório aqui porque, diferente
// da reposição, um ajuste esconde um problema (perda, furto, erro de
// contagem anterior) que vale a pena registrar o porquê.
type AjustarInput struct {
	ProdutoID  uint
	VariacaoID *uint
	NovoValor  int
	Motivo     string
}

// Repor soma Quantidade ao estoque atual do produto/variação — Fase 8,
// nível 2 (Scale). Devolve o estoque resultante.
func (s *EstoqueService) Repor(lojaID uint, input ReporInput) (int, error) {
	if input.Quantidade <= 0 {
		return 0, errors.New("quantidade de reposição precisa ser maior que zero")
	}

	atual, err := s.validarDonoEBuscarEstoque(lojaID, input.ProdutoID, input.VariacaoID)
	if err != nil {
		return 0, err
	}
	if atual == nil {
		return 0, errors.New("esse item não tem controle de estoque ativo")
	}

	novoValor := *atual + input.Quantidade
	if err := s.definirEstoque(input.ProdutoID, input.VariacaoID, novoValor); err != nil {
		return 0, err
	}

	s.registrarMovimentacao(lojaID, input.ProdutoID, input.VariacaoID, domain.MovimentoEstoqueReposicao, input.Quantidade, novoValor, input.Motivo, nil)
	return novoValor, nil
}

// Ajustar sobrescreve o estoque pra NovoValor — Fase 8, nível 2 (Scale).
func (s *EstoqueService) Ajustar(lojaID uint, input AjustarInput) (int, error) {
	if input.NovoValor < 0 {
		return 0, errors.New("estoque não pode ser negativo")
	}
	if input.Motivo == "" {
		return 0, errors.New("motivo é obrigatório num ajuste manual")
	}

	atual, err := s.validarDonoEBuscarEstoque(lojaID, input.ProdutoID, input.VariacaoID)
	if err != nil {
		return 0, err
	}
	if atual == nil {
		return 0, errors.New("esse item não tem controle de estoque ativo")
	}

	delta := input.NovoValor - *atual
	if err := s.definirEstoque(input.ProdutoID, input.VariacaoID, input.NovoValor); err != nil {
		return 0, err
	}

	s.registrarMovimentacao(lojaID, input.ProdutoID, input.VariacaoID, domain.MovimentoEstoqueAjuste, delta, input.NovoValor, input.Motivo, nil)
	return input.NovoValor, nil
}

// validarDonoEBuscarEstoque confere que o produto (e a variação, se
// informada) pertence à loja, e devolve o EstoqueAtual encontrado (nil
// se o item não tem controle de estoque ativado).
func (s *EstoqueService) validarDonoEBuscarEstoque(lojaID, produtoID uint, variacaoID *uint) (*int, error) {
	produto, err := s.produtoRepo.BuscarPorID(produtoID)
	if err != nil {
		return nil, errors.New("produto não encontrado")
	}
	if produto.LojaID != lojaID {
		return nil, errors.New("produto não pertence a essa loja")
	}

	if variacaoID == nil {
		return produto.EstoqueAtual, nil
	}

	v, err := s.variacaoRepo.BuscarPorID(*variacaoID)
	if err != nil || v.ProdutoID != produtoID {
		return nil, errors.New("variação não encontrada")
	}
	return v.EstoqueAtual, nil
}

func (s *EstoqueService) definirEstoque(produtoID uint, variacaoID *uint, novoValor int) error {
	if variacaoID == nil {
		produto, err := s.produtoRepo.BuscarPorID(produtoID)
		if err != nil {
			return err
		}
		produto.EstoqueAtual = &novoValor
		if novoValor > 0 {
			produto.Disponivel = true
		}
		if err := s.produtoRepo.Atualizar(produto); err != nil {
			return fmt.Errorf("atualizando estoque do produto: %w", err)
		}
		return nil
	}

	v, err := s.variacaoRepo.BuscarPorID(*variacaoID)
	if err != nil {
		return err
	}
	v.EstoqueAtual = &novoValor
	if novoValor > 0 {
		v.Disponivel = true
	}
	if err := s.variacaoRepo.Atualizar(v); err != nil {
		return fmt.Errorf("atualizando estoque da variação: %w", err)
	}
	return nil
}

func (s *EstoqueService) registrarMovimentacao(lojaID, produtoID uint, variacaoID *uint, tipo domain.TipoMovimentoEstoque, quantidade, estoqueResultante int, motivo string, pedidoID *uint) {
	mov := domain.MovimentacaoEstoque{
		LojaID:            lojaID,
		ProdutoID:         produtoID,
		VariacaoID:        variacaoID,
		Tipo:              tipo,
		Quantidade:        quantidade,
		EstoqueResultante: estoqueResultante,
		Motivo:            motivo,
		PedidoID:          pedidoID,
	}
	// Falha ao registrar o histórico não deve impedir a reposição/ajuste
	// em si (o estoque já foi corrigido) — só perde a rastreabilidade
	// desse evento específico.
	if err := s.movimentacaoRepo.Criar(&mov); err != nil {
		log.Printf("aviso: não foi possível registrar movimentação de estoque (produto %d): %v", produtoID, err)
	}
}

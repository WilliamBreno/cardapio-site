package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type SugestaoProdutoInput struct {
	ProdutoOrigemID   uint
	ProdutoSugeridoID uint
	TipoDesconto      string // "" = sem desconto, senão "percentual" ou "fixo"
	ValorDesconto     float64
}

type SugestaoProdutoService struct {
	sugestaoRepo *repository.SugestaoProdutoRepository
	produtoRepo  *repository.ProdutoRepository
	lojaRepo     *repository.LojaRepository
}

func NewSugestaoProdutoService(db *gorm.DB) *SugestaoProdutoService {
	return &SugestaoProdutoService{
		sugestaoRepo: repository.NewSugestaoProdutoRepository(db),
		produtoRepo:  repository.NewProdutoRepository(db),
		lojaRepo:     repository.NewLojaRepository(db),
	}
}

// SugestaoProdutoComStatus é a versão da tela de configuração — mostra se
// o vínculo está "ativo" de verdade (ver regra do gostinho grátis abaixo).
type SugestaoProdutoComStatus struct {
	domain.SugestaoProduto
	Ativo bool `json:"ativo"`
}

// Listar devolve todos os vínculos da loja, marcando quais estão
// realmente ativos hoje. Sem a Sugestão Inteligente contratada, só o
// vínculo mais antigo (o "grátis") fica ativo — os demais (cadastrados
// enquanto a loja tinha o recurso contratado e depois cancelou, ver
// Fase 6 Parte 3) continuam salvos, só ficam ocultos/inativos até uma
// nova assinatura, sem precisar recadastrar nada.
func (s *SugestaoProdutoService) Listar(lojaID uint) ([]SugestaoProdutoComStatus, error) {
	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return nil, fmt.Errorf("buscando loja: %w", err)
	}
	sugestoes, err := s.sugestaoRepo.ListarPorLoja(lojaID)
	if err != nil {
		return nil, err
	}

	var primeiraID uint
	if !loja.SugestaoInteligenteContratada {
		primeiraID, _ = s.sugestaoRepo.PrimeiraID(lojaID)
	}

	resultado := make([]SugestaoProdutoComStatus, len(sugestoes))
	for i, sugestao := range sugestoes {
		ativo := loja.SugestaoInteligenteContratada || sugestao.ID == primeiraID
		resultado[i] = SugestaoProdutoComStatus{SugestaoProduto: sugestao, Ativo: ativo}
	}
	return resultado, nil
}

// Criar valida a regra central da Fase 6: nunca sugerir o próprio
// produto, e nunca deixar dois produtos sugeridos da MESMA categoria
// vinculados à mesma origem — isso reforça, já na configuração, a regra
// de exibição que só mostra uma sugestão por categoria no carrinho (ver
// MontarSugestoesCarrinho). Sem a Sugestão Inteligente contratada, a loja
// só pode ter 1 vínculo no total — o "gostinho grátis" que funciona de
// verdade pro cliente final, sem limite de uso, só de quantidade.
func (s *SugestaoProdutoService) Criar(lojaID uint, input SugestaoProdutoInput) (*domain.SugestaoProduto, error) {
	if input.ProdutoOrigemID == input.ProdutoSugeridoID {
		return nil, errors.New("um produto não pode sugerir ele mesmo")
	}

	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return nil, fmt.Errorf("buscando loja: %w", err)
	}
	if !loja.SugestaoInteligenteContratada {
		total, err := s.sugestaoRepo.ContarPorLoja(lojaID)
		if err != nil {
			return nil, fmt.Errorf("verificando limite de vínculos: %w", err)
		}
		if total >= 1 {
			return nil, errors.New("você já testou grátis — assine a Sugestão Inteligente pra liberar sem limite")
		}
	}

	origem, err := s.produtoRepo.BuscarPorID(input.ProdutoOrigemID)
	if err != nil || origem.LojaID != lojaID {
		return nil, errors.New("produto de origem não encontrado")
	}
	sugerido, err := s.produtoRepo.BuscarPorID(input.ProdutoSugeridoID)
	if err != nil || sugerido.LojaID != lojaID {
		return nil, errors.New("produto sugerido não encontrado")
	}

	existentes, err := s.sugestaoRepo.ListarPorProdutosOrigem(lojaID, []uint{input.ProdutoOrigemID})
	if err != nil {
		return nil, fmt.Errorf("verificando sugestões existentes: %w", err)
	}
	for _, existente := range existentes {
		if existente.ProdutoSugerido.CategoriaID == sugerido.CategoriaID {
			return nil, fmt.Errorf("já existe uma sugestão da categoria %q pra esse produto — só uma sugestão por categoria", sugerido.Categoria.Nome)
		}
	}

	var tipoDesconto *domain.TipoDesconto
	var valorDesconto *float64
	if input.TipoDesconto != "" {
		t := domain.TipoDesconto(input.TipoDesconto)
		if t != domain.TipoDescontoPercentual && t != domain.TipoDescontoFixo {
			return nil, errors.New("tipo de desconto inválido")
		}
		if input.ValorDesconto <= 0 {
			return nil, errors.New("valor do desconto precisa ser maior que zero")
		}
		if t == domain.TipoDescontoPercentual && input.ValorDesconto > 100 {
			return nil, errors.New("desconto percentual não pode passar de 100%")
		}
		tipoDesconto = &t
		valorDesconto = &input.ValorDesconto
	}

	sugestao := domain.SugestaoProduto{
		LojaID:            lojaID,
		ProdutoOrigemID:   input.ProdutoOrigemID,
		ProdutoSugeridoID: input.ProdutoSugeridoID,
		TipoDesconto:      tipoDesconto,
		ValorDesconto:     valorDesconto,
	}
	if err := s.sugestaoRepo.Criar(&sugestao); err != nil {
		return nil, fmt.Errorf("não foi possível criar a sugestão (vínculo já existe?): %w", err)
	}
	return &sugestao, nil
}

// SugestaoCarrinhoItem é a versão pronta pra exibir na revisão do
// carrinho — já resolvida (uma por categoria, sem produto repetido) e com
// o preço com desconto já calculado.
type SugestaoCarrinhoItem struct {
	SugestaoID       uint    `json:"sugestao_id"`
	ProdutoID        uint    `json:"produto_id"`
	Nome             string  `json:"nome"`
	FotoURL          string  `json:"foto_url"`
	Preco            float64 `json:"preco"`
	PrecoComDesconto float64 `json:"preco_com_desconto"`
	TipoDesconto     string  `json:"tipo_desconto,omitempty"`
	ValorDesconto    float64 `json:"valor_desconto,omitempty"`
}

// MontarSugestoesCarrinho monta a seção consolidada de sugestões pra
// mostrar na revisão do carrinho antes de finalizar (não popup a cada
// produto adicionado) — reúne os vínculos de TODOS os produtos já no
// carrinho (avulso ou componente de combo, por isso recebe a lista pronta
// em vez de calcular aqui) e aplica as três regras da Fase 6: nunca
// sugerir produto que já está no carrinho, nunca duplicar o mesmo produto
// sugerido por mais de uma origem, e no máximo uma sugestão por categoria
// — priorizando sempre a de maior desconto quando há mais de uma opção
// disputando a mesma vaga (mesmo produto ou mesma categoria).
func (s *SugestaoProdutoService) MontarSugestoesCarrinho(lojaID uint, produtosNoCarrinho []uint) ([]SugestaoCarrinhoItem, error) {
	if len(produtosNoCarrinho) == 0 {
		return nil, nil
	}

	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return nil, fmt.Errorf("buscando loja: %w", err)
	}

	sugestoes, err := s.sugestaoRepo.ListarPorProdutosOrigem(lojaID, produtosNoCarrinho)
	if err != nil {
		return nil, fmt.Errorf("buscando sugestões: %w", err)
	}

	// Sem a Sugestão Inteligente contratada, só o vínculo mais antigo da
	// loja (o "gostinho grátis") pode aparecer de verdade pro cliente —
	// os demais, se existirem (loja que já foi assinante e cancelou),
	// ficam ocultos até uma nova assinatura, mas continuam salvos.
	if !loja.SugestaoInteligenteContratada {
		primeiraID, err := s.sugestaoRepo.PrimeiraID(lojaID)
		if err != nil {
			return nil, nil
		}
		filtradas := make([]domain.SugestaoProduto, 0, 1)
		for _, sugestao := range sugestoes {
			if sugestao.ID == primeiraID {
				filtradas = append(filtradas, sugestao)
			}
		}
		sugestoes = filtradas
	}

	noCarrinho := make(map[uint]bool, len(produtosNoCarrinho))
	for _, id := range produtosNoCarrinho {
		noCarrinho[id] = true
	}

	melhorPorProduto := make(map[uint]domain.SugestaoProduto)
	for _, sugestao := range sugestoes {
		if noCarrinho[sugestao.ProdutoSugeridoID] {
			continue
		}
		atual, existe := melhorPorProduto[sugestao.ProdutoSugeridoID]
		if !existe || descontoEmReais(sugestao) > descontoEmReais(atual) {
			melhorPorProduto[sugestao.ProdutoSugeridoID] = sugestao
		}
	}

	melhorPorCategoria := make(map[uint]domain.SugestaoProduto)
	for _, sugestao := range melhorPorProduto {
		categoriaID := sugestao.ProdutoSugerido.CategoriaID
		atual, existe := melhorPorCategoria[categoriaID]
		if !existe || descontoEmReais(sugestao) > descontoEmReais(atual) {
			melhorPorCategoria[categoriaID] = sugestao
		}
	}

	resultado := make([]SugestaoCarrinhoItem, 0, len(melhorPorCategoria))
	for _, sugestao := range melhorPorCategoria {
		base := sugestao.ProdutoSugerido.Preco
		item := SugestaoCarrinhoItem{
			SugestaoID:       sugestao.ID,
			ProdutoID:        sugestao.ProdutoSugeridoID,
			Nome:             sugestao.ProdutoSugerido.Nome,
			FotoURL:          sugestao.ProdutoSugerido.FotoURL,
			Preco:            base,
			PrecoComDesconto: sugestao.PrecoComDesconto(base),
		}
		if sugestao.TipoDesconto != nil {
			item.TipoDesconto = string(*sugestao.TipoDesconto)
			item.ValorDesconto = *sugestao.ValorDesconto
		}
		resultado = append(resultado, item)
	}
	return resultado, nil
}

func descontoEmReais(sugestao domain.SugestaoProduto) float64 {
	base := sugestao.ProdutoSugerido.Preco
	return base - sugestao.PrecoComDesconto(base)
}

func (s *SugestaoProdutoService) Deletar(lojaID, sugestaoID uint) error {
	sugestao, err := s.sugestaoRepo.BuscarPorID(sugestaoID)
	if err != nil {
		return errors.New("sugestão não encontrada")
	}
	if sugestao.LojaID != lojaID {
		return errors.New("sugestão não pertence a essa loja")
	}
	return s.sugestaoRepo.Deletar(sugestaoID)
}

package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type CatalogoService struct {
	lojaRepo         *repository.LojaRepository
	categoriaRepo    *repository.CategoriaRepository
	subcategoriaRepo *repository.SubcategoriaRepository
	grupoCorRepo     *repository.GrupoCorRepository
	produtoRepo      *repository.ProdutoRepository
	comboRepo        *repository.ComboRepository
}

func NewCatalogoService(db *gorm.DB) *CatalogoService {
	return &CatalogoService{
		lojaRepo:         repository.NewLojaRepository(db),
		categoriaRepo:    repository.NewCategoriaRepository(db),
		subcategoriaRepo: repository.NewSubcategoriaRepository(db),
		grupoCorRepo:     repository.NewGrupoCorRepository(db),
		produtoRepo:      repository.NewProdutoRepository(db),
		comboRepo:        repository.NewComboRepository(db),
	}
}

// CardapioPublico é tudo que a página pública de uma loja precisa pra
// renderizar de uma vez: dados da loja, abas (categorias), produtos e
// combos. Subcategorias/GruposCor só têm conteúdo pra lojas "mercadoria" —
// o catálogo em formato lista (alimentício) simplesmente ignora os campos.
type CardapioPublico struct {
	Loja          domain.Loja
	Categorias    []domain.Categoria
	Subcategorias []domain.Subcategoria
	GruposCor     []domain.GrupoCor
	Produtos      []domain.Produto
	Combos        []domain.Combo
}

// BuscarCardapioPorSlug é o que alimenta a rota pública GET /lojas/:slug.
func (s *CatalogoService) BuscarCardapioPorSlug(slug string) (*CardapioPublico, error) {
	loja, err := s.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}

	categorias, err := s.categoriaRepo.ListarPorLoja(loja.ID)
	if err != nil {
		return nil, fmt.Errorf("listando categorias: %w", err)
	}

	subcategorias, err := s.subcategoriaRepo.ListarPorLoja(loja.ID)
	if err != nil {
		return nil, fmt.Errorf("listando subcategorias: %w", err)
	}

	gruposCor, err := s.grupoCorRepo.ListarPorLoja(loja.ID)
	if err != nil {
		return nil, fmt.Errorf("listando grupos de cor: %w", err)
	}

	// true = só produtos disponíveis — cardápio público não mostra item
	// pausado pelo dono.
	produtos, err := s.produtoRepo.ListarPorLoja(loja.ID, true)
	if err != nil {
		return nil, fmt.Errorf("listando produtos: %w", err)
	}

	combos, err := s.comboRepo.ListarDisponiveisPorLoja(loja.ID)
	if err != nil {
		return nil, fmt.Errorf("listando combos: %w", err)
	}

	return &CardapioPublico{
		Loja:          *loja,
		Categorias:    categorias,
		Subcategorias: subcategorias,
		GruposCor:     gruposCor,
		Produtos:      produtos,
		Combos:        combos,
	}, nil
}

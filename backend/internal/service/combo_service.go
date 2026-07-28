package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type ComboItemInput struct {
	ProdutoID  uint
	Quantidade int
}

type ComboInput struct {
	Nome       string
	Descricao  string
	FotoURL    string
	Preco      float64
	Disponivel bool
	Ordem      int
	Itens      []ComboItemInput
}

type ComboService struct {
	comboRepo   *repository.ComboRepository
	produtoRepo *repository.ProdutoRepository
}

func NewComboService(db *gorm.DB) *ComboService {
	return &ComboService{
		comboRepo:   repository.NewComboRepository(db),
		produtoRepo: repository.NewProdutoRepository(db),
	}
}

func (s *ComboService) Listar(lojaID uint) ([]domain.Combo, error) {
	return s.comboRepo.ListarPorLoja(lojaID)
}

// ListarDisponiveis é a versão pro cardápio público.
func (s *ComboService) ListarDisponiveis(lojaID uint) ([]domain.Combo, error) {
	return s.comboRepo.ListarDisponiveisPorLoja(lojaID)
}

func (s *ComboService) Criar(lojaID uint, input ComboInput) (*domain.Combo, error) {
	itens, err := s.validarItens(lojaID, input)
	if err != nil {
		return nil, err
	}

	combo := domain.Combo{
		LojaID:     lojaID,
		Nome:       input.Nome,
		Descricao:  input.Descricao,
		FotoURL:    input.FotoURL,
		Preco:      input.Preco,
		Disponivel: input.Disponivel,
		Ordem:      input.Ordem,
		Itens:      itens,
	}
	if err := s.comboRepo.Criar(&combo); err != nil {
		return nil, fmt.Errorf("criando combo: %w", err)
	}
	return s.comboRepo.BuscarPorID(combo.ID)
}

func (s *ComboService) Atualizar(lojaID, comboID uint, input ComboInput) (*domain.Combo, error) {
	combo, err := s.buscarDaLoja(lojaID, comboID)
	if err != nil {
		return nil, err
	}

	itens, err := s.validarItens(lojaID, input)
	if err != nil {
		return nil, err
	}

	combo.Nome = input.Nome
	combo.Descricao = input.Descricao
	combo.FotoURL = input.FotoURL
	combo.Preco = input.Preco
	combo.Disponivel = input.Disponivel
	combo.Ordem = input.Ordem
	combo.Itens = itens

	if err := s.comboRepo.Atualizar(combo); err != nil {
		return nil, fmt.Errorf("atualizando combo: %w", err)
	}
	return s.comboRepo.BuscarPorID(combo.ID)
}

func (s *ComboService) Deletar(lojaID, comboID uint) error {
	if _, err := s.buscarDaLoja(lojaID, comboID); err != nil {
		return err
	}
	return s.comboRepo.Deletar(comboID)
}

func (s *ComboService) buscarDaLoja(lojaID, comboID uint) (*domain.Combo, error) {
	combo, err := s.comboRepo.BuscarPorID(comboID)
	if err != nil {
		return nil, errors.New("combo não encontrado")
	}
	if combo.LojaID != lojaID {
		return nil, errors.New("combo não pertence a essa loja")
	}
	return combo, nil
}

// validarItens garante que o combo tem pelo menos um item, e que todo
// produto componente pertence de fato à mesma loja do combo — sem isso,
// um lojista poderia (por engano ou não) montar um combo com produto de
// outra loja.
func (s *ComboService) validarItens(lojaID uint, input ComboInput) ([]domain.ComboItem, error) {
	if input.Nome == "" {
		return nil, errors.New("nome do combo é obrigatório")
	}
	if input.Preco <= 0 {
		return nil, errors.New("preço do combo precisa ser maior que zero")
	}
	if len(input.Itens) == 0 {
		return nil, errors.New("o combo precisa ter pelo menos um produto")
	}

	itens := make([]domain.ComboItem, 0, len(input.Itens))
	for _, itemInput := range input.Itens {
		if itemInput.Quantidade <= 0 {
			return nil, fmt.Errorf("quantidade inválida pro produto %d", itemInput.ProdutoID)
		}
		produto, err := s.produtoRepo.BuscarPorID(itemInput.ProdutoID)
		if err != nil {
			return nil, fmt.Errorf("produto %d não encontrado", itemInput.ProdutoID)
		}
		if produto.LojaID != lojaID {
			return nil, fmt.Errorf("produto %q não pertence a essa loja", produto.Nome)
		}
		itens = append(itens, domain.ComboItem{
			ProdutoID:  itemInput.ProdutoID,
			Quantidade: itemInput.Quantidade,
		})
	}
	return itens, nil
}

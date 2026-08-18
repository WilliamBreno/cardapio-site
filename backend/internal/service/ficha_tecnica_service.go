package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// FichaTecnicaItemInput é um insumo + quantidade recebido do formulário —
// separado do domain.FichaTecnicaItem pelo mesmo motivo de ComboItemInput.
type FichaTecnicaItemInput struct {
	InsumoID   uint
	Quantidade float64
}

// FichaTecnica é a resposta montada pro admin: os itens (já com o insumo
// carregado) mais o CMV e a margem já calculados — o cliente da API não
// precisa saber a fórmula, só ler o resultado.
type FichaTecnica struct {
	Itens  []domain.FichaTecnicaItem `json:"itens"`
	CMV    float64                   `json:"cmv"`
	Preco  float64                   `json:"preco"`
	Margem float64                   `json:"margem"`
}

type FichaTecnicaService struct {
	fichaTecnicaRepo *repository.FichaTecnicaRepository
	produtoRepo      *repository.ProdutoRepository
	insumoRepo       *repository.InsumoRepository
}

func NewFichaTecnicaService(db *gorm.DB) *FichaTecnicaService {
	return &FichaTecnicaService{
		fichaTecnicaRepo: repository.NewFichaTecnicaRepository(db),
		produtoRepo:      repository.NewProdutoRepository(db),
		insumoRepo:       repository.NewInsumoRepository(db),
	}
}

// Buscar devolve a ficha técnica atual de um produto (vazia se ainda não
// tiver sido cadastrada) com CMV e margem já calculados. CMV nunca fica
// desatualizado porque é somado na hora, a partir do custo por unidade de
// uso de cada insumo (ver Insumo.CustoPorUnidadeUso) — não existe um
// valor de CMV armazenado que precisaria ser recalculado/invalidado.
func (s *FichaTecnicaService) Buscar(lojaID, produtoID uint) (*FichaTecnica, error) {
	produto, err := s.buscarProdutoDaLoja(lojaID, produtoID)
	if err != nil {
		return nil, err
	}

	itens, err := s.fichaTecnicaRepo.BuscarPorProduto(produtoID)
	if err != nil {
		return nil, fmt.Errorf("buscando ficha técnica: %w", err)
	}

	return montarFichaTecnica(itens, produto.Preco), nil
}

// Salvar substitui a ficha técnica de um produto por completo — mesmo
// padrão de "salva a lista inteira de uma vez" já usado em
// ComboService.Atualizar. Lista vazia é permitida (remove a ficha
// técnica do produto, ele volta a usar o estoque simples de sempre).
func (s *FichaTecnicaService) Salvar(lojaID, produtoID uint, itensInput []FichaTecnicaItemInput) (*FichaTecnica, error) {
	produto, err := s.buscarProdutoDaLoja(lojaID, produtoID)
	if err != nil {
		return nil, err
	}

	itens := make([]domain.FichaTecnicaItem, 0, len(itensInput))
	for _, itemInput := range itensInput {
		if itemInput.Quantidade <= 0 {
			return nil, fmt.Errorf("quantidade inválida pro insumo %d", itemInput.InsumoID)
		}
		insumo, err := s.insumoRepo.BuscarPorID(itemInput.InsumoID)
		if err != nil {
			return nil, fmt.Errorf("insumo %d não encontrado", itemInput.InsumoID)
		}
		if insumo.LojaID != lojaID {
			return nil, fmt.Errorf("insumo %q não pertence a essa loja", insumo.Nome)
		}
		itens = append(itens, domain.FichaTecnicaItem{
			InsumoID:   itemInput.InsumoID,
			Quantidade: itemInput.Quantidade,
		})
	}

	if err := s.fichaTecnicaRepo.Salvar(produtoID, itens); err != nil {
		return nil, fmt.Errorf("salvando ficha técnica: %w", err)
	}

	itensSalvos, err := s.fichaTecnicaRepo.BuscarPorProduto(produtoID)
	if err != nil {
		return nil, fmt.Errorf("recarregando ficha técnica: %w", err)
	}
	return montarFichaTecnica(itensSalvos, produto.Preco), nil
}

func (s *FichaTecnicaService) buscarProdutoDaLoja(lojaID, produtoID uint) (*domain.Produto, error) {
	produto, err := s.produtoRepo.BuscarPorID(produtoID)
	if err != nil {
		return nil, errors.New("produto não encontrado")
	}
	if produto.LojaID != lojaID {
		return nil, errors.New("produto não pertence a essa loja")
	}
	return produto, nil
}

func montarFichaTecnica(itens []domain.FichaTecnicaItem, preco float64) *FichaTecnica {
	cmv := 0.0
	for _, item := range itens {
		cmv += item.Quantidade * item.Insumo.CustoPorUnidadeUso()
	}
	return &FichaTecnica{
		Itens:  itens,
		CMV:    cmv,
		Preco:  preco,
		Margem: preco - cmv,
	}
}

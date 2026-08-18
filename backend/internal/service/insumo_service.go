package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// InsumoInput são os campos editáveis de um Insumo — separado do
// domain.Insumo pra não deixar o handler montar a struct de banco direto
// a partir do JSON recebido (mesmo padrão de ComboInput/ProdutoInput).
type InsumoInput struct {
	Nome               string
	UnidadeCompra      string
	UnidadeUso         string
	FatorConversao     float64
	CustoUnidadeCompra float64
	EstoqueAtual       *float64
	EstoqueAlerta      *float64
}

type InsumoService struct {
	insumoRepo       *repository.InsumoRepository
	fichaTecnicaRepo *repository.FichaTecnicaRepository
}

func NewInsumoService(db *gorm.DB) *InsumoService {
	return &InsumoService{
		insumoRepo:       repository.NewInsumoRepository(db),
		fichaTecnicaRepo: repository.NewFichaTecnicaRepository(db),
	}
}

func (s *InsumoService) Listar(lojaID uint) ([]domain.Insumo, error) {
	return s.insumoRepo.ListarPorLoja(lojaID)
}

func (s *InsumoService) Criar(lojaID uint, input InsumoInput) (*domain.Insumo, error) {
	if err := validarInsumoInput(input); err != nil {
		return nil, err
	}
	insumo := domain.Insumo{
		LojaID:             lojaID,
		Nome:               input.Nome,
		UnidadeCompra:      input.UnidadeCompra,
		UnidadeUso:         input.UnidadeUso,
		FatorConversao:     input.FatorConversao,
		CustoUnidadeCompra: input.CustoUnidadeCompra,
		EstoqueAtual:       input.EstoqueAtual,
		EstoqueAlerta:      input.EstoqueAlerta,
	}
	if err := s.insumoRepo.Criar(&insumo); err != nil {
		return nil, fmt.Errorf("criando insumo: %w", err)
	}
	return &insumo, nil
}

func (s *InsumoService) Atualizar(lojaID, insumoID uint, input InsumoInput) (*domain.Insumo, error) {
	if err := validarInsumoInput(input); err != nil {
		return nil, err
	}
	insumo, err := s.buscarDaLoja(lojaID, insumoID)
	if err != nil {
		return nil, err
	}

	insumo.Nome = input.Nome
	insumo.UnidadeCompra = input.UnidadeCompra
	insumo.UnidadeUso = input.UnidadeUso
	insumo.FatorConversao = input.FatorConversao
	insumo.CustoUnidadeCompra = input.CustoUnidadeCompra
	insumo.EstoqueAtual = input.EstoqueAtual
	insumo.EstoqueAlerta = input.EstoqueAlerta

	if err := s.insumoRepo.Atualizar(insumo); err != nil {
		return nil, fmt.Errorf("atualizando insumo: %w", err)
	}
	return insumo, nil
}

// Deletar recusa a exclusão se o insumo ainda estiver numa ficha técnica
// — mesmo espírito de SubcategoriaService.Deletar/ComboRepository — sem
// isso, a ficha técnica de um produto ficaria referenciando um insumo
// que não existe mais.
func (s *InsumoService) Deletar(lojaID, insumoID uint) error {
	insumo, err := s.buscarDaLoja(lojaID, insumoID)
	if err != nil {
		return err
	}

	usos, err := s.fichaTecnicaRepo.ContarUsosInsumo(insumo.ID)
	if err != nil {
		return fmt.Errorf("verificando uso do insumo: %w", err)
	}
	if usos > 0 {
		return errors.New("não é possível excluir um insumo usado em alguma ficha técnica — remova-o das fichas técnicas primeiro")
	}

	return s.insumoRepo.Deletar(insumo.ID)
}

func (s *InsumoService) buscarDaLoja(lojaID, insumoID uint) (*domain.Insumo, error) {
	insumo, err := s.insumoRepo.BuscarPorID(insumoID)
	if err != nil {
		return nil, errors.New("insumo não encontrado")
	}
	if insumo.LojaID != lojaID {
		return nil, errors.New("insumo não pertence a essa loja")
	}
	return insumo, nil
}

func validarInsumoInput(input InsumoInput) error {
	if input.Nome == "" {
		return errors.New("nome do insumo é obrigatório")
	}
	if input.UnidadeCompra == "" || input.UnidadeUso == "" {
		return errors.New("unidade de compra e unidade de uso são obrigatórias")
	}
	if input.FatorConversao <= 0 {
		return errors.New("fator de conversão precisa ser maior que zero")
	}
	if input.CustoUnidadeCompra < 0 {
		return errors.New("custo não pode ser negativo")
	}
	if input.EstoqueAtual != nil && *input.EstoqueAtual < 0 {
		return errors.New("estoque não pode ser negativo")
	}
	return nil
}

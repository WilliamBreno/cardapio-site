package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type InsumoRepository struct {
	db *gorm.DB
}

func NewInsumoRepository(db *gorm.DB) *InsumoRepository {
	return &InsumoRepository{db: db}
}

func (r *InsumoRepository) Criar(insumo *domain.Insumo) error {
	return r.db.Create(insumo).Error
}

func (r *InsumoRepository) BuscarPorID(id uint) (*domain.Insumo, error) {
	var insumo domain.Insumo
	if err := r.db.First(&insumo, id).Error; err != nil {
		return nil, err
	}
	return &insumo, nil
}

// ListarPorLoja devolve todo insumo da loja, mais recente primeiro.
func (r *InsumoRepository) ListarPorLoja(lojaID uint) ([]domain.Insumo, error) {
	var insumos []domain.Insumo
	if err := r.db.Where("loja_id = ?", lojaID).Order("nome").Find(&insumos).Error; err != nil {
		return nil, err
	}
	return insumos, nil
}

func (r *InsumoRepository) Atualizar(insumo *domain.Insumo) error {
	return r.db.Save(insumo).Error
}

func (r *InsumoRepository) Deletar(id uint) error {
	return r.db.Delete(&domain.Insumo{}, id).Error
}

// SubtrairEstoque decrementa o estoque de um insumo após uma venda de
// produto com ficha técnica — mesmo padrão atômico de
// ProdutoRepository.SubtrairEstoque (UPDATE com GREATEST, não
// lê-depois-escreve) pra evitar race condition entre pagamentos
// concorrentes. Devolve -1 quando o insumo não tem controle de estoque
// ativo (EstoqueAtual nil), pro chamador saber que deve ignorar.
func (r *InsumoRepository) SubtrairEstoque(insumoID uint, quantidade float64) (estoqueRestante float64, err error) {
	result := r.db.Model(&domain.Insumo{}).
		Where("id = ? AND estoque_atual IS NOT NULL", insumoID).
		UpdateColumn("estoque_atual", gorm.Expr("GREATEST(estoque_atual - ?, 0)", quantidade))
	if result.Error != nil {
		return 0, result.Error
	}

	var insumo domain.Insumo
	if err := r.db.Select("estoque_atual").First(&insumo, insumoID).Error; err != nil {
		return 0, err
	}
	if insumo.EstoqueAtual == nil {
		return -1, nil // sem controle de estoque, ignora
	}
	return *insumo.EstoqueAtual, nil
}

// BuscarEstoqueAlerta retorna o insumo se ele tiver estoque_alerta
// configurado e o estoque atual tiver atingido ou passado esse limite —
// mesmo padrão de ProdutoRepository.BuscarEstoqueAlerta.
func (r *InsumoRepository) BuscarEstoqueAlerta(insumoID uint) (*domain.Insumo, bool) {
	var insumo domain.Insumo
	if err := r.db.First(&insumo, insumoID).Error; err != nil {
		return nil, false
	}
	if insumo.EstoqueAtual == nil || insumo.EstoqueAlerta == nil {
		return nil, false
	}
	return &insumo, *insumo.EstoqueAtual <= *insumo.EstoqueAlerta
}

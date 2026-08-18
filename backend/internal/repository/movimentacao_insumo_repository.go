package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type MovimentacaoInsumoRepository struct {
	db *gorm.DB
}

func NewMovimentacaoInsumoRepository(db *gorm.DB) *MovimentacaoInsumoRepository {
	return &MovimentacaoInsumoRepository{db: db}
}

func (r *MovimentacaoInsumoRepository) Criar(m *domain.MovimentacaoInsumo) error {
	return r.db.Create(m).Error
}

// ListarPorLoja devolve o histórico de movimentação de insumo de uma
// loja, mais recente primeiro — opcionalmente filtrado por insumo
// (insumoID == 0 não filtra). Mesmo limite de 500 linhas de
// MovimentacaoEstoqueRepository.ListarPorLoja.
func (r *MovimentacaoInsumoRepository) ListarPorLoja(lojaID, insumoID uint) ([]domain.MovimentacaoInsumo, error) {
	query := r.db.Where("loja_id = ?", lojaID)
	if insumoID != 0 {
		query = query.Where("insumo_id = ?", insumoID)
	}
	var movimentacoes []domain.MovimentacaoInsumo
	if err := query.Order("created_at DESC, id DESC").Limit(500).Find(&movimentacoes).Error; err != nil {
		return nil, err
	}
	return movimentacoes, nil
}

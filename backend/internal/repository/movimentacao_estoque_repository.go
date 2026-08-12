package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type MovimentacaoEstoqueRepository struct {
	db *gorm.DB
}

func NewMovimentacaoEstoqueRepository(db *gorm.DB) *MovimentacaoEstoqueRepository {
	return &MovimentacaoEstoqueRepository{db: db}
}

func (r *MovimentacaoEstoqueRepository) Criar(m *domain.MovimentacaoEstoque) error {
	return r.db.Create(m).Error
}

// ListarPorLoja devolve o histórico de movimentação de uma loja, mais
// recente primeiro — opcionalmente filtrado por produto (produtoID == 0
// não filtra). Limitado a 500 linhas: é um histórico de consulta rápida,
// não um relatório de exportação completa.
func (r *MovimentacaoEstoqueRepository) ListarPorLoja(lojaID, produtoID uint) ([]domain.MovimentacaoEstoque, error) {
	query := r.db.Where("loja_id = ?", lojaID)
	if produtoID != 0 {
		query = query.Where("produto_id = ?", produtoID)
	}
	var movimentacoes []domain.MovimentacaoEstoque
	if err := query.Order("created_at DESC, id DESC").Limit(500).Find(&movimentacoes).Error; err != nil {
		return nil, err
	}
	return movimentacoes, nil
}

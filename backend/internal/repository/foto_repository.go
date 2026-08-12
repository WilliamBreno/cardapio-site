package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type FotoRepository struct {
	db *gorm.DB
}

func NewFotoRepository(db *gorm.DB) *FotoRepository {
	return &FotoRepository{db: db}
}

func (r *FotoRepository) ListarPorProduto(produtoID uint) ([]domain.FotoProduto, error) {
	var fotos []domain.FotoProduto
	err := r.db.Where("produto_id = ?", produtoID).Order("ordem, id").Find(&fotos).Error
	return fotos, err
}

func (r *FotoRepository) Adicionar(foto *domain.FotoProduto) error {
	return r.db.Create(foto).Error
}

func (r *FotoRepository) Deletar(id, produtoID uint) error {
	return r.db.Where("id = ? AND produto_id = ?", id, produtoID).Delete(&domain.FotoProduto{}).Error
}

// ReordenarTodas grava a nova ordem de todas as fotos de uma vez, dentro
// de uma transação — sem isso, uma falha no meio do laço (ex: conexão
// caindo) deixava a ordem inconsistente (algumas fotos já com o índice
// novo, outras ainda com o antigo), sem nenhum jeito de saber que
// aconteceu uma escrita parcial.
func (r *FotoRepository) ReordenarTodas(produtoID uint, ids []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&domain.FotoProduto{}).
				Where("id = ? AND produto_id = ?", id, produtoID).
				Update("ordem", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

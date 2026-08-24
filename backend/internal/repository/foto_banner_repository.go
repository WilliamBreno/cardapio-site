package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type FotoBannerRepository struct {
	db *gorm.DB
}

func NewFotoBannerRepository(db *gorm.DB) *FotoBannerRepository {
	return &FotoBannerRepository{db: db}
}

func (r *FotoBannerRepository) ListarPorLoja(lojaID uint) ([]domain.FotoBanner, error) {
	var fotos []domain.FotoBanner
	err := r.db.Where("loja_id = ?", lojaID).Order("ordem, id").Find(&fotos).Error
	return fotos, err
}

func (r *FotoBannerRepository) Adicionar(foto *domain.FotoBanner) error {
	return r.db.Create(foto).Error
}

func (r *FotoBannerRepository) Deletar(id, lojaID uint) error {
	return r.db.Where("id = ? AND loja_id = ?", id, lojaID).Delete(&domain.FotoBanner{}).Error
}

// ReordenarTodas grava a nova ordem de todas as fotos de uma vez, dentro
// de uma transação — mesmo padrão de FotoRepository.ReordenarTodas
// (produto), evita ordem inconsistente se a escrita falhar no meio.
func (r *FotoBannerRepository) ReordenarTodas(lojaID uint, ids []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&domain.FotoBanner{}).
				Where("id = ? AND loja_id = ?", id, lojaID).
				Update("ordem", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MigrarBannerUnico é uma migração de dado, chamada uma vez no boot da API
// (main.go, depois do AutoMigrate) — cria uma FotoBanner a partir do
// antigo Loja.BannerURL (campo único, formato anterior a este redesign)
// pra toda loja que já tinha um banner configurado e ainda não tem
// nenhuma linha na tabela nova. Sem isso, quem já tinha configurado um
// banner via o campo antigo o perderia silenciosamente do cardápio
// público na primeira vez que essa versão subisse.
func (r *FotoBannerRepository) MigrarBannerUnico() error {
	return r.db.Exec(`
		INSERT INTO fotos_banner (loja_id, url, ordem)
		SELECT id, banner_url, 0 FROM lojas
		WHERE banner_url <> ''
		AND id NOT IN (SELECT DISTINCT loja_id FROM fotos_banner)
	`).Error
}

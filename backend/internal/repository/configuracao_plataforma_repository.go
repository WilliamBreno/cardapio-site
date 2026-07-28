package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type ConfiguracaoPlataformaRepository struct {
	db *gorm.DB
}

func NewConfiguracaoPlataformaRepository(db *gorm.DB) *ConfiguracaoPlataformaRepository {
	return &ConfiguracaoPlataformaRepository{db: db}
}

// Buscar devolve a linha única de configuração (ID 1) — ver
// cmd/api/main.go, que garante essa linha existir (com os valores
// padrão) logo após rodar as migrations.
func (r *ConfiguracaoPlataformaRepository) Buscar() (*domain.ConfiguracaoPlataforma, error) {
	var cfg domain.ConfiguracaoPlataforma
	if err := r.db.First(&cfg, 1).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

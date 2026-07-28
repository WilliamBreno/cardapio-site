package repository

import (
	"fmt"

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

// SalvarPreapprovalPlanID cacheia o ID do preapproval_plan do Mercado
// Pago criado sob demanda (get-or-create) pra "pro", "scale" ou
// "sugestao" — ver MercadoPagoAssinaturaService.obterOuCriarPreapprovalPlan.
func (r *ConfiguracaoPlataformaRepository) SalvarPreapprovalPlanID(plano, id string) error {
	coluna := map[string]string{
		"pro":      "mercado_pago_preapproval_plan_pro_id",
		"scale":    "mercado_pago_preapproval_plan_scale_id",
		"sugestao": "mercado_pago_preapproval_plan_sugestao_id",
	}[plano]
	if coluna == "" {
		return fmt.Errorf("plano desconhecido pra salvar preapproval_plan: %q", plano)
	}
	return r.db.Model(&domain.ConfiguracaoPlataforma{}).Where("id = 1").Update(coluna, id).Error
}

package service

import (
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type ConfiguracaoPlataformaService struct {
	repo *repository.ConfiguracaoPlataformaRepository
}

func NewConfiguracaoPlataformaService(db *gorm.DB) *ConfiguracaoPlataformaService {
	return &ConfiguracaoPlataformaService{repo: repository.NewConfiguracaoPlataformaRepository(db)}
}

// PrecoSugestaoInteligente lê o valor mensal direto da tabela — nunca
// hardcoded, pra dar pra ajustar sem redeploy (ver
// docs/plano-melhorias-drenux.md, Fase 6). Se por algum motivo a linha
// não existir (não deveria — main.go garante ela no boot), cai num
// padrão de segurança em vez de quebrar a tela que pediu o preço.
func (s *ConfiguracaoPlataformaService) PrecoSugestaoInteligente() float64 {
	cfg, err := s.repo.Buscar()
	if err != nil {
		return 19.90
	}
	return cfg.SugestaoInteligentePrecoMensal
}

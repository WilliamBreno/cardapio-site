package service

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

type LojaService struct {
	lojaRepo *repository.LojaRepository
}

func NewLojaService(db *gorm.DB) *LojaService {
	return &LojaService{lojaRepo: repository.NewLojaRepository(db)}
}

func (s *LojaService) Buscar(lojaID uint) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorID(lojaID)
}

func (s *LojaService) BuscarPorSlug(slug string) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorSlug(slug)
}

func (s *LojaService) AtualizarConfiguracoes(lojaID uint, cfg repository.ConfiguracoesLoja) error {
	// Achado investigando entrega de WhatsApp (ver Tarefa 0): esse campo
	// nunca era normalizado — um número salvo com formatação (espaço,
	// hífen, sem o 55) resolve pro JID errado na hora de notificar o
	// lojista, sem erro nenhum visível pra ele. Ver NormalizarTelefone.
	cfg.WhatsappNumero = NormalizarTelefone(cfg.WhatsappNumero)
	return s.lojaRepo.AtualizarConfiguracoes(lojaID, cfg)
}

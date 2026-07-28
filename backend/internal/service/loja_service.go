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

	// Sugestão Inteligente (Fase 6) é um recurso pago à parte — o toggle só
	// pode ficar ligado se a loja realmente contratou. Não confia no que
	// veio do form: relê o estado atual da loja e força desligado se
	// SugestaoInteligenteContratada for false, em vez de rejeitar a
	// atualização inteira (o resto do form continua salvando normalmente).
	if cfg.SugestaoInteligenteAtiva {
		loja, err := s.lojaRepo.BuscarPorID(lojaID)
		if err != nil {
			return err
		}
		if !loja.SugestaoInteligenteContratada {
			cfg.SugestaoInteligenteAtiva = false
		}
	}

	return s.lojaRepo.AtualizarConfiguracoes(lojaID, cfg)
}

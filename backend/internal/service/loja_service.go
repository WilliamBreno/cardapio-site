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

	// Sugestão Inteligente (Fase 6): o toggle liga/desliga a exibição no
	// carrinho do cliente livremente, contratada ou não — mesmo sem
	// contratar, a loja tem direito a 1 vínculo grátis que precisa
	// aparecer de verdade se o toggle estiver ligado (ver
	// SugestaoProdutoService, que já limita a QUANTIDADE de vínculos, não
	// a exibição). Não há mais trava aqui.

	return s.lojaRepo.AtualizarConfiguracoes(lojaID, cfg)
}

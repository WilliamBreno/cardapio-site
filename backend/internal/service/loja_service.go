package service

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// contasComPermissaoTrocarSegmento lista os únicos emails que podem
// alterar o SegmentoPrincipal de uma loja depois do cadastro — pra
// qualquer outra conta, a escolha feita no cadastro é definitiva (pedido
// do William: sem isso, a pergunta do cadastro perde o sentido, já que
// dela dependem regras de negócio inteiras — Variação x Subcategoria/
// Grupo, categoria sugerida no onboarding do Mercado Pago, etc). São
// contas de teste/administração, não lojistas reais.
var contasComPermissaoTrocarSegmento = map[string]bool{
	"teste-mp-assinatura@example.com": true,
	"drenuxtest@test.com":             true,
	"lojasapato@test.com":             true,
}

type LojaService struct {
	lojaRepo    *repository.LojaRepository
	usuarioRepo *repository.UsuarioRepository
}

func NewLojaService(db *gorm.DB) *LojaService {
	return &LojaService{
		lojaRepo:    repository.NewLojaRepository(db),
		usuarioRepo: repository.NewUsuarioRepository(db),
	}
}

func (s *LojaService) Buscar(lojaID uint) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorID(lojaID)
}

func (s *LojaService) BuscarPorSlug(slug string) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorSlug(slug)
}

// PodeEditarSegmento diz se o usuário logado pode alterar o
// SegmentoPrincipal da própria loja depois do cadastro — ver
// contasComPermissaoTrocarSegmento.
func (s *LojaService) PodeEditarSegmento(usuarioID uint) bool {
	usuario, err := s.usuarioRepo.BuscarPorID(usuarioID)
	if err != nil {
		return false
	}
	return contasComPermissaoTrocarSegmento[usuario.Email]
}

func (s *LojaService) AtualizarConfiguracoes(lojaID, usuarioID uint, cfg repository.ConfiguracoesLoja) error {
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

	// SegmentoPrincipal é definitivo desde o cadastro — só as contas em
	// contasComPermissaoTrocarSegmento podem mudar depois. Pra qualquer
	// outra conta, ignora silenciosamente o valor que veio do formulário
	// e mantém o segmento atual, sem travar o resto do salvamento (mesmo
	// espírito da trava que existia pra Sugestão Inteligente: nunca falha
	// o form inteiro por causa de um campo só). O frontend já deixa esse
	// campo travado pra quem não tem permissão — isso aqui é defesa
	// contra uma chamada direta na API.
	lojaAtual, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return err
	}
	if cfg.SegmentoPrincipal != string(lojaAtual.SegmentoPrincipal) && !s.PodeEditarSegmento(usuarioID) {
		cfg.SegmentoPrincipal = string(lojaAtual.SegmentoPrincipal)
	}

	return s.lojaRepo.AtualizarConfiguracoes(lojaID, cfg)
}

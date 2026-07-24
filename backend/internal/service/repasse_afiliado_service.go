package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// RepasseAfiliadoService cuida do controle manual de comissão de afiliado
// pra pedidos pagos via Mercado Pago (ver domain.RepasseAfiliado e
// docs/plano-melhorias-drenux.md, Fase 5.5) — não movimenta dinheiro, só
// registra quanto é devido e se já foi pago.
type RepasseAfiliadoService struct {
	repasseRepo *repository.RepasseAfiliadoRepository
}

func NewRepasseAfiliadoService(db *gorm.DB) *RepasseAfiliadoService {
	return &RepasseAfiliadoService{
		repasseRepo: repository.NewRepasseAfiliadoRepository(db),
	}
}

// RegistrarPendente cria o lançamento de comissão devida ao afiliado de
// um pedido pago via Mercado Pago — chamado na confirmação do pagamento
// (ver MercadoPagoService.ProcessarNotificacaoPagamento). Idempotente: se
// a notificação repetir pro mesmo pedido, só ignora em vez de duplicar.
func (s *RepasseAfiliadoService) RegistrarPendente(afiliadoID, pedidoID, lojaID uint, valor float64) error {
	if valor <= 0 {
		return nil
	}

	existe, err := s.repasseRepo.ExistePorPedido(pedidoID)
	if err != nil {
		return fmt.Errorf("verificando repasse existente do pedido %d: %w", pedidoID, err)
	}
	if existe {
		return nil
	}

	repasse := domain.RepasseAfiliado{
		AfiliadoID: afiliadoID,
		PedidoID:   pedidoID,
		LojaID:     lojaID,
		Valor:      valor,
		Status:     domain.StatusRepassePendente,
	}
	if err := s.repasseRepo.Criar(&repasse); err != nil {
		return fmt.Errorf("criando repasse do pedido %d: %w", pedidoID, err)
	}
	return nil
}

// ExtratoAfiliado é o que o próprio afiliado vê no painel dele: histórico
// completo (pendente + pago) e o total ainda pendente.
type ExtratoAfiliado struct {
	Repasses      []domain.RepasseAfiliado `json:"repasses"`
	TotalPendente float64                  `json:"total_pendente"`
}

func (s *RepasseAfiliadoService) ExtratoAfiliado(afiliadoID uint) (*ExtratoAfiliado, error) {
	repasses, err := s.repasseRepo.ListarPorAfiliado(afiliadoID)
	if err != nil {
		return nil, err
	}
	total, err := s.repasseRepo.SomarPendentePorAfiliado(afiliadoID)
	if err != nil {
		return nil, err
	}
	return &ExtratoAfiliado{Repasses: repasses, TotalPendente: total}, nil
}

// PendentesAgrupado é a visão geral do admin Drenux: um afiliado por
// linha, com o total pendente somado — usada pra decidir quem repassar.
func (s *RepasseAfiliadoService) PendentesAgrupado() ([]repository.PendentePorAfiliado, error) {
	return s.repasseRepo.ListarPendentesAgrupado()
}

// DetalheAfiliado é o extrato completo de um afiliado específico, visto
// pelo admin Drenux (mesma consulta do painel do afiliado, só que
// acessada por ID em vez de pelo token de quem está logado).
func (s *RepasseAfiliadoService) DetalheAfiliado(afiliadoID uint) ([]domain.RepasseAfiliado, error) {
	return s.repasseRepo.ListarPorAfiliado(afiliadoID)
}

// MarcarComoPago marca um ou mais lançamentos como pagos — chamado pelo
// admin Drenux depois de fazer o repasse via Pix manualmente, fora do
// sistema. Não movimenta dinheiro nenhum, só registra a confirmação.
func (s *RepasseAfiliadoService) MarcarComoPago(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("nenhum lançamento selecionado")
	}
	return s.repasseRepo.MarcarComoPago(ids)
}

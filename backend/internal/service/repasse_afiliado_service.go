package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// RepasseAfiliadoService cuida do controle manual de comissão de afiliado
// pra pedidos pagos via Mercado Pago (ver domain.RepasseAfiliado e
// docs/plano-melhorias-drenux.md, Fase 5.5) — não movimenta dinheiro, só
// registra quanto é devido e se já foi pago.
type RepasseAfiliadoService struct {
	repasseRepo  *repository.RepasseAfiliadoRepository
	afiliadoRepo *repository.AfiliadoRepository
	pedidoRepo   *repository.PedidoRepository
}

func NewRepasseAfiliadoService(db *gorm.DB) *RepasseAfiliadoService {
	return &RepasseAfiliadoService{
		repasseRepo:  repository.NewRepasseAfiliadoRepository(db),
		afiliadoRepo: repository.NewAfiliadoRepository(db),
		pedidoRepo:   repository.NewPedidoRepository(db),
	}
}

// RegistrarPendente calcula (usando o percentual de comissão próprio
// desse afiliado — ver domain.Afiliado.ComissaoPercentual) e registra o
// valor devido por um pedido pago via Mercado Pago. baseComissao já vem
// sem a taxa de entrega (ver MercadoPagoService.CriarCheckout — comissão
// não incide sobre frete). gmvAntes é o GMV do mês da loja antes desse
// pedido, usado pra achar a faixa de comissão certa (ver
// calcularComissaoEscalonada, Fase 7.2). Idempotente: se a notificação do
// Mercado Pago repetir pro mesmo pedido, só ignora em vez de duplicar.
func (s *RepasseAfiliadoService) RegistrarPendente(afiliadoID, pedidoID, lojaID uint, baseComissao float64, plano string, gmvAntes float64) error {
	afiliado, err := s.afiliadoRepo.BuscarPorID(afiliadoID)
	if err != nil {
		return fmt.Errorf("buscando afiliado %d: %w", afiliadoID, err)
	}

	valor := calcularComissaoAfiliado(baseComissao, plano, gmvAntes, afiliado.ComissaoPercentual)
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
		PedidoID:   &pedidoID,
		Tipo:       domain.TipoRepasseRecorrente,
		LojaID:     lojaID,
		Valor:      valor,
		Status:     domain.StatusRepassePendente,
	}
	if err := s.repasseRepo.Criar(&repasse); err != nil {
		return fmt.Errorf("criando repasse do pedido %d: %w", pedidoID, err)
	}
	return nil
}

// valorBonusAtivacaoPorPlano define o bônus de ativação, pagamento único,
// por plano ativado pela loja indicada (Fase 7.5) — Start não entra aqui:
// sem split de pagamento, não gera comissão nenhuma, então não gera
// bônus.
var valorBonusAtivacaoPorPlano = map[string]float64{
	"basic": 60.0,
	"pro":   150.0,
	"scale": 400.0,
}

// prazoBonusAtivacao é a janela, a partir do cadastro da loja indicada,
// dentro da qual ela precisa atingir minimoPedidosBonusAtivacao pedidos
// pra gerar o bônus de ativação (Fase 7.5).
const prazoBonusAtivacao = 60 * 24 * time.Hour

// minimoPedidosBonusAtivacao é o mesmo número do limite mensal do plano
// Start (Fase 7.3) — não por coincidência de regra, só reaproveitando o
// mesmo patamar de "loja com movimento real" já usado em outro lugar.
const minimoPedidosBonusAtivacao = 30

// VerificarBonusAtivacao checa se a loja indicada por um afiliado já
// completou os critérios do bônus de ativação (loja ativou plano pago +
// atingiu o mínimo de pedidos dentro do prazo, a contar do cadastro) e
// ainda não recebeu o bônus — se sim, registra como pendente.
// Idempotente (checa domain.TipoRepasseBonusAtivacao existente antes de
// criar). Chamado depois de todo pedido pago via Mercado Pago de uma loja
// com afiliado (mesmo ponto de RegistrarPendente, ver
// MercadoPagoService.ProcessarNotificacaoPagamento) — só é alcançável
// depois que a loja sai do Start pra um plano pago (Start não gera pedido
// pago via Mercado Pago, não tem split), então não precisa checar o
// plano separado disso além de achar o valor certo na tabela.
func (s *RepasseAfiliadoService) VerificarBonusAtivacao(loja *domain.Loja) error {
	if loja.AfiliadoID == nil {
		return nil
	}
	valorBonus, temBonus := valorBonusAtivacaoPorPlano[loja.Plano]
	if !temBonus {
		return nil
	}
	if time.Since(loja.CreatedAt) > prazoBonusAtivacao {
		return nil
	}

	totalPedidos, err := s.pedidoRepo.ContarPedidosDesde(loja.ID, loja.CreatedAt)
	if err != nil {
		return fmt.Errorf("contando pedidos da loja %d pro bônus de ativação: %w", loja.ID, err)
	}
	if totalPedidos < minimoPedidosBonusAtivacao {
		return nil
	}

	jaTemBonus, err := s.repasseRepo.ExisteBonusAtivacao(loja.ID)
	if err != nil {
		return fmt.Errorf("verificando bônus de ativação existente da loja %d: %w", loja.ID, err)
	}
	if jaTemBonus {
		return nil
	}

	repasse := domain.RepasseAfiliado{
		AfiliadoID: *loja.AfiliadoID,
		PedidoID:   nil,
		Tipo:       domain.TipoRepasseBonusAtivacao,
		LojaID:     loja.ID,
		Valor:      valorBonus,
		Status:     domain.StatusRepassePendente,
	}
	if err := s.repasseRepo.Criar(&repasse); err != nil {
		return fmt.Errorf("criando bônus de ativação da loja %d: %w", loja.ID, err)
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

// ListarTodosComTotais é a visão geral do admin Drenux: TODOS os
// afiliados cadastrados (mesmo sem nenhum lançamento ainda), com o total
// pago e pendente de cada um — pra ver quem existe, não só quem tem
// saldo pra receber.
func (s *RepasseAfiliadoService) ListarTodosComTotais() ([]repository.AfiliadoComTotais, error) {
	return s.repasseRepo.ListarTodosComTotais()
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

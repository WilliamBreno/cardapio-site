package repository

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type RepasseAfiliadoRepository struct {
	db *gorm.DB
}

func NewRepasseAfiliadoRepository(db *gorm.DB) *RepasseAfiliadoRepository {
	return &RepasseAfiliadoRepository{db: db}
}

func (r *RepasseAfiliadoRepository) Criar(repasse *domain.RepasseAfiliado) error {
	return r.db.Create(repasse).Error
}

// ExistePorPedido evita duplicar o lançamento se a notificação do Mercado
// Pago chegar mais de uma vez pro mesmo pedido — checagem explícita, além
// da uniqueIndex no banco, pra devolver um no-op silencioso em vez de
// estourar erro de constraint.
func (r *RepasseAfiliadoRepository) ExistePorPedido(pedidoID uint) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.RepasseAfiliado{}).Where("pedido_id = ?", pedidoID).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// ExisteBonusAtivacao diz se essa loja já tem um lançamento de bônus de
// ativação registrado (pendente ou pago) — garante que o bônus (Fase
// 7.5) só é gerado uma vez por loja, mesmo que VerificarBonusAtivacao
// rode de novo em pedidos seguintes.
func (r *RepasseAfiliadoRepository) ExisteBonusAtivacao(lojaID uint) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.RepasseAfiliado{}).
		Where("loja_id = ? AND tipo = ?", lojaID, domain.TipoRepasseBonusAtivacao).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// ListarPorAfiliado devolve o extrato completo (pendente + pago) de um
// afiliado, mais recente primeiro — usado tanto no painel do próprio
// afiliado quanto no detalhe que o admin Drenux vê.
func (r *RepasseAfiliadoRepository) ListarPorAfiliado(afiliadoID uint) ([]domain.RepasseAfiliado, error) {
	var repasses []domain.RepasseAfiliado
	err := r.db.Preload("Loja").
		Where("afiliado_id = ?", afiliadoID).
		Order("created_at DESC").
		Find(&repasses).Error
	return repasses, err
}

func (r *RepasseAfiliadoRepository) SomarPendentePorAfiliado(afiliadoID uint) (float64, error) {
	var total float64
	err := r.db.Model(&domain.RepasseAfiliado{}).
		Where("afiliado_id = ? AND status = ?", afiliadoID, domain.StatusRepassePendente).
		Select("COALESCE(SUM(valor), 0)").
		Scan(&total).Error
	return total, err
}

// AfiliadoComTotais é a linha da visão geral do admin Drenux — TODO
// afiliado cadastrado (mesmo sem nenhum lançamento ainda, por isso é
// LEFT JOIN), com quanto já foi pago e quanto ainda está pendente. O
// detalhe (lançamento por lançamento) vem de ListarPorAfiliado a partir
// do ID.
type AfiliadoComTotais struct {
	AfiliadoID         uint    `json:"afiliado_id"`
	Nome               string  `json:"nome"`
	Email              string  `json:"email"`
	Codigo             string  `json:"codigo"`
	ComissaoPercentual float64 `json:"comissao_percentual"`
	TotalPendente      float64 `json:"total_pendente"`
	TotalPago          float64 `json:"total_pago"`
	Quantidade         int64   `json:"quantidade"`
}

func (r *RepasseAfiliadoRepository) ListarTodosComTotais() ([]AfiliadoComTotais, error) {
	var resultado []AfiliadoComTotais
	err := r.db.Table("afiliados").
		Joins("LEFT JOIN repasses_afiliado ON repasses_afiliado.afiliado_id = afiliados.id").
		Group("afiliados.id, afiliados.nome, afiliados.email, afiliados.codigo, afiliados.comissao_percentual").
		Order("afiliados.nome").
		Select(`afiliados.id AS afiliado_id,
			afiliados.nome AS nome,
			afiliados.email AS email,
			afiliados.codigo AS codigo,
			afiliados.comissao_percentual AS comissao_percentual,
			COALESCE(SUM(CASE WHEN repasses_afiliado.status = 'pendente' THEN repasses_afiliado.valor ELSE 0 END), 0) AS total_pendente,
			COALESCE(SUM(CASE WHEN repasses_afiliado.status = 'pago' THEN repasses_afiliado.valor ELSE 0 END), 0) AS total_pago,
			COUNT(repasses_afiliado.id) AS quantidade`).
		Scan(&resultado).Error
	return resultado, err
}

// MarcarComoPago marca um ou mais lançamentos como pagos de uma vez — só
// atualiza os que ainda estavam pendentes, pra nunca sobrescrever o
// PagoEm de um lançamento já quitado antes por engano (ex: clique
// duplicado na tela do admin).
func (r *RepasseAfiliadoRepository) MarcarComoPago(ids []uint) (int64, error) {
	agora := time.Now()
	resultado := r.db.Model(&domain.RepasseAfiliado{}).
		Where("id IN ? AND status = ?", ids, domain.StatusRepassePendente).
		Updates(map[string]interface{}{
			"status":  domain.StatusRepassePago,
			"pago_em": agora,
		})
	return resultado.RowsAffected, resultado.Error
}

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

// PendentePorAfiliado é a linha da visão geral do admin Drenux — um
// afiliado por linha, com o total pendente somado. O detalhe
// (lançamento por lançamento) vem de ListarPorAfiliado a partir do ID.
type PendentePorAfiliado struct {
	AfiliadoID uint    `json:"afiliado_id"`
	Nome       string  `json:"nome"`
	Email      string  `json:"email"`
	Total      float64 `json:"total_pendente"`
	Quantidade int64   `json:"quantidade"`
}

func (r *RepasseAfiliadoRepository) ListarPendentesAgrupado() ([]PendentePorAfiliado, error) {
	var resultado []PendentePorAfiliado
	err := r.db.Table("repasses_afiliado").
		Joins("JOIN afiliados ON afiliados.id = repasses_afiliado.afiliado_id").
		Where("repasses_afiliado.status = ?", domain.StatusRepassePendente).
		Group("afiliados.id, afiliados.nome, afiliados.email").
		Select("afiliados.id AS afiliado_id, afiliados.nome AS nome, afiliados.email AS email, COALESCE(SUM(repasses_afiliado.valor),0) AS total, COUNT(*) AS quantidade").
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

package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type AssinaturaPendenteRepository struct {
	db *gorm.DB
}

func NewAssinaturaPendenteRepository(db *gorm.DB) *AssinaturaPendenteRepository {
	return &AssinaturaPendenteRepository{db: db}
}

func (r *AssinaturaPendenteRepository) Criar(a *domain.AssinaturaPendente) error {
	return r.db.Create(a).Error
}

// BuscarPorToken é usado tanto pra mostrar a tela de "finalizar cadastro"
// quanto, na hora do submit, pra validar de novo antes de criar a loja —
// só devolve um registro já CONFIRMADO (webhook já processou o
// pagamento). Ver BuscarPorTokenIncluindoPendente pro polling que
// acontece antes disso.
func (r *AssinaturaPendenteRepository) BuscarPorToken(token string) (*domain.AssinaturaPendente, error) {
	var a domain.AssinaturaPendente
	if err := r.db.Where("token = ? AND usado = ? AND confirmada = ?", token, false, true).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// BuscarPorTokenIncluindoPendente devolve o registro independente de já
// ter sido confirmado pelo webhook ou não — usado pela tela de
// "finalizar cadastro" pra distinguir "token inválido" (não encontrou
// nada) de "ainda processando o pagamento" (encontrou, mas Confirmada
// ainda é false), decidindo se continua tentando de novo ou desiste.
func (r *AssinaturaPendenteRepository) BuscarPorTokenIncluindoPendente(token string) (*domain.AssinaturaPendente, error) {
	var a domain.AssinaturaPendente
	if err := r.db.Where("token = ? AND usado = ?", token, false).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ConfirmarPagamento é chamado pelo webhook de assinatura do Mercado
// Pago quando o preapproval é aprovado — preenche o email de verdade do
// pagador (a gente não sabia esse dado no momento do checkout, já que o
// cliente ainda nem tinha conta) e marca como confirmada, liberando o
// registro pra BuscarPorToken.
func (r *AssinaturaPendenteRepository) ConfirmarPagamento(id uint, email, preapprovalID string) error {
	return r.db.Model(&domain.AssinaturaPendente{}).Where("id = ?", id).Updates(map[string]interface{}{
		"email":                       email,
		"mercado_pago_preapproval_id": preapprovalID,
		"confirmada":                  true,
	}).Error
}

// BuscarPorMercadoPagoPreapprovalID é usado pelo webhook pra achar de
// volta o registro criado no momento do checkout, quando o único dado
// disponível é o ID do preapproval (ex: eventos depois da confirmação
// inicial).
func (r *AssinaturaPendenteRepository) BuscarPorMercadoPagoPreapprovalID(preapprovalID string) (*domain.AssinaturaPendente, error) {
	var a domain.AssinaturaPendente
	if err := r.db.Where("mercado_pago_preapproval_id = ?", preapprovalID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// BuscarPorSessionID é usado quando o cliente é redirecionado direto da
// Stripe (?session_id=...), antes do webhook necessariamente ter
// processado ainda — o frontend tenta de novo por alguns segundos.
func (r *AssinaturaPendenteRepository) BuscarPorSessionID(sessionID string) (*domain.AssinaturaPendente, error) {
	var a domain.AssinaturaPendente
	if err := r.db.Where("stripe_session_id = ? AND usado = ?", sessionID, false).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// MarcarUsado impede que a mesma assinatura seja usada pra criar duas
// lojas diferentes.
func (r *AssinaturaPendenteRepository) MarcarUsado(id uint) error {
	return r.db.Model(&domain.AssinaturaPendente{}).Where("id = ?", id).Update("usado", true).Error
}

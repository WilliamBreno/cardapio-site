package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type ComboRepository struct {
	db *gorm.DB
}

func NewComboRepository(db *gorm.DB) *ComboRepository {
	return &ComboRepository{db: db}
}

// ExisteComComponente diz se algum combo da loja usa esse produto como
// componente — usado pra recusar a exclusão do produto com uma mensagem
// amigável, em vez de deixar a constraint de FK do banco estourar um erro
// cru (ver ProdutoService.Deletar).
func (r *ComboRepository) ExisteComComponente(produtoID uint) (bool, error) {
	var total int64
	err := r.db.Model(&domain.ComboItem{}).Where("produto_id = ?", produtoID).Count(&total).Error
	return total > 0, err
}

// ListarPorLoja devolve todo combo da loja, disponível ou não — o dono
// precisa ver tudo pra poder reativar (mesmo padrão de ProdutoRepository).
func (r *ComboRepository) ListarPorLoja(lojaID uint) ([]domain.Combo, error) {
	var combos []domain.Combo
	err := r.db.Preload("Itens.Produto").
		Where("loja_id = ?", lojaID).
		Order("ordem, id").
		Find(&combos).Error
	return combos, err
}

// ListarDisponiveisPorLoja é a versão pro cardápio público — só combo
// disponível, e só com componentes que ainda existem e estão disponíveis
// (um combo com um item pausado não devia aparecer vendável).
func (r *ComboRepository) ListarDisponiveisPorLoja(lojaID uint) ([]domain.Combo, error) {
	var combos []domain.Combo
	err := r.db.Preload("Itens.Produto").
		Where("loja_id = ? AND disponivel = ?", lojaID, true).
		Order("ordem, id").
		Find(&combos).Error
	return combos, err
}

func (r *ComboRepository) BuscarPorID(id uint) (*domain.Combo, error) {
	var combo domain.Combo
	if err := r.db.Preload("Itens.Produto").First(&combo, id).Error; err != nil {
		return nil, err
	}
	return &combo, nil
}

func (r *ComboRepository) Criar(combo *domain.Combo) error {
	return r.db.Create(combo).Error
}

// Atualizar salva os campos do combo e substitui a lista de itens por
// completo (apaga os antigos, insere os novos) — mais simples e seguro
// que tentar fazer diff item a item numa lista pequena como essa.
func (r *ComboRepository) Atualizar(combo *domain.Combo) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("combo_id = ?", combo.ID).Delete(&domain.ComboItem{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Combo{}).Where("id = ?", combo.ID).Updates(map[string]interface{}{
			"nome":       combo.Nome,
			"descricao":  combo.Descricao,
			"foto_url":   combo.FotoURL,
			"preco":      combo.Preco,
			"disponivel": combo.Disponivel,
			"ordem":      combo.Ordem,
		}).Error; err != nil {
			return err
		}
		for i := range combo.Itens {
			combo.Itens[i].ID = 0
			combo.Itens[i].ComboID = combo.ID
		}
		if len(combo.Itens) > 0 {
			if err := tx.Create(&combo.Itens).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ComboRepository) Deletar(id uint) error {
	return r.db.Delete(&domain.Combo{}, id).Error
}

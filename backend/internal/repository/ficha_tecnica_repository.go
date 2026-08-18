package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type FichaTecnicaRepository struct {
	db *gorm.DB
}

func NewFichaTecnicaRepository(db *gorm.DB) *FichaTecnicaRepository {
	return &FichaTecnicaRepository{db: db}
}

// BuscarPorProduto devolve os itens da ficha técnica de um produto, com
// o insumo de cada item já carregado (precisa do custo/unidade pra
// calcular o CMV).
func (r *FichaTecnicaRepository) BuscarPorProduto(produtoID uint) ([]domain.FichaTecnicaItem, error) {
	var itens []domain.FichaTecnicaItem
	if err := r.db.Preload("Insumo").Where("produto_id = ?", produtoID).Order("id").Find(&itens).Error; err != nil {
		return nil, err
	}
	return itens, nil
}

// ExisteFichaTecnica diz se o produto tem ficha técnica cadastrada — é o
// que decide, na hora da venda, se o desconto de estoque vai pros
// insumos (ficha técnica) ou pro estoque simples do produto/variação
// (comportamento anterior, Fase 8), ver
// PosPagamentoService.descontarInsumosSeFichaTecnica.
func (r *FichaTecnicaRepository) ExisteFichaTecnica(produtoID uint) (bool, error) {
	var total int64
	err := r.db.Model(&domain.FichaTecnicaItem{}).Where("produto_id = ?", produtoID).Count(&total).Error
	return total > 0, err
}

// ContarUsosInsumo diz em quantas fichas técnicas (de quaisquer produtos)
// um insumo aparece — usado pra recusar a exclusão do insumo com uma
// mensagem amigável, mesmo padrão de ComboRepository.ExisteComComponente.
func (r *FichaTecnicaRepository) ContarUsosInsumo(insumoID uint) (int64, error) {
	var total int64
	err := r.db.Model(&domain.FichaTecnicaItem{}).Where("insumo_id = ?", insumoID).Count(&total).Error
	return total, err
}

// Salvar substitui a ficha técnica de um produto por completo (apaga os
// itens antigos, insere os novos) — mesmo padrão de
// ComboRepository.Atualizar: mais simples e seguro que tentar fazer diff
// item a item numa lista pequena como essa.
func (r *FichaTecnicaRepository) Salvar(produtoID uint, itens []domain.FichaTecnicaItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("produto_id = ?", produtoID).Delete(&domain.FichaTecnicaItem{}).Error; err != nil {
			return err
		}
		for i := range itens {
			itens[i].ID = 0
			itens[i].ProdutoID = produtoID
		}
		if len(itens) > 0 {
			if err := tx.Create(&itens).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

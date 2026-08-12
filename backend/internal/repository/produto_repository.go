package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type ProdutoRepository struct {
	db *gorm.DB
}

func NewProdutoRepository(db *gorm.DB) *ProdutoRepository {
	return &ProdutoRepository{db: db}
}

func (r *ProdutoRepository) Criar(produto *domain.Produto) error {
	return r.db.Create(produto).Error
}

// BuscarPorID já vem com a Categoria, Variacoes e Fotos carregadas.
func (r *ProdutoRepository) BuscarPorID(id uint) (*domain.Produto, error) {
	var produto domain.Produto
	if err := r.db.
		Preload("Categoria").
		Preload("Subcategoria").
		Preload("GrupoCor").
		Preload("Variacoes", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") }).
		Preload("Variacoes.Fotos", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") }).
		Preload("Fotos", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") }).
		First(&produto, id).Error; err != nil {
		return nil, err
	}
	return &produto, nil
}

func (r *ProdutoRepository) Atualizar(produto *domain.Produto) error {
	return r.db.Save(produto).Error
}

func (r *ProdutoRepository) Deletar(id uint) error {
	return r.db.Delete(&domain.Produto{}, id).Error
}

// ContarPorLoja conta todos os produtos da loja (inclusive pausados) —
// usado pelo limite de 30 produtos do plano Start (Fase 7.4).
func (r *ProdutoRepository) ContarPorLoja(lojaID uint) (int64, error) {
	var total int64
	if err := r.db.Model(&domain.Produto{}).Where("loja_id = ?", lojaID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// ContarPorCategoria é usado pra impedir excluir uma categoria que ainda
// tem produtos dentro dela.
func (r *ProdutoRepository) ContarPorCategoria(categoriaID uint) (int64, error) {
	var total int64
	if err := r.db.Model(&domain.Produto{}).Where("categoria_id = ?", categoriaID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// ContarPorSubcategoria/ContarPorGrupoCor impedem excluir uma
// subcategoria/grupo de cor que ainda tem produto associado.
func (r *ProdutoRepository) ContarPorSubcategoria(subcategoriaID uint) (int64, error) {
	var total int64
	if err := r.db.Model(&domain.Produto{}).Where("subcategoria_id = ?", subcategoriaID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ProdutoRepository) ContarPorGrupoCor(grupoCorID uint) (int64, error) {
	var total int64
	if err := r.db.Model(&domain.Produto{}).Where("grupo_cor_id = ?", grupoCorID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// SubtrairEstoque decrementa o estoque de um produto após pagamento
// confirmado. Usa UPDATE atômico (não lê + escreve separado) pra evitar
// race condition se dois pagamentos chegarem ao mesmo tempo.
// Devolve o estoque restante após a subtração, pra decidir se precisa
// disparar alerta ou marcar como esgotado.
func (r *ProdutoRepository) SubtrairEstoque(produtoID uint, quantidade int) (estoqueRestante int, err error) {
	result := r.db.Model(&domain.Produto{}).
		Where("id = ? AND estoque_atual IS NOT NULL", produtoID).
		UpdateColumn("estoque_atual", gorm.Expr("GREATEST(estoque_atual - ?, 0)", quantidade))
	if result.Error != nil {
		return 0, result.Error
	}

	// Busca o valor atual pra decidir o que fazer a seguir
	var produto domain.Produto
	if err := r.db.Select("estoque_atual, estoque_alerta").First(&produto, produtoID).Error; err != nil {
		return 0, err
	}
	if produto.EstoqueAtual == nil {
		return -1, nil // sem controle de estoque, ignora
	}

	// Se zerou, marca como indisponível
	if *produto.EstoqueAtual == 0 {
		r.db.Model(&domain.Produto{}).Where("id = ?", produtoID).Update("disponivel", false)
	}

	return *produto.EstoqueAtual, nil
}

// BuscarEstoqueAlerta retorna o produto se ele tiver estoque_alerta
// configurado e o estoque atual tiver atingido ou passado esse limite.
func (r *ProdutoRepository) BuscarEstoqueAlerta(produtoID uint) (*domain.Produto, bool) {
	var produto domain.Produto
	if err := r.db.First(&produto, produtoID).Error; err != nil {
		return nil, false
	}
	if produto.EstoqueAtual == nil || produto.EstoqueAlerta == nil {
		return nil, false
	}
	return &produto, *produto.EstoqueAtual <= *produto.EstoqueAlerta
}

// apenasDisponiveis controla se produtos marcados como indisponíveis
// entram na lista — true pro cardápio público (cliente não deve ver item
// fora de estoque), false pro painel admin (o dono precisa ver tudo,
// inclusive o que está pausado, pra poder reativar).
func (r *ProdutoRepository) ListarPorLoja(lojaID uint, apenasDisponiveis bool) ([]domain.Produto, error) {
	query := r.db.Where("loja_id = ?", lojaID).
		Preload("Categoria").
		Preload("Subcategoria").
		Preload("GrupoCor").
		Preload("Variacoes", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") }).
		Preload("Variacoes.Fotos", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") }).
		Preload("Fotos", func(db *gorm.DB) *gorm.DB { return db.Order("ordem, id") })
	if apenasDisponiveis {
		query = query.Where("disponivel = ?", true)
	}
	var produtos []domain.Produto
	if err := query.Order("id").Find(&produtos).Error; err != nil {
		return nil, err
	}
	return produtos, nil
}

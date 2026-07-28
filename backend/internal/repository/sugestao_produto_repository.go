package repository

import (
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type SugestaoProdutoRepository struct {
	db *gorm.DB
}

func NewSugestaoProdutoRepository(db *gorm.DB) *SugestaoProdutoRepository {
	return &SugestaoProdutoRepository{db: db}
}

// ListarPorLoja devolve todos os vínculos da loja, com o produto sugerido
// (nome, categoria) já carregado — a tela de configuração agrupa isso
// por produto origem no frontend.
func (r *SugestaoProdutoRepository) ListarPorLoja(lojaID uint) ([]domain.SugestaoProduto, error) {
	var sugestoes []domain.SugestaoProduto
	err := r.db.Preload("ProdutoSugerido").Preload("ProdutoSugerido.Categoria").
		Where("loja_id = ?", lojaID).
		Order("produto_origem_id, id").
		Find(&sugestoes).Error
	return sugestoes, err
}

// ListarPorProdutosOrigem busca todos os vínculos configurados pra uma
// lista de produtos (os que já estão no carrinho do cliente) — usado
// pra montar a seção de sugestões antes do checkout.
func (r *SugestaoProdutoRepository) ListarPorProdutosOrigem(lojaID uint, produtoIDs []uint) ([]domain.SugestaoProduto, error) {
	if len(produtoIDs) == 0 {
		return nil, nil
	}
	var sugestoes []domain.SugestaoProduto
	err := r.db.Preload("ProdutoSugerido").Preload("ProdutoSugerido.Categoria").
		Where("loja_id = ? AND produto_origem_id IN ?", lojaID, produtoIDs).
		Find(&sugestoes).Error
	return sugestoes, err
}

func (r *SugestaoProdutoRepository) BuscarPorID(id uint) (*domain.SugestaoProduto, error) {
	var sugestao domain.SugestaoProduto
	if err := r.db.First(&sugestao, id).Error; err != nil {
		return nil, err
	}
	return &sugestao, nil
}

func (r *SugestaoProdutoRepository) Criar(sugestao *domain.SugestaoProduto) error {
	return r.db.Create(sugestao).Error
}

func (r *SugestaoProdutoRepository) Deletar(id uint) error {
	return r.db.Delete(&domain.SugestaoProduto{}, id).Error
}

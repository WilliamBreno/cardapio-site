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
		Preload("ProdutoSugerido.Fotos", func(db *gorm.DB) *gorm.DB {
			return db.Order("ordem, id")
		}).
		Where("loja_id = ? AND produto_origem_id IN ?", lojaID, produtoIDs).
		Find(&sugestoes).Error
	return sugestoes, err
}

// ContarPorLoja devolve quantos vínculos a loja já tem no total — usado
// pra aplicar o limite de 1 vínculo grátis quando a Sugestão Inteligente
// não está contratada (ver SugestaoProdutoService.Criar).
func (r *SugestaoProdutoRepository) ContarPorLoja(lojaID uint) (int64, error) {
	var total int64
	err := r.db.Model(&domain.SugestaoProduto{}).Where("loja_id = ?", lojaID).Count(&total).Error
	return total, err
}

// PrimeiraID devolve o ID do vínculo mais antigo da loja (o "vínculo
// grátis" quando a loja não tem a Sugestão Inteligente contratada) — ver
// SugestaoProdutoService.MontarSugestoesCarrinho e Listar.
func (r *SugestaoProdutoRepository) PrimeiraID(lojaID uint) (uint, error) {
	var sugestao domain.SugestaoProduto
	if err := r.db.Where("loja_id = ?", lojaID).Order("id asc").First(&sugestao).Error; err != nil {
		return 0, err
	}
	return sugestao.ID, nil
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

// DeletarPorProduto remove toda sugestão que envolva esse produto, dos
// dois lados do vínculo (origem ou sugerido) — chamado antes de excluir
// um produto (ver ProdutoService.Deletar), pra não deixar a exclusão
// travar numa constraint de FK (lado ProdutoSugeridoID) nem deixar linha
// órfã pra trás (lado ProdutoOrigemID, que não tinha nenhuma trava antes
// dessa correção).
func (r *SugestaoProdutoRepository) DeletarPorProduto(produtoID uint) error {
	return r.db.Where("produto_origem_id = ? OR produto_sugerido_id = ?", produtoID, produtoID).
		Delete(&domain.SugestaoProduto{}).Error
}

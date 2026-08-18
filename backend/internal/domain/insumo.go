package domain

import "time"

// Insumo é um ingrediente/matéria-prima usado na ficha técnica de um
// produto (Fase 9.1, plano Scale) — diferente de Produto (item vendável no
// cardápio), um Insumo nunca é vendido diretamente: só é consumido quando
// um produto que o usa é vendido (ver FichaTecnicaItem).
type Insumo struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	LojaID uint `gorm:"not null;index" json:"loja_id"`

	Nome string `gorm:"size:100;not null" json:"nome"`

	// UnidadeCompra é a unidade em que o insumo é comprado do fornecedor
	// (ex: "kg", "fardo", "caixa", "litro"). UnidadeUso é a unidade em que
	// ele é consumido na ficha técnica de um produto (ex: "g", "ml",
	// "unidade") — quase sempre mais granular que a de compra.
	UnidadeCompra string `gorm:"size:20;not null" json:"unidade_compra"`
	UnidadeUso    string `gorm:"size:20;not null" json:"unidade_uso"`

	// FatorConversao é quantas UnidadeUso equivalem a 1 UnidadeCompra (ex:
	// 1kg = 1000g → FatorConversao = 1000). Quando compra e uso são a
	// mesma unidade (ex: "unidade"/"unidade"), FatorConversao = 1.
	FatorConversao float64 `gorm:"not null;default:1" json:"fator_conversao"`

	// CustoUnidadeCompra é o custo de 1 UnidadeCompra (ex: R$/kg) — o
	// custo por UnidadeUso é sempre derivado (ver CustoPorUnidadeUso),
	// nunca guardado separado, pra nunca ficar desatualizado quando esse
	// valor muda.
	CustoUnidadeCompra float64 `gorm:"not null;default:0" json:"custo_unidade_compra"`

	// Controle de estoque do insumo — opcional, mesmo espírito de
	// Produto.EstoqueAtual (nil = sem controle). Guardado em UnidadeUso (a
	// granularidade em que é consumido pela venda), não em UnidadeCompra.
	// float64 porque insumo se mede em massa/volume fracionário (ex:
	// "347g restantes"), diferente de Produto que é sempre unidade inteira.
	EstoqueAtual  *float64 `gorm:"default:null" json:"estoque_atual"`
	EstoqueAlerta *float64 `gorm:"default:null" json:"estoque_alerta"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Insumo) TableName() string {
	return "insumos"
}

// CustoPorUnidadeUso deriva o custo de 1 UnidadeUso a partir do custo de
// compra — sempre calculado na hora, nunca armazenado, pra o CMV de todo
// produto que usa esse insumo ficar automaticamente em dia assim que
// CustoUnidadeCompra é atualizado (Fase 9.1: "CMV automático").
func (i Insumo) CustoPorUnidadeUso() float64 {
	if i.FatorConversao <= 0 {
		return 0
	}
	return i.CustoUnidadeCompra / i.FatorConversao
}

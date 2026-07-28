package domain

import "time"

// ConfiguracaoPlataforma é uma tabela de linha única (ID sempre 1) com
// valores da própria Drenux que precisam poder ser ajustados sem
// redeploy — hoje só o preço da Sugestão Inteligente (Fase 6), lido
// daqui em qualquer lugar que precise exibir ou calcular a cobrança
// desse recurso. Ajustar o valor é uma UPDATE direta nessa linha pelo
// console do Neon, de propósito (não existe tela de admin pra isso
// ainda — ver docs/plano-melhorias-drenux.md, Fase 6).
type ConfiguracaoPlataforma struct {
	ID                             uint      `gorm:"primaryKey" json:"id"`
	SugestaoInteligentePrecoMensal float64   `gorm:"not null;default:19.90" json:"sugestao_inteligente_preco_mensal"`
	UpdatedAt                      time.Time `json:"updated_at"`
}

func (ConfiguracaoPlataforma) TableName() string {
	return "configuracoes_plataforma"
}

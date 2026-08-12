package service

import "testing"

func TestMontarItemEstoqueCritico(t *testing.T) {
	alerta5 := 5

	casos := []struct {
		nome          string
		estoqueAtual  int
		estoqueAlerta *int
		esperaCritico bool
	}{
		{"zerado é sempre crítico", 0, nil, true},
		{"abaixo do alerta é crítico", 3, &alerta5, true},
		{"no limite do alerta é crítico", 5, &alerta5, true},
		{"acima do alerta não é crítico", 6, &alerta5, false},
		{"sem alerta configurado, só zerado é crítico", 100, nil, false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			item := montarItemEstoque(1, "Produto", nil, "", c.estoqueAtual, c.estoqueAlerta)
			if item.Critico != c.esperaCritico {
				t.Fatalf("estoque=%d alerta=%v: esperava crítico=%v, veio %v", c.estoqueAtual, c.estoqueAlerta, c.esperaCritico, item.Critico)
			}
		})
	}
}

func TestCriticidadeOrdenaZeradoPrimeiro(t *testing.T) {
	alerta5 := 5
	zerado := montarItemEstoque(1, "Zerado", nil, "", 0, nil)
	baixo := montarItemEstoque(2, "Baixo", nil, "", 3, &alerta5)
	ok := montarItemEstoque(3, "OK", nil, "", 50, &alerta5)

	if !(criticidade(zerado) < criticidade(baixo) && criticidade(baixo) < criticidade(ok)) {
		t.Fatalf("ordem esperada zerado < baixo < ok, veio %d, %d, %d", criticidade(zerado), criticidade(baixo), criticidade(ok))
	}
}

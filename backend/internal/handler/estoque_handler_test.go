package handler

import "testing"

func TestRelatorioEstoqueDisponivel(t *testing.T) {
	casos := map[string]bool{
		"start": false,
		"basic": false,
		"pro":   true,
		"scale": true,
	}
	for plano, esperado := range casos {
		if got := relatorioEstoqueDisponivel(plano); got != esperado {
			t.Errorf("relatorioEstoqueDisponivel(%q) = %v, esperava %v", plano, got, esperado)
		}
	}
}

func TestControleEstoqueCompletoDisponivel(t *testing.T) {
	casos := map[string]bool{
		"start": false,
		"basic": false,
		"pro":   false,
		"scale": true,
	}
	for plano, esperado := range casos {
		if got := controleEstoqueCompletoDisponivel(plano); got != esperado {
			t.Errorf("controleEstoqueCompletoDisponivel(%q) = %v, esperava %v", plano, got, esperado)
		}
	}
}

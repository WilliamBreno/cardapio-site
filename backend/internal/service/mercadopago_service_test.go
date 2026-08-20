package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

// calcularComissaoEscalonada é a fórmula de comissão da Fase 7.2 — testada
// isolada porque é lógica financeira pura (sem banco/HTTP), mas que decide
// diretamente quanto a Drenux retém de cada pedido via marketplace_fee.
func TestCalcularComissaoEscalonada(t *testing.T) {
	casos := []struct {
		nome        string
		plano       string
		gmvAntes    float64
		valorPedido float64
		esperado    float64
	}{
		{"plano sem comissão (start) devolve zero", "start", 0, 1000, 0},
		{"plano desconhecido devolve zero", "inexistente", 0, 1000, 0},
		{"pedido de valor zero devolve zero", "basic", 0, 0, 0},
		{"pedido de valor negativo devolve zero", "basic", 0, -50, 0},
		{"basic inteiro na primeira faixa", "basic", 0, 3000, 72.00},                      // 3000 * 2,4%
		{"basic cruzando o teto de 5000", "basic", 4000, 2000, 39.00},                     // 1000@2,4% + 1000@1,5%
		{"basic todo além de 20000 (última faixa sem teto)", "basic", 25000, 1000, 13.00}, // 1000 * 1,3%
		{"pro exatamente no teto da primeira faixa", "pro", 0, 5000, 90.00},               // 5000 * 1,8%
		{"scale é flat, não tem faixa", "scale", 100000, 10000, 99.00},                    // 10000 * 0,99%
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := calcularComissaoEscalonada(c.plano, c.gmvAntes, c.valorPedido)
			esperadoArredondado := arredondarCentavos(c.esperado)
			if got != esperadoArredondado {
				t.Fatalf("plano=%s gmvAntes=%.2f valorPedido=%.2f: esperava %.2f, veio %.2f", c.plano, c.gmvAntes, c.valorPedido, esperadoArredondado, got)
			}
		})
	}
}

func arredondarCentavos(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// TestCalcularComissaoEscalonadaCasoPro verifica separadamente o cenário de
// "pro cruzando duas faixas de uma vez" (5000@1,8% + 15000@1,2% +
// 500@1,05%, dado gmvAntes=4500 e valorPedido=16000) — feito à parte da
// tabela acima porque a soma manual das 3 fatias é mais fácil de auditar
// aqui do que dentro de um literal de struct.
func TestCalcularComissaoEscalonadaCasoPro(t *testing.T) {
	// gmvAntes=4500 já ocupa 4500 da primeira faixa (teto 5000) — sobram só
	// 500 nela. Os 16000 do pedido se distribuem assim:
	//   500  na faixa 1 (até 5000)   a 1,8%
	//   15000 na faixa 2 (até 20000) a 1,2%  -> mas só cabem 15000 (20000-5000)
	//   0    sobra pra faixa 3
	// 500 + 15000 = 15500, restam 500 pra faixa 3 a 1,05%
	esperado := 500*1.8/100 + 15000*1.2/100 + 500*1.05/100
	got := calcularComissaoEscalonada("pro", 4500, 16000)
	if diff := got - arredondarCentavos(esperado); diff > 0.005 || diff < -0.005 {
		t.Fatalf("esperava aproximadamente %.2f, veio %.2f", esperado, got)
	}
}

// gerarAssinaturaTeste monta um header x-signature válido pro manifest
// dado, exatamente como o Mercado Pago faz do lado deles — usado pra criar
// o "caso feliz" nos testes de ValidarAssinaturaWebhook sem depender de
// credenciais reais.
func gerarAssinaturaTeste(secret, dataID, requestID, ts string) string {
	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, requestID, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	v1 := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("ts=%s,v1=%s", ts, v1)
}

func TestValidarAssinaturaWebhook(t *testing.T) {
	const secret = "segredo-de-teste"
	const dataID = "123456789"
	const requestID = "req-abc"
	const ts = "1700000000"

	assinaturaValida := gerarAssinaturaTeste(secret, dataID, requestID, ts)

	t.Run("assinatura válida passa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: secret}
		if err := s.ValidarAssinaturaWebhook(assinaturaValida, requestID, dataID); err != nil {
			t.Fatalf("esperava passar, veio erro: %v", err)
		}
	})

	t.Run("sem webhook secret configurado, recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: ""}
		if err := s.ValidarAssinaturaWebhook(assinaturaValida, requestID, dataID); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("sem header x-signature, recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: secret}
		if err := s.ValidarAssinaturaWebhook("", requestID, dataID); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("header em formato inesperado (sem ts/v1), recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: secret}
		if err := s.ValidarAssinaturaWebhook("formato=errado", requestID, dataID); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("assinatura de outro segredo, recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: "segredo-diferente"}
		if err := s.ValidarAssinaturaWebhook(assinaturaValida, requestID, dataID); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("dataID trocado (payload adulterado), recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: secret}
		if err := s.ValidarAssinaturaWebhook(assinaturaValida, requestID, "999999999"); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("requestID trocado, recusa", func(t *testing.T) {
		s := &MercadoPagoService{webhookSecret: secret}
		if err := s.ValidarAssinaturaWebhook(assinaturaValida, "outro-request-id", dataID); err == nil {
			t.Fatal("esperava erro, veio nil")
		}
	})

	t.Run("dataID em caixa diferente ainda valida (manifest usa lowercase)", func(t *testing.T) {
		// ValidarAssinaturaWebhook aplica strings.ToLower(dataID) antes de montar
		// o manifest — confirma que isso realmente normaliza, não só documenta.
		s := &MercadoPagoService{webhookSecret: secret}
		assinaturaMinuscula := gerarAssinaturaTeste(secret, "abc123", requestID, ts)
		if err := s.ValidarAssinaturaWebhook(assinaturaMinuscula, requestID, "ABC123"); err != nil {
			t.Fatalf("esperava passar (dataID normalizado pra minúsculo), veio erro: %v", err)
		}
	})
}

package notification

import (
	"strings"
	"testing"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
)

func TestMontarMensagemSaiuParaEntrega(t *testing.T) {
	pedido := &domain.Pedido{ID: 24, ClienteNome: "Cliente QA"}

	t.Run("com link (Pro/Scale)", func(t *testing.T) {
		msg := montarMensagemSaiuParaEntrega(pedido, "Loja QA", "https://drenux.com.br/loja-qa/pedido/24/rastrear")
		if !strings.Contains(msg, "Acompanhe em tempo real") || !strings.Contains(msg, "https://drenux.com.br/loja-qa/pedido/24/rastrear") {
			t.Fatalf("esperava o link de rastreamento na mensagem, veio: %q", msg)
		}
	})

	t.Run("sem link (Start/Basic, Fase 7.4)", func(t *testing.T) {
		msg := montarMensagemSaiuParaEntrega(pedido, "Loja QA", "")
		if strings.Contains(msg, "Acompanhe em tempo real") || strings.Contains(msg, "http") {
			t.Fatalf("não esperava link nenhum na mensagem sem rastreamento, veio: %q", msg)
		}
		if !strings.Contains(msg, "Cliente QA") || !strings.Contains(msg, "saiu para entrega") {
			t.Fatalf("mensagem sem link perdeu o aviso básico de status, veio: %q", msg)
		}
	})
}

package notification

import (
	"fmt"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
)

const dataHoraFormato = "02/01/2006 às 15:04"

// montarMensagemCliente monta o texto livre enviado ao cliente. Cita o
// nome da loja porque as mensagens vêm todas do mesmo número de
// WhatsApp da plataforma — sem isso, ficaria confuso de qual pedido é.
func montarMensagemCliente(pedido *domain.Pedido, lojaNome string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Oi, %s! Seu pedido #%d na %s foi confirmado.\n\n", pedido.ClienteNome, pedido.ID, lojaNome))
	sb.WriteString("Itens:\n")
	for _, item := range pedido.Itens {
		subtotal := item.PrecoUnit * float64(item.Quantidade)
		sb.WriteString(fmt.Sprintf("- %dx %s - R$ %.2f\n", item.Quantidade, item.ProdutoNome, subtotal))
	}
	sb.WriteString(fmt.Sprintf("\nTotal: R$ %.2f\n", pedido.Total))
	sb.WriteString(fmt.Sprintf("Retirada: %s\n", pedido.DataRetirada.Format(dataHoraFormato)))
	sb.WriteString("\nObrigado pela preferência!")
	return sb.String()
}

// montarMensagemAdmin monta o texto livre enviado pro dono da loja.
func montarMensagemAdmin(pedido *domain.Pedido, lojaNome string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Novo pedido pago - %s - #%d\n\n", lojaNome, pedido.ID))
	sb.WriteString(fmt.Sprintf("Cliente: %s (%s)\n", pedido.ClienteNome, pedido.ClienteTelefone))
	sb.WriteString("Itens:\n")
	for _, item := range pedido.Itens {
		sb.WriteString(fmt.Sprintf("- %dx %s\n", item.Quantidade, item.ProdutoNome))
	}
	sb.WriteString(fmt.Sprintf("\nTotal: R$ %.2f\n", pedido.Total))
	sb.WriteString(fmt.Sprintf("Retirada: %s", pedido.DataRetirada.Format(dataHoraFormato)))
	return sb.String()
}

// montarMensagemSaiuParaEntrega avisa o cliente que o pedido saiu pra
// entrega. Separada da confirmação de pagamento porque acontece num
// momento diferente do fluxo (só depois que o dono/entregador marca
// "saiu para entrega"). linkRastreamento vazio = loja sem rastreamento em
// tempo real no plano (Fase 7.4, Start/Basic) — manda só o aviso de
// status, sem a linha do link.
func montarMensagemSaiuParaEntrega(pedido *domain.Pedido, lojaNome, linkRastreamento string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Oba, %s! Seu pedido #%d na %s saiu para entrega. 🛵", pedido.ClienteNome, pedido.ID, lojaNome))
	if linkRastreamento != "" {
		sb.WriteString(fmt.Sprintf("\n\nAcompanhe em tempo real: %s", linkRastreamento))
	}
	return sb.String()
}

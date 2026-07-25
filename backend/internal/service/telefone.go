package service

import "strings"

// NormalizarTelefone garante que um número de WhatsApp salvo no sistema
// sempre vire só dígitos com o DDI 55 na frente — o mesmo formato que
// notification.WhatsmeowSender espera pra montar o JID de envio. Sem
// isso, um número digitado com espaço, hífen, parênteses ou sem o 55
// resolve pro contato errado (ou nenhum) na hora de mandar mensagem, sem
// gerar nenhum erro visível — o lojista só nunca recebe a notificação.
// Mesma normalização já usada nos telefones de cliente (frontend,
// handler/guardados_handler.go), só que essa faltava pro WhatsApp da
// própria loja (Configurações).
func NormalizarTelefone(telefone string) string {
	var sb strings.Builder
	for _, r := range telefone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	limpo := sb.String()
	if limpo == "" {
		return ""
	}
	if !strings.HasPrefix(limpo, "55") {
		limpo = "55" + limpo
	}
	return limpo
}

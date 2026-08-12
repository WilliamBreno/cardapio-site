package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
)

// EmailSender envia emails transacionais via API do Resend (resend.com).
// Free tier: 3.000 emails/mês, 100/dia — mais do que suficiente pra
// reset de senha nessa fase do produto.
type EmailSender struct {
	apiKey    string
	remetente string // ex: "Drenux <naoresponda@drenux.com.br>"
}

func NewEmailSender(apiKey, remetente string) *EmailSender {
	return &EmailSender{apiKey: apiKey, remetente: remetente}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func (e *EmailSender) enviar(destinatario, assunto, html string) error {
	payload := resendPayload{
		From:    e.remetente,
		To:      []string{destinatario},
		Subject: assunto,
		Html:    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("montando payload do email: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("criando requisição pro Resend: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("enviando email via Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Resend retornou status %d ao enviar email", resp.StatusCode)
	}

	return nil
}

// EnviarResetSenha manda o email com o link de redefinição de senha.
func (e *EmailSender) EnviarResetSenha(destinatario, nomeUsuario, linkReset string) error {
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
			<h2>Redefinir senha</h2>
			<p>Olá, %s.</p>
			<p>Recebemos uma solicitação para redefinir a senha da sua conta. Se foi você, clique no botão abaixo:</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="background: #b45f4d; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold;">
					Redefinir minha senha
				</a>
			</p>
			<p style="color: #888; font-size: 14px;">Esse link expira em 1 hora. Se você não pediu essa alteração, pode ignorar este email com segurança — sua senha continua a mesma.</p>
		</div>
	`, nomeUsuario, linkReset)

	return e.enviar(destinatario, "Redefinir sua senha", html)
}

// EnviarAssinaturaConfirmada avisa que o pagamento da mensalidade (plano
// Pro/Scale) foi confirmado, com o link pra finalizar o cadastro da loja.
// Sem prazo de expiração mencionado no texto de propósito — o link em si
// nunca expira.
func (e *EmailSender) EnviarAssinaturaConfirmada(destinatario, plano, linkFinalizar string) error {
	nomePlano := "Pro"
	if plano == "scale" {
		nomePlano = "Scale"
	}

	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
			<h2>Pagamento confirmado!</h2>
			<p>Sua assinatura do plano <strong>%s</strong> foi confirmada com sucesso.</p>
			<p>Falta só um passo: finalize o cadastro da sua loja pra começar a vender.</p>
			<p style="margin: 24px 0;">
				<a href="%s" style="background: #b45f4d; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold;">
					Finalizar cadastro da loja
				</a>
			</p>
			<p style="color: #888; font-size: 14px;">Pode clicar quando quiser — esse link não expira.</p>
		</div>
	`, nomePlano, linkFinalizar)

	return e.enviar(destinatario, "Pagamento confirmado — finalize sua loja Drenux", html)
}

// EnviarNovoPedido avisa o dono de uma loja Start que um pedido foi
// pago — substitui o aviso por WhatsApp (EnviarNotificacaoAdmin) só pra
// esse plano, que não tem interação de WhatsApp incluída (Fase 8.5, a
// pedido do William: "no máximo por email" no plano grátis). Basic/Pro/
// Scale continuam recebendo por WhatsApp, sem mudança.
func (e *EmailSender) EnviarNovoPedido(destinatario, lojaNome string, pedido *domain.Pedido) error {
	var itens strings.Builder
	for _, item := range pedido.Itens {
		itens.WriteString(fmt.Sprintf("<li>%dx %s</li>", item.Quantidade, item.ProdutoNome))
	}

	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
			<h2>Novo pedido pago — #%d</h2>
			<p>Cliente: <strong>%s</strong> (%s)</p>
			<ul>%s</ul>
			<p><strong>Total: R$ %.2f</strong></p>
			<p>Retirada: %s</p>
			<p style="color: #888; font-size: 14px; margin-top: 24px;">Acesse o painel da %s pra ver todos os detalhes do pedido.</p>
		</div>
	`, pedido.ID, pedido.ClienteNome, pedido.ClienteTelefone, itens.String(), pedido.Total, pedido.DataRetirada.Format("02/01/2006 15:04"), lojaNome)

	return e.enviar(destinatario, fmt.Sprintf("Novo pedido pago — %s — #%d", lojaNome, pedido.ID), html)
}

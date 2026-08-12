package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// MercadoPagoAssinaturaService implementa a Fase 6 Parte 3 do roadmap
// (ver docs/plano-melhorias-drenux.md): cobrança recorrente CONSOLIDADA —
// assinatura de plano da loja (Pro/Scale) e Sugestão Inteligente — direto
// na conta da PRÓPRIA Drenux via Mercado Pago Subscriptions
// (preapproval_plan/preapproval). Sem OAuth, sem split: esse dinheiro é
// receita direta da plataforma, diferente de MercadoPagoService (Fase 5),
// que lida com o checkout de PEDIDO na conta de cada loja.
//
// A validação de assinatura do webhook NÃO mora aqui — a aplicação do
// Mercado Pago só aceita uma URL de notificação por ambiente (achado
// configurando o painel real em 28/07/2026), então os eventos de
// "Planos e assinaturas" chegam na MESMA URL /webhooks/mercadopago do
// checkout de pedido, validados pelo MercadoPagoService.ValidarAssinaturaWebhook
// já existente (mesmo secret, MERCADOPAGO_WEBHOOK_SECRET) — ver
// MercadoPagoHandler.Webhook, que despacha pra cá depois de validar.
//
// IMPORTANTE: nunca testado contra a API real do Mercado Pago (sem
// credenciais de teste nesse ambiente) — em especial, o mecanismo de
// correlação via external_reference no link de checkout hospedado
// (?preapproval_plan_id=...&external_reference=...) segue o padrão
// documentado publicamente, mas precisa ser confirmado com uma assinatura
// de teste real antes de confiar em produção. Se o external_reference não
// chegar no preapproval por algum motivo, o webhook loga um aviso e não
// derruba nada — mas a confirmação automática não vai funcionar até isso
// ser corrigido.
type MercadoPagoAssinaturaService struct {
	accessToken string
	frontendURL string
	httpClient  *http.Client

	lojaRepo         *repository.LojaRepository
	assinaturaRepo   *repository.AssinaturaPendenteRepository
	configuracaoRepo *repository.ConfiguracaoPlataformaRepository
	emailSender      *notification.EmailSender
}

func NewMercadoPagoAssinaturaService(accessToken, frontendURL string, db *gorm.DB, emailSender *notification.EmailSender) *MercadoPagoAssinaturaService {
	return &MercadoPagoAssinaturaService{
		accessToken:      accessToken,
		frontendURL:      frontendURL,
		httpClient:       &http.Client{Timeout: 20 * time.Second},
		lojaRepo:         repository.NewLojaRepository(db),
		assinaturaRepo:   repository.NewAssinaturaPendenteRepository(db),
		configuracaoRepo: repository.NewConfiguracaoPlataformaRepository(db),
		emailSender:      emailSender,
	}
}

// obterOuCriarPreapprovalPlan acha (ou cria, na primeira vez) o
// preapproval_plan de "pro", "scale" ou "sugestao" — cacheado em
// ConfiguracaoPlataforma pra não duplicar plano a cada checkout (mesmo
// espírito do obterOuCriarPriceAssinatura da Stripe, adaptado pra API do
// Mercado Pago).
func (s *MercadoPagoAssinaturaService) obterOuCriarPreapprovalPlan(ctx context.Context, tipo string) (string, error) {
	cfg, err := s.configuracaoRepo.Buscar()
	if err != nil {
		return "", fmt.Errorf("buscando configuração da plataforma: %w", err)
	}

	var idExistente, nome, backURL string
	var valor float64
	switch tipo {
	case "pro":
		idExistente, valor, nome, backURL = cfg.MercadoPagoPreapprovalPlanProID, valoresMensalidadePlano["pro"], "Drenux Pro", s.frontendURL+"/cadastro/finalizar"
	case "scale":
		idExistente, valor, nome, backURL = cfg.MercadoPagoPreapprovalPlanScaleID, valoresMensalidadePlano["scale"], "Drenux Scale", s.frontendURL+"/cadastro/finalizar"
	case "sugestao":
		idExistente, valor, nome, backURL = cfg.MercadoPagoPreapprovalPlanSugestaoID, cfg.SugestaoInteligentePrecoMensal, "Drenux — Sugestão Inteligente", s.frontendURL+"/admin/configuracoes"
	default:
		return "", fmt.Errorf("tipo de assinatura desconhecido: %q", tipo)
	}

	if idExistente != "" {
		return idExistente, nil
	}

	corpo := map[string]interface{}{
		"reason": nome,
		"auto_recurring": map[string]interface{}{
			"frequency":          1,
			"frequency_type":     "months",
			"transaction_amount": valor,
			"currency_id":        "BRL",
		},
		"back_url": backURL,
	}

	resultado, err := s.chamarAPI(ctx, http.MethodPost, "https://api.mercadopago.com/preapproval_plan", corpo)
	if err != nil {
		return "", fmt.Errorf("criando preapproval_plan (%s): %w", tipo, err)
	}
	id, _ := resultado["id"].(string)
	if id == "" {
		return "", fmt.Errorf("Mercado Pago não devolveu o ID do preapproval_plan (%s)", tipo)
	}

	if err := s.configuracaoRepo.SalvarPreapprovalPlanID(tipo, id); err != nil {
		log.Printf("aviso: não foi possível cachear preapproval_plan_id (%s): %v", tipo, err)
	}
	return id, nil
}

func montarCheckoutURL(preapprovalPlanID, externalReference string) string {
	query := url.Values{}
	query.Set("preapproval_plan_id", preapprovalPlanID)
	query.Set("external_reference", externalReference)
	return "https://www.mercadopago.com.br/subscriptions/checkout?" + query.Encode()
}

// CriarCheckoutPlano atende o fluxo de PRÉ-CADASTRO (Planos.tsx, clique
// em "Escolher Pro/Scale" antes de ter conta) — cria um registro
// AssinaturaPendente com um token só nosso (a loja ainda não existe, o
// Mercado Pago não sabe de nada disso ainda) e devolve a URL do checkout
// hospedado, com esse token como external_reference. Quando o webhook
// confirmar o pagamento, o registro é atualizado e um email é disparado
// com o link de finalização (ver ProcessarWebhook) — é o mecanismo
// principal e mais confiável; o cliente também pode, em tese, ser
// redirecionado de volta já com o token na URL, mas isso não foi testado
// contra o checkout real do Mercado Pago (ver aviso no topo do arquivo),
// por isso o frontend não deve depender só disso.
func (s *MercadoPagoAssinaturaService) CriarCheckoutPlano(ctx context.Context, plano string) (string, error) {
	if plano != "pro" && plano != "scale" {
		return "", fmt.Errorf("plano inválido: %s", plano)
	}
	if s.accessToken == "" {
		return "", errors.New("assinatura via Mercado Pago não está configurada")
	}

	planID, err := s.obterOuCriarPreapprovalPlan(ctx, plano)
	if err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("gerando token da assinatura: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	assinatura := domain.AssinaturaPendente{Plano: plano, Token: token}
	if err := s.assinaturaRepo.Criar(&assinatura); err != nil {
		return "", fmt.Errorf("criando assinatura pendente: %w", err)
	}

	return montarCheckoutURL(planID, token), nil
}

// CriarCheckoutSugestaoInteligente atende o botão "Assinar" em
// Configurações (loja já logada, já existe) — usa "loja:{id}" como
// external_reference, então não precisa do vai-e-volta de
// AssinaturaPendente: o webhook já sabe direto de qual loja é.
func (s *MercadoPagoAssinaturaService) CriarCheckoutSugestaoInteligente(ctx context.Context, lojaID uint) (string, error) {
	if s.accessToken == "" {
		return "", errors.New("assinatura via Mercado Pago não está configurada")
	}

	planID, err := s.obterOuCriarPreapprovalPlan(ctx, "sugestao")
	if err != nil {
		return "", err
	}

	externalRef := fmt.Sprintf("loja:%d", lojaID)
	return montarCheckoutURL(planID, externalRef), nil
}

// CancelarSugestaoInteligente atende o botão "Cancelar assinatura" —
// cancela o preapproval no Mercado Pago (se houver um salvo) e desativa o
// recurso na loja imediatamente, sem esperar o webhook confirmar.
func (s *MercadoPagoAssinaturaService) CancelarSugestaoInteligente(ctx context.Context, lojaID uint) error {
	loja, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return errors.New("loja não encontrada")
	}

	if loja.SugestaoInteligenteMercadoPagoPreapprovalID != "" {
		if _, err := s.chamarAPI(ctx, http.MethodPut, "https://api.mercadopago.com/preapproval/"+loja.SugestaoInteligenteMercadoPagoPreapprovalID, map[string]interface{}{
			"status": "cancelled",
		}); err != nil {
			return fmt.Errorf("cancelando assinatura no Mercado Pago: %w", err)
		}
	}

	return s.lojaRepo.DesativarSugestaoInteligente(lojaID)
}

// VerificarToken devolve o registro de assinatura pendente independente
// de já estar confirmado ou não — quem chama decide o que fazer com
// AssinaturaPendente.Confirmada (ver PlanoHandler.VerificarToken).
func (s *MercadoPagoAssinaturaService) VerificarToken(token string) (*domain.AssinaturaPendente, error) {
	return s.assinaturaRepo.BuscarPorTokenIncluindoPendente(token)
}

type preapprovalResposta struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ExternalReference string `json:"external_reference"`
	PayerEmail        string `json:"payer_email"`
}

func (s *MercadoPagoAssinaturaService) buscarPreapproval(ctx context.Context, id string) (*preapprovalResposta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.mercadopago.com/preapproval/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("buscando preapproval %s: %w", id, err)
	}
	defer resp.Body.Close()

	corpo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mercado Pago recusou consulta do preapproval %s (status %d): %s", id, resp.StatusCode, string(corpo))
	}

	var p preapprovalResposta
	if err := json.Unmarshal(corpo, &p); err != nil {
		return nil, fmt.Errorf("lendo resposta do preapproval %s: %w", id, err)
	}
	return &p, nil
}

// ProcessarWebhook trata uma notificação type=subscription_preapproval —
// idempotente (reprocessar o mesmo evento não duplica efeito nenhum).
// Cobre três situações, nessa ordem de checagem:
//  1. Preapproval já vinculado à assinatura de PLANO de uma loja — evento
//     de renovação/cancelamento/pausa ao longo da vida da assinatura.
//  2. Preapproval já vinculado à assinatura de SUGESTÃO INTELIGENTE de
//     uma loja — mesma ideia, recurso separado.
//  3. Primeira confirmação (nenhuma loja ainda vinculada a esse
//     preapproval) — external_reference decide se é um token de
//     pré-cadastro (assinatura de plano) ou "loja:{id}" (Sugestão
//     Inteligente de loja já existente).
func (s *MercadoPagoAssinaturaService) ProcessarWebhook(ctx context.Context, preapprovalID string) error {
	preapproval, err := s.buscarPreapproval(ctx, preapprovalID)
	if err != nil {
		return err
	}

	log.Printf("assinatura Mercado Pago %s: status=%q external_reference=%q", preapprovalID, preapproval.Status, preapproval.ExternalReference)

	if loja, err := s.lojaRepo.BuscarPorMercadoPagoPreapprovalIDPlano(preapprovalID); err == nil {
		return s.aplicarStatusPlano(loja, preapproval.Status)
	}
	if loja, err := s.lojaRepo.BuscarPorSugestaoInteligenteMercadoPagoPreapprovalID(preapprovalID); err == nil {
		return s.aplicarStatusSugestao(loja, preapproval.Status)
	}

	if strings.HasPrefix(preapproval.ExternalReference, "loja:") {
		return s.confirmarPrimeiraSugestao(preapproval)
	}
	if preapproval.ExternalReference != "" {
		return s.confirmarPrimeiraAssinaturaPlano(preapproval)
	}

	log.Printf("aviso: assinatura Mercado Pago %s sem external_reference reconhecível — ignorando", preapprovalID)
	return nil
}

func (s *MercadoPagoAssinaturaService) aplicarStatusPlano(loja *domain.Loja, status string) error {
	if loja.PlanoVitalicio {
		return nil // conta de teste interna — nunca rebaixa, mesmo que algum preapproval acabe vinculado a ela
	}
	switch status {
	case "authorized":
		return nil // já ativa, isso é só a recorrência confirmando de novo
	case "cancelled", "paused":
		log.Printf("loja %d: assinatura de plano %s no Mercado Pago — voltando pro Start", loja.ID, status)
		return s.lojaRepo.CancelarAssinaturaPlano(loja.ID)
	default:
		log.Printf("loja %d: status de assinatura de plano %q ainda sem tratamento específico — ignorando", loja.ID, status)
		return nil
	}
}

func (s *MercadoPagoAssinaturaService) aplicarStatusSugestao(loja *domain.Loja, status string) error {
	switch status {
	case "authorized":
		return nil
	case "cancelled", "paused":
		log.Printf("loja %d: assinatura da Sugestão Inteligente %s no Mercado Pago — desativando", loja.ID, status)
		return s.lojaRepo.DesativarSugestaoInteligente(loja.ID)
	default:
		log.Printf("loja %d: status de assinatura da Sugestão Inteligente %q ainda sem tratamento específico — ignorando", loja.ID, status)
		return nil
	}
}

func (s *MercadoPagoAssinaturaService) confirmarPrimeiraSugestao(p *preapprovalResposta) error {
	if p.Status != "authorized" {
		log.Printf("assinatura Mercado Pago %s (Sugestão Inteligente): status %q ainda não autorizado — aguardando", p.ID, p.Status)
		return nil
	}
	lojaIDStr := strings.TrimPrefix(p.ExternalReference, "loja:")
	lojaID, err := strconv.ParseUint(lojaIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("external_reference de loja inválido (%q): %w", p.ExternalReference, err)
	}
	if err := s.lojaRepo.AtivarSugestaoInteligente(uint(lojaID), p.ID, time.Now()); err != nil {
		return fmt.Errorf("ativando Sugestão Inteligente da loja %d: %w", lojaID, err)
	}
	log.Printf("loja %d: Sugestão Inteligente contratada via Mercado Pago (preapproval=%s)", lojaID, p.ID)
	return nil
}

func (s *MercadoPagoAssinaturaService) confirmarPrimeiraAssinaturaPlano(p *preapprovalResposta) error {
	if p.Status != "authorized" {
		log.Printf("assinatura Mercado Pago %s (plano): status %q ainda não autorizado — aguardando", p.ID, p.Status)
		return nil
	}

	assinatura, err := s.assinaturaRepo.BuscarPorTokenIncluindoPendente(p.ExternalReference)
	if err != nil {
		return fmt.Errorf("token de assinatura pendente %q não encontrado: %w", p.ExternalReference, err)
	}
	if assinatura.Confirmada {
		return nil // idempotente — o Mercado Pago pode repetir a notificação
	}
	if p.PayerEmail == "" {
		return errors.New("assinatura aprovada sem payer_email")
	}

	if err := s.assinaturaRepo.ConfirmarPagamento(assinatura.ID, p.PayerEmail, p.ID); err != nil {
		return fmt.Errorf("confirmando assinatura pendente %d: %w", assinatura.ID, err)
	}
	log.Printf("assinatura pendente %d confirmada via Mercado Pago (preapproval=%s, plano=%s)", assinatura.ID, p.ID, assinatura.Plano)

	if s.emailSender == nil {
		log.Printf("aviso: email não configurado — assinatura pendente %d confirmada mas email não enviado. Token: %s", assinatura.ID, assinatura.Token)
		return nil
	}
	link := fmt.Sprintf("%s/cadastro/finalizar?token=%s", s.frontendURL, assinatura.Token)
	if err := s.emailSender.EnviarAssinaturaConfirmada(p.PayerEmail, assinatura.Plano, link); err != nil {
		log.Printf("falha ao enviar email de assinatura confirmada: %v", err)
	}
	return nil
}

func (s *MercadoPagoAssinaturaService) chamarAPI(ctx context.Context, metodo, url string, corpo map[string]interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(corpo)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, metodo, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando API do Mercado Pago: %w", err)
	}
	defer resp.Body.Close()

	corpoResposta, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var resultado map[string]interface{}
	if err := json.Unmarshal(corpoResposta, &resultado); err != nil {
		return nil, fmt.Errorf("lendo resposta do Mercado Pago: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mercado Pago recusou a requisição (status %d): %s", resp.StatusCode, string(corpoResposta))
	}

	return resultado, nil
}

package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config centraliza a leitura de variáveis de ambiente. Em vez de
// espalhar os.Getenv pelo código, todo mundo pede uma *Config.
type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	StripeSecretKey     string
	StripeWebhookSecret string
	FrontendURLs        []string
	CronSecret          string
	ResendAPIKey        string
	EmailRemetente      string

	// Credenciais da aplicação Mercado Pago (marketplace) — usadas pro
	// OAuth de conexão da loja (Fase 5 do roadmap, ver
	// docs/plano-melhorias-drenux.md). MercadoPagoWebhookSecret é a chave
	// secreta configurada no painel do Mercado Pago pra validar a
	// assinatura (header x-signature) das notificações de pagamento.
	MercadoPagoClientID      string
	MercadoPagoClientSecret  string
	MercadoPagoWebhookSecret string

	// APIPublicURL é o endereço público desta própria API (não o do
	// frontend) — precisa bater exatamente com o redirect_uri cadastrado
	// na aplicação do Mercado Pago, já que é pra cá que o OAuth redireciona
	// o navegador depois da autorização (GET /admin/mercadopago/callback).
	APIPublicURL string

	// DrenuxAdminSecret protege as rotas internas /drenux/* (repasse de
	// comissão de afiliado — Fase 5.5) — mesmo padrão do CronSecret,
	// sem sistema de login próprio: só um secret compartilhado no header
	// X-Drenux-Admin-Secret, já que só o William usa essas rotas.
	DrenuxAdminSecret string
}

func Load() *Config {
	// Em produção (Render) as variáveis já vêm definidas no ambiente, então
	// não tem problema o arquivo .env não existir — só avisamos no log,
	// não derrubamos a aplicação.
	if err := godotenv.Load(); err != nil {
		log.Println("aviso: .env não encontrado, lendo variáveis do ambiente do sistema")
	}
	frontendURLsRaw := getEnv("FRONTEND_URL", "http://localhost:5173")
	frontendURLs := strings.Split(frontendURLsRaw, ",")
	for i := range frontendURLs {
		frontendURLs[i] = strings.TrimSpace(frontendURLs[i])
	}
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		// Padrão já bate com a porta do Vite em desenvolvimento — quando
		// fizer o deploy do frontend (Vercel), define essa variável com
		// a URL real em produção.
		FrontendURLs:   frontendURLs,
		CronSecret:     getEnv("CRON_SECRET", ""),
		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		EmailRemetente: getEnv("EMAIL_REMETENTE", "Drenux <naoresponda@drenux.com.br>"),

		MercadoPagoClientID:      getEnv("MERCADOPAGO_CLIENT_ID", ""),
		MercadoPagoClientSecret:  getEnv("MERCADOPAGO_CLIENT_SECRET", ""),
		MercadoPagoWebhookSecret: getEnv("MERCADOPAGO_WEBHOOK_SECRET", ""),
		APIPublicURL:             getEnv("API_PUBLIC_URL", "http://localhost:8080"),
		DrenuxAdminSecret:        getEnv("DRENUX_ADMIN_SECRET", ""),
	}

	if cfg.DatabaseURL == "" {
		log.Println("aviso: DATABASE_URL não definida (vamos precisar dela já na próxima etapa)")
	}

	if cfg.JWTSecret == "" {
		log.Println("aviso: JWT_SECRET não definida — qualquer um conseguiria forjar token. Defina antes de qualquer deploy real.")
	}

	if cfg.StripeSecretKey == "" {
		log.Println("aviso: STRIPE_SECRET_KEY não definida — endpoints de Stripe vão falhar")
	}

	if cfg.StripeWebhookSecret == "" {
		log.Println("aviso: STRIPE_WEBHOOK_SECRET não definida — o webhook vai rejeitar todos os eventos")
	}

	if cfg.MercadoPagoClientID == "" || cfg.MercadoPagoClientSecret == "" {
		log.Println("aviso: MERCADOPAGO_CLIENT_ID/MERCADOPAGO_CLIENT_SECRET não definidas — onboarding de pagamento via Mercado Pago vai falhar")
	}

	if cfg.MercadoPagoWebhookSecret == "" {
		log.Println("aviso: MERCADOPAGO_WEBHOOK_SECRET não definida — o webhook do Mercado Pago vai rejeitar todas as notificações")
	}

	if cfg.DrenuxAdminSecret == "" {
		log.Println("aviso: DRENUX_ADMIN_SECRET não definida — as rotas /drenux/* vão ficar abertas sem proteção nenhuma")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

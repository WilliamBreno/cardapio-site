package config

import (
	"log"
	"os"
	"strconv"
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

	// MercadoPagoAccessToken é o access_token da PRÓPRIA conta Mercado
	// Pago da Drenux (não é OAuth de loja nenhuma) — usado só pra
	// assinaturas recorrentes (plano da loja + Sugestão Inteligente, Fase
	// 6 Parte 3), já que esse dinheiro é receita direta da plataforma, sem
	// split. Pego direto no painel do Mercado Pago ("Suas integrações" →
	// credenciais de produção/teste da aplicação), diferente do
	// CLIENT_ID/CLIENT_SECRET acima (esses autenticam OUTRAS contas via
	// OAuth).
	//
	// Achado configurando o painel real do Mercado Pago (28/07/2026): a
	// aplicação só aceita UMA URL de notificação por ambiente (teste ou
	// produção), com os tópicos marcados por checkbox apontando todos pra
	// ela — não dá pra ter uma URL/secret separada por tópico como o
	// desenho original desta fase previa. Por isso as notificações de
	// assinatura ("Planos e assinaturas") chegam na MESMA URL
	// /webhooks/mercadopago do checkout de pedido, validadas com o MESMO
	// MercadoPagoWebhookSecret acima — não existe mais um secret
	// separado pra assinatura.
	MercadoPagoAccessToken string

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

	// Rate limiting por IP (ver internal/middleware/rate_limit.go).
	// "Geral" se aplica a toda a API; "Auth" é mais apertado, só nas
	// rotas sensíveis a força bruta (login, cadastro, esqueci senha,
	// /drenux/*). Configurável por env var pra dar pra apertar/afrouxar
	// sem precisar de deploy de código novo.
	RateLimitGeralPorMinuto int
	RateLimitGeralBurst     int
	RateLimitAuthPorMinuto  int
	RateLimitAuthBurst      int
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

		MercadoPagoAccessToken: getEnv("MERCADOPAGO_ACCESS_TOKEN", ""),
		APIPublicURL:           getEnv("API_PUBLIC_URL", "http://localhost:8080"),
		DrenuxAdminSecret:      getEnv("DRENUX_ADMIN_SECRET", ""),

		RateLimitGeralPorMinuto: getEnvInt("RATE_LIMIT_GERAL_POR_MINUTO", 100),
		RateLimitGeralBurst:     getEnvInt("RATE_LIMIT_GERAL_BURST", 40),
		RateLimitAuthPorMinuto:  getEnvInt("RATE_LIMIT_AUTH_POR_MINUTO", 10),
		RateLimitAuthBurst:      getEnvInt("RATE_LIMIT_AUTH_BURST", 5),
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

	if cfg.MercadoPagoAccessToken == "" {
		log.Println("aviso: MERCADOPAGO_ACCESS_TOKEN não definida — checkout de assinatura (plano/Sugestão Inteligente) vai falhar")
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

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("aviso: %s=%q não é um número válido, usando o padrão (%d)", key, value, fallback)
		return fallback
	}
	return parsed
}

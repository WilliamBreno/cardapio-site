package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/config"
	"github.com/WilliamBreno/cardapio-backend/internal/database"
	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/handler"
	"github.com/WilliamBreno/cardapio-backend/internal/middleware"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/WilliamBreno/cardapio-backend/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	log.Println("conectado ao banco com sucesso")

	if err := db.AutoMigrate(
		&domain.Usuario{},
		&domain.Loja{},
		&domain.Categoria{},
		&domain.Subcategoria{},
		&domain.GrupoCor{},
		&domain.Produto{},
		&domain.FotoProduto{},
		&domain.VariacaoProduto{},
		&domain.FotoVariacao{},
		&domain.Cupom{},
		&domain.Pedido{},
		&domain.ItemPedido{},
		&domain.SolicitacaoEntrega{},
		&domain.Afiliado{},
		&domain.AssinaturaPendente{},
		&domain.RepasseAfiliado{},
	); err != nil {
		log.Fatalf("erro ao migrar o banco: %v", err)
	}
	log.Println("migrations aplicadas com sucesso")

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: cfg.FrontendURLs,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// X-Drenux-Admin-Secret é o header customizado usado pela área
		// interna /drenux/* (ver middleware.DrenuxAdminRequired) — sem
		// declarar aqui, o navegador bloqueia a requisição no preflight
		// de CORS antes dela sequer chegar no backend, e o front recebe
		// só um erro de rede genérico, sem status HTTP nenhum pra
		// diagnosticar.
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Drenux-Admin-Secret"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	emailSender := notification.NewEmailSender(cfg.ResendAPIKey, cfg.EmailRemetente)
	authService := service.NewAuthService(db, cfg.JWTSecret, emailSender, cfg.FrontendURLs[0])
	authHandler := handler.NewAuthHandler(authService)

	catalogoService := service.NewCatalogoService(db)
	catalogoHandler := handler.NewCatalogoHandler(catalogoService, db)

	categoriaService := service.NewCategoriaService(db)
	categoriaHandler := handler.NewCategoriaHandler(categoriaService)

	subcategoriaService := service.NewSubcategoriaService(db)
	subcategoriaHandler := handler.NewSubcategoriaHandler(subcategoriaService)

	grupoCorService := service.NewGrupoCorService(db)
	grupoCorHandler := handler.NewGrupoCorHandler(grupoCorService)

	produtoService := service.NewProdutoService(db)
	produtoHandler := handler.NewProdutoHandler(produtoService)

	variacaoService := service.NewVariacaoService(db)
	variacaoHandler := handler.NewVariacaoHandler(variacaoService)
	fotoVariacaoHandler := handler.NewFotoVariacaoHandler(db)

	distanciaService := service.NewDistanciaService()

	pedidoService := service.NewPedidoService(db, distanciaService)

	var whatsappSender notification.NotificationSender
	ws, err := notification.NewWhatsmeowSender(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Printf("aviso: WhatsApp não conectado (%v) — pedidos pagos não vão notificar até isso ser resolvido. Rode 'go run ./cmd/whatsapp-pair' apontando pro banco certo e reinicie.", err)
	} else {
		whatsappSender = ws
		defer ws.Close()
		log.Println("WhatsApp conectado com sucesso")
	}

	lojaService := service.NewLojaService(db)
	lojaRepoParaPedido := repository.NewLojaRepository(db)

	pedidoHandler := handler.NewPedidoHandler(
		pedidoService,
		repository.NewPedidoRepository(db),
		lojaRepoParaPedido,
		whatsappSender,
		cfg.FrontendURLs[0],
	)

	// posPagamentoService concentra estoque + notificação pós-pagamento —
	// compartilhado entre Stripe (assinatura, frete) e Mercado Pago
	// (pedido, ver Fase 5 do roadmap) pra não duplicar essa lógica.
	posPagamentoService := service.NewPosPagamentoService(db, whatsappSender)

	// StripeService agora também recebe emailSender e frontendURL — usados
	// pro fluxo de assinatura de plano (Pro/Scale).
	stripeService := service.NewStripeService(cfg.StripeSecretKey, cfg.StripeWebhookSecret, db, whatsappSender, emailSender, cfg.FrontendURLs[0], posPagamentoService)
	stripeHandler := handler.NewStripeHandler(stripeService, cfg.FrontendURLs[0])
	planoHandler := handler.NewPlanoHandler(stripeService)

	// repasseAfiliadoService controla o repasse manual de comissão de
	// afiliado pra pedidos pagos via Mercado Pago (Fase 5.5 — split 1:N
	// exigiria contato comercial sem prazo, então o repasse em si continua
	// manual/Pix, só o controle é automático).
	repasseAfiliadoService := service.NewRepasseAfiliadoService(db)

	// MercadoPagoService assume o checkout de PEDIDO (não o de assinatura
	// de plano, que continua na Stripe) — ver Fase 5 do roadmap em
	// docs/plano-melhorias-drenux.md.
	mercadoPagoService := service.NewMercadoPagoService(
		cfg.MercadoPagoClientID, cfg.MercadoPagoClientSecret, cfg.MercadoPagoWebhookSecret,
		cfg.JWTSecret, cfg.APIPublicURL, cfg.FrontendURLs[0], db, posPagamentoService, repasseAfiliadoService,
	)
	mercadoPagoHandler := handler.NewMercadoPagoHandler(mercadoPagoService, cfg.FrontendURLs[0], cfg.CronSecret)

	lojaHandler := handler.NewLojaHandler(lojaService, distanciaService)

	dashboardService := service.NewDashboardService(db)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	fotoHandler := handler.NewFotoHandler(db)

	cupomService := service.NewCupomService(db)
	cupomHandler := handler.NewCupomHandler(cupomService)

	relatorioService := service.NewRelatorioService(db, whatsappSender)
	relatorioHandler := handler.NewRelatorioHandler(relatorioService, cfg.CronSecret)

	freteHandler := handler.NewFreteHandler(lojaService, distanciaService)

	guardadosService := service.NewGuardadosService(db, distanciaService)
	guardadosHandler := handler.NewGuardadosHandler(guardadosService)
	solicitacaoHandler := handler.NewSolicitacaoHandler(repository.NewSolicitacaoEntregaRepository(db))

	afiliadoService := service.NewAfiliadoService(db, cfg.JWTSecret, cfg.StripeSecretKey, emailSender, cfg.FrontendURLs[0])
	afiliadoHandler := handler.NewAfiliadoHandler(afiliadoService, repasseAfiliadoService, cfg.FrontendURLs[0])
	drenuxAdminHandler := handler.NewDrenuxAdminHandler(repasseAfiliadoService, afiliadoService)

	router.POST("/auth/cadastro", authHandler.Cadastrar)
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/esqueci-senha", authHandler.EsqueciSenha)
	router.POST("/auth/redefinir-senha", authHandler.RedefinirSenha)

	router.POST("/afiliados/login", afiliadoHandler.Login)

	// Assinatura de plano (Pro/Scale) — rotas públicas
	router.POST("/planos/checkout", planoHandler.CriarCheckout)
	router.GET("/planos/verificar-token", planoHandler.VerificarToken)
	router.GET("/planos/verificar-sessao", planoHandler.VerificarSessao)

	router.GET("/lojas/:slug", catalogoHandler.BuscarCardapio)
	router.GET("/lojas/:slug/historico", catalogoHandler.BuscarHistorico)
	router.POST("/lojas/:slug/pedidos", pedidoHandler.Criar)
	router.GET("/lojas/:slug/pedidos/:id/rastrear", pedidoHandler.Rastrear)
	router.POST("/lojas/:slug/cupons/validar", func(c *gin.Context) {
		loja, err := lojaService.BuscarPorSlug(c.Param("slug"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"erro": "loja não encontrada"})
			return
		}
		c.Set("loja_id_publico", loja.ID)
		cupomHandler.Validar(c)
	})
	router.POST("/lojas/:slug/cotar-frete", freteHandler.Cotar)

	// Checkout de PEDIDO migrou pra Mercado Pago (Fase 5.2 do roadmap) —
	// o handler/service da Stripe continuam no repositório (usados pra
	// assinatura de plano e frete), só pararam de ser chamados aqui.
	router.POST("/pedidos/:id/checkout", mercadoPagoHandler.Checkout)

	router.GET("/lojas/:slug/guardados", guardadosHandler.Listar)
	router.POST("/lojas/:slug/guardados/cotar-frete", guardadosHandler.CotarFrete)
	router.POST("/lojas/:slug/guardados/solicitar-entrega", guardadosHandler.SolicitarEntrega)
	router.GET("/lojas/:slug/solicitacoes/:id/rastrear", solicitacaoHandler.Rastrear)
	router.POST("/solicitacoes/:id/checkout", stripeHandler.CheckoutFrete)

	router.POST("/webhooks/stripe", stripeHandler.Webhook)
	router.POST("/webhooks/mercadopago", mercadoPagoHandler.Webhook)

	// Callback do OAuth do Mercado Pago — rota pública de propósito (é o
	// próprio Mercado Pago que redireciona o navegador do dono pra cá,
	// sem token de sessão nosso; a identidade da loja vem do "state"
	// assinado, ver MercadoPagoService.IniciarOnboarding).
	router.GET("/admin/mercadopago/callback", mercadoPagoHandler.Callback)

	router.POST("/relatorio/semanal", relatorioHandler.EnviarSemanal)
	router.POST("/mercadopago/renovar-tokens", mercadoPagoHandler.RenovarTokens)

	admin := router.Group("/admin")
	admin.Use(middleware.AuthRequired(cfg.JWTSecret))
	admin.GET("/me", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"usuario_id": c.GetUint("usuario_id"),
			"loja_id":    c.GetUint("loja_id"),
		})
	})

	admin.GET("/categorias", categoriaHandler.Listar)
	admin.POST("/categorias", categoriaHandler.Criar)
	admin.PUT("/categorias/:id", categoriaHandler.Atualizar)
	admin.DELETE("/categorias/:id", categoriaHandler.Deletar)

	// Hierarquia Categoria → Subcategoria → Grupo de Cor — exclusiva do
	// segmento "mercadoria" (ver docs/plano-melhorias-drenux.md, Fase 3).
	admin.GET("/subcategorias", subcategoriaHandler.Listar)
	admin.POST("/categorias/:categoriaId/subcategorias", subcategoriaHandler.Criar)
	admin.PUT("/subcategorias/:id", subcategoriaHandler.Atualizar)
	admin.DELETE("/subcategorias/:id", subcategoriaHandler.Deletar)

	admin.GET("/grupos-cor", grupoCorHandler.Listar)
	admin.POST("/subcategorias/:subcategoriaId/grupos-cor", grupoCorHandler.Criar)
	admin.PUT("/grupos-cor/:id", grupoCorHandler.Atualizar)
	admin.DELETE("/grupos-cor/:id", grupoCorHandler.Deletar)

	admin.GET("/produtos", produtoHandler.Listar)
	admin.POST("/produtos", produtoHandler.Criar)
	admin.PUT("/produtos/:id", produtoHandler.Atualizar)
	admin.DELETE("/produtos/:id", produtoHandler.Deletar)

	admin.GET("/dashboard", dashboardHandler.Dados)

	admin.GET("/cupons", cupomHandler.Listar)
	admin.POST("/cupons", cupomHandler.Criar)
	admin.PUT("/cupons/:id", cupomHandler.Atualizar)
	admin.DELETE("/cupons/:id", cupomHandler.Deletar)

	admin.POST("/fotos/:produtoId", fotoHandler.Adicionar)
	admin.PUT("/fotos/:produtoId/reordenar", fotoHandler.Reordenar)
	admin.DELETE("/fotos/:produtoId/:fotoId", fotoHandler.Deletar)

	variacoes := admin.Group("/variacoes")
	variacoes.GET("/:produtoId", variacaoHandler.Listar)
	variacoes.POST("/:produtoId", variacaoHandler.Criar)
	variacoes.PUT("/:produtoId/:variacaoId", variacaoHandler.Atualizar)
	variacoes.DELETE("/:produtoId/:variacaoId", variacaoHandler.Deletar)
	variacoes.POST("/:produtoId/:variacaoId/fotos", fotoVariacaoHandler.Adicionar)
	variacoes.DELETE("/:produtoId/:variacaoId/fotos/:fotoId", fotoVariacaoHandler.Deletar)

	admin.GET("/pedidos", pedidoHandler.Listar)
	admin.PUT("/pedidos/:id/status-entrega", pedidoHandler.AtualizarStatusEntrega)
	admin.POST("/pedidos/:id/localizacao", pedidoHandler.AtualizarLocalizacao)

	admin.GET("/solicitacoes", solicitacaoHandler.Listar)
	admin.PUT("/solicitacoes/:id/status-entrega", solicitacaoHandler.AtualizarStatusEntrega)
	admin.POST("/solicitacoes/:id/localizacao", solicitacaoHandler.AtualizarLocalizacao)

	admin.POST("/stripe/onboarding", stripeHandler.IniciarOnboarding)
	admin.GET("/stripe/status", stripeHandler.Status)

	admin.GET("/mercadopago/onboarding", mercadoPagoHandler.IniciarOnboarding)
	admin.GET("/mercadopago/status", mercadoPagoHandler.Status)

	admin.GET("/loja", lojaHandler.Buscar)
	admin.PUT("/loja", lojaHandler.AtualizarConfiguracoes)

	afiliado := router.Group("/afiliado")
	afiliado.Use(middleware.AfiliadoRequired(cfg.JWTSecret))
	afiliado.GET("/dashboard", afiliadoHandler.Dashboard)
	afiliado.POST("/stripe/onboarding", afiliadoHandler.IniciarOnboarding)
	afiliado.GET("/repasses", afiliadoHandler.Extrato)

	// /drenux/* — controle interno de repasse de comissão de afiliado
	// (Fase 5.5), sem sistema de login de staff próprio: protegido só por
	// um secret compartilhado (ver middleware.DrenuxAdminRequired).
	drenux := router.Group("/drenux")
	drenux.Use(middleware.DrenuxAdminRequired(cfg.DrenuxAdminSecret))
	drenux.POST("/afiliados", drenuxAdminHandler.CriarAfiliado)
	drenux.GET("/afiliados/pendentes", drenuxAdminHandler.PendentesPorAfiliado)
	drenux.GET("/afiliados/:id/repasses", drenuxAdminHandler.DetalheAfiliado)
	drenux.POST("/repasses/marcar-pago", drenuxAdminHandler.MarcarComoPago)

	router.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(500, gin.H{"status": "erro", "banco": "indisponível"})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "banco": "conectado"})
	})

	log.Printf("servidor rodando na porta %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}

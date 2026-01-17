package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/auth"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/email"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/notification"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/ocr"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/payment"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/upload"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/handlers"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/config"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/database"

	_ "github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/cmd/saas-api/docs"
)

// @title WhatsApp Bot SaaS API
// @version 2.0
// @description API documentation for WhatsApp Bot SaaS (Modular Architecture)
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@whatsapp-saas.com
// @license.name MIT
// @host localhost:8080
// @BasePath /
func main() {
	// Load config
	cfg := config.LoadConfig()
	log.Printf("🚀 Starting saas-api on port %s", cfg.Port)

	// Init database
	db := database.NewDB(cfg.DatabaseURL)
	defer db.Close()

	// Init repositories (use GORM instance)
	clientRepo := repositories.NewClientRepo(db.GORM)
	conversationRepo := repositories.NewConversationRepo(db.GORM)
	kbRepo := repositories.NewKBRepo(db.GORM)
	transactionRepo := repositories.NewTransactionRepo(db.GORM)
	workflowRepo := repositories.NewWorkflowRepo(db.GORM)
	orderRepo := repositories.NewOrderRepo(db.GORM)
	cartRepo := repositories.NewCartRepo(db.GORM)
	productRepo := repositories.NewProductRepo(db.GORM)
	kbRetriever := kb.NewRetriever(db.GORM)

	// Init tenant resolver (for multi-tenant/multi-module routing)
	tenantResolver := tenant.NewResolver(db.DB)

	// Init LLM service (multi-provider support)
	llmService := llm.NewService()

	// Init WhatsApp service
	waService := whatsapp.NewService(cfg.WhatsAppStoreURL)

	// Init OCR service (multi-provider support)
	var ocrProvider ocr.Provider
	switch cfg.OCRProvider {
	case "ocrspace":
		ocrProvider = ocr.NewOCRSpaceProvider(cfg.OCRSpaceAPIKey)
	case "tesseract":
		ocrProvider = ocr.NewTesseractProvider(cfg.TesseractLanguage)
	default:
		// Default to Google Cloud Vision
		ocrProvider = ocr.NewGoogleVisionProvider(cfg.GoogleVisionAPIKey)
	}
	ocrService := ocr.NewService(ocrProvider)

	// Init email service (multi-provider support)
	var emailProvider email.Provider
	switch cfg.EmailProvider {
	case "resend":
		emailProvider = email.NewResendProvider(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
	case "brevo":
		emailProvider = email.NewBrevoProvider(cfg.BrevoAPIKey, cfg.EmailFrom, cfg.EmailFromName)
	default:
		// Default to Brevo
		if cfg.BrevoAPIKey != "" {
			emailProvider = email.NewBrevoProvider(cfg.BrevoAPIKey, cfg.EmailFrom, cfg.EmailFromName)
		} else if cfg.ResendAPIKey != "" {
			emailProvider = email.NewResendProvider(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
		}
	}
	var emailService *email.Service
	if emailProvider != nil {
		emailService = email.NewService(emailProvider)
	}

	// Init notification service (multi-channel)
	var notificationService *notification.Service
	if emailService != nil && cfg.AdminPhone != "" && cfg.AdminEmail != "" {
		notificationService = notification.NewService(waService, emailService, cfg.AdminPhone, cfg.AdminEmail)
	}

	// Log provider info
	log.Printf("📱 Using WhatsApp provider: %s", waService.GetProviderName())
	log.Printf("🤖 Using LLM provider: %s", llmService.GetProviderName())
	log.Printf("🔍 Using OCR provider: %s", ocrService.GetProviderName())
	if emailService != nil {
		log.Printf("📧 Using Email provider: %s", emailService.GetProviderName())
	} else {
		log.Printf("⚠️  Email service not configured")
	}
	if notificationService != nil {
		log.Printf("🔔 Notification service enabled (Admin: %s, %s)", cfg.AdminPhone, cfg.AdminEmail)
	}

	// Init payment gateway based on config
	paymentGateway, err := payment.NewGateway(cfg, db.GORM)
	if err != nil {
		log.Fatalf("Failed to initialize payment gateway: %v", err)
	}
	log.Printf("💳 Payment mode: %s", cfg.PaymentMode)

	// Init services
	workflowService := services.NewWorkflowService(workflowRepo, db.GORM, waService, llmService)
	if err := workflowService.Initialize(); err != nil {
		log.Fatalf("Failed to initialize workflow service: %v", err)
	}
	defer workflowService.Shutdown()

	// Init order service with payment gateway and notification
	orderService := services.NewOrderService(orderRepo, clientRepo, paymentGateway, waService, notificationService)

	// Init cart service
	cartService := services.NewCartService(cartRepo, orderRepo)

	// Init product service
	productService := services.NewProductService(productRepo)

	// Init webhook service with cart and order services
	webhookService := services.NewWebhookService(clientRepo, conversationRepo, transactionRepo, kbRetriever, llmService, waService, ocrService, tenantResolver, cartService, orderService, cfg)

	// Init auth service
	authService := auth.NewService(db.GORM, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, cfg.GoogleClientID)
	log.Printf("🔐 Authentication service initialized")

	// Init upload service (multi-provider support)
	var uploadProvider upload.Provider
	switch cfg.UploadProvider {
	case "cloudinary":
		if cfg.CloudinaryCloudName != "" && cfg.CloudinaryAPIKey != "" && cfg.CloudinaryAPISecret != "" {
			provider, err := upload.NewCloudinaryProvider(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
			if err != nil {
				log.Fatalf("Failed to initialize Cloudinary: %v", err)
			}
			uploadProvider = provider
			log.Printf("📤 Using Upload provider: Cloudinary")
		} else {
			log.Println("⚠️  Cloudinary credentials not configured, falling back to local storage")
			provider, err := upload.NewLocalProvider(cfg.UploadBasePath, cfg.UploadBaseURL)
			if err != nil {
				log.Fatalf("Failed to initialize local storage: %v", err)
			}
			uploadProvider = provider
			log.Printf("📤 Using Upload provider: Local Storage")
		}
	case "s3":
		if cfg.S3AccessKeyID != "" && cfg.S3SecretAccessKey != "" && cfg.S3Region != "" && cfg.S3BucketName != "" {
			provider, err := upload.NewS3Provider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, cfg.S3Region, cfg.S3BucketName)
			if err != nil {
				log.Fatalf("Failed to initialize S3: %v", err)
			}
			uploadProvider = provider
			log.Printf("📤 Using Upload provider: AWS S3")
		} else {
			log.Println("⚠️  S3 credentials not configured, falling back to local storage")
			provider, err := upload.NewLocalProvider(cfg.UploadBasePath, cfg.UploadBaseURL)
			if err != nil {
				log.Fatalf("Failed to initialize local storage: %v", err)
			}
			uploadProvider = provider
			log.Printf("📤 Using Upload provider: Local Storage")
		}
	default:
		// Default to local storage
		provider, err := upload.NewLocalProvider(cfg.UploadBasePath, cfg.UploadBaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
		uploadProvider = provider
		log.Printf("📤 Using Upload provider: Local Storage")
	}
	uploadService := upload.NewService(uploadProvider)

	// Init handlers
	clientHandler := handlers.NewClientHandler(clientRepo)
	kbHandler := handlers.NewKBHandler(kbRetriever, kbRepo)
	healthHandler := handlers.NewHealthHandler(waService)
	whatsappHandler := handlers.NewWhatsAppHandler(waService, clientRepo)
	webhookHandler := handlers.NewWebhookHandler(webhookService)
	ocrHandler := handlers.NewOCRHandler(ocrService, llmService, transactionRepo, workflowService)
	workflowHandler := handlers.NewWorkflowHandler(workflowService)
	paymentHandler := handlers.NewPaymentHandler(orderService)
	cartHandler := handlers.NewCartHandler(cartService)
	productHandler := handlers.NewProductHandler(productService)
	uploadHandler := upload.NewHandler(uploadService)

	// Init Fiber app
	app := fiber.New(fiber.Config{
		AppName: "WhatsApp Bot SaaS API",
	})

	// Middleware
	app.Use(cors.New())

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check
	app.Get("/health", healthHandler.GetHealth)

	// Authentication routes (public - no auth required)
	authGroup := app.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/google", authHandler.LoginWithGoogle)
	authGroup.Post("/refresh", authHandler.RefreshToken)

	// Protected auth routes (require authentication)
	authGroup.Post("/logout", auth.AuthMiddleware(authService), authHandler.Logout)
	authGroup.Get("/me", auth.AuthMiddleware(authService), authHandler.Me)

	// Product routes (protected - require authentication)
	productsGroup := app.Group("/products", auth.AuthMiddleware(authService))
	productsGroup.Post("/", productHandler.CreateProduct)
	productsGroup.Get("/", productHandler.ListProducts)
	productsGroup.Get("/:id", productHandler.GetProduct)
	productsGroup.Put("/:id", productHandler.UpdateProduct)
	productsGroup.Delete("/:id", productHandler.DeleteProduct)
	productsGroup.Patch("/:id/stock", productHandler.UpdateStock)
	productsGroup.Patch("/:id/toggle", productHandler.ToggleProductStatus)

	// Upload routes (protected - require authentication)
	uploadGroup := app.Group("/upload", auth.AuthMiddleware(authService))
	uploadGroup.Post("/", uploadHandler.UploadFile)
	uploadGroup.Post("/product", uploadHandler.UploadProductImage)
	uploadGroup.Delete("/", uploadHandler.DeleteFile)
	uploadGroup.Get("/info", uploadHandler.GetProviderInfo)

	// Static file serving for local uploads
	app.Static("/uploads", cfg.UploadBasePath)

	// Client routes
	app.Get("/clients", clientHandler.GetActiveClients)
	app.Get("/clients/:id", clientHandler.GetClientByID)

	// Knowledge Base routes
	app.Get("/knowledge-base", kbHandler.GetKnowledgeBase)
	app.Post("/knowledge-base", kbHandler.AddKnowledgeItem)

	// WhatsApp routes
	app.Get("/whatsapp/qr", whatsappHandler.GetQRCode)
	app.Post("/whatsapp/session/start", whatsappHandler.StartSession)
	app.Post("/whatsapp/session/stop", whatsappHandler.StopSession)
	app.Post("/whatsapp/session/restart", whatsappHandler.RestartSession)
	app.Get("/whatsapp/session/status", whatsappHandler.GetSessionStatus)
	app.Post("/whatsapp/webhook/configure", whatsappHandler.ConfigureWebhook)

	// Webhook route
	app.Post("/webhook", webhookHandler.ReceiveWebhook)

	// OCR routes
	app.Post("/ocr/process-receipt", ocrHandler.ProcessReceipt)
	app.Get("/transactions", ocrHandler.GetTransactions)

	// Workflow routes
	app.Post("/workflows", workflowHandler.CreateWorkflow)
	app.Get("/workflows", workflowHandler.ListWorkflows)
	app.Get("/workflows/:id", workflowHandler.GetWorkflow)
	app.Put("/workflows/:id", workflowHandler.UpdateWorkflow)
	app.Delete("/workflows/:id", workflowHandler.DeleteWorkflow)
	app.Post("/workflows/:id/execute", workflowHandler.ExecuteWorkflow)
	app.Get("/workflows/:id/executions", workflowHandler.GetWorkflowExecutions)

	// Shopping Cart routes
	app.Post("/cart/add", cartHandler.AddToCart)
	app.Put("/cart/update", cartHandler.UpdateCartItem)
	app.Delete("/cart/remove", cartHandler.RemoveFromCart)
	app.Get("/cart", cartHandler.ViewCart)
	app.Delete("/cart/clear", cartHandler.ClearCart)
	app.Post("/cart/checkout", cartHandler.CheckoutCart)

	// Order/Payment routes
	app.Post("/orders", paymentHandler.CreateOrder)
	app.Get("/orders", paymentHandler.ListOrders)
	app.Get("/orders/customer", paymentHandler.ListCustomerOrders)
	app.Get("/orders/status/:orderNumber", paymentHandler.GetOrderStatus)
	app.Get("/orders/:id", paymentHandler.GetOrderByID)
	app.Put("/orders/:id", paymentHandler.UpdateOrder)
	app.Post("/orders/:id/confirm-payment", paymentHandler.ManualPaymentConfirm)
	app.Post("/orders/:id/cancel", paymentHandler.CancelOrder)

	// Payment webhook routes
	app.Post("/webhooks/midtrans", paymentHandler.MidtransWebhook)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("✅ saas-api running at :%s", port)
	log.Printf("📄 Swagger UI: http://localhost:%s/swagger/", port)
	log.Printf("🔗 QR Endpoint: http://localhost:%s/whatsapp/qr", port)
	log.Fatal(app.Listen(":" + port))
}

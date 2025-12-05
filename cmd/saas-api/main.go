package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/email"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/notification"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/ocr"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/payment"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
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

	// Init webhook service with cart and order services
	webhookService := services.NewWebhookService(clientRepo, conversationRepo, transactionRepo, kbRetriever, llmService, waService, ocrService, tenantResolver, cartService, orderService)

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

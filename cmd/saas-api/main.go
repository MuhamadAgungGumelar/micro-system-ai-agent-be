package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/handlers"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
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
	kbRetriever := kb.NewRetriever(db.GORM)

	// Init WhatsApp service (untuk QR endpoint)
	waService := whatsapp.NewService(cfg.WhatsAppStoreURL)

	// Log provider info
	log.Printf("📱 Using WhatsApp provider: %s", waService.GetProviderName())

	// Init handlers
	clientHandler := handlers.NewClientHandler(clientRepo)
	kbHandler := handlers.NewKBHandler(kbRetriever)
	healthHandler := handlers.NewHealthHandler(waService)
	whatsappHandler := handlers.NewWhatsAppHandler(waService, clientRepo)
	webhookHandler := handlers.NewWebhookHandler()

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
	app.Get("/whatsapp/session/status", whatsappHandler.GetSessionStatus)

	// Webhook route
	app.Post("/webhook", webhookHandler.ReceiveWebhook)

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

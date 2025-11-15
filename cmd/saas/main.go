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

	_ "github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/docs"
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

	// Init repositories
	clientRepo := repositories.NewClientRepo(db.DB)
	kbRetriever := kb.NewRetriever(db.DB)

	// Init handlers
	clientHandler := handlers.NewClientHandler(clientRepo)
	kbHandler := handlers.NewKBHandler(kbRetriever)

	// Init WhatsApp service (untuk QR endpoint)
	waService := whatsapp.NewService(cfg.WhatsAppStoreURL)

	// Init Fiber app
	app := fiber.New(fiber.Config{
		AppName: "WhatsApp Bot SaaS API",
	})

	// Middleware
	app.Use(cors.New())

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "saas-api",
		})
	})

	// Client routes
	app.Get("/clients", clientHandler.GetActiveClients)
	app.Get("/clients/:id", clientHandler.GetClientByID)

	// Knowledge Base routes
	app.Get("/knowledge-base", kbHandler.GetKnowledgeBase)
	app.Post("/knowledge-base", kbHandler.AddKnowledgeItem)

	// WhatsApp QR route
	app.Get("/whatsapp/qr", func(c *fiber.Ctx) error {
		clientID := c.Query("client_id")
		if clientID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "client_id is required",
			})
		}

		qr, err := waService.GenerateQR()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Set("Content-Type", "image/png")
		c.Set("Content-Disposition", "attachment; filename=whatsapp-qr.png")
		return c.Send(qr)
	})

	// Webhook route (placeholder)
	app.Post("/webhook", func(c *fiber.Ctx) error {
		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid payload",
			})
		}

		log.Printf("📨 Webhook received: %+v", payload)
		return c.JSON(fiber.Map{"status": "received"})
	})

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("✅ saas-api running at :%s", port)
	log.Fatal(app.Listen(":" + port))
}

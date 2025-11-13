package main

import (
	"log"

	_ "github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/docs"
	_ "github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/models"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/config"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/database"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/repositories"
)

var clientRepo repositories.ClientRepo
var kbRepo repositories.KBRepo

// @title WhatsApp Bot SaaS API
// @version 1.0
// @description API documentation for WhatsApp Bot SaaS
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@whatsapp-saas.com
// @license.name MIT
// @host localhost:8080
// @BasePath /
func main() {
	_ = godotenv.Load()
	cfg := config.LoadConfig()
	db := database.NewDB(cfg.DatabaseURL)
	defer db.Close()

	clientRepo = repositories.NewClientRepo(db.DB)
	kbRepo = repositories.NewKBRepo(db.DB)

	app := fiber.New()

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Routes
	app.Get("/clients", GetClients)
	app.Get("/knowledge-base", GetKnowledgeBase)
	app.Post("/knowledge-base", AddKnowledgeItem)
	app.Post("/webhook", HandleWebhook)
	app.Get("/whatsapp/qr", GetWhatsAppQR)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 API running at :%s", port)
	log.Fatal(app.Listen(":" + port))
}

// @Summary Get all active clients
// @Description Returns all clients with active subscription
// @Tags Clients
// @Produce json
// @Success 200 {array} models.Client
// @Router /clients [get]
func GetClients(c *fiber.Ctx) error {
	clients, err := clientRepo.GetActiveClients()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch clients"})
	}
	return c.JSON(clients)
}

// @Summary Get Knowledge Base by Client ID
// @Description Returns knowledge base data for a client
// @Tags KnowledgeBase
// @Produce json
// @Param client_id query string true "Client ID"
// @Success 200 {object} models.KnowledgeBase
// @Router /knowledge-base [get]
func GetKnowledgeBase(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "client_id is required"})
	}
	kb, err := kbRepo.GetKnowledgeBase(clientID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch knowledge base"})
	}
	return c.JSON(kb)
}

// @Summary Add new knowledge base item
// @Description Adds FAQ or product entry to knowledge base
// @Tags KnowledgeBase
// @Accept json
// @Produce json
// @Param data body map[string]interface{} true "Knowledge base data"
// @Success 201 {object} map[string]string
// @Router /knowledge-base [post]
func AddKnowledgeItem(c *fiber.Ctx) error {
	var req struct {
		ClientID string  `json:"client_id"`
		Type     string  `json:"type"` // faq / product
		Question string  `json:"question,omitempty"`
		Answer   string  `json:"answer,omitempty"`
		Name     string  `json:"name,omitempty"`
		Price    float64 `json:"price,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	log.Printf("📚 Knowledge item added: %+v", req)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "ok"})
}

// @Summary Webhook handler
// @Description Receive events from WhatsApp
// @Tags Webhook
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /webhook [post]
func HandleWebhook(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	log.Printf("📨 Webhook received: %+v", payload)
	return c.JSON(fiber.Map{"status": "received"})
}

// @Summary Generate WhatsApp QR Code for client
// @Description Returns a QR Code image (base64) for the given client
// @Tags WhatsApp
// @Produce json
// @Param client_id query string true "Client ID"
// @Success 200 {object} map[string]string
// @Router /whatsapp/qr [get]
func GetWhatsAppQR(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	waService := services.NewWhatsAppService()
	qr, err := waService.GenerateQRForClient(clientID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"client_id": clientID,
		"qr_image":  qr, // base64 image
	})
}

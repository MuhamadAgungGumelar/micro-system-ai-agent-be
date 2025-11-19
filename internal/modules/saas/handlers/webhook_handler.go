package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct{}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// ReceiveWebhook godoc
// @Summary WhatsApp webhook receiver
// @Description Receive webhook events from WhatsApp Provider
// @Tags Webhook
// @Accept json
// @Produce json
// @Param payload body map[string]interface{} true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /webhook [post]
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}

	log.Printf("📨 Webhook received: %+v", payload)

	// TODO: Process webhook message
	// Forward ke agent core untuk processing

	return c.JSON(fiber.Map{"status": "received"})
}

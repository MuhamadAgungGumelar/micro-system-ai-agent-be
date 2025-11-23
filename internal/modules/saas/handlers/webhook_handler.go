package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles HTTP webhook requests (thin layer)
type WebhookHandler struct {
	webhookService *services.WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// WAHAWebhookPayload represents incoming WAHA webhook message
type WAHAWebhookPayload struct {
	Event   string `json:"event"`
	Session string `json:"session"`
	Payload struct {
		ID        string                 `json:"id"`
		Timestamp int64                  `json:"timestamp"`
		From      string                 `json:"from"` // Format: 628xxx@c.us
		FromMe    bool                   `json:"fromMe"`
		To        string                 `json:"to"`
		Body      string                 `json:"body"`
		HasMedia  bool                   `json:"hasMedia"`
		MediaURL  string                 `json:"mediaUrl"`  // URL to download media
		MimeType  string                 `json:"mimeType"`  // image/jpeg, image/png, etc
		Media     map[string]interface{} `json:"media"`     // WAHA media object (fallback)
		Ack       int                    `json:"ack"`
	} `json:"payload"`
}

// ReceiveWebhook godoc
// @Summary WhatsApp webhook receiver
// @Description Receive webhook events from WhatsApp Provider (WAHA/GreenAPI)
// @Tags Webhook
// @Accept json
// @Produce json
// @Param payload body map[string]interface{} true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /webhook [post]
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error {
	// Log raw body for debugging
	rawBody := c.Body()
	log.Printf("📥 Raw webhook payload: %s", string(rawBody))

	// Parse webhook payload
	var payload WAHAWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		log.Printf("❌ Failed to parse webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}

	log.Printf("📨 Webhook received - Event: %s, From: %s, FromMe: %v, HasMedia: %v, MimeType: %s, MediaURL: %s, Body: %s",
		payload.Event, payload.Payload.From, payload.Payload.FromMe, payload.Payload.HasMedia, payload.Payload.MimeType, payload.Payload.MediaURL, payload.Payload.Body)

	// Skip invalid messages
	if payload.Event != "message" || payload.Payload.FromMe || payload.Payload.From == "" {
		log.Printf("⏭️ Skipping event - Event: %s, FromMe: %v, From: %s",
			payload.Event, payload.Payload.FromMe, payload.Payload.From)
		return c.JSON(fiber.Map{"status": "ignored"})
	}

	// Check if this is an image message (receipt photo)
	isImageMessage := payload.Payload.HasMedia

	// Skip if neither text nor image
	if !isImageMessage && (payload.Payload.Body == "" ||
		strings.Contains(payload.Payload.Body, "@c.us") ||
		strings.Contains(payload.Payload.Body, "@s.whatsapp.net")) {
		log.Printf("⏭️ Skipping - Not a valid text or image message")
		return c.JSON(fiber.Map{"status": "ignored"})
	}

	// Extract phone number from 'from' field (format: 628xxx@c.us)
	phoneNumber := extractPhoneNumber(payload.Payload.From)
	if phoneNumber == "" {
		log.Printf("⚠️ Invalid phone number format: %s", payload.Payload.From)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid phone number",
		})
	}

	// Process message based on type
	if isImageMessage {
		// Extract media URL from various possible fields
		mediaURL := extractMediaURL(&payload)
		if mediaURL == "" {
			log.Printf("⚠️ Image message but no media URL found")
			return c.JSON(fiber.Map{"status": "ignored", "reason": "no_media_url"})
		}

		log.Printf("📸 Image message detected from %s - MediaURL: %s", phoneNumber, mediaURL)
		// Process image message (OCR for receipt) - delegate to service
		go h.webhookService.ProcessImageMessage(payload.Session, phoneNumber, mediaURL)
	} else {
		log.Printf("✅ Text message detected from %s: %s", phoneNumber, payload.Payload.Body)
		// Process text message (AI chat) - delegate to service
		go h.webhookService.ProcessTextMessage(payload.Session, phoneNumber, payload.Payload.Body)
	}

	return c.JSON(fiber.Map{"status": "received"})
}

// extractMediaURL tries to extract media URL from various possible fields
func extractMediaURL(payload *WAHAWebhookPayload) string {
	// Try direct mediaUrl field first
	if payload.Payload.MediaURL != "" {
		return payload.Payload.MediaURL
	}

	// Try media object (WAHA sometimes puts URL in media.url or media.link)
	if payload.Payload.Media != nil {
		if url, ok := payload.Payload.Media["url"].(string); ok && url != "" {
			return url
		}
		if link, ok := payload.Payload.Media["link"].(string); ok && link != "" {
			return link
		}
		if mediaUrl, ok := payload.Payload.Media["mediaUrl"].(string); ok && mediaUrl != "" {
			return mediaUrl
		}
	}

	// Check if we need to construct URL manually from message ID
	// WAHA format: GET /api/messages/{id}/media
	if payload.Payload.ID != "" {
		// Construct media URL using WAHA API
		// Format: http://localhost:3000/api/sessions/{session}/messages/{id}/media
		baseURL := "http://localhost:3000" // You can get this from config
		return fmt.Sprintf("%s/api/sessions/%s/messages/%s/media", baseURL, payload.Session, payload.Payload.ID)
	}

	return ""
}

// extractPhoneNumber extracts clean phone number from WhatsApp format (e.g., "628xxx@c.us" -> "628xxx")
func extractPhoneNumber(from string) string {
	// Format: 628123456789@c.us or 628123456789@s.whatsapp.net
	parts := strings.Split(from, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return from
}

// Helper to pretty print webhook payload for debugging
func prettyPrint(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

package handlers

import (
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/gofiber/fiber/v2"
)

type WhatsAppHandler struct {
	whatsappService *whatsapp.Service
	clientRepo      repositories.ClientRepo
}

func NewWhatsAppHandler(whatsappService *whatsapp.Service, clientRepo repositories.ClientRepo) *WhatsAppHandler {
	return &WhatsAppHandler{
		whatsappService: whatsappService,
		clientRepo:      clientRepo,
	}
}

// GetQRCode godoc
// @Summary Get WhatsApp QR Code
// @Description Generate QR code for WhatsApp authentication
// @Tags WhatsApp
// @Produce image/png
// @Param session_id query string false "Session ID" default(default)
// @Success 200 {file} image/png
// @Failure 500 {object} map[string]interface{}
// @Router /whatsapp/qr [get]
func (h *WhatsAppHandler) GetQRCode(c *fiber.Ctx) error {
	sessionID := c.Query("session_id", "default")

	log.Printf("🔍 Generating QR for session: %s (provider: %s)", sessionID, h.whatsappService.GetProviderName())

	// Pass sessionID to service
	qr, err := h.whatsappService.GenerateQR(sessionID)
	if err != nil {
		log.Printf("❌ Failed to generate QR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return as image
	c.Set("Content-Type", "image/png")
	c.Set("Content-Disposition", "inline; filename=whatsapp-qr.png")
	return c.Send(qr)
}

// StartSession godoc
// @Summary Start WhatsApp session
// @Description Start a new WhatsApp session for a client
// @Tags WhatsApp
// @Accept json
// @Produce json
// @Param data body object{session_id=string,client_id=string} true "Session data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /whatsapp/session/start [post]
func (h *WhatsAppHandler) StartSession(c *fiber.Ctx) error {
	var req struct {
		SessionID string `json:"session_id"`
		ClientID  string `json:"client_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	if req.SessionID == "" {
		req.SessionID = "default"
	}

	log.Printf("🚀 Starting session '%s' for client '%s'", req.SessionID, req.ClientID)

	// Start session via service
	if err := h.whatsappService.StartSession(req.SessionID); err != nil {
		log.Printf("❌ Failed to start session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Store session mapping in database (client_id -> session_id)
	if req.ClientID != "" {
		client, err := h.clientRepo.GetByID(req.ClientID)
		if err != nil {
			log.Printf("⚠️ Failed to get client: %v", err)
		} else {
			// Update client with whatsapp_session_id
			client.WhatsAppSessionID = req.SessionID
			if err := h.clientRepo.Update(client); err != nil {
				log.Printf("⚠️ Failed to update client session: %v", err)
			} else {
				log.Printf("✅ Session mapping stored: client=%s -> session=%s", req.ClientID, req.SessionID)
			}
		}
	}

	return c.JSON(fiber.Map{
		"status":     "ok",
		"message":    "Session started successfully",
		"session_id": req.SessionID,
		"client_id":  req.ClientID,
		"provider":   h.whatsappService.GetProviderName(),
	})
}

// GetSessionStatus godoc
// @Summary Get WhatsApp session status
// @Description Check if WhatsApp session is connected
// @Tags WhatsApp
// @Produce json
// @Param session_id query string false "Session ID" default(default)
// @Success 200 {object} map[string]interface{}
// @Router /whatsapp/session/status [get]
func (h *WhatsAppHandler) GetSessionStatus(c *fiber.Ctx) error {
	sessionID := c.Query("session_id", "default")

	log.Printf("📊 Checking status for session: %s", sessionID)

	// Check session status via service
	connected, err := h.whatsappService.GetSessionStatus(sessionID)
	if err != nil {
		log.Printf("⚠️ Failed to get session status: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"session_id": sessionID,
		"connected":  connected,
		"provider":   h.whatsappService.GetProviderName(),
	})
}

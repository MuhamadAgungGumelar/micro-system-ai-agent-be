package handlers

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	whatsappService *whatsapp.Service
}

func NewHealthHandler(whatsappService *whatsapp.Service) *HealthHandler {
	return &HealthHandler{whatsappService: whatsappService}
}

// GetHealth godoc
// @Summary Service health check
// @Description Check if API is alive
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *HealthHandler) GetHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":   "ok",
		"service":  "saas-api",
		"provider": h.whatsappService.GetProviderName(),
	})
}

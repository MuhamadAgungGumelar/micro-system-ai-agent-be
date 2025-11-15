package handlers

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/gofiber/fiber/v2"
)

type ClientHandler struct {
	clientRepo repositories.ClientRepo
}

func NewClientHandler(repo repositories.ClientRepo) *ClientHandler {
	return &ClientHandler{clientRepo: repo}
}

// GetActiveClients godoc
// @Summary Get all active clients
// @Description Returns all clients with active subscription
// @Tags Clients
// @Produce json
// @Success 200 {array} models.Client
// @Router /clients [get]
func (h *ClientHandler) GetActiveClients(c *fiber.Ctx) error {
	clients, err := h.clientRepo.GetActiveClients()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch clients",
		})
	}

	return c.JSON(clients)
}

// GetClientByID godoc
// @Summary Get client by ID
// @Description Returns a single client by ID
// @Tags Clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} models.Client
// @Router /clients/{id} [get]
func (h *ClientHandler) GetClientByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	client, err := h.clientRepo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "client not found",
		})
	}

	return c.JSON(client)
}

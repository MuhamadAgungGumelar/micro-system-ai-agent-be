package handlers

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/gofiber/fiber/v2"
)

type KBHandler struct {
	kbRetriever *kb.Retriever
}

func NewKBHandler(retriever *kb.Retriever) *KBHandler {
	return &KBHandler{kbRetriever: retriever}
}

// GetKnowledgeBase godoc
// @Summary Get Knowledge Base by Client ID
// @Description Returns knowledge base data for a client
// @Tags KnowledgeBase
// @Produce json
// @Param client_id query string true "Client ID"
// @Success 200 {object} llm.KnowledgeBase
// @Router /knowledge-base [get]
func (h *KBHandler) GetKnowledgeBase(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	kb, err := h.kbRetriever.GetKnowledgeBase(clientID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch knowledge base",
		})
	}

	return c.JSON(kb)
}

// AddKnowledgeItem godoc
// @Summary Add new knowledge base item
// @Description Adds FAQ or product entry to knowledge base
// @Tags KnowledgeBase
// @Accept json
// @Produce json
// @Param data body map[string]interface{} true "Knowledge base data"
// @Success 201 {object} map[string]string
// @Router /knowledge-base [post]
func (h *KBHandler) AddKnowledgeItem(c *fiber.Ctx) error {
	var req struct {
		ClientID string  `json:"client_id"`
		Type     string  `json:"type"` // faq / product
		Question string  `json:"question,omitempty"`
		Answer   string  `json:"answer,omitempty"`
		Name     string  `json:"name,omitempty"`
		Price    float64 `json:"price,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	// TODO: Implement insert to knowledge_base table
	// For now, just return success

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "ok",
	})
}

package handlers

import (
	"encoding/json"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type KBHandler struct {
	kbRetriever *kb.Retriever
	kbRepo      repositories.KBRepo
}

func NewKBHandler(retriever *kb.Retriever, repo repositories.KBRepo) *KBHandler {
	return &KBHandler{
		kbRetriever: retriever,
		kbRepo:      repo,
	}
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

// KnowledgeBaseRequest represents request body for adding knowledge base item
type KnowledgeBaseRequest struct {
	ClientID string                 `json:"client_id" example:"7a393015-15b8-4bcf-8ce6-840f753bfb1c"`
	Type     string                 `json:"type" example:"faq"` // faq, product, service, policy, or any custom type
	Title    string                 `json:"title" example:"Cara Order"`
	Content  map[string]interface{} `json:"content" swaggertype:"object"`
	Tags     []string               `json:"tags,omitempty" example:"order,howto"`
}

// AddKnowledgeItem godoc
// @Summary Add new knowledge base item
// @Description Adds knowledge base entry with flexible JSONB content. The 'content' field accepts any JSON structure. Examples: For FAQ use {"question":"...","answer":"..."}, for Product use {"name":"...","price":50000,"description":"...","stock":100}
// @Tags KnowledgeBase
// @Accept json
// @Produce json
// @Param data body KnowledgeBaseRequest true "Knowledge base data - content field accepts any JSON structure"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /knowledge-base [post]
func (h *KBHandler) AddKnowledgeItem(c *fiber.Ctx) error {
	var req KnowledgeBaseRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	// Validate required fields
	if req.ClientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	if req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "type is required",
		})
	}

	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "title is required",
		})
	}

	if req.Content == nil || len(req.Content) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "content is required and cannot be empty",
		})
	}

	// Parse client_id to UUID
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid client_id format",
		})
	}

	// Convert content map to JSON bytes for datatypes.JSON
	contentJSON, err := json.Marshal(req.Content)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid content format",
		})
	}

	// Create knowledge base entry
	entry := &models.KnowledgeBaseEntry{
		ClientID: clientUUID,
		Type:     req.Type,
		Title:    req.Title,
		Content:  datatypes.JSON(contentJSON),     // Convert to datatypes.JSON
		Tags:     pq.StringArray(req.Tags),        // Convert []string to pq.StringArray
		IsActive: true,
	}

	// Save to database
	if err := h.kbRepo.Create(entry); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create knowledge base entry",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "ok",
		"message": "Knowledge base entry created successfully",
		"id":      entry.ID.String(),
	})
}

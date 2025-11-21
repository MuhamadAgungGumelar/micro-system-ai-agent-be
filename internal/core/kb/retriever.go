package kb

import (
	"encoding/json"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Retriever struct {
	db *gorm.DB
}

func NewRetriever(db *gorm.DB) *Retriever {
	return &Retriever{db: db}
}

// GetKnowledgeBase mengambil knowledge base untuk client tertentu
func (r *Retriever) GetKnowledgeBase(clientID string) (*llm.KnowledgeBase, error) {
	kb := &llm.KnowledgeBase{}

	// Parse UUID
	uid, err := uuid.Parse(clientID)
	if err != nil {
		return nil, err
	}

	// Get client info
	var client models.Client
	if err := r.db.First(&client, "id = ?", uid).Error; err != nil {
		return nil, err
	}

	kb.BusinessName = client.BusinessName
	kb.Tone = client.Tone

	// Get all knowledge base entries
	var entries []models.KnowledgeBaseEntry
	if err := r.db.Where("client_id = ? AND is_active = ?", uid, true).
		Order("created_at DESC").
		Limit(100).
		Find(&entries).Error; err != nil {
		return nil, err
	}

	// Parse entries based on type
	for _, entry := range entries {
		// Unmarshal JSONB content
		var content map[string]interface{}
		contentBytes, err := entry.Content.MarshalJSON()
		if err != nil {
			continue // Skip if can't get JSON
		}

		if err := json.Unmarshal(contentBytes, &content); err != nil {
			continue // Skip if can't unmarshal
		}

		switch entry.Type {
		case "faq":
			// Extract FAQ from JSONB content
			if question, ok := content["question"].(string); ok {
				if answer, ok := content["answer"].(string); ok {
					kb.FAQs = append(kb.FAQs, llm.FAQ{
						Question: question,
						Answer:   answer,
					})
				}
			}

		case "product":
			// Extract Product from JSONB content
			if name, ok := content["name"].(string); ok {
				price := 0.0
				if p, ok := content["price"].(float64); ok {
					price = p
				}
				kb.Products = append(kb.Products, llm.Product{
					Name:  name,
					Price: price,
				})
			}

		default:
			// All other types (service, policy, promo, info, contact, etc.)
			// Add to RawEntries for flexible handling
			kb.RawEntries = append(kb.RawEntries, llm.RawKBEntry{
				Type:    entry.Type,
				Title:   entry.Title,
				Content: content,
			})
		}
	}

	return kb, nil
}

package kb

import (
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
		switch entry.Type {
		case "faq":
			// Extract FAQ from JSONB content
			if question, ok := entry.Content["question"].(string); ok {
				if answer, ok := entry.Content["answer"].(string); ok {
					kb.FAQs = append(kb.FAQs, llm.FAQ{
						Question: question,
						Answer:   answer,
					})
				}
			}

		case "product":
			// Extract Product from JSONB content
			if name, ok := entry.Content["name"].(string); ok {
				price := 0.0
				if p, ok := entry.Content["price"].(float64); ok {
					price = p
				}
				kb.Products = append(kb.Products, llm.Product{
					Name:  name,
					Price: price,
				})
			}
		}
	}

	return kb, nil
}

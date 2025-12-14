package repositories

import (
	"encoding/json"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"gorm.io/gorm"
)

type KBRepo interface {
	GetKnowledgeBase(clientID string) (*models.KnowledgeBase, error)
	Create(entry *models.KnowledgeBaseEntry) error
}

type kbRepo struct {
	db *gorm.DB
}

func NewKBRepo(db *gorm.DB) KBRepo {
	return &kbRepo{db: db}
}

func (r *kbRepo) GetKnowledgeBase(clientID string) (*models.KnowledgeBase, error) {
	kb := &models.KnowledgeBase{}

	// Get client info
	var client models.Client
	if err := r.db.Where("id = ?", clientID).First(&client).Error; err != nil {
		return nil, err
	}
	kb.BusinessName = client.BusinessName
	kb.Tone = client.Tone

	// Get all knowledge base entries for this client
	var entries []models.KnowledgeBaseEntry
	if err := r.db.Where("client_id = ? AND is_active = ?", clientID, true).Find(&entries).Error; err != nil {
		return nil, err
	}

	// Parse entries into FAQs and Products
	for _, entry := range entries {
		// Use GormDataType method to get the value
		var content map[string]interface{}
		contentBytes, err := entry.Content.MarshalJSON()
		if err != nil {
			continue // Skip if can't get JSON
		}

		if err := json.Unmarshal(contentBytes, &content); err != nil {
			continue // Skip if can't unmarshal
		}

		if entry.Type == "faq" {
			if question, ok := content["question"].(string); ok {
				if answer, ok := content["answer"].(string); ok {
					kb.FAQs = append(kb.FAQs, models.FAQ{
						Question: question,
						Answer:   answer,
					})
				}
			}
		} else if entry.Type == "product" {
			if name, ok := content["name"].(string); ok {
				price := 0.0
				if priceVal, ok := content["price"].(float64); ok {
					price = priceVal
				}
				kb.Products = append(kb.Products, models.KBProduct{
					Name:  name,
					Price: price,
				})
			}
		}
	}

	return kb, nil
}

func (r *kbRepo) Create(entry *models.KnowledgeBaseEntry) error {
	// Set default value for IsActive if not set
	if !entry.IsActive {
		entry.IsActive = true
	}

	// Use GORM to create the entry
	return r.db.Create(entry).Error
}

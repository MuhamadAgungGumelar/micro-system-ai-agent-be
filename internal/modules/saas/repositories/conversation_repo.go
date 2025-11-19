package repositories

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationRepo interface {
	LogConversation(clientID, customerPhone, message, response string) error
	GetByClientID(clientID string, limit int) ([]models.Conversation, error)
}

type conversationRepo struct {
	db *gorm.DB
}

func NewConversationRepo(db *gorm.DB) ConversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) LogConversation(clientID, customerPhone, message, response string) error {
	// Parse UUID
	uid, err := uuid.Parse(clientID)
	if err != nil {
		return err
	}

	// Create conversation record
	conversation := models.Conversation{
		ClientID:      uid,
		CustomerPhone: customerPhone,
		MessageType:   "incoming",
		MessageText:   message,
		AIResponse:    response,
	}

	if err := r.db.Create(&conversation).Error; err != nil {
		return err
	}

	// Update credits (best effort) - using raw SQL for complex date logic
	r.db.Exec(`
		UPDATE saas_credits
		SET credits_used = credits_used + 1
		WHERE client_id = ?
		AND CURRENT_DATE BETWEEN period_start AND period_end
	`, uid)

	return nil
}

func (r *conversationRepo) GetByClientID(clientID string, limit int) ([]models.Conversation, error) {
	uid, err := uuid.Parse(clientID)
	if err != nil {
		return nil, err
	}

	var conversations []models.Conversation
	err = r.db.Where("client_id = ?", uid).
		Order("created_at DESC").
		Limit(limit).
		Find(&conversations).Error

	return conversations, err
}

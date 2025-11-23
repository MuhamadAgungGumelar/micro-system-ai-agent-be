package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Conversation represents a conversation between client and customer
type Conversation struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID      uuid.UUID `gorm:"type:uuid;not null;index" json:"client_id"`
	CustomerPhone string    `gorm:"type:text;not null" json:"customer_phone"`
	MessageType   string    `gorm:"type:text;default:'incoming'" json:"message_type"`
	MessageText   string    `gorm:"type:text" json:"message_text"`
	AIResponse    string    `gorm:"type:text" json:"ai_response"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name
func (Conversation) TableName() string {
	return "saas_conversations"
}

// BeforeCreate sets UUID before creating
func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

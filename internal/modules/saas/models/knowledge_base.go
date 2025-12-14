package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// KnowledgeBaseEntry represents a single knowledge base item with flexible JSONB content
type KnowledgeBaseEntry struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID  uuid.UUID      `gorm:"type:uuid;not null;index:idx_client_type" json:"client_id"`
	Type      string         `gorm:"type:text;not null;index:idx_client_type" json:"type"` // 'faq', 'product', 'service', 'policy'
	Title     string         `gorm:"type:text;not null" json:"title"`
	Content   datatypes.JSON `gorm:"type:jsonb;not null" json:"content"` // Flexible JSONB content using GORM datatypes
	Tags      pq.StringArray `gorm:"type:text[]" json:"tags"`            // PostgreSQL text array
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name
func (KnowledgeBaseEntry) TableName() string {
	return "saas_knowledge_base"
}

// BeforeCreate sets UUID before creating
func (kb *KnowledgeBaseEntry) BeforeCreate(tx *gorm.DB) error {
	if kb.ID == uuid.Nil {
		kb.ID = uuid.New()
	}
	return nil
}

// Legacy structs for backward compatibility with existing code
type KnowledgeBase struct {
	BusinessName string      `json:"business_name"`
	Tone         string      `json:"tone"`
	FAQs         []FAQ       `json:"faqs"`
	Products     []KBProduct `json:"products"`
}

type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// KBProduct represents a simple product in knowledge base (legacy)
type KBProduct struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

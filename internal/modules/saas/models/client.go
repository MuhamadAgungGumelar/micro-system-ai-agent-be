package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Client struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WhatsAppNumber     string    `gorm:"column:whatsapp_number;type:text;not null" json:"whatsapp_number"`
	BusinessName       string    `gorm:"column:business_name;type:text;not null" json:"business_name"`
	SubscriptionPlan   string    `gorm:"column:subscription_plan;type:text;default:'free'" json:"subscription_plan"`
	SubscriptionStatus string    `gorm:"column:subscription_status;type:text;default:'active'" json:"subscription_status"`
	Tone               string    `gorm:"column:tone;type:text;default:'neutral'" json:"tone"`
	WADeviceID         string    `gorm:"column:wa_device_id;type:text" json:"wa_device_id"`
	SessionID          string    `gorm:"column:session_id;type:text" json:"session_id"` // WhatsApp session ID for multi-session providers
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (Client) TableName() string {
	return "saas_clients"
}

// BeforeCreate will set a UUID rather than numeric ID.
func (c *Client) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// KnowledgeBaseEntry represents a single knowledge base item with flexible JSONB content
type KnowledgeBaseEntry struct {
	ID        uuid.UUID              `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID  uuid.UUID              `gorm:"type:uuid;not null;index:idx_client_type" json:"client_id"`
	Type      string                 `gorm:"type:text;not null;index:idx_client_type" json:"type"` // 'faq', 'product', 'service', 'policy'
	Title     string                 `gorm:"type:text;not null" json:"title"`
	Content   map[string]interface{} `gorm:"type:jsonb;not null" json:"content"` // Flexible JSONB content
	Tags      []string               `gorm:"type:text[]" json:"tags"`
	IsActive  bool                   `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time              `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time              `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (KnowledgeBaseEntry) TableName() string {
	return "saas_knowledge_base"
}

func (kb *KnowledgeBaseEntry) BeforeCreate(tx *gorm.DB) error {
	if kb.ID == uuid.Nil {
		kb.ID = uuid.New()
	}
	return nil
}

// Legacy structs for backward compatibility with existing code
type KnowledgeBase struct {
	BusinessName string    `json:"business_name"`
	Tone         string    `json:"tone"`
	FAQs         []FAQ     `json:"faqs"`
	Products     []Product `json:"products"`
}

type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Product struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

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

func (Conversation) TableName() string {
	return "saas_conversations"
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Credit struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"client_id"`
	CreditsUsed int        `gorm:"default:0" json:"credits_used"`
	PeriodStart *time.Time `gorm:"type:date;default:CURRENT_DATE" json:"period_start"`
	PeriodEnd   *time.Time `gorm:"type:date" json:"period_end"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (Credit) TableName() string {
	return "saas_credits"
}

func (cr *Credit) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == uuid.Nil {
		cr.ID = uuid.New()
	}
	return nil
}

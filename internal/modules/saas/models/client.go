package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Client represents a SaaS client/business
type Client struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WhatsAppNumber     string    `gorm:"column:whatsapp_number;type:text" json:"whatsapp_number"`
	BusinessName       string    `gorm:"column:business_name;type:text;not null" json:"business_name"`
	Module             string    `gorm:"column:module;type:text;default:'saas'" json:"module"` // Module: saas, umkm, farmasi, manufacturing
	SubscriptionPlan   string    `gorm:"column:subscription_plan;type:text;default:'free'" json:"subscription_plan"`
	SubscriptionStatus string    `gorm:"column:subscription_status;type:text;default:'active'" json:"subscription_status"`
	Tone               string    `gorm:"column:tone;type:text;default:'neutral'" json:"tone"`
	Timezone           string    `gorm:"column:timezone;type:text;default:'Asia/Jakarta'" json:"timezone"`
	WADeviceID         string    `gorm:"column:wa_device_id;type:text" json:"wa_device_id"`
	WhatsAppSessionID  string    `gorm:"column:whatsapp_session_id;type:text" json:"whatsapp_session_id"` // WhatsApp session ID for multi-session providers (WAHA, etc)
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (Client) TableName() string {
	return "clients"
}

// BeforeCreate sets UUID before creating
func (c *Client) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

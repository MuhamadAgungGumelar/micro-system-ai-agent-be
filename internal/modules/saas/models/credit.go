package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Credit represents client credit usage tracking
type Credit struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"client_id"`
	CreditsUsed int        `gorm:"default:0" json:"credits_used"`
	PeriodStart *time.Time `gorm:"type:date;default:CURRENT_DATE" json:"period_start"`
	PeriodEnd   *time.Time `gorm:"type:date" json:"period_end"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name
func (Credit) TableName() string {
	return "saas_credits"
}

// BeforeCreate sets UUID before creating
func (cr *Credit) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == uuid.Nil {
		cr.ID = uuid.New()
	}
	return nil
}

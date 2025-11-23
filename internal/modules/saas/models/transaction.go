package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Transaction represents a business transaction (from receipt/invoice or manual entry)
type Transaction struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID        uuid.UUID      `gorm:"type:uuid;not null;index:idx_transactions_client" json:"client_id"`
	TotalAmount     float64        `gorm:"type:decimal(15,2);not null;default:0" json:"total_amount"`
	TransactionDate time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"transaction_date"`
	StoreName       string         `gorm:"type:varchar(255)" json:"store_name,omitempty"`
	Items           datatypes.JSON `gorm:"type:jsonb" json:"items,omitempty"` // Array of items as JSONB
	CreatedFrom     string         `gorm:"type:varchar(20);not null;default:'manual'" json:"created_from"` // 'ocr' or 'manual'
	SourceType      string         `gorm:"type:varchar(20);not null;default:'manual'" json:"source_type"`  // 'receipt', 'invoice', 'manual'
	OCRConfidence   *float64       `gorm:"type:float" json:"ocr_confidence,omitempty"`                     // OCR confidence score (0-1)
	OCRRawText      string         `gorm:"type:text" json:"ocr_raw_text,omitempty"`                        // Original OCR extracted text
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationship
	Client Client `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName specifies the table name
func (Transaction) TableName() string {
	return "saas_transactions"
}

// BeforeCreate sets UUID before creating
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

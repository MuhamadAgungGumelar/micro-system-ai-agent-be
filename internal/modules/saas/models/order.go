package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Order represents a customer order
type Order struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID     uuid.UUID `gorm:"type:uuid;not null" json:"client_id"`
	OrderNumber  string    `gorm:"type:text;unique;not null" json:"order_number"`

	// Customer Info
	CustomerPhone string `gorm:"type:text;not null" json:"customer_phone"`
	CustomerName  string `gorm:"type:text" json:"customer_name"`
	CustomerEmail string `gorm:"type:text" json:"customer_email"`

	// Order Details
	Items       string  `gorm:"type:jsonb;not null" json:"items"` // JSON array of OrderItem
	TotalAmount float64 `gorm:"type:decimal(12,2);not null" json:"total_amount"`
	Currency    string  `gorm:"type:text;default:'IDR'" json:"currency"`

	// Payment
	PaymentMethod   string  `gorm:"type:text" json:"payment_method"`
	PaymentStatus   string  `gorm:"type:text;default:'pending'" json:"payment_status"`
	PaymentGateway  string  `gorm:"type:text" json:"payment_gateway"` // manual, midtrans, xendit
	PaymentLink     string  `gorm:"type:text" json:"payment_link"`
	PaymentToken    string  `gorm:"type:text" json:"payment_token"`
	PaymentReference string `gorm:"type:text" json:"payment_reference"` // Transaction ID
	PaidAt          *time.Time `json:"paid_at"`

	// Fulfillment
	FulfillmentStatus string  `gorm:"type:text;default:'pending'" json:"fulfillment_status"`
	TrackingNumber    string  `gorm:"type:text" json:"tracking_number"`
	ShippedAt         *time.Time `json:"shipped_at"`
	DeliveredAt       *time.Time `json:"delivered_at"`

	// Shipping Address (optional)
	ShippingAddress string `gorm:"type:text" json:"shipping_address"`
	ShippingCity    string `gorm:"type:text" json:"shipping_city"`
	ShippingZip     string `gorm:"type:text" json:"shipping_zip"`

	// Notes
	CustomerNotes string `gorm:"type:text" json:"customer_notes"`
	AdminNotes    string `gorm:"type:text" json:"admin_notes"`

	// Metadata
	Metadata  string `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (Order) TableName() string {
	return "saas_orders"
}

// BeforeCreate sets UUID before creating
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// Order status constants
const (
	// Payment Status
	PaymentStatusPending   = "pending"
	PaymentStatusPaid      = "paid"
	PaymentStatusFailed    = "failed"
	PaymentStatusCancelled = "cancelled"
	PaymentStatusRefunded  = "refunded"

	// Fulfillment Status
	FulfillmentStatusPending    = "pending"
	FulfillmentStatusProcessing = "processing"
	FulfillmentStatusShipped    = "shipped"
	FulfillmentStatusDelivered  = "delivered"
	FulfillmentStatusCancelled  = "cancelled"
)

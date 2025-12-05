package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Subtotal    float64 `json:"subtotal"`
}

// Order represents a customer order (simplified version)
type Order struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID    uuid.UUID      `gorm:"type:uuid;not null" json:"client_id"`
	OrderNumber string         `gorm:"type:text;unique;not null" json:"order_number"`

	// Customer
	CustomerPhone string `gorm:"type:text;not null" json:"customer_phone"`
	CustomerName  string `gorm:"type:text" json:"customer_name"`

	// Order Details
	Items       datatypes.JSON `gorm:"type:jsonb;not null" json:"items"`
	TotalAmount float64        `gorm:"type:decimal(12,2);not null" json:"total_amount"`

	// Payment
	PaymentMethod    string     `gorm:"type:text" json:"payment_method"`
	PaymentStatus    string     `gorm:"type:text;default:'pending'" json:"payment_status"`
	PaymentGateway   string     `gorm:"type:text" json:"payment_gateway"`
	PaymentLink      string     `gorm:"type:text" json:"payment_link"`
	PaymentReference string     `gorm:"type:text" json:"payment_reference"`
	PaidAt           *time.Time `json:"paid_at"`

	// Fulfillment
	FulfillmentStatus string `gorm:"type:text;default:'pending'" json:"fulfillment_status"`

	// Timestamps
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

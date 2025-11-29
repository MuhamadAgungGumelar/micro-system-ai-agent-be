package payment

import (
	"time"

	"github.com/google/uuid"
)

// Gateway defines the interface for payment processing
// This allows us to swap between manual and automated payment methods
type Gateway interface {
	// Process initiates payment for an order
	// For manual: creates handoff request for admin
	// For automated: generates payment link
	Process(order *Order) (*ProcessResult, error)

	// GetStatus retrieves current payment status
	GetStatus(orderID string) (*PaymentStatus, error)

	// Cancel cancels a pending payment
	Cancel(orderID string) error

	// Name returns the gateway provider name
	Name() string
}

// Order represents an order that needs payment
type Order struct {
	ID            uuid.UUID   `json:"id"`
	ClientID      uuid.UUID   `json:"client_id"`
	OrderNumber   string      `json:"order_number"`
	CustomerPhone string      `json:"customer_phone"`
	CustomerName  string      `json:"customer_name"`
	Items         []OrderItem `json:"items"`
	TotalAmount   float64     `json:"total_amount"`
	Currency      string      `json:"currency"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
}

// OrderItem represents a single item in an order
type OrderItem struct {
	ProductID   uuid.UUID `json:"product_id"`
	VariantID   uuid.UUID `json:"variant_id"`
	ProductName string    `json:"product_name"`
	VariantName string    `json:"variant_name"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Subtotal    float64   `json:"subtotal"`
}

// ProcessResult contains the result of payment processing
type ProcessResult struct {
	Success      bool       `json:"success"`
	PaymentLink  string     `json:"payment_link,omitempty"`  // For automated
	HandoffID    *uuid.UUID `json:"handoff_id,omitempty"`    // For manual
	Message      string     `json:"message"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`    // Payment link expiry
	Instructions string     `json:"instructions,omitempty"` // Payment instructions
}

// PaymentStatus represents the current status of a payment
type PaymentStatus struct {
	OrderID     string     `json:"order_id"`
	Status      string     `json:"status"` // pending, paid, failed, cancelled, expired
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	PaymentLink string     `json:"payment_link,omitempty"`
	Reference   string     `json:"reference,omitempty"` // Transaction ID or handoff ID
	Method      string     `json:"method,omitempty"`    // bank_transfer, qris, ewallet, manual
}

// Payment status constants
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
)

// Payment method constants
const (
	MethodManual       = "manual"
	MethodBankTransfer = "bank_transfer"
	MethodQRIS         = "qris"
	MethodEWallet      = "ewallet"
	MethodCreditCard   = "credit_card"
)

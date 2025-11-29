package payment

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ManualPaymentGateway handles payment through manual admin verification
// Used for MVP - admin confirms payment manually via WhatsApp/Dashboard
type ManualPaymentGateway struct {
	db *gorm.DB
}

// NewManualPaymentGateway creates a new manual payment gateway
func NewManualPaymentGateway(db *gorm.DB) *ManualPaymentGateway {
	return &ManualPaymentGateway{
		db: db,
	}
}

// Process creates a handoff request for admin to handle payment
func (g *ManualPaymentGateway) Process(order *Order) (*ProcessResult, error) {
	// Create handoff request for admin
	handoffID := uuid.New()

	// Store order items as JSON
	itemsJSON, _ := json.Marshal(order.Items)

	handoff := map[string]interface{}{
		"id":                  handoffID,
		"client_id":           order.ClientID,
		"customer_phone":      order.CustomerPhone,
		"customer_name":       order.CustomerName,
		"reason":              "payment_pending",
		"conversation_summary": g.buildOrderSummary(order),
		"detected_intent":     string(itemsJSON),
		"status":              "pending",
		"metadata": fmt.Sprintf(`{
			"order_id": "%s",
			"order_number": "%s",
			"total_amount": %f,
			"payment_mode": "manual"
		}`, order.ID, order.OrderNumber, order.TotalAmount),
		"created_at": time.Now(),
	}

	err := g.db.Table("saas_handoff_requests").Create(handoff).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create handoff request: %w", err)
	}

	log.Printf("✅ Manual payment handoff created: %s for order %s", handoffID, order.OrderNumber)

	// Build payment instructions
	instructions := g.buildPaymentInstructions(order)

	return &ProcessResult{
		Success:      true,
		HandoffID:    &handoffID,
		Message:      "Pesanan Anda telah dibuat. Admin kami akan menghubungi Anda untuk pembayaran.",
		Instructions: instructions,
	}, nil
}

// GetStatus retrieves payment status from handoff request
func (g *ManualPaymentGateway) GetStatus(orderID string) (*PaymentStatus, error) {
	var handoff struct {
		ID        uuid.UUID
		Status    string
		CreatedAt time.Time
		Metadata  string
	}

	err := g.db.Table("saas_handoff_requests").
		Where("metadata->>'order_id' = ?", orderID).
		First(&handoff).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &PaymentStatus{
				OrderID: orderID,
				Status:  StatusPending,
			}, nil
		}
		return nil, err
	}

	// Map handoff status to payment status
	var paymentStatus string
	var paidAt *time.Time

	switch handoff.Status {
	case "pending", "assigned":
		paymentStatus = StatusPending
	case "completed":
		paymentStatus = StatusPaid
		now := time.Now()
		paidAt = &now
	case "cancelled":
		paymentStatus = StatusCancelled
	default:
		paymentStatus = StatusPending
	}

	return &PaymentStatus{
		OrderID:   orderID,
		Status:    paymentStatus,
		PaidAt:    paidAt,
		Reference: handoff.ID.String(),
		Method:    MethodManual,
	}, nil
}

// Cancel cancels a pending manual payment
func (g *ManualPaymentGateway) Cancel(orderID string) error {
	result := g.db.Table("saas_handoff_requests").
		Where("metadata->>'order_id' = ?", orderID).
		Where("status IN ('pending', 'assigned')").
		Update("status", "cancelled")

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no pending payment found for order %s", orderID)
	}

	log.Printf("✅ Manual payment cancelled for order %s", orderID)
	return nil
}

// Name returns the gateway name
func (g *ManualPaymentGateway) Name() string {
	return "Manual Payment Gateway"
}

// buildOrderSummary creates a summary of the order for admin
func (g *ManualPaymentGateway) buildOrderSummary(order *Order) string {
	summary := fmt.Sprintf("📦 Order #%s\n\n", order.OrderNumber)
	summary += fmt.Sprintf("Customer: %s (%s)\n\n", order.CustomerName, order.CustomerPhone)
	summary += "Items:\n"

	for _, item := range order.Items {
		summary += fmt.Sprintf("- %s", item.ProductName)
		if item.VariantName != "" {
			summary += fmt.Sprintf(" (%s)", item.VariantName)
		}
		summary += fmt.Sprintf(" x%d @ Rp %s = Rp %s\n",
			item.Quantity,
			formatPrice(item.UnitPrice),
			formatPrice(item.Subtotal))
	}

	summary += fmt.Sprintf("\nTotal: Rp %s", formatPrice(order.TotalAmount))

	return summary
}

// buildPaymentInstructions creates payment instructions for customer
func (g *ManualPaymentGateway) buildPaymentInstructions(order *Order) string {
	instructions := fmt.Sprintf(
		"📝 *Instruksi Pembayaran*\n\n"+
			"Nomor Pesanan: *#%s*\n"+
			"Total Pembayaran: *Rp %s*\n\n"+
			"Admin kami akan segera menghubungi Anda untuk memberikan detail pembayaran.\n\n"+
			"Metode pembayaran yang tersedia:\n"+
			"• Transfer Bank\n"+
			"• QRIS\n"+
			"• E-Wallet\n"+
			"• Cash on Delivery (COD)\n\n"+
			"Mohon tunggu, admin akan menghubungi dalam 5-10 menit. 🙏",
		order.OrderNumber,
		formatPrice(order.TotalAmount),
	)

	return instructions
}

// Helper function to format price
func formatPrice(amount float64) string {
	return fmt.Sprintf("%.0f", amount)
}

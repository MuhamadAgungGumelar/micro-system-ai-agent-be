package payment

import (
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

// Process creates a manual payment request (simplified - no handoff table)
func (g *ManualPaymentGateway) Process(order *Order) (*ProcessResult, error) {
	log.Printf("✅ Manual payment mode for order %s - admin will be notified", order.OrderNumber)

	// Build payment instructions for customer
	instructions := g.buildPaymentInstructions(order)

	// Note: Order is already saved in saas_orders table
	// No need for separate handoff table
	// Admin will be notified via WhatsApp (handled by OrderService)

	return &ProcessResult{
		Success:      true,
		Message:      "Pesanan Anda telah dibuat. Admin kami akan menghubungi Anda untuk pembayaran.",
		Instructions: instructions,
	}, nil
}

// GetStatus retrieves payment status from order table directly
func (g *ManualPaymentGateway) GetStatus(orderID string) (*PaymentStatus, error) {
	// Query order from saas_orders table
	var order struct {
		ID            uuid.UUID
		OrderNumber   string
		PaymentStatus string
		PaymentMethod string
		PaidAt        *time.Time
	}

	err := g.db.Table("saas_orders").
		Where("id = ? OR order_number = ?", orderID, orderID).
		First(&order).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &PaymentStatus{
				OrderID: orderID,
				Status:  StatusPending,
			}, nil
		}
		return nil, err
	}

	return &PaymentStatus{
		OrderID:   order.OrderNumber,
		Status:    order.PaymentStatus,
		PaidAt:    order.PaidAt,
		Reference: order.ID.String(),
		Method:    order.PaymentMethod,
	}, nil
}

// Cancel cancels a pending manual payment
func (g *ManualPaymentGateway) Cancel(orderID string) error {
	// Update order status in saas_orders table
	result := g.db.Table("saas_orders").
		Where("id = ? OR order_number = ?", orderID, orderID).
		Where("payment_status = ?", StatusPending).
		Update("payment_status", StatusCancelled)

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

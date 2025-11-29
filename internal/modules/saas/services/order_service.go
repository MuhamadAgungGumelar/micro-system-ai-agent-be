package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/payment"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo      repositories.OrderRepo
	paymentGateway payment.Gateway
	whatsappSvc    WhatsAppService
}

func NewOrderService(
	orderRepo repositories.OrderRepo,
	paymentGateway payment.Gateway,
	whatsappSvc WhatsAppService,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		paymentGateway: paymentGateway,
		whatsappSvc:    whatsappSvc,
	}
}

// CreateOrderRequest represents the request to create an order
type CreateOrderRequest struct {
	ClientID      string
	CustomerPhone string
	CustomerName  string
	Items         []payment.OrderItem
	TotalAmount   float64
}

// CreateOrder creates a new order and initiates payment
func (s *OrderService) CreateOrder(req *CreateOrderRequest) (*models.Order, *payment.ProcessResult, error) {
	// Generate order number
	orderNumber := s.generateOrderNumber()

	// Convert items to JSON
	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	// Create order
	order := &models.Order{
		ClientID:          uuid.MustParse(req.ClientID),
		OrderNumber:       orderNumber,
		CustomerPhone:     req.CustomerPhone,
		CustomerName:      req.CustomerName,
		Items:             string(itemsJSON),
		TotalAmount:       req.TotalAmount,
		Currency:          "IDR",
		PaymentStatus:     models.PaymentStatusPending,
		PaymentGateway:    s.paymentGateway.Name(),
		FulfillmentStatus: models.FulfillmentStatusPending,
	}

	// Save to database
	err = s.orderRepo.Create(order)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create order: %w", err)
	}

	log.Printf("✅ Order created: %s (Client: %s, Total: %.2f)", orderNumber, req.ClientID, req.TotalAmount)

	// Process payment
	paymentOrder := &payment.Order{
		ID:            order.ID,
		ClientID:      order.ClientID,
		OrderNumber:   order.OrderNumber,
		CustomerPhone: order.CustomerPhone,
		CustomerName:  order.CustomerName,
		Items:         req.Items,
		TotalAmount:   order.TotalAmount,
		Currency:      order.Currency,
		Status:        order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}

	result, err := s.paymentGateway.Process(paymentOrder)
	if err != nil {
		log.Printf("❌ Payment processing failed for order %s: %v", orderNumber, err)
		return order, nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Update order with payment details
	if result.PaymentLink != "" {
		order.PaymentLink = result.PaymentLink
		s.orderRepo.Update(order)
	}

	log.Printf("✅ Payment initiated for order %s via %s", orderNumber, s.paymentGateway.Name())

	// Send payment instructions to customer via WhatsApp
	s.sendPaymentInstructions(req.CustomerPhone, order, result)

	return order, result, nil
}

// ConfirmPayment confirms payment for an order (used by admin for manual mode)
func (s *OrderService) ConfirmPayment(orderID string, paymentMethod, reference string) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}

	if order.PaymentStatus == models.PaymentStatusPaid {
		return fmt.Errorf("order already paid")
	}

	// Update payment status
	now := time.Now()
	order.PaymentStatus = models.PaymentStatusPaid
	order.PaymentMethod = paymentMethod
	order.PaymentReference = reference
	order.PaidAt = &now
	order.FulfillmentStatus = models.FulfillmentStatusProcessing

	err = s.orderRepo.Update(order)
	if err != nil {
		return err
	}

	log.Printf("✅ Payment confirmed for order %s (Method: %s)", order.OrderNumber, paymentMethod)

	// Notify customer
	s.sendPaymentConfirmation(order)

	return nil
}

// CancelOrder cancels an order and its payment
func (s *OrderService) CancelOrder(orderID string) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}

	if order.PaymentStatus == models.PaymentStatusPaid {
		return fmt.Errorf("cannot cancel paid order")
	}

	// Cancel payment
	err = s.paymentGateway.Cancel(order.OrderNumber)
	if err != nil {
		log.Printf("⚠️  Failed to cancel payment for order %s: %v", order.OrderNumber, err)
		// Continue anyway to cancel order
	}

	// Update order status
	order.PaymentStatus = models.PaymentStatusCancelled
	order.FulfillmentStatus = models.FulfillmentStatusCancelled

	err = s.orderRepo.Update(order)
	if err != nil {
		return err
	}

	log.Printf("✅ Order cancelled: %s", order.OrderNumber)

	// Notify customer
	s.whatsappSvc.SendMessage(order.CustomerPhone,
		fmt.Sprintf("❌ Order #%s telah dibatalkan.", order.OrderNumber))

	return nil
}

// GetOrderStatus retrieves order and payment status
func (s *OrderService) GetOrderStatus(orderNumber string) (*models.Order, *payment.PaymentStatus, error) {
	order, err := s.orderRepo.GetByOrderNumber(orderNumber)
	if err != nil {
		return nil, nil, err
	}

	// Get payment status from gateway
	paymentStatus, err := s.paymentGateway.GetStatus(orderNumber)
	if err != nil {
		log.Printf("⚠️  Failed to get payment status for %s: %v", orderNumber, err)
		paymentStatus = &payment.PaymentStatus{
			OrderID: orderNumber,
			Status:  order.PaymentStatus,
		}
	}

	// Sync payment status if different
	if paymentStatus.Status != order.PaymentStatus {
		s.syncPaymentStatus(order, paymentStatus)
	}

	return order, paymentStatus, nil
}

// SyncPaymentStatus syncs payment status from gateway to order
func (s *OrderService) syncPaymentStatus(order *models.Order, paymentStatus *payment.PaymentStatus) {
	order.PaymentStatus = paymentStatus.Status

	if paymentStatus.Status == payment.StatusPaid && order.PaidAt == nil {
		order.PaidAt = paymentStatus.PaidAt
		order.FulfillmentStatus = models.FulfillmentStatusProcessing
		order.PaymentMethod = paymentStatus.Method
		order.PaymentReference = paymentStatus.Reference

		// Notify customer
		s.sendPaymentConfirmation(order)
	}

	s.orderRepo.Update(order)
}

// generateOrderNumber generates a unique order number
func (s *OrderService) generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("ORD-%s-%d",
		now.Format("20060102"),
		now.Unix()%100000,
	)
}

// sendPaymentInstructions sends payment instructions to customer
func (s *OrderService) sendPaymentInstructions(customerPhone string, order *models.Order, result *payment.ProcessResult) {
	message := fmt.Sprintf(
		"✅ *Pesanan Berhasil Dibuat*\n\n"+
			"No. Pesanan: *#%s*\n"+
			"Total: *Rp %s*\n\n"+
			"%s",
		order.OrderNumber,
		formatPrice(order.TotalAmount),
		result.Instructions,
	)

	s.whatsappSvc.SendMessage(customerPhone, message)
}

// sendPaymentConfirmation sends payment confirmation to customer
func (s *OrderService) sendPaymentConfirmation(order *models.Order) {
	message := fmt.Sprintf(
		"✅ *Pembayaran Diterima!*\n\n"+
			"No. Pesanan: *#%s*\n"+
			"Total: *Rp %s*\n"+
			"Status: *Sedang Diproses*\n\n"+
			"Pesanan Anda akan segera kami kirim. Terima kasih! 🙏",
		order.OrderNumber,
		formatPrice(order.TotalAmount),
	)

	s.whatsappSvc.SendMessage(order.CustomerPhone, message)
}

// Helper function to format price
func formatPrice(amount float64) string {
	return fmt.Sprintf("%.0f", amount)
}

// WhatsAppService interface for dependency injection
type WhatsAppService interface {
	SendMessage(to, message string) error
}

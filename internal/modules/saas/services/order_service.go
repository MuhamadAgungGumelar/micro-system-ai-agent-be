package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/notification"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/payment"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OrderService struct {
	orderRepo       repositories.OrderRepo
	clientRepo      repositories.ClientRepo
	paymentGateway  payment.Gateway
	whatsappSvc     WhatsAppService
	notificationSvc NotificationService
}

func NewOrderService(
	orderRepo repositories.OrderRepo,
	clientRepo repositories.ClientRepo,
	paymentGateway payment.Gateway,
	whatsappSvc WhatsAppService,
	notificationSvc NotificationService,
) *OrderService {
	return &OrderService{
		orderRepo:       orderRepo,
		clientRepo:      clientRepo,
		paymentGateway:  paymentGateway,
		whatsappSvc:     whatsappSvc,
		notificationSvc: notificationSvc,
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

	// Convert payment.OrderItem to models.OrderItem and marshal to JSON
	orderItems := make([]models.OrderItem, len(req.Items))
	for i, item := range req.Items {
		orderItems[i] = models.OrderItem{
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.UnitPrice,
			Subtotal:    item.Subtotal,
		}
	}

	itemsJSON, err := json.Marshal(orderItems)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	// Create order
	order := &models.Order{
		ClientID:          uuid.MustParse(req.ClientID),
		OrderNumber:       orderNumber,
		CustomerPhone:     req.CustomerPhone,
		CustomerName:      req.CustomerName,
		Items:             datatypes.JSON(itemsJSON),
		TotalAmount:       req.TotalAmount,
		PaymentStatus:     models.PaymentStatusPending,
		PaymentGateway:    s.paymentGateway.Name(),
		FulfillmentStatus: models.FulfillmentStatusPending,
	}

	// Save to database
	if err = s.orderRepo.Create(order); err != nil {
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
		Currency:      "IDR",
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

	// Notify tenant admin about new order
	if s.notificationSvc != nil {
		tenantAdmin := s.getTenantAdminContact(order.ClientID)
		if tenantAdmin != nil {
			itemsText := s.formatItemsForNotification(req.Items)
			if err := s.notificationSvc.NotifyNewOrder(tenantAdmin, orderNumber, req.CustomerPhone, req.TotalAmount, itemsText); err != nil {
				log.Printf("⚠️  Failed to send admin notification: %v", err)
			}
		}
	}

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

	// Notify tenant admin
	if s.notificationSvc != nil {
		tenantAdmin := s.getTenantAdminContact(order.ClientID)
		if tenantAdmin != nil {
			if err := s.notificationSvc.NotifyPaymentConfirmed(tenantAdmin, order.OrderNumber, order.CustomerPhone, order.TotalAmount); err != nil {
				log.Printf("⚠️  Failed to send payment confirmation notification to admin: %v", err)
			}
		}
	}

	return nil
}

// CancelOrder cancels an order and its payment with optional reason
func (s *OrderService) CancelOrder(orderID string, reason string) error {
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

	log.Printf("✅ Order cancelled: %s (Reason: %s)", order.OrderNumber, reason)

	// Default reason if not provided
	if reason == "" {
		reason = "Maaf, pesanan tidak dapat diproses"
	}

	// Notify customer with friendly message
	customerMessage := fmt.Sprintf(
		"😔 *Mohon Maaf*\n\n"+
			"Pesanan Anda *#%s* telah dibatalkan.\n\n"+
			"*Alasan:* %s\n\n"+
			"Silakan hubungi kami jika ada pertanyaan. Terima kasih atas pengertiannya! 🙏",
		order.OrderNumber,
		reason,
	)
	s.whatsappSvc.SendMessage(order.CustomerPhone, customerMessage)

	// Notify tenant admin
	if s.notificationSvc != nil {
		tenantAdmin := s.getTenantAdminContact(order.ClientID)
		if tenantAdmin != nil {
			if err := s.notificationSvc.NotifyOrderCancelled(tenantAdmin, order.OrderNumber, order.CustomerPhone, reason); err != nil {
				log.Printf("⚠️  Failed to send cancellation notification to admin: %v", err)
			}
		}
	}

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

// formatItemsForNotification formats order items for notification message
func (s *OrderService) formatItemsForNotification(items []payment.OrderItem) string {
	var itemsText string
	for i, item := range items {
		itemsText += fmt.Sprintf("%d. %s - %dx @ Rp %.0f = Rp %.0f",
			i+1,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
		)
		if i < len(items)-1 {
			itemsText += "\n"
		}
	}
	return itemsText
}

// UpdateOrderRequest represents the request to update an order
type UpdateOrderRequest struct {
	Items       []models.OrderItem `json:"items,omitempty"`
	TotalAmount *float64           `json:"total_amount,omitempty"`
	AdminNotes  string             `json:"admin_notes,omitempty"`
}

// UpdateOrder updates an order (used by admin when stock verification changes order)
func (s *OrderService) UpdateOrder(orderID string, req *UpdateOrderRequest) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, err
	}

	// Only allow updating pending orders
	if order.PaymentStatus != models.PaymentStatusPending {
		return nil, fmt.Errorf("cannot update order with status %s", order.PaymentStatus)
	}

	// Update items if provided
	if req.Items != nil && len(req.Items) > 0 {
		itemsJSON, err := json.Marshal(req.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal items: %w", err)
		}
		order.Items = datatypes.JSON(itemsJSON)
	}

	// Update total amount if provided
	if req.TotalAmount != nil {
		order.TotalAmount = *req.TotalAmount
	}

	// Update admin notes if provided

	err = s.orderRepo.Update(order)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Order updated: %s (Total: %.2f)", order.OrderNumber, order.TotalAmount)

	// Notify customer about order update
	s.sendOrderUpdateNotification(order)

	return order, nil
}

// ListOrders lists orders with optional filtering
func (s *OrderService) ListOrders(clientID string, limit int) ([]models.Order, error) {
	return s.orderRepo.GetByClientID(clientID, limit)
}

// ListCustomerOrders lists orders for a specific customer
func (s *OrderService) ListCustomerOrders(clientID, customerPhone string, limit int) ([]models.Order, error) {
	return s.orderRepo.GetByCustomerPhone(clientID, customerPhone, limit)
}

// GetOrderByID retrieves an order by ID
func (s *OrderService) GetOrderByID(orderID string) (*models.Order, error) {
	return s.orderRepo.GetByID(orderID)
}

// GetOrderByOrderNumber retrieves an order by order number
func (s *OrderService) GetOrderByOrderNumber(orderNumber string) (*models.Order, error) {
	return s.orderRepo.GetByOrderNumber(orderNumber)
}

// sendOrderUpdateNotification sends notification when order is updated
func (s *OrderService) sendOrderUpdateNotification(order *models.Order) {
	message := fmt.Sprintf(
		"📝 *Pesanan Diperbarui*\n\n"+
			"No. Pesanan: *#%s*\n"+
			"Total Baru: *Rp %s*\n\n"+
			"%s",
		order.OrderNumber,
		formatPrice(order.TotalAmount),
		"",
	)

	s.whatsappSvc.SendMessage(order.CustomerPhone, message)
}

// WhatsAppService interface for dependency injection
type WhatsAppService interface {
	SendMessage(to, message string) error
}

// getTenantAdminContact retrieves tenant admin contact info from client
func (s *OrderService) getTenantAdminContact(clientID uuid.UUID) *notification.AdminContact {
	client, err := s.clientRepo.GetByID(clientID.String())
	if err != nil {
		log.Printf("⚠️  Failed to get client info for notifications: %v", err)
		return nil
	}

	return &notification.AdminContact{
		Phone: client.WhatsAppNumber, // Tenant admin WhatsApp number
		Email: "",                     // TODO: Add admin_email field to clients table
		Name:  client.BusinessName,    // Business name as admin identifier
	}
}

// NotificationService interface for dependency injection
type NotificationService interface {
	SendToCustomer(customerPhone, message string) error
	NotifyNewOrder(tenantAdmin *notification.AdminContact, orderNumber, customerPhone string, totalAmount float64, items string) error
	NotifyPaymentConfirmed(tenantAdmin *notification.AdminContact, orderNumber, customerPhone string, totalAmount float64) error
	NotifyOrderCancelled(tenantAdmin *notification.AdminContact, orderNumber, customerPhone string, reason string) error
}

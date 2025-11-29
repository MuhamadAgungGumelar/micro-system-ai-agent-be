package handlers

import (
	"fmt"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	orderService *services.OrderService
}

func NewPaymentHandler(orderService *services.OrderService) *PaymentHandler {
	return &PaymentHandler{
		orderService: orderService,
	}
}

// CreateOrder godoc
// @Summary Create a new order
// @Description Create a new order for a customer (admin only)
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body services.CreateOrderRequest true "Order details"
// @Success 200 {object} map[string]interface{}
// @Router /orders [post]
func (h *PaymentHandler) CreateOrder(c *fiber.Ctx) error {
	var req services.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Validate required fields
	if req.ClientID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id is required"})
	}
	if req.CustomerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "customer_phone is required"})
	}
	if len(req.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "items is required"})
	}
	if req.TotalAmount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "total_amount must be greater than 0"})
	}

	// Create order
	order, paymentResult, err := h.orderService.CreateOrder(&req)
	if err != nil {
		log.Printf("❌ Failed to create order: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Order created successfully",
		"order":   order,
		"payment": paymentResult,
	})
}

// MidtransWebhook godoc
// @Summary Midtrans payment webhook
// @Description Handle Midtrans payment notifications
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param notification body map[string]interface{} true "Midtrans notification"
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/midtrans [post]
func (h *PaymentHandler) MidtransWebhook(c *fiber.Ctx) error {
	var notification map[string]interface{}
	if err := c.BodyParser(&notification); err != nil {
		log.Printf("❌ Failed to parse Midtrans webhook: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	log.Printf("📥 Midtrans webhook received: %v", notification)

	// Extract order ID and transaction status
	orderID, ok := notification["order_id"].(string)
	if !ok {
		log.Printf("❌ Missing order_id in Midtrans webhook")
		return c.Status(400).JSON(fiber.Map{"error": "missing order_id"})
	}

	transactionStatus, ok := notification["transaction_status"].(string)
	if !ok {
		log.Printf("❌ Missing transaction_status in Midtrans webhook")
		return c.Status(400).JSON(fiber.Map{"error": "missing transaction_status"})
	}

	paymentType := notification["payment_type"].(string)
	transactionID := notification["transaction_id"].(string)

	log.Printf("📋 Order: %s, Status: %s, Type: %s, TxID: %s",
		orderID, transactionStatus, paymentType, transactionID)

	// Handle based on transaction status
	switch transactionStatus {
	case "capture", "settlement":
		// Payment successful!
		log.Printf("✅ Payment successful for order %s", orderID)

		err := h.orderService.ConfirmPayment(orderID, paymentType, transactionID)
		if err != nil {
			log.Printf("❌ Failed to confirm payment for order %s: %v", orderID, err)
			// Return 200 anyway to prevent Midtrans from retrying
			return c.JSON(fiber.Map{
				"status":  "received",
				"message": "payment received but confirmation failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "payment confirmed",
		})

	case "pending":
		log.Printf("⏳ Payment pending for order %s", orderID)
		return c.JSON(fiber.Map{
			"status":  "received",
			"message": "payment pending",
		})

	case "deny", "cancel", "expire":
		log.Printf("❌ Payment %s for order %s", transactionStatus, orderID)

		err := h.orderService.CancelOrder(orderID)
		if err != nil {
			log.Printf("❌ Failed to cancel order %s: %v", orderID, err)
		}

		return c.JSON(fiber.Map{
			"status":  "received",
			"message": fmt.Sprintf("payment %s", transactionStatus),
		})

	default:
		log.Printf("⚠️  Unknown transaction status: %s for order %s", transactionStatus, orderID)
		return c.JSON(fiber.Map{
			"status":  "received",
			"message": "unknown status",
		})
	}
}

// ManualPaymentConfirm godoc
// @Summary Manually confirm payment (Admin)
// @Description Admin manually confirms payment for an order
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param payment body object{payment_method=string,reference=string,notes=string} true "Payment confirmation details"
// @Success 200 {object} map[string]interface{}
// @Router /orders/{id}/confirm-payment [post]
func (h *PaymentHandler) ManualPaymentConfirm(c *fiber.Ctx) error {
	orderID := c.Params("id")

	var req struct {
		PaymentMethod string `json:"payment_method"` // bank_transfer, qris, cod, etc
		Reference     string `json:"reference"`      // Transaction reference
		Notes         string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Admin confirms payment
	err := h.orderService.ConfirmPayment(orderID, req.PaymentMethod, req.Reference)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Payment confirmed successfully",
		"order_id": orderID,
	})
}

// GetOrderStatus godoc
// @Summary Get order status
// @Description Retrieve order and payment status by order number
// @Tags Orders
// @Produce json
// @Param orderNumber path string true "Order Number"
// @Success 200 {object} map[string]interface{}
// @Router /orders/{orderNumber}/status [get]
func (h *PaymentHandler) GetOrderStatus(c *fiber.Ctx) error {
	orderNumber := c.Params("orderNumber")

	order, paymentStatus, err := h.orderService.GetOrderStatus(orderNumber)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "order not found"})
	}

	return c.JSON(fiber.Map{
		"order":          order,
		"payment_status": paymentStatus,
	})
}

// CancelOrder godoc
// @Summary Cancel an order
// @Description Cancel a pending order
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{}
// @Router /orders/{id}/cancel [post]
func (h *PaymentHandler) CancelOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")

	err := h.orderService.CancelOrder(orderID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Order cancelled successfully",
	})
}

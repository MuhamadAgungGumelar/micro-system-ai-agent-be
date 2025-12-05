package services

import (
	"context"
	"log"
	"regexp"
	"strings"
)

// handleAdminCommand processes admin commands from WhatsApp
// Returns true if command was handled, false if message should be processed normally
func (s *WebhookService) handleAdminCommand(ctx context.Context, clientID, adminPhone, message string) bool {
	message = strings.TrimSpace(message)
	messageUpper := strings.ToUpper(message)

	// Check for CANCEL command
	// Format: CANCEL ORD-20251130-5863 Stok habis
	if strings.HasPrefix(messageUpper, "CANCEL ") {
		s.handleCancelCommand(adminPhone, message)
		return true
	}

	// Check for CONFIRM command
	// Format: CONFIRM ORD-20251130-5863 transfer TRF123456
	if strings.HasPrefix(messageUpper, "CONFIRM ") {
		s.handleConfirmCommand(adminPhone, message)
		return true
	}

	// Not an admin command
	return false
}

// handleCancelCommand processes order cancellation
// Format: CANCEL ORD-20251130-5863 Stok habis
func (s *WebhookService) handleCancelCommand(adminPhone, message string) {
	// Parse: CANCEL <order-number> <reason>
	parts := strings.SplitN(message, " ", 3)

	if len(parts) < 2 {
		s.whatsappService.SendMessage(adminPhone,
			"❌ Format salah!\n\n"+
			"Gunakan:\n"+
			"CANCEL <order-number> <alasan>\n\n"+
			"Contoh:\n"+
			"CANCEL ORD-20251130-5863 Stok habis")
		return
	}

	orderNumber := strings.TrimSpace(parts[1])
	reason := "Dibatalkan oleh admin"
	if len(parts) == 3 {
		reason = strings.TrimSpace(parts[2])
	}

	// Validate order number format
	orderPattern := regexp.MustCompile(`^ORD-\d{8}-\d+$`)
	if !orderPattern.MatchString(orderNumber) {
		s.whatsappService.SendMessage(adminPhone,
			"❌ Nomor order tidak valid!\n\n"+
			"Format yang benar: ORD-YYYYMMDD-XXXXX\n"+
			"Contoh: ORD-20251130-5863")
		return
	}

	log.Printf("🔧 Admin %s cancelling order %s: %s", adminPhone, orderNumber, reason)

	// Get order by order number
	order, err := s.orderService.GetOrderByOrderNumber(orderNumber)
	if err != nil {
		log.Printf("❌ Order not found: %s - %v", orderNumber, err)
		s.whatsappService.SendMessage(adminPhone,
			"❌ Order tidak ditemukan!\n\n"+
			"Nomor order: "+orderNumber+"\n"+
			"Pastikan nomor order benar.")
		return
	}

	// Cancel the order
	err = s.orderService.CancelOrder(order.ID.String(), reason)
	if err != nil {
		log.Printf("❌ Failed to cancel order: %v", err)
		s.whatsappService.SendMessage(adminPhone,
			"❌ Gagal membatalkan order!\n\n"+
			"Error: "+err.Error())
		return
	}

	// Success response to admin
	s.whatsappService.SendMessage(adminPhone,
		"✅ *Order Dibatalkan*\n\n"+
		"📦 Order: "+orderNumber+"\n"+
		"📝 Alasan: "+reason+"\n\n"+
		"Customer telah menerima notifikasi pembatalan.")

	log.Printf("✅ Admin %s successfully cancelled order %s", adminPhone, orderNumber)
}

// handleConfirmCommand processes payment confirmation
// Format: CONFIRM ORD-20251130-5863 transfer TRF123456
func (s *WebhookService) handleConfirmCommand(adminPhone, message string) {
	// Parse: CONFIRM <order-number> <payment-method> <reference>
	parts := strings.SplitN(message, " ", 4)

	if len(parts) < 4 {
		s.whatsappService.SendMessage(adminPhone,
			"❌ Format salah!\n\n"+
			"Gunakan:\n"+
			"CONFIRM <order-number> <metode> <referensi>\n\n"+
			"Contoh:\n"+
			"CONFIRM ORD-20251130-5863 transfer TRF123456\n"+
			"CONFIRM ORD-20251130-5863 cash NOTA-001\n"+
			"CONFIRM ORD-20251130-5863 gopay GP-987654")
		return
	}

	orderNumber := strings.TrimSpace(parts[1])
	paymentMethod := strings.TrimSpace(parts[2])
	reference := strings.TrimSpace(parts[3])

	// Validate order number format
	orderPattern := regexp.MustCompile(`^ORD-\d{8}-\d+$`)
	if !orderPattern.MatchString(orderNumber) {
		s.whatsappService.SendMessage(adminPhone,
			"❌ Nomor order tidak valid!\n\n"+
			"Format yang benar: ORD-YYYYMMDD-XXXXX\n"+
			"Contoh: ORD-20251130-5863")
		return
	}

	log.Printf("🔧 Admin %s confirming payment for order %s: %s %s", adminPhone, orderNumber, paymentMethod, reference)

	// Get order by order number
	order, err := s.orderService.GetOrderByOrderNumber(orderNumber)
	if err != nil {
		log.Printf("❌ Order not found: %s - %v", orderNumber, err)
		s.whatsappService.SendMessage(adminPhone,
			"❌ Order tidak ditemukan!\n\n"+
			"Nomor order: "+orderNumber+"\n"+
			"Pastikan nomor order benar.")
		return
	}

	// Confirm payment
	err = s.orderService.ConfirmPayment(order.ID.String(), paymentMethod, reference)
	if err != nil {
		log.Printf("❌ Failed to confirm payment: %v", err)
		s.whatsappService.SendMessage(adminPhone,
			"❌ Gagal konfirmasi pembayaran!\n\n"+
			"Error: "+err.Error())
		return
	}

	// Success response to admin
	s.whatsappService.SendMessage(adminPhone,
		"✅ *Pembayaran Dikonfirmasi*\n\n"+
		"📦 Order: "+orderNumber+"\n"+
		"💳 Metode: "+paymentMethod+"\n"+
		"🔖 Referensi: "+reference+"\n\n"+
		"Customer telah menerima notifikasi pembayaran diterima.")

	log.Printf("✅ Admin %s successfully confirmed payment for order %s", adminPhone, orderNumber)
}

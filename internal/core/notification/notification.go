package notification

import (
	"fmt"
	"log"
)

// Channel represents a notification channel
type Channel string

const (
	ChannelWhatsApp Channel = "whatsapp"
	ChannelEmail    Channel = "email"
	ChannelDatabase Channel = "database" // For future CMS/dashboard
)

// Notification represents a notification message
type Notification struct {
	To       string
	Subject  string
	Message  string
	Channels []Channel
	Data     map[string]interface{} // Additional metadata
}

// AdminContact represents admin contact information
type AdminContact struct {
	Phone string
	Email string
	Name  string
}

// WhatsAppService interface for sending WhatsApp messages
type WhatsAppService interface {
	SendMessage(to, message string) error
}

// EmailService interface for sending emails
type EmailService interface {
	SendEmail(to, subject, body string) error
	GetProviderName() string
}

// Service handles multi-channel notifications
type Service struct {
	whatsappService  WhatsAppService
	emailService     EmailService
	superAdminPhone  string // Super admin (SaaS owner) - optional
	superAdminEmail  string // Super admin email - optional
	notifySuperAdmin bool   // Whether to notify super admin
}

// NewService creates a new notification service
func NewService(whatsappSvc WhatsAppService, emailSvc EmailService, superAdminPhone, superAdminEmail string) *Service {
	return &Service{
		whatsappService:  whatsappSvc,
		emailService:     emailSvc,
		superAdminPhone:  superAdminPhone,
		superAdminEmail:  superAdminEmail,
		notifySuperAdmin: superAdminPhone != "" || superAdminEmail != "",
	}
}

// SendToTenantAdmin sends notification to tenant admin (primary recipient)
func (s *Service) SendToTenantAdmin(admin *AdminContact, subject, message string, data map[string]interface{}) error {
	var errors []error

	// Send to tenant admin via WhatsApp (primary)
	if admin.Phone != "" {
		if err := s.whatsappService.SendMessage(admin.Phone, message); err != nil {
			log.Printf("❌ Failed to send WhatsApp to tenant admin %s: %v", admin.Phone, err)
			errors = append(errors, err)
		} else {
			log.Printf("✅ WhatsApp notification sent to tenant admin: %s", admin.Phone)
		}
	}

	// Send to tenant admin via Email (if available)
	if admin.Email != "" && s.emailService != nil {
		htmlBody := s.formatEmailBody(subject, message, data)
		if err := s.emailService.SendEmail(admin.Email, subject, htmlBody); err != nil {
			log.Printf("❌ Failed to send email to tenant admin %s: %v", admin.Email, err)
			errors = append(errors, err)
		} else {
			log.Printf("✅ Email notification sent to tenant admin: %s", admin.Email)
		}
	}

	// Optionally send to super admin (for monitoring)
	if s.notifySuperAdmin {
		s.sendToSuperAdmin(admin, subject, message, data)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send notifications: %v", errors)
	}

	return nil
}

// sendToSuperAdmin sends notification copy to super admin (monitoring)
func (s *Service) sendToSuperAdmin(tenantAdmin *AdminContact, subject, message string, data map[string]interface{}) {
	// Prefix subject to indicate it's from a tenant
	superAdminSubject := fmt.Sprintf("[Tenant: %s] %s", tenantAdmin.Name, subject)

	// Add tenant info to message
	superAdminMessage := fmt.Sprintf(
		"🏢 *Tenant Admin:* %s (%s)\n\n%s",
		tenantAdmin.Name,
		tenantAdmin.Phone,
		message,
	)

	// Send via WhatsApp if configured
	if s.superAdminPhone != "" {
		if err := s.whatsappService.SendMessage(s.superAdminPhone, superAdminMessage); err != nil {
			log.Printf("⚠️  Failed to send WhatsApp to super admin: %v", err)
		} else {
			log.Printf("📨 WhatsApp notification sent to super admin: %s", s.superAdminPhone)
		}
	}

	// Send via Email if configured
	if s.superAdminEmail != "" && s.emailService != nil {
		htmlBody := s.formatEmailBody(superAdminSubject, superAdminMessage, data)
		if err := s.emailService.SendEmail(s.superAdminEmail, superAdminSubject, htmlBody); err != nil {
			log.Printf("⚠️  Failed to send email to super admin: %v", err)
		} else {
			log.Printf("📨 Email notification sent to super admin: %s", s.superAdminEmail)
		}
	}
}

// SendToCustomer sends a notification to a customer (typically via WhatsApp)
func (s *Service) SendToCustomer(customerPhone, message string) error {
	if s.whatsappService == nil {
		return fmt.Errorf("whatsapp service not configured")
	}
	return s.whatsappService.SendMessage(customerPhone, message)
}

// formatEmailBody formats the notification message as HTML
func (s *Service) formatEmailBody(subject, message string, data map[string]interface{}) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2196F3; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { padding: 20px; background: #f9f9f9; border: 1px solid #ddd; border-top: none; }
        .message { background: white; padding: 15px; border-left: 4px solid #2196F3; margin: 10px 0; }
        .data { margin-top: 20px; }
        .data-item { padding: 8px; background: white; margin: 5px 0; border-radius: 3px; }
        .label { font-weight: bold; color: #555; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #666; background: #f0f0f0; border-radius: 0 0 5px 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>🔔 ` + subject + `</h2>
        </div>
        <div class="content">
            <div class="message">
                <pre style="white-space: pre-wrap; font-family: Arial, sans-serif; margin: 0;">` + message + `</pre>
            </div>`

	// Add additional data if present
	if len(data) > 0 {
		html += `<div class="data"><h3>Additional Details:</h3>`
		for key, value := range data {
			html += fmt.Sprintf(`<div class="data-item"><span class="label">%s:</span> %v</div>`, key, value)
		}
		html += `</div>`
	}

	html += `
        </div>
        <div class="footer">
            <p>WhatsApp Bot SaaS - Automated Notification System</p>
        </div>
    </div>
</body>
</html>`

	return html
}

// NotifyNewOrder sends notification about a new order to tenant admin
func (s *Service) NotifyNewOrder(tenantAdmin *AdminContact, orderNumber, customerPhone string, totalAmount float64, items string) error {
	subject := fmt.Sprintf("🛒 New Order: %s", orderNumber)
	message := fmt.Sprintf(
		"*New Order Received!*\n\n"+
			"📦 Order Number: *%s*\n"+
			"👤 Customer: %s\n"+
			"💰 Total Amount: Rp %.0f\n"+
			"📝 Items:\n%s\n\n"+
			"Please verify stock and confirm payment.",
		orderNumber,
		customerPhone,
		totalAmount,
		items,
	)

	data := map[string]interface{}{
		"order_number":   orderNumber,
		"customer_phone": customerPhone,
		"total_amount":   totalAmount,
		"items":          items,
	}

	return s.SendToTenantAdmin(tenantAdmin, subject, message, data)
}

// NotifyPaymentConfirmed sends notification when payment is confirmed
func (s *Service) NotifyPaymentConfirmed(tenantAdmin *AdminContact, orderNumber, customerPhone string, totalAmount float64) error {
	subject := fmt.Sprintf("✅ Payment Confirmed: %s", orderNumber)
	message := fmt.Sprintf(
		"*Payment Confirmed!*\n\n"+
			"📦 Order Number: *%s*\n"+
			"👤 Customer: %s\n"+
			"💰 Amount Paid: Rp %.0f\n\n"+
			"Please prepare the order for shipment.",
		orderNumber,
		customerPhone,
		totalAmount,
	)

	data := map[string]interface{}{
		"order_number":   orderNumber,
		"customer_phone": customerPhone,
		"total_amount":   totalAmount,
	}

	return s.SendToTenantAdmin(tenantAdmin, subject, message, data)
}

// NotifyOrderCancelled sends notification when order is cancelled
func (s *Service) NotifyOrderCancelled(tenantAdmin *AdminContact, orderNumber, customerPhone string, reason string) error {
	subject := fmt.Sprintf("❌ Order Cancelled: %s", orderNumber)
	message := fmt.Sprintf(
		"*Order Cancelled*\n\n"+
			"📦 Order Number: *%s*\n"+
			"👤 Customer: %s\n"+
			"📝 Reason: %s",
		orderNumber,
		customerPhone,
		reason,
	)

	data := map[string]interface{}{
		"order_number":   orderNumber,
		"customer_phone": customerPhone,
		"reason":         reason,
	}

	return s.SendToTenantAdmin(tenantAdmin, subject, message, data)
}

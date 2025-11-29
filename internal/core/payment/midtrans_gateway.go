package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// MidtransPaymentGateway handles automated payment through Midtrans
// Supports QRIS, Bank Transfer, E-Wallet, Credit Card
type MidtransPaymentGateway struct {
	serverKey   string
	isProduction bool
	baseURL     string
	snapURL     string
	client      *http.Client
	db          *gorm.DB
}

// NewMidtransPaymentGateway creates a new Midtrans payment gateway
func NewMidtransPaymentGateway(serverKey string, isProduction bool, db *gorm.DB) *MidtransPaymentGateway {
	baseURL := "https://api.sandbox.midtrans.com/v2"
	snapURL := "https://app.sandbox.midtrans.com/snap/v1"

	if isProduction {
		baseURL = "https://api.midtrans.com/v2"
		snapURL = "https://app.midtrans.com/snap/v1"
	}

	return &MidtransPaymentGateway{
		serverKey:    serverKey,
		isProduction: isProduction,
		baseURL:      baseURL,
		snapURL:      snapURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		db: db,
	}
}

// Process creates a Midtrans Snap payment link
func (g *MidtransPaymentGateway) Process(order *Order) (*ProcessResult, error) {
	// Build item details for Midtrans
	var itemDetails []map[string]interface{}
	for _, item := range order.Items {
		itemName := item.ProductName
		if item.VariantName != "" {
			itemName += " - " + item.VariantName
		}

		itemDetails = append(itemDetails, map[string]interface{}{
			"id":       item.VariantID.String(),
			"name":     itemName,
			"price":    item.UnitPrice,
			"quantity": item.Quantity,
		})
	}

	// Create Snap transaction
	payload := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     order.OrderNumber,
			"gross_amount": order.TotalAmount,
		},
		"item_details": itemDetails,
		"customer_details": map[string]interface{}{
			"first_name": order.CustomerName,
			"phone":      order.CustomerPhone,
		},
		"enabled_payments": []string{
			"credit_card", "bca_va", "bni_va", "bri_va",
			"mandiri_va", "permata_va", "other_va",
			"gopay", "shopeepay", "qris",
		},
		"callbacks": map[string]interface{}{
			"finish": "https://your-app.com/payment/finish",
		},
		"expiry": map[string]interface{}{
			"unit":     "minutes",
			"duration": 60, // 1 hour expiry
		},
	}

	// Call Midtrans Snap API
	resp, err := g.createSnapTransaction(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create Midtrans transaction: %w", err)
	}

	// Store payment link in order metadata (optional - could use separate table)
	g.storePaymentLink(order.ID.String(), resp.RedirectURL, resp.Token)

	log.Printf("✅ Midtrans payment link created for order %s: %s", order.OrderNumber, resp.RedirectURL)

	expiresAt := time.Now().Add(60 * time.Minute)

	return &ProcessResult{
		Success:     true,
		PaymentLink: resp.RedirectURL,
		Message:     "Silakan lakukan pembayaran melalui link yang diberikan.",
		ExpiresAt:   &expiresAt,
		Instructions: g.buildPaymentInstructions(order, resp.RedirectURL),
	}, nil
}

// GetStatus retrieves payment status from Midtrans
func (g *MidtransPaymentGateway) GetStatus(orderID string) (*PaymentStatus, error) {
	// Query Midtrans transaction status
	url := fmt.Sprintf("%s/%s/status", g.baseURL, orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(g.serverKey, "")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Midtrans: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Transaction not found - probably pending/not created yet
		return &PaymentStatus{
			OrderID: orderID,
			Status:  StatusPending,
		}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("midtrans returned status %d", resp.StatusCode)
	}

	var result struct {
		TransactionStatus string `json:"transaction_status"`
		PaymentType       string `json:"payment_type"`
		TransactionTime   string `json:"transaction_time"`
		TransactionID     string `json:"transaction_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Map Midtrans status to our status
	var status string
	var paidAt *time.Time

	switch result.TransactionStatus {
	case "capture", "settlement":
		status = StatusPaid
		if result.TransactionTime != "" {
			t, _ := time.Parse("2006-01-02 15:04:05", result.TransactionTime)
			paidAt = &t
		}
	case "pending":
		status = StatusPending
	case "deny", "cancel":
		status = StatusCancelled
	case "expire":
		status = StatusExpired
	case "failure":
		status = StatusFailed
	default:
		status = StatusPending
	}

	return &PaymentStatus{
		OrderID:   orderID,
		Status:    status,
		PaidAt:    paidAt,
		Reference: result.TransactionID,
		Method:    g.mapPaymentMethod(result.PaymentType),
	}, nil
}

// Cancel cancels a pending Midtrans transaction
func (g *MidtransPaymentGateway) Cancel(orderID string) error {
	url := fmt.Sprintf("%s/%s/cancel", g.baseURL, orderID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(g.serverKey, "")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel Midtrans transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("midtrans cancel failed with status %d", resp.StatusCode)
	}

	log.Printf("✅ Midtrans payment cancelled for order %s", orderID)
	return nil
}

// Name returns the gateway name
func (g *MidtransPaymentGateway) Name() string {
	return "Midtrans Payment Gateway"
}

// createSnapTransaction creates a Snap transaction
func (g *MidtransPaymentGateway) createSnapTransaction(payload map[string]interface{}) (*SnapResponse, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := g.snapURL + "/transactions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(g.serverKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("midtrans API error: %v", errorResp)
	}

	var snapResp SnapResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapResp); err != nil {
		return nil, err
	}

	return &snapResp, nil
}

// storePaymentLink stores payment link in database
func (g *MidtransPaymentGateway) storePaymentLink(orderID, paymentLink, token string) {
	// Store in a payments table or order metadata
	// For now, we'll update the order's payment_link field
	// This assumes orders table has payment_link and payment_token columns
	g.db.Table("saas_orders").
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"payment_link":  paymentLink,
			"payment_token": token,
		})
}

// buildPaymentInstructions creates payment instructions for customer
func (g *MidtransPaymentGateway) buildPaymentInstructions(order *Order, paymentLink string) string {
	instructions := fmt.Sprintf(
		"💳 *Pembayaran Order #%s*\n\n"+
			"Total: *Rp %s*\n\n"+
			"Silakan bayar melalui link berikut:\n"+
			"%s\n\n"+
			"Metode pembayaran tersedia:\n"+
			"• Transfer Bank (BCA, BNI, BRI, Mandiri)\n"+
			"• QRIS (Scan & Pay)\n"+
			"• E-Wallet (GoPay, ShopeePay)\n"+
			"• Kartu Kredit/Debit\n\n"+
			"Link berlaku selama 1 jam.\n"+
			"Pembayaran akan otomatis dikonfirmasi. ✅",
		order.OrderNumber,
		formatPrice(order.TotalAmount),
		paymentLink,
	)

	return instructions
}

// mapPaymentMethod maps Midtrans payment type to our method
func (g *MidtransPaymentGateway) mapPaymentMethod(midtransType string) string {
	switch midtransType {
	case "gopay", "shopeepay":
		return MethodEWallet
	case "qris":
		return MethodQRIS
	case "credit_card", "debit_card":
		return MethodCreditCard
	default:
		return MethodBankTransfer
	}
}

// SnapResponse is the response from Midtrans Snap API
type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

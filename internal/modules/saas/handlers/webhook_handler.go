package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/ocr"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

type WebhookHandler struct {
	clientRepo       repositories.ClientRepo
	conversationRepo repositories.ConversationRepo
	transactionRepo  repositories.TransactionRepo
	kbRetriever      *kb.Retriever
	llmService       *llm.Service
	whatsappService  *whatsapp.Service
	ocrService       *ocr.Service
}

func NewWebhookHandler(
	clientRepo repositories.ClientRepo,
	conversationRepo repositories.ConversationRepo,
	transactionRepo repositories.TransactionRepo,
	kbRetriever *kb.Retriever,
	llmService *llm.Service,
	whatsappService *whatsapp.Service,
	ocrService *ocr.Service,
) *WebhookHandler {
	return &WebhookHandler{
		clientRepo:       clientRepo,
		conversationRepo: conversationRepo,
		transactionRepo:  transactionRepo,
		kbRetriever:      kbRetriever,
		llmService:       llmService,
		whatsappService:  whatsappService,
		ocrService:       ocrService,
	}
}

// WAHAWebhookPayload represents incoming WAHA webhook message
type WAHAWebhookPayload struct {
	Event   string `json:"event"`
	Session string `json:"session"`
	Payload struct {
		ID        string                 `json:"id"`
		Timestamp int64                  `json:"timestamp"`
		From      string                 `json:"from"` // Format: 628xxx@c.us
		FromMe    bool                   `json:"fromMe"`
		To        string                 `json:"to"`
		Body      string                 `json:"body"`
		HasMedia  bool                   `json:"hasMedia"`
		MediaURL  string                 `json:"mediaUrl"`  // URL to download media
		MimeType  string                 `json:"mimeType"`  // image/jpeg, image/png, etc
		Media     map[string]interface{} `json:"media"`     // WAHA media object (fallback)
		Ack       int                    `json:"ack"`
	} `json:"payload"`
}

// ReceiveWebhook godoc
// @Summary WhatsApp webhook receiver
// @Description Receive webhook events from WhatsApp Provider (WAHA/GreenAPI)
// @Tags Webhook
// @Accept json
// @Produce json
// @Param payload body map[string]interface{} true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /webhook [post]
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error {
	// Log raw body for debugging
	rawBody := c.Body()
	log.Printf("📥 Raw webhook payload: %s", string(rawBody))

	var payload WAHAWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		log.Printf("❌ Failed to parse webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}

	log.Printf("📨 Webhook received - Event: %s, From: %s, FromMe: %v, HasMedia: %v, MimeType: %s, MediaURL: %s, Body: %s",
		payload.Event, payload.Payload.From, payload.Payload.FromMe, payload.Payload.HasMedia, payload.Payload.MimeType, payload.Payload.MediaURL, payload.Payload.Body)

	// Skip invalid messages
	if payload.Event != "message" || payload.Payload.FromMe || payload.Payload.From == "" {
		log.Printf("⏭️ Skipping event - Event: %s, FromMe: %v, From: %s",
			payload.Event, payload.Payload.FromMe, payload.Payload.From)
		return c.JSON(fiber.Map{"status": "ignored"})
	}

	// Check if this is an image message (receipt photo)
	// Note: WAHA might not send mimeType, so we check HasMedia only
	isImageMessage := payload.Payload.HasMedia

	// Skip if neither text nor image
	if !isImageMessage && (payload.Payload.Body == "" ||
		strings.Contains(payload.Payload.Body, "@c.us") ||
		strings.Contains(payload.Payload.Body, "@s.whatsapp.net")) {
		log.Printf("⏭️ Skipping - Not a valid text or image message")
		return c.JSON(fiber.Map{"status": "ignored"})
	}

	// Extract phone number from 'from' field (format: 628xxx@c.us)
	phoneNumber := extractPhoneNumber(payload.Payload.From)
	if phoneNumber == "" {
		log.Printf("⚠️ Invalid phone number format: %s", payload.Payload.From)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid phone number",
		})
	}

	if isImageMessage {
		// Extract media URL from various possible fields
		mediaURL := h.extractMediaURL(&payload)
		if mediaURL == "" {
			log.Printf("⚠️ Image message but no media URL found")
			return c.JSON(fiber.Map{"status": "ignored", "reason": "no_media_url"})
		}

		log.Printf("📸 Image message detected from %s - MediaURL: %s", phoneNumber, mediaURL)
		// Process image message (OCR for receipt)
		go h.processImageMessage(payload.Session, phoneNumber, mediaURL)
	} else {
		log.Printf("✅ Text message detected from %s: %s", phoneNumber, payload.Payload.Body)
		// Process text message (AI chat)
		go h.processMessage(payload.Session, phoneNumber, payload.Payload.Body)
	}

	return c.JSON(fiber.Map{"status": "received"})
}

func (h *WebhookHandler) processMessage(sessionID, customerPhone, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("🔄 Processing message from %s (session: %s): %s", customerPhone, sessionID, message)

	// 1. Find client by WhatsApp session ID
	client, err := h.clientRepo.GetClientByWhatsAppSession(sessionID)
	if err != nil {
		log.Printf("❌ No client found for session '%s': %v", sessionID, err)
		return
	}

	log.Printf("📋 Using client: %s (%s)", client.BusinessName, client.ID.String())

	// 2. Start typing indicator
	if err := h.whatsappService.StartTyping(customerPhone); err != nil {
		log.Printf("⚠️ Failed to start typing indicator: %v", err)
	} else {
		log.Printf("⌨️ Typing indicator started for %s", customerPhone)
	}

	// Ensure typing stops when function exits
	defer func() {
		if err := h.whatsappService.StopTyping(customerPhone); err != nil {
			log.Printf("⚠️ Failed to stop typing indicator: %v", err)
		}
	}()

	// 3. Retrieve knowledge base for this client
	knowledgeBase, err := h.kbRetriever.GetKnowledgeBase(client.ID.String())
	if err != nil {
		log.Printf("⚠️ Failed to get knowledge base: %v", err)
		knowledgeBase = &llm.KnowledgeBase{
			BusinessName: client.BusinessName,
			Tone:         client.Tone,
		}
	}

	// 4. Build system prompt with knowledge base
	systemPrompt := llm.BuildSystemPrompt(knowledgeBase)

	// 5. Call LLM to generate response
	log.Printf("🤖 Calling LLM: %s", h.llmService.GetProviderName())
	aiResponse, err := h.llmService.GenerateResponse(ctx, systemPrompt, message)
	if err != nil {
		log.Printf("❌ LLM error (%s): %v", h.llmService.GetProviderName(), err)
		aiResponse = "Maaf, saya sedang mengalami gangguan. Silakan coba lagi nanti."
	}

	log.Printf("🤖 AI Response: %s", aiResponse)

	// 6. Send response back via WhatsApp
	if err := h.whatsappService.SendMessage(customerPhone, aiResponse); err != nil {
		log.Printf("❌ Failed to send WhatsApp message: %v", err)
		return
	}

	log.Printf("✅ Message sent to %s", customerPhone)

	// 7. Log conversation to database
	if err := h.conversationRepo.LogConversation(client.ID.String(), customerPhone, message, aiResponse); err != nil {
		log.Printf("⚠️ Failed to log conversation: %v", err)
	}

	log.Printf("💾 Conversation logged successfully")
}

// processImageMessage handles incoming image messages for OCR processing
func (h *WebhookHandler) processImageMessage(sessionID, customerPhone, mediaURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("📸 Processing image from %s (session: %s): %s", customerPhone, sessionID, mediaURL)

	// 1. Find client by WhatsApp session ID
	client, err := h.clientRepo.GetClientByWhatsAppSession(sessionID)
	if err != nil {
		log.Printf("❌ No client found for session '%s': %v", sessionID, err)
		h.whatsappService.SendMessage(customerPhone, "⚠️ Maaf, terjadi kesalahan sistem. Silakan coba lagi nanti.")
		return
	}

	log.Printf("📋 Using client: %s (%s)", client.BusinessName, client.ID.String())

	// 2. Start typing indicator
	if err := h.whatsappService.StartTyping(customerPhone); err != nil {
		log.Printf("⚠️ Failed to start typing indicator: %v", err)
	}

	defer func() {
		if err := h.whatsappService.StopTyping(customerPhone); err != nil {
			log.Printf("⚠️ Failed to stop typing indicator: %v", err)
		}
	}()

	// 3. Download image from WhatsApp media URL
	log.Printf("⬇️ Downloading image from: %s", mediaURL)
	imageData, err := h.downloadImage(mediaURL)
	if err != nil {
		log.Printf("❌ Failed to download image: %v", err)
		h.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal mengunduh gambar. Pastikan gambar terkirim dengan baik.")
		return
	}

	log.Printf("✅ Image downloaded successfully (%d bytes)", len(imageData))

	// 4. Process with OCR
	log.Printf("🔍 Processing with OCR: %s", h.ocrService.GetProviderName())
	ocrResult, err := h.ocrService.ExtractText(ctx, imageData)
	if err != nil {
		log.Printf("❌ OCR extraction failed: %v", err)
		h.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal membaca teks dari gambar. Pastikan foto struk jelas dan tidak buram.")
		return
	}

	log.Printf("✅ OCR extracted text (confidence: %.2f%%): %s", ocrResult.Confidence*100, ocrResult.Text)

	// 5. Parse receipt data using LLM (much more accurate than regex)
	llmParser := ocr.NewLLMParser(h.llmService)
	receiptData, err := llmParser.ParseReceiptWithLLM(ctx, ocrResult.Text)
	if err != nil {
		log.Printf("❌ Failed to parse receipt: %v", err)
		h.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal memproses data struk. Silakan coba lagi dengan foto yang lebih jelas.")
		return
	}

	log.Printf("📊 Parsed receipt: Total=%.2f, Date=%s, Items=%d, Store=%s",
		receiptData.TotalAmount, receiptData.TransactionDate.Format("2006-01-02"), len(receiptData.Items), receiptData.StoreName)

	// 6. Convert items to JSONB
	itemsJSON, err := json.Marshal(receiptData.Items)
	if err != nil {
		log.Printf("❌ Failed to marshal items: %v", err)
		h.whatsappService.SendMessage(customerPhone, "❌ Maaf, terjadi kesalahan saat menyimpan data.")
		return
	}

	// 7. Create transaction record
	transaction := &models.Transaction{
		ClientID:        client.ID,
		TotalAmount:     receiptData.TotalAmount,
		TransactionDate: receiptData.TransactionDate,
		StoreName:       receiptData.StoreName,
		Items:           datatypes.JSON(itemsJSON),
		CreatedFrom:     "ocr",
		SourceType:      "receipt",
		OCRConfidence:   &ocrResult.Confidence,
		OCRRawText:      ocrResult.Text,
	}

	if err := h.transactionRepo.Create(transaction); err != nil {
		log.Printf("❌ Failed to save transaction: %v", err)
		h.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal menyimpan transaksi ke database.")
		return
	}

	log.Printf("✅ Transaction saved successfully: %s", transaction.ID.String())

	// 8. Send success response to user
	responseMessage := h.buildReceiptResponseMessage(transaction, receiptData)
	if err := h.whatsappService.SendMessage(customerPhone, responseMessage); err != nil {
		log.Printf("❌ Failed to send response: %v", err)
		return
	}

	log.Printf("✅ Response sent to %s", customerPhone)
}

// downloadImage downloads image from WhatsApp media URL
func (h *WebhookHandler) downloadImage(mediaURL string) ([]byte, error) {
	// Create HTTP request
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add WAHA API key header if this is a WAHA URL
	if strings.Contains(mediaURL, "localhost:3000") || strings.Contains(mediaURL, "/api/sessions/") {
		// Get WAHA API key from environment
		wahaAPIKey := "fa11b2e40b13445d97cd7008cf7b6245" // TODO: Move to config
		req.Header.Set("X-Api-Key", wahaAPIKey)
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// buildReceiptResponseMessage creates a user-friendly message with receipt details
func (h *WebhookHandler) buildReceiptResponseMessage(transaction *models.Transaction, receipt *ocr.ReceiptData) string {
	var msg strings.Builder

	msg.WriteString("✅ *Struk berhasil diproses!*\n\n")

	if receipt.StoreName != "" {
		msg.WriteString(fmt.Sprintf("🏪 *Toko:* %s\n", receipt.StoreName))
	}

	msg.WriteString(fmt.Sprintf("📅 *Tanggal:* %s\n", receipt.TransactionDate.Format("02 Jan 2006")))

	if receipt.TotalAmount > 0 {
		msg.WriteString(fmt.Sprintf("💰 *Total:* Rp %s\n", formatCurrency(receipt.TotalAmount)))
	}

	if len(receipt.Items) > 0 {
		msg.WriteString(fmt.Sprintf("\n📦 *Item (%d):*\n", len(receipt.Items)))
		for i, item := range receipt.Items {
			if i >= 5 {
				msg.WriteString(fmt.Sprintf("   ... dan %d item lainnya\n", len(receipt.Items)-5))
				break
			}
			msg.WriteString(fmt.Sprintf("   • %s (%dx) - Rp %s\n", item.Name, item.Quantity, formatCurrency(item.Price)))
		}
	}

	msg.WriteString(fmt.Sprintf("\n🎯 *Akurasi OCR:* %.0f%%\n", *transaction.OCRConfidence*100))
	msg.WriteString(fmt.Sprintf("🆔 *ID Transaksi:* %s\n", transaction.ID.String()[:8]))

	msg.WriteString("\n_Transaksi telah tersimpan di sistem._")

	return msg.String()
}

// formatCurrency formats number as Indonesian currency
func formatCurrency(amount float64) string {
	// Simple formatting: 1000000 -> 1.000.000
	amountStr := fmt.Sprintf("%.0f", amount)

	// Add thousand separators
	var result strings.Builder
	length := len(amountStr)

	for i, char := range amountStr {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteString(".")
		}
		result.WriteRune(char)
	}

	return result.String()
}

// extractMediaURL tries to extract media URL from various possible fields
func (h *WebhookHandler) extractMediaURL(payload *WAHAWebhookPayload) string {
	// Try direct mediaUrl field first
	if payload.Payload.MediaURL != "" {
		return payload.Payload.MediaURL
	}

	// Try media object (WAHA sometimes puts URL in media.url or media.link)
	if payload.Payload.Media != nil {
		if url, ok := payload.Payload.Media["url"].(string); ok && url != "" {
			return url
		}
		if link, ok := payload.Payload.Media["link"].(string); ok && link != "" {
			return link
		}
		if mediaUrl, ok := payload.Payload.Media["mediaUrl"].(string); ok && mediaUrl != "" {
			return mediaUrl
		}
	}

	// Check if we need to construct URL manually from message ID
	// WAHA format: GET /api/messages/{id}/media
	if payload.Payload.ID != "" {
		// Construct media URL using WAHA API
		// Format: http://localhost:3000/api/sessions/{session}/messages/{id}/media
		baseURL := "http://localhost:3000" // You can get this from config
		return fmt.Sprintf("%s/api/sessions/%s/messages/%s/media", baseURL, payload.Session, payload.Payload.ID)
	}

	return ""
}

// extractPhoneNumber extracts clean phone number from WhatsApp format (e.g., "628xxx@c.us" -> "628xxx")
func extractPhoneNumber(from string) string {
	// Format: 628123456789@c.us or 628123456789@s.whatsapp.net
	parts := strings.Split(from, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return from
}

// DEPRECATED: buildSystemPrompt is no longer used - we now use llm.BuildSystemPrompt() instead
// This function is kept for reference only
// func buildSystemPrompt(kb *llm.KnowledgeBase) string { ... }

// Helper to pretty print webhook payload for debugging
func prettyPrint(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

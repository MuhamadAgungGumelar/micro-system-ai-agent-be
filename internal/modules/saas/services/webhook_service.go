package services

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
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"gorm.io/datatypes"
)

// WebhookService handles business logic for incoming WhatsApp webhooks
type WebhookService struct {
	clientRepo       repositories.ClientRepo
	conversationRepo repositories.ConversationRepo
	transactionRepo  repositories.TransactionRepo
	kbRetriever      *kb.Retriever
	llmService       *llm.Service
	whatsappService  *whatsapp.Service
	ocrService       *ocr.Service
	tenantResolver   *tenant.Resolver
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	clientRepo repositories.ClientRepo,
	conversationRepo repositories.ConversationRepo,
	transactionRepo repositories.TransactionRepo,
	kbRetriever *kb.Retriever,
	llmService *llm.Service,
	whatsappService *whatsapp.Service,
	ocrService *ocr.Service,
	tenantResolver *tenant.Resolver,
) *WebhookService {
	return &WebhookService{
		clientRepo:       clientRepo,
		conversationRepo: conversationRepo,
		transactionRepo:  transactionRepo,
		kbRetriever:      kbRetriever,
		llmService:       llmService,
		whatsappService:  whatsappService,
		ocrService:       ocrService,
		tenantResolver:   tenantResolver,
	}
}

// ProcessTextMessage handles incoming text messages with AI chat
func (s *WebhookService) ProcessTextMessage(sessionID, customerPhone, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("🔄 Processing message from %s (session: %s): %s", customerPhone, sessionID, message)

	// 1. Resolve tenant context (determine role, module, client)
	tenantCtx, err := s.tenantResolver.ResolveFromPhone(customerPhone)
	if err != nil {
		log.Printf("❌ Failed to resolve tenant for %s: %v", customerPhone, err)
		s.whatsappService.SendMessage(customerPhone, "Maaf, sistem sedang bermasalah. Silakan hubungi administrator.")
		return
	}

	log.Printf("👤 Resolved tenant: ClientID=%s, Module=%s, Role=%s", tenantCtx.ClientID, tenantCtx.Module, tenantCtx.Role)

	// 2. Get client details
	client, err := s.clientRepo.GetByID(tenantCtx.ClientID)
	if err != nil {
		log.Printf("❌ No client found for ID '%s': %v", tenantCtx.ClientID, err)
		return
	}

	log.Printf("📋 Using client: %s (%s) [Role: %s]", client.BusinessName, client.ID.String(), tenantCtx.Role)

	// 2. Start typing indicator
	if err := s.whatsappService.StartTyping(customerPhone); err != nil {
		log.Printf("⚠️ Failed to start typing indicator: %v", err)
	} else {
		log.Printf("⌨️ Typing indicator started for %s", customerPhone)
	}

	// Ensure typing stops when function exits
	defer func() {
		if err := s.whatsappService.StopTyping(customerPhone); err != nil {
			log.Printf("⚠️ Failed to stop typing indicator: %v", err)
		}
	}()

	// 3. Retrieve knowledge base for this client
	knowledgeBase, err := s.kbRetriever.GetKnowledgeBase(client.ID.String())
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
	log.Printf("🤖 Calling LLM: %s", s.llmService.GetProviderName())
	aiResponse, err := s.llmService.GenerateResponse(ctx, systemPrompt, message)
	if err != nil {
		log.Printf("❌ LLM error (%s): %v", s.llmService.GetProviderName(), err)
		aiResponse = "Maaf, saya sedang mengalami gangguan. Silakan coba lagi nanti."
	}

	log.Printf("🤖 AI Response: %s", aiResponse)

	// 6. Send response back via WhatsApp
	if err := s.whatsappService.SendMessage(customerPhone, aiResponse); err != nil {
		log.Printf("❌ Failed to send WhatsApp message: %v", err)
		return
	}

	log.Printf("✅ Message sent to %s", customerPhone)

	// 7. Log conversation to database
	if err := s.conversationRepo.LogConversation(client.ID.String(), customerPhone, message, aiResponse); err != nil {
		log.Printf("⚠️ Failed to log conversation: %v", err)
	}

	log.Printf("💾 Conversation logged successfully")
}

// ProcessImageMessage handles incoming image messages for OCR processing
func (s *WebhookService) ProcessImageMessage(sessionID, customerPhone, mediaURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("📸 Processing image from %s (session: %s): %s", customerPhone, sessionID, mediaURL)

	// 1. Resolve tenant context
	tenantCtx, err := s.tenantResolver.ResolveFromPhone(customerPhone)
	if err != nil {
		log.Printf("❌ Failed to resolve tenant for %s: %v", customerPhone, err)
		s.whatsappService.SendMessage(customerPhone, "Maaf, sistem sedang bermasalah. Silakan hubungi administrator.")
		return
	}

	log.Printf("👤 Resolved tenant: ClientID=%s, Module=%s, Role=%s", tenantCtx.ClientID, tenantCtx.Module, tenantCtx.Role)

	// 2. Get client details
	client, err := s.clientRepo.GetByID(tenantCtx.ClientID)
	if err != nil {
		log.Printf("❌ No client found for ID '%s': %v", tenantCtx.ClientID, err)
		return
	}

	log.Printf("📋 Using client: %s (%s) [Role: %s]", client.BusinessName, client.ID.String(), tenantCtx.Role)

	// 2. Start typing indicator
	if err := s.whatsappService.StartTyping(customerPhone); err != nil {
		log.Printf("⚠️ Failed to start typing indicator: %v", err)
	}

	defer func() {
		if err := s.whatsappService.StopTyping(customerPhone); err != nil {
			log.Printf("⚠️ Failed to stop typing indicator: %v", err)
		}
	}()

	// 3. Download image from WhatsApp media URL
	log.Printf("⬇️ Downloading image from: %s", mediaURL)
	imageData, err := s.downloadImage(mediaURL)
	if err != nil {
		log.Printf("❌ Failed to download image: %v", err)
		s.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal mengunduh gambar. Pastikan gambar terkirim dengan baik.")
		return
	}

	log.Printf("✅ Image downloaded successfully (%d bytes)", len(imageData))

	// 4. Process with OCR
	log.Printf("🔍 Processing with OCR: %s", s.ocrService.GetProviderName())
	ocrResult, err := s.ocrService.ExtractText(ctx, imageData)
	if err != nil {
		log.Printf("❌ OCR extraction failed: %v", err)
		s.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal membaca teks dari gambar. Pastikan foto struk jelas dan tidak buram.")
		return
	}

	log.Printf("✅ OCR extracted text (confidence: %.2f%%): %s", ocrResult.Confidence*100, ocrResult.Text)

	// 5. Parse receipt data using LLM (much more accurate than regex)
	llmParser := ocr.NewLLMParser(s.llmService)
	receiptData, err := llmParser.ParseReceiptWithLLM(ctx, ocrResult.Text)
	if err != nil {
		log.Printf("❌ Failed to parse receipt: %v", err)
		s.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal memproses data struk. Silakan coba lagi dengan foto yang lebih jelas.")
		return
	}

	log.Printf("📊 Parsed receipt: Total=%.2f, Date=%s, Items=%d, Store=%s",
		receiptData.TotalAmount, receiptData.TransactionDate.Format("2006-01-02"), len(receiptData.Items), receiptData.StoreName)

	// 6. Convert items to JSONB
	itemsJSON, err := json.Marshal(receiptData.Items)
	if err != nil {
		log.Printf("❌ Failed to marshal items: %v", err)
		s.whatsappService.SendMessage(customerPhone, "❌ Maaf, terjadi kesalahan saat menyimpan data.")
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

	if err := s.transactionRepo.Create(transaction); err != nil {
		log.Printf("❌ Failed to save transaction: %v", err)
		s.whatsappService.SendMessage(customerPhone, "❌ Maaf, gagal menyimpan transaksi ke database.")
		return
	}

	log.Printf("✅ Transaction saved successfully: %s", transaction.ID.String())

	// 8. Send success response to user
	responseMessage := s.buildReceiptResponseMessage(transaction, receiptData)
	if err := s.whatsappService.SendMessage(customerPhone, responseMessage); err != nil {
		log.Printf("❌ Failed to send response: %v", err)
		return
	}

	log.Printf("✅ Response sent to %s", customerPhone)
}

// downloadImage downloads image from WhatsApp media URL
func (s *WebhookService) downloadImage(mediaURL string) ([]byte, error) {
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
func (s *WebhookService) buildReceiptResponseMessage(transaction *models.Transaction, receipt *ocr.ReceiptData) string {
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

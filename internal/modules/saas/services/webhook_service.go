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
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/payment"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/config"
	"github.com/google/uuid"
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
	cartService      *CartService
	orderService     *OrderService
	config           *config.Config
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
	cartService *CartService,
	orderService *OrderService,
	cfg *config.Config,
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
		cartService:      cartService,
		orderService:     orderService,
		config:           cfg,
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

	// Check if message is admin command (for admin_tenant or super_admin)
	if tenantCtx.Role == "admin_tenant" || tenantCtx.Role == "super_admin" {
		if handled := s.handleAdminCommand(ctx, client.ID.String(), customerPhone, message); handled {
			return // Command handled, don't process as regular message
		}
	}

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

	// 6. Parse cart commands from AI response
	cleanResponse, commands := s.parseCartCommands(aiResponse)

	// 7. Send clean response back via WhatsApp (without commands)
	if err := s.whatsappService.SendMessage(customerPhone, cleanResponse); err != nil {
		log.Printf("❌ Failed to send WhatsApp message: %v", err)
		return
	}

	log.Printf("✅ Message sent to %s", customerPhone)

	// 8. Execute cart commands if any
	if len(commands) > 0 {
		s.executeCartCommands(ctx, client.ID.String(), customerPhone, commands, knowledgeBase.Products)
	}

	// 9. Log conversation to database
	if err := s.conversationRepo.LogConversation(client.ID.String(), customerPhone, message, cleanResponse); err != nil {
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
		// Get WAHA API key from config
		if s.config.WameoAPIKey != "" {
			req.Header.Set("X-Api-Key", s.config.WameoAPIKey)
		}
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

// CartCommand represents a cart operation command
type CartCommand struct {
	Action      string // ADD_TO_CART, VIEW_CART, CHECKOUT
	ProductName string
	Quantity    int
}

// parseCartCommands extracts cart commands from AI response
func (s *WebhookService) parseCartCommands(aiResponse string) (string, []CartCommand) {
	lines := strings.Split(aiResponse, "\n")
	var cleanLines []string
	var commands []CartCommand

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for ADD_TO_CART command
		if strings.HasPrefix(trimmed, "[ADD_TO_CART:") && strings.HasSuffix(trimmed, "]") {
			// Extract: [ADD_TO_CART:product_name|quantity]
			content := strings.TrimPrefix(trimmed, "[ADD_TO_CART:")
			content = strings.TrimSuffix(content, "]")
			parts := strings.Split(content, "|")

			if len(parts) == 2 {
				productName := strings.TrimSpace(parts[0])
				quantity := 1
				fmt.Sscanf(parts[1], "%d", &quantity)

				commands = append(commands, CartCommand{
					Action:      "ADD_TO_CART",
					ProductName: productName,
					Quantity:    quantity,
				})
				log.Printf("🛒 Parsed ADD_TO_CART command: %s x%d", productName, quantity)
			}
		} else if trimmed == "[VIEW_CART]" {
			commands = append(commands, CartCommand{Action: "VIEW_CART"})
			log.Printf("🛒 Parsed VIEW_CART command")
		} else if trimmed == "[CHECKOUT]" {
			commands = append(commands, CartCommand{Action: "CHECKOUT"})
			log.Printf("🛒 Parsed CHECKOUT command")
		} else {
			// Not a command, keep in clean response
			cleanLines = append(cleanLines, line)
		}
	}

	cleanResponse := strings.Join(cleanLines, "\n")
	cleanResponse = strings.TrimSpace(cleanResponse)

	return cleanResponse, commands
}

// executeCartCommands processes cart commands
func (s *WebhookService) executeCartCommands(ctx context.Context, clientID, customerPhone string, commands []CartCommand, products []llm.Product) {
	for _, cmd := range commands {
		switch cmd.Action {
		case "ADD_TO_CART":
			s.handleAddToCart(clientID, customerPhone, cmd.ProductName, cmd.Quantity, products)

		case "VIEW_CART":
			s.handleViewCart(clientID, customerPhone)

		case "CHECKOUT":
			s.handleCheckout(clientID, customerPhone)
		}
	}
}

// handleAddToCart adds item to cart
func (s *WebhookService) handleAddToCart(clientID, customerPhone, productName string, quantity int, products []llm.Product) {
	// Find product price from knowledge base
	var productPrice float64
	for _, p := range products {
		if strings.EqualFold(p.Name, productName) {
			productPrice = p.Price
			break
		}
	}

	if productPrice == 0 {
		log.Printf("⚠️  Product not found in knowledge base: %s", productName)
		s.whatsappService.SendMessage(customerPhone, fmt.Sprintf("Maaf, produk '%s' tidak ditemukan dalam katalog.", productName))
		return
	}

	// Add to cart
	req := &AddToCartRequest{
		ClientID:      clientID,
		CustomerPhone: customerPhone,
		ProductID:     strings.ToLower(strings.ReplaceAll(productName, " ", "_")),
		ProductName:   productName,
		Quantity:      quantity,
		Price:         productPrice,
	}

	cart, err := s.cartService.AddToCart(req)
	if err != nil {
		log.Printf("❌ Failed to add to cart: %v", err)
		s.whatsappService.SendMessage(customerPhone, "Maaf, terjadi kesalahan saat menambahkan ke keranjang.")
		return
	}

	log.Printf("✅ Added %s x%d to cart for %s", productName, quantity, customerPhone)

	// Send confirmation
	message := fmt.Sprintf(
		"✅ *Berhasil ditambahkan!*\n\n"+
			"🛒 Total item di keranjang: %d\n"+
			"💰 Total belanja: Rp %s\n\n"+
			"Ketik 'checkout' untuk lanjut pembayaran atau 'lihat keranjang' untuk cek pesanan.",
		len(cart.Items),
		formatCurrency(cart.TotalAmount),
	)
	s.whatsappService.SendMessage(customerPhone, message)
}

// handleViewCart shows cart contents
func (s *WebhookService) handleViewCart(clientID, customerPhone string) {
	cart, err := s.cartService.ViewCart(clientID, customerPhone)
	if err != nil {
		log.Printf("⚠️  No cart found: %v", err)
		s.whatsappService.SendMessage(customerPhone, "Keranjang Anda masih kosong. Yuk pesan sesuatu! 😊")
		return
	}

	if cart.IsEmpty() {
		s.whatsappService.SendMessage(customerPhone, "Keranjang Anda masih kosong. Yuk pesan sesuatu! 😊")
		return
	}

	// Build cart summary
	var msg strings.Builder
	msg.WriteString("🛒 *Keranjang Belanja Anda:*\n\n")

	for i, item := range cart.Items {
		msg.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.ProductName))
		msg.WriteString(fmt.Sprintf("   %dx @ Rp %s = Rp %s\n\n",
			item.Quantity,
			formatCurrency(item.Price),
			formatCurrency(item.Subtotal),
		))
	}

	msg.WriteString(fmt.Sprintf("💰 *Total: Rp %s*\n\n", formatCurrency(cart.TotalAmount)))
	msg.WriteString("Ketik 'checkout' untuk lanjut pembayaran.")

	s.whatsappService.SendMessage(customerPhone, msg.String())
}

// handleCheckout processes checkout
func (s *WebhookService) handleCheckout(clientID, customerPhone string) {
	// Get cart
	cart, err := s.cartService.ViewCart(clientID, customerPhone)
	if err != nil {
		log.Printf("⚠️  No cart found: %v", err)
		s.whatsappService.SendMessage(customerPhone, "Keranjang Anda masih kosong. Silakan pesan terlebih dahulu.")
		return
	}

	if cart.IsEmpty() {
		s.whatsappService.SendMessage(customerPhone, "Keranjang Anda masih kosong. Silakan pesan terlebih dahulu.")
		return
	}

	// Convert cart items to payment.OrderItem format
	orderItems := make([]payment.OrderItem, len(cart.Items))
	for i, item := range cart.Items {
		// Generate UUID from product ID string
		productUUID := uuid.MustParse("00000000-0000-0000-0000-000000000000") // Placeholder
		variantUUID := uuid.MustParse("00000000-0000-0000-0000-000000000000") // Placeholder

		orderItems[i] = payment.OrderItem{
			ProductID:   productUUID,
			VariantID:   variantUUID,
			ProductName: item.ProductName,
			VariantName: "",
			Quantity:    item.Quantity,
			UnitPrice:   item.Price,
			Subtotal:    item.Subtotal,
		}
	}

	// Create order via OrderService
	orderReq := &CreateOrderRequest{
		ClientID:      clientID,
		CustomerPhone: customerPhone,
		CustomerName:  customerPhone, // Use phone as name for now
		Items:         orderItems,
		TotalAmount:   cart.TotalAmount,
	}

	order, paymentResult, err := s.orderService.CreateOrder(orderReq)
	if err != nil {
		log.Printf("❌ Failed to create order: %v", err)
		s.whatsappService.SendMessage(customerPhone, "Maaf, terjadi kesalahan saat memproses pesanan. Silakan coba lagi.")
		return
	}

	log.Printf("✅ Order created from cart: %s", order.OrderNumber)

	// Clear cart after successful checkout
	s.cartService.ClearCart(clientID, customerPhone)

	// Send success notification (payment instructions already sent by OrderService)
	log.Printf("🎉 Checkout completed for %s - Order %s", customerPhone, order.OrderNumber)

	// Note: Notifications to tenant admin and super admin are automatically sent by OrderService.CreateOrder
	_ = paymentResult // Payment result already handled in OrderService
}

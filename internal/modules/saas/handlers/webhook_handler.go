package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	clientRepo       repositories.ClientRepo
	conversationRepo repositories.ConversationRepo
	kbRetriever      *kb.Retriever
	llmService       *llm.Service
	whatsappService  *whatsapp.Service
}

func NewWebhookHandler(
	clientRepo repositories.ClientRepo,
	conversationRepo repositories.ConversationRepo,
	kbRetriever *kb.Retriever,
	llmService *llm.Service,
	whatsappService *whatsapp.Service,
) *WebhookHandler {
	return &WebhookHandler{
		clientRepo:       clientRepo,
		conversationRepo: conversationRepo,
		kbRetriever:      kbRetriever,
		llmService:       llmService,
		whatsappService:  whatsappService,
	}
}

// WAHAWebhookPayload represents incoming WAHA webhook message
type WAHAWebhookPayload struct {
	Event   string `json:"event"`
	Session string `json:"session"`
	Payload struct {
		ID        string `json:"id"`
		Timestamp int64  `json:"timestamp"`
		From      string `json:"from"`      // Format: 628xxx@c.us
		FromMe    bool   `json:"fromMe"`
		To        string `json:"to"`
		Body      string `json:"body"`
		HasMedia  bool   `json:"hasMedia"`
		Ack       int    `json:"ack"`
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
	var payload WAHAWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		log.Printf("❌ Failed to parse webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}

	log.Printf("📨 Webhook received - Event: %s, From: %s, FromMe: %v, Body: %s",
		payload.Event, payload.Payload.From, payload.Payload.FromMe, payload.Payload.Body)

	// Only process incoming text messages from users
	// Based on WAHA documentation: https://waha.devlike.pro/docs/how-to/receive-messages/
	// Skip if:
	// - Not a "message" event (could be "message.ack", "message.reaction", "session.status", etc)
	// - Message is from me (bot sent it)
	// - Body is empty (no text content)
	// - Body contains '@c.us' or '@s.whatsapp.net' (likely a system/connection event metadata)
	// - From field is empty (invalid message)
	if payload.Event != "message" ||
	   payload.Payload.FromMe ||
	   payload.Payload.Body == "" ||
	   payload.Payload.From == "" ||
	   strings.Contains(payload.Payload.Body, "@c.us") ||
	   strings.Contains(payload.Payload.Body, "@s.whatsapp.net") {
		log.Printf("⏭️ Skipping event - Event: %s, FromMe: %v, From: %s, Body: %s",
			payload.Event, payload.Payload.FromMe, payload.Payload.From, payload.Payload.Body)
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

	log.Printf("✅ Valid message detected from %s: %s", phoneNumber, payload.Payload.Body)

	// Process message asynchronously with session context
	go h.processMessage(payload.Session, phoneNumber, payload.Payload.Body)

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
	systemPrompt := buildSystemPrompt(knowledgeBase)

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

// extractPhoneNumber extracts clean phone number from WhatsApp format (e.g., "628xxx@c.us" -> "628xxx")
func extractPhoneNumber(from string) string {
	// Format: 628123456789@c.us or 628123456789@s.whatsapp.net
	parts := strings.Split(from, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return from
}

// buildSystemPrompt creates system prompt with knowledge base context
func buildSystemPrompt(kb *llm.KnowledgeBase) string {
	prompt := fmt.Sprintf(`Kamu adalah asisten AI untuk %s dengan tone %s.

**Informasi Bisnis:**
- Nama: %s
- Tone: %s

`, kb.BusinessName, kb.Tone, kb.BusinessName, kb.Tone)

	// Add FAQs if available
	if len(kb.FAQs) > 0 {
		prompt += "**Frequently Asked Questions:**\n"
		for i, faq := range kb.FAQs {
			prompt += fmt.Sprintf("%d. Q: %s\n   A: %s\n", i+1, faq.Question, faq.Answer)
		}
		prompt += "\n"
	}

	// Add Products if available
	if len(kb.Products) > 0 {
		prompt += "**Daftar Produk:**\n"
		for i, product := range kb.Products {
			prompt += fmt.Sprintf("%d. %s - Rp %.0f\n", i+1, product.Name, product.Price)
		}
		prompt += "\n"
	}

	prompt += `**Instruksi:**
- Jawab pertanyaan customer dengan ramah dan informatif
- Gunakan informasi dari FAQ dan daftar produk di atas
- Jika pertanyaan di luar knowledge base, arahkan customer untuk contact langsung
- Gunakan tone yang sesuai dengan brand
- Maksimal 2-3 kalimat per response
- Jangan gunakan markdown formatting (bold, italic, dll)

Contoh response yang baik:
"Halo! Kopi Arabica Premium kami harganya Rp 50.000. Ini kopi pilihan dari petani lokal dengan rasa yang premium. Mau pesan berapa pak/bu?"
`

	return prompt
}

// Helper to pretty print webhook payload for debugging
func prettyPrint(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

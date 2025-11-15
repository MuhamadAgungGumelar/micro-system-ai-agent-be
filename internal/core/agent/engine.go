package agent

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
)

type Engine struct {
	waService       *whatsapp.Service
	llmClient       *llm.Client
	kbRetriever     *kb.Retriever
	tenantResolver  *tenant.Resolver
	conversationLog ConversationLogger
	lastMessageTime map[string]time.Time
	messageMutex    sync.Mutex
}

// ConversationLogger interface untuk log conversation (akan diimplementasi di module)
type ConversationLogger interface {
	LogConversation(clientID, customerPhone, message, response string) error
}

func NewEngine(
	waService *whatsapp.Service,
	llmClient *llm.Client,
	kbRetriever *kb.Retriever,
	tenantResolver *tenant.Resolver,
	conversationLog ConversationLogger,
) *Engine {
	return &Engine{
		waService:       waService,
		llmClient:       llmClient,
		kbRetriever:     kbRetriever,
		tenantResolver:  tenantResolver,
		conversationLog: conversationLog,
		lastMessageTime: make(map[string]time.Time),
	}
}

// HandleMessage adalah entry point utama untuk semua pesan masuk
func (e *Engine) HandleMessage(evt *events.Message) {
	// Extract info
	from := evt.Info.Sender.User // nomor pengirim
	text := evt.Message.GetConversation()

	if text == "" {
		return // Ignore non-text messages untuk saat ini
	}

	// Rate limiting per user
	e.messageMutex.Lock()
	lastTime, exists := e.lastMessageTime[from]
	if exists && time.Since(lastTime) < 2*time.Second {
		e.messageMutex.Unlock()
		log.Printf("⚠️ Rate limit: ignoring message from %s (too fast)", from)
		return
	}
	e.lastMessageTime[from] = time.Now()
	e.messageMutex.Unlock()

	// Resolve tenant context
	ctx, err := e.tenantResolver.ResolveFromPhone(from)
	if err != nil {
		log.Printf("❌ Failed to resolve tenant for %s: %v", from, err)
		e.waService.SendMessage(from, "Maaf, sistem sedang bermasalah.")
		return
	}

	log.Printf("📩 [%s|%s|%s] Message from %s: %s", ctx.Module, ctx.Role, ctx.CompanyID, from, text)

	// Route ke handler berdasarkan module
	switch ctx.Module {
	case "saas":
		e.handleSaaSMessage(ctx, from, text)
	case "farmasi":
		e.handleFarmasiMessage(ctx, from, text)
	case "umkm":
		e.handleUMKMMessage(ctx, from, text)
	default:
		e.handleSaaSMessage(ctx, from, text) // Default fallback
	}
}

// handleSaaSMessage untuk module SaaS (knowledge base answering)
func (e *Engine) handleSaaSMessage(ctx *tenant.TenantContext, from, text string) {
	// Get knowledge base
	kb, err := e.kbRetriever.GetKnowledgeBase(ctx.ClientID)
	if err != nil {
		log.Printf("❌ Failed to get KB for client %s: %v", ctx.ClientID, err)
		e.waService.SendMessage(from, "Maaf, sistem sedang bermasalah.")
		return
	}

	// Build system prompt
	systemPrompt := llm.BuildSystemPrompt(kb)

	// Generate response
	llmCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := e.llmClient.GenerateResponse(llmCtx, systemPrompt, text)
	if err != nil {
		log.Printf("❌ AI error: %v", err)
		reply = "Maaf, saya sedang tidak bisa menjawab saat ini."
	}

	// Send reply
	if err := e.waService.SendMessage(from, reply); err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return
	}

	// Log conversation (async)
	go func() {
		if e.conversationLog != nil {
			_ = e.conversationLog.LogConversation(ctx.ClientID, from, text, reply)
		}
	}()
}

// handleFarmasiMessage untuk module Farmasi (akan dikembangkan nanti)
func (e *Engine) handleFarmasiMessage(ctx *tenant.TenantContext, from, text string) {
	// TODO: Implement farmasi-specific logic
	// Untuk saat ini, fallback ke SaaS handler
	log.Printf("ℹ️ Farmasi module not yet implemented, using SaaS handler")
	e.handleSaaSMessage(ctx, from, text)
}

// handleUMKMMessage untuk module UMKM (akan dikembangkan nanti)
func (e *Engine) handleUMKMMessage(ctx *tenant.TenantContext, from, text string) {
	// TODO: Implement UMKM-specific logic
	log.Printf("ℹ️ UMKM module not yet implemented, using SaaS handler")
	e.handleSaaSMessage(ctx, from, text)
}

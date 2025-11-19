// internal/core/agent/engine.go
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

// HandleMessage adalah entry point untuk semua pesan masuk
// Menerima interface{} untuk support multi-provider (whatsmeow, greenapi, waha)
func (e *Engine) HandleMessage(evt interface{}) {
	// Extract phone number dan message text dari berbagai provider
	var from, text string

	switch v := evt.(type) {
	case *events.Message:
		// Whatsmeow native
		if v.Info.IsFromMe {
			return // Skip pesan dari diri sendiri
		}
		from = v.Info.Sender.User
		text = v.Message.GetConversation()

	case *whatsapp.GreenAPIMessage:
		// Green API
		from = v.From
		text = v.Message
		// Remove @c.us suffix jika ada
		if len(from) > 5 && from[len(from)-5:] == "@c.us" {
			from = from[:len(from)-5]
		}

	case *whatsapp.WAHAMessage:
		// WAHA
		from = v.From
		text = v.Message
		// Remove @c.us suffix jika ada
		if len(from) > 5 && from[len(from)-5:] == "@c.us" {
			from = from[:len(from)-5]
		}

	default:
		log.Printf("⚠️ Unknown message type: %T", evt)
		return
	}

	if text == "" {
		return
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
		e.handleSaaSMessage(ctx, from, text)
	}
}

func (e *Engine) handleSaaSMessage(ctx *tenant.TenantContext, from, text string) {
	kb, err := e.kbRetriever.GetKnowledgeBase(ctx.ClientID)
	if err != nil {
		log.Printf("❌ Failed to get KB for client %s: %v", ctx.ClientID, err)
		e.waService.SendMessage(from, "Maaf, sistem sedang bermasalah.")
		return
	}

	systemPrompt := llm.BuildSystemPrompt(kb)

	llmCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := e.llmClient.GenerateResponse(llmCtx, systemPrompt, text)
	if err != nil {
		log.Printf("❌ AI error: %v", err)
		reply = "Maaf, saya sedang tidak bisa menjawab saat ini."
	}

	if err := e.waService.SendMessage(from, reply); err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return
	}

	go func() {
		if e.conversationLog != nil {
			_ = e.conversationLog.LogConversation(ctx.ClientID, from, text, reply)
		}
	}()
}

func (e *Engine) handleFarmasiMessage(ctx *tenant.TenantContext, from, text string) {
	log.Printf("ℹ️ Farmasi module not yet implemented, using SaaS handler")
	e.handleSaaSMessage(ctx, from, text)
}

func (e *Engine) handleUMKMMessage(ctx *tenant.TenantContext, from, text string) {
	log.Printf("ℹ️ UMKM module not yet implemented, using SaaS handler")
	e.handleSaaSMessage(ctx, from, text)
}

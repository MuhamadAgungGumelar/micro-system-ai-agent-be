package services

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
)

// MessageService - DEPRECATED: Use core/agent/engine.go instead
// This is kept for backward compatibility only
type MessageService struct {
	waService   *whatsapp.Service
	llmClient   *llm.Client
	kbRetriever *kb.Retriever
	clientRepo  repositories.ClientRepo
	convRepo    repositories.ConversationRepo
	creditsRepo *repositories.CreditsRepo
}

var lastMessageTime = make(map[string]time.Time)
var messageMutex sync.Mutex

func NewMessageService(
	wa *whatsapp.Service,
	llmClient *llm.Client,
	kbRetriever *kb.Retriever,
	client repositories.ClientRepo,
	conv repositories.ConversationRepo,
	credits *repositories.CreditsRepo,
) *MessageService {
	return &MessageService{
		waService:   wa,
		llmClient:   llmClient,
		kbRetriever: kbRetriever,
		clientRepo:  client,
		convRepo:    conv,
		creditsRepo: credits,
	}
}

func (s *MessageService) HandleIncomingMessage(clientID string, evt *events.Message) {
	from := evt.Info.Sender.User
	text := evt.Message.GetConversation()
	if text == "" {
		return
	}

	// Rate limiting
	messageMutex.Lock()
	lastTime, exists := lastMessageTime[from]
	if exists && time.Since(lastTime) < 2*time.Second {
		messageMutex.Unlock()
		log.Printf("⚠️ Rate limit: ignoring message from %s (too fast)", from)
		return
	}
	lastMessageTime[from] = time.Now()
	messageMutex.Unlock()

	log.Printf("📩 [%s] message from %s: %s", clientID, from, text)

	// Get knowledge base (using core retriever)
	kb, err := s.kbRetriever.GetKnowledgeBase(clientID)
	if err != nil {
		log.Printf("failed to get KB for client %s: %v", clientID, err)
		s.waService.SendMessage(from, "Maaf, sistem sedang bermasalah.")
		return
	}

	// Build system prompt
	systemPrompt := llm.BuildSystemPrompt(kb)

	// Generate response
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := s.llmClient.GenerateResponse(ctx, systemPrompt, text)
	if err != nil {
		log.Printf("AI error: %v", err)
		reply = "Maaf, saya sedang tidak bisa menjawab saat ini."
	}

	// Send reply
	if err := s.waService.SendMessage(from, reply); err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}

	// Log conversation
	go func() {
		_ = s.convRepo.LogConversation(clientID, from, text, reply)
		if s.creditsRepo != nil {
			_ = s.creditsRepo.IncrementUsage(clientID)
		}
	}()
}

package services

import (
	"context"
	"log"
	"time"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/repositories"
	"go.mau.fi/whatsmeow/types/events"
)

type MessageService struct {
	waService   *WhatsAppService
	aiService   *AIService
	clientRepo  repositories.ClientRepo
	kbRepo      repositories.KBRepo
	convRepo    repositories.ConversationRepo
	creditsRepo *repositories.CreditsRepo
}

func NewMessageService(
	wa *WhatsAppService,
	ai *AIService,
	client repositories.ClientRepo,
	kb repositories.KBRepo,
	conv repositories.ConversationRepo,
	credits *repositories.CreditsRepo,
) *MessageService {
	return &MessageService{
		waService:   wa,
		aiService:   ai,
		clientRepo:  client,
		kbRepo:      kb,
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

	log.Printf("📩 [%s] message from %s: %s", clientID, from, text)

	// Get knowledge base
	kb, err := s.kbRepo.GetKnowledgeBase(clientID)
	if err != nil {
		log.Printf("failed to get KB for client %s: %v", clientID, err)
		s.waService.SendMessage(from, "Maaf, sistem sedang bermasalah.")
		return
	}

	// Generate response
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := s.aiService.GenerateResponse(ctx, text, kb)
	if err != nil {
		log.Printf("AI error: %v", err)
		reply = "Maaf, saya sedang tidak bisa menjawab saat ini."
	}

	if err := s.waService.SendMessage(from, reply); err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}

	// Log conversation
	go func() {
		_ = s.convRepo.LogConversation(clientID, from, text, reply)
		_ = s.creditsRepo.IncrementUsage(clientID)

	}()
}

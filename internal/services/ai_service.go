package services

import (
	"context"
	"fmt"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
}

func NewAIService(apiKey string) *AIService {
	return &AIService{
		client: openai.NewClient(apiKey),
	}
}

func (s *AIService) GenerateResponse(ctx context.Context, userMessage string, kb *models.KnowledgeBase) (string, error) {
	systemPrompt := s.buildSystemPrompt(kb)
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMessage},
		},
		Temperature: 0.6,
		MaxTokens:   300,
	})
	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choice")
	}
	return resp.Choices[0].Message.Content, nil
}

func (s *AIService) buildSystemPrompt(kb *models.KnowledgeBase) string {
	// same as your builder...
	// build prompt using kb.FAQs and kb.Products
	return "..." // keep your implementation
}

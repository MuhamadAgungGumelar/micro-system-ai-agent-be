package llm

import (
	"context"
	"log"
)

// Service wraps LLM provider untuk dependency injection
type Service struct {
	provider LLMProvider
}

// NewService creates LLM service with provider from environment
func NewService() *Service {
	cfg, err := LoadProviderFromEnv()
	if err != nil {
		log.Fatalf("❌ Failed to load LLM config: %v", err)
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create LLM provider: %v", err)
	}

	log.Printf("🤖 Using LLM provider: %s (model: %s)", provider.GetProviderName(), cfg.Model)

	return &Service{provider: provider}
}

// NewServiceWithProvider creates service with custom provider (for testing)
func NewServiceWithProvider(provider LLMProvider) *Service {
	return &Service{provider: provider}
}

// GenerateResponse generates AI response
func (s *Service) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return s.provider.GenerateResponse(ctx, systemPrompt, userMessage)
}

// GetProviderName returns current provider name
func (s *Service) GetProviderName() string {
	return s.provider.GetProviderName()
}

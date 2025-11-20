package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type GroqProvider struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

func NewGroqProvider(apiKey string, model string, temperature float32, maxTokens int) *GroqProvider {
	if model == "" {
		model = "llama-3.1-8b-instant"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxTokens == 0 {
		maxTokens = 2048
	}

	// Groq uses OpenAI-compatible API with custom base URL
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"

	return &GroqProvider{
		client:      openai.NewClientWithConfig(config),
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

func (p *GroqProvider) GetProviderName() string {
	return "Groq"
}

func (p *GroqProvider) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMessage},
		},
		Temperature: p.temperature,
		MaxTokens:   p.maxTokens,
	})

	if err != nil {
		return "", fmt.Errorf("groq error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return resp.Choices[0].Message.Content, nil
}

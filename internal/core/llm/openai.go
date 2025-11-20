package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

func NewOpenAIProvider(apiKey string, model string, temperature float32, maxTokens int) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxTokens == 0 {
		maxTokens = 300
	}

	return &OpenAIProvider{
		client:      openai.NewClient(apiKey),
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

func (p *OpenAIProvider) GetProviderName() string {
	return "OpenAI"
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
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
		return "", fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

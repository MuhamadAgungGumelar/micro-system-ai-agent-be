package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type DeepSeekProvider struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

func NewDeepSeekProvider(apiKey string, model string, temperature float32, maxTokens int) *DeepSeekProvider {
	if model == "" {
		model = "deepseek-chat"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxTokens == 0 {
		maxTokens = 2048
	}

	// DeepSeek uses OpenAI-compatible API with custom base URL
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.deepseek.com"
	config.HTTPClient = &http.Client{
		Timeout: 60 * time.Second, // Increase timeout to 60 seconds
	}

	return &DeepSeekProvider{
		client:      openai.NewClientWithConfig(config),
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

func (p *DeepSeekProvider) GetProviderName() string {
	return "DeepSeek"
}

func (p *DeepSeekProvider) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
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
		return "", fmt.Errorf("deepseek error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	return resp.Choices[0].Message.Content, nil
}

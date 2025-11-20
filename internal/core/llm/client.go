package llm

import (
	"context"
	"fmt"
	"log"

	openai "github.com/sashabaranov/go-openai"
)

// Client wraps LLM provider (backward compatible)
// DEPRECATED: Use Service instead for multi-provider support
type Client struct {
	provider LLMProvider
	client   *openai.Client // Keep for GenerateResponseWithFunctions
}

// NewClient creates a client with OpenAI provider (backward compatible)
// DEPRECATED: Use NewService() instead
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		log.Println("⚠️ OPENAI_API_KEY is empty, LLM will not work")
	}

	return &Client{
		provider: NewOpenAIProvider(apiKey, "gpt-4o-mini", 0.7, 300),
		client:   openai.NewClient(apiKey),
	}
}

// GenerateResponse generates AI response (uses provider pattern now)
func (c *Client) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return c.provider.GenerateResponse(ctx, systemPrompt, userMessage)
}

// GenerateResponseWithFunctions untuk AI Actions (nanti)
func (c *Client) GenerateResponseWithFunctions(ctx context.Context, systemPrompt, userMessage string, functions []openai.FunctionDefinition) (*openai.ChatCompletionResponse, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMessage},
		},
		Functions:   functions,
		Temperature: 0.6,
		MaxTokens:   500,
	})

	if err != nil {
		return nil, fmt.Errorf("openai error: %w", err)
	}

	return &resp, nil
}

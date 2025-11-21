package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ClaudeProvider struct {
	apiKey      string
	model       string
	temperature float32
	maxTokens   int
	client      *http.Client
}

func NewClaudeProvider(apiKey string, model string, temperature float32, maxTokens int) *ClaudeProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxTokens == 0 {
		maxTokens = 2048
	}

	return &ClaudeProvider{
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		client: &http.Client{
			Timeout: 60 * time.Second, // Increase timeout to 60 seconds
		},
	}
}

func (p *ClaudeProvider) GetProviderName() string {
	return "Anthropic Claude"
}

// Claude API request/response structures
type claudeRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float32          `json:"temperature"`
	Messages    []claudeMessage  `json:"messages"`
	System      string           `json:"system,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (p *ClaudeProvider) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	reqBody := claudeRequest{
		Model:       p.model,
		MaxTokens:   p.maxTokens,
		Temperature: p.temperature,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	// Add system prompt if provided
	if systemPrompt != "" {
		reqBody.System = systemPrompt
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude error (model: %s, status: %d): %s", p.model, resp.StatusCode, string(body))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	return claudeResp.Content[0].Text, nil
}

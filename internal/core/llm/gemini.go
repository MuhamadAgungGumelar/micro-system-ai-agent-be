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

type GeminiProvider struct {
	apiKey      string
	model       string
	temperature float32
	maxTokens   int
	client      *http.Client
}

func NewGeminiProvider(apiKey string, model string, temperature float32, maxTokens int) *GeminiProvider {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxTokens == 0 {
		maxTokens = 8192 // Increase for receipt parsing (needs more tokens for JSON output)
	}

	return &GeminiProvider{
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		client: &http.Client{
			Timeout: 60 * time.Second, // Increase timeout to 60 seconds
		},
	}
}

func (p *GeminiProvider) GetProviderName() string {
	return "Google Gemini"
}

// Gemini REST API request/response structures
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float32 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (p *GeminiProvider) GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// Use REST API v1 endpoint (not v1beta)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s",
		p.model, p.apiKey)

	// Build contents - combine system prompt with user message
	var contents []geminiContent

	// For Gemini v1 API, system instruction should be part of the first user message
	userText := userMessage
	if systemPrompt != "" {
		userText = systemPrompt + "\n\n" + userMessage
	}

	contents = append(contents, geminiContent{
		Parts: []geminiPart{{Text: userText}},
		Role:  "user",
	})

	reqBody := geminiRequest{
		Contents: contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     p.temperature,
			MaxOutputTokens: p.maxTokens,
		},
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

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini error (model: %s, status: %d): %s", p.model, resp.StatusCode, string(body))
	}

	// Log raw response for debugging
	fmt.Printf("🔍 Gemini raw response: %s\n", string(body))

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Log parsed response structure
	fmt.Printf("🔍 Gemini candidates count: %d\n", len(geminiResp.Candidates))
	if len(geminiResp.Candidates) > 0 {
		fmt.Printf("🔍 Gemini parts count: %d\n", len(geminiResp.Candidates[0].Content.Parts))
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini (candidates: %d)", len(geminiResp.Candidates))
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

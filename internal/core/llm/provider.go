package llm

import (
	"context"
	"fmt"
	"os"
)

// LLMProvider interface untuk multiple AI providers
type LLMProvider interface {
	GenerateResponse(ctx context.Context, systemPrompt, userMessage string) (string, error)
	GetProviderName() string
}

// ProviderType untuk factory
type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderGemini   ProviderType = "gemini"
	ProviderGroq     ProviderType = "groq"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderClaude   ProviderType = "claude"
)

// ProviderConfig untuk create provider
type ProviderConfig struct {
	Type ProviderType

	// API Keys
	OpenAIKey   string
	GeminiKey   string
	GroqKey     string
	DeepSeekKey string
	ClaudeKey   string

	// Model configs
	Model       string
	Temperature float32
	MaxTokens   int
}

// NewProvider factory untuk create LLM provider
func NewProvider(cfg *ProviderConfig) (LLMProvider, error) {
	switch cfg.Type {
	case ProviderOpenAI:
		if cfg.OpenAIKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is required")
		}
		return NewOpenAIProvider(cfg.OpenAIKey, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil

	case ProviderGemini:
		if cfg.GeminiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required")
		}
		return NewGeminiProvider(cfg.GeminiKey, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil

	case ProviderGroq:
		if cfg.GroqKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY is required")
		}
		return NewGroqProvider(cfg.GroqKey, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil

	case ProviderDeepSeek:
		if cfg.DeepSeekKey == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY is required")
		}
		return NewDeepSeekProvider(cfg.DeepSeekKey, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil

	case ProviderClaude:
		if cfg.ClaudeKey == "" {
			return nil, fmt.Errorf("CLAUDE_API_KEY is required")
		}
		return NewClaudeProvider(cfg.ClaudeKey, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil

	default:
		return nil, fmt.Errorf("unknown LLM provider type: %s", cfg.Type)
	}
}

// LoadProviderFromEnv load config dari environment variables
func LoadProviderFromEnv() (*ProviderConfig, error) {
	providerType := os.Getenv("LLM_PROVIDER")
	if providerType == "" {
		providerType = "openai" // default
	}

	cfg := &ProviderConfig{
		Type:        ProviderType(providerType),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		GeminiKey:   os.Getenv("GEMINI_API_KEY"),
		GroqKey:     os.Getenv("GROQ_API_KEY"),
		DeepSeekKey: os.Getenv("DEEPSEEK_API_KEY"),
		ClaudeKey:   os.Getenv("CLAUDE_API_KEY"),
	}

	// Set defaults for model configs
	if model := os.Getenv("LLM_MODEL"); model != "" {
		cfg.Model = model
	} else {
		// Provider-specific defaults
		switch cfg.Type {
		case ProviderOpenAI:
			cfg.Model = "gpt-4o-mini"
		case ProviderGemini:
			cfg.Model = "gemini-2.5-flash"
		case ProviderGroq:
			cfg.Model = "llama-3.1-70b-versatile"
		case ProviderDeepSeek:
			cfg.Model = "deepseek-chat"
		case ProviderClaude:
			cfg.Model = "claude-3-5-sonnet-20241022"
		}
	}

	// Temperature
	cfg.Temperature = 0.7
	cfg.MaxTokens = 1024

	return cfg, nil
}

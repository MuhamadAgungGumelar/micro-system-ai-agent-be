package vector

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingProvider defines the interface for text embedding generation
type EmbeddingProvider interface {
	// GenerateEmbedding generates an embedding vector for a single text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateBatchEmbeddings generates embeddings for multiple texts
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimensions returns the dimension size of the embeddings
	GetDimensions() int

	// GetProviderName returns the provider name
	GetProviderName() string
}

// OpenAIEmbeddingProvider implements EmbeddingProvider using OpenAI
type OpenAIEmbeddingProvider struct {
	client *openai.Client
	model  string
	dims   int
}

// NewOpenAIEmbeddingProvider creates a new OpenAI embedding provider
// Default model: text-embedding-3-small (1536 dimensions)
func NewOpenAIEmbeddingProvider(apiKey string, model string) (*OpenAIEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if model == "" {
		model = "text-embedding-3-small" // Default model
	}

	// Determine dimensions based on model
	dims := 1536 // Default for text-embedding-3-small
	switch model {
	case "text-embedding-3-small":
		dims = 1536
	case "text-embedding-3-large":
		dims = 3072
	case "text-embedding-ada-002":
		dims = 1536
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbeddingProvider{
		client: client,
		model:  model,
		dims:   dims,
	}, nil
}

// GenerateEmbedding generates an embedding for a single text
func (p *OpenAIEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(p.model),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func (p *OpenAIEmbeddingProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(p.model),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// GetDimensions returns the dimension size
func (p *OpenAIEmbeddingProvider) GetDimensions() int {
	return p.dims
}

// GetProviderName returns the provider name
func (p *OpenAIEmbeddingProvider) GetProviderName() string {
	return fmt.Sprintf("openai_%s", p.model)
}

// GeminiEmbeddingProvider implements EmbeddingProvider using Google Gemini
// Note: This is a placeholder. Implement when Gemini embeddings API is available
type GeminiEmbeddingProvider struct {
	apiKey string
	model  string
	dims   int
}

// NewGeminiEmbeddingProvider creates a new Gemini embedding provider
func NewGeminiEmbeddingProvider(apiKey string) (*GeminiEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	return &GeminiEmbeddingProvider{
		apiKey: apiKey,
		model:  "text-embedding-004", // Placeholder model
		dims:   768,                   // Placeholder dimensions
	}, nil
}

// GenerateEmbedding generates an embedding (placeholder)
func (p *GeminiEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// TODO: Implement when Gemini embeddings API is available
	return nil, fmt.Errorf("Gemini embeddings not yet implemented")
}

// GenerateBatchEmbeddings generates batch embeddings (placeholder)
func (p *GeminiEmbeddingProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO: Implement when Gemini embeddings API is available
	return nil, fmt.Errorf("Gemini embeddings not yet implemented")
}

// GetDimensions returns the dimension size
func (p *GeminiEmbeddingProvider) GetDimensions() int {
	return p.dims
}

// GetProviderName returns the provider name
func (p *GeminiEmbeddingProvider) GetProviderName() string {
	return "gemini_embeddings"
}

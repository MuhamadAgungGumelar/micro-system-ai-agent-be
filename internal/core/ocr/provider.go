package ocr

import "context"

// Provider interface for OCR services
type Provider interface {
	// ExtractText extracts text from image
	ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// OCRResult contains the extracted text and metadata
type OCRResult struct {
	Text       string  `json:"text"`       // Raw extracted text
	Confidence float64 `json:"confidence"` // OCR confidence score (0-1)
}

// Service wraps the OCR provider
type Service struct {
	provider Provider
}

// NewService creates a new OCR service with the given provider
func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

// ExtractText extracts text from image using the configured provider
func (s *Service) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	return s.provider.ExtractText(ctx, imageData)
}

// GetProviderName returns the name of the current provider
func (s *Service) GetProviderName() string {
	return s.provider.GetProviderName()
}

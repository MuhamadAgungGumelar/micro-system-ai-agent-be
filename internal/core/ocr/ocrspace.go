package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// OCRSpaceProvider implements OCR using OCR.space API
type OCRSpaceProvider struct {
	apiKey string
	client *http.Client
}

// NewOCRSpaceProvider creates a new OCR.space provider
func NewOCRSpaceProvider(apiKey string) *OCRSpaceProvider {
	return &OCRSpaceProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetProviderName returns the provider name
func (p *OCRSpaceProvider) GetProviderName() string {
	return "OCR.space"
}

// OCR.space API response structure
type ocrSpaceResponse struct {
	ParsedResults []struct {
		TextOverlay struct {
			Lines []interface{} `json:"Lines"`
		} `json:"TextOverlay"`
		ParsedText      string  `json:"ParsedText"`
		FileParseExitCode int   `json:"FileParseExitCode"`
	} `json:"ParsedResults"`
	OCRExitCode       int     `json:"OCRExitCode"`
	IsErroredOnProcessing bool `json:"IsErroredOnProcessing"`
	ErrorMessage      []string `json:"ErrorMessage,omitempty"`
	ProcessingTimeInMilliseconds string `json:"ProcessingTimeInMilliseconds"`
}

// ExtractText extracts text from image using OCR.space API
func (p *OCRSpaceProvider) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add image file
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("failed to write image data: %w", err)
	}

	// Add API key
	if err := writer.WriteField("apikey", p.apiKey); err != nil {
		return nil, fmt.Errorf("failed to write api key: %w", err)
	}

	// Add language (Indonesian + English)
	if err := writer.WriteField("language", "eng"); err != nil {
		return nil, fmt.Errorf("failed to write language: %w", err)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	url := "https://api.ocr.space/parse/image"
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocrspace request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ocrspace error (status: %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ocrResp ocrSpaceResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if ocrResp.IsErroredOnProcessing {
		errMsg := "unknown error"
		if len(ocrResp.ErrorMessage) > 0 {
			errMsg = ocrResp.ErrorMessage[0]
		}
		return nil, fmt.Errorf("ocrspace processing error: %s", errMsg)
	}

	if ocrResp.OCRExitCode != 1 {
		return nil, fmt.Errorf("ocrspace exit code: %d", ocrResp.OCRExitCode)
	}

	// Extract text
	if len(ocrResp.ParsedResults) == 0 {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
		}, nil
	}

	text := ocrResp.ParsedResults[0].ParsedText

	// OCR.space doesn't provide confidence score, use default
	confidence := 0.85

	return &OCRResult{
		Text:       text,
		Confidence: confidence,
	}, nil
}

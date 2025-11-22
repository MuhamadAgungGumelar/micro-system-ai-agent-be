package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GoogleVisionProvider implements OCR using Google Cloud Vision API
type GoogleVisionProvider struct {
	apiKey string
	client *http.Client
}

// NewGoogleVisionProvider creates a new Google Vision OCR provider
func NewGoogleVisionProvider(apiKey string) *GoogleVisionProvider {
	return &GoogleVisionProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetProviderName returns the provider name
func (p *GoogleVisionProvider) GetProviderName() string {
	return "Google Cloud Vision"
}

// Google Vision API request/response structures
type visionRequest struct {
	Requests []visionRequestItem `json:"requests"`
}

type visionRequestItem struct {
	Image    visionImage    `json:"image"`
	Features []visionFeature `json:"features"`
}

type visionImage struct {
	Content string `json:"content"` // base64 encoded image
}

type visionFeature struct {
	Type       string `json:"type"`
	MaxResults int    `json:"maxResults,omitempty"`
}

type visionResponse struct {
	Responses []struct {
		TextAnnotations []struct {
			Description string  `json:"description"`
			Score       float64 `json:"score,omitempty"`
		} `json:"textAnnotations"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"responses"`
}

// ExtractText extracts text from image using Google Cloud Vision API
func (p *GoogleVisionProvider) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Prepare request
	reqBody := visionRequest{
		Requests: []visionRequestItem{
			{
				Image: visionImage{
					Content: base64Image,
				},
				Features: []visionFeature{
					{
						Type:       "TEXT_DETECTION",
						MaxResults: 1,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Google Vision API
	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google vision request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google vision error (status: %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var visionResp visionResponse
	if err := json.Unmarshal(body, &visionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(visionResp.Responses) == 0 {
		return nil, fmt.Errorf("no response from Google Vision")
	}

	// Check for API errors
	if visionResp.Responses[0].Error != nil {
		return nil, fmt.Errorf("google vision API error: %s", visionResp.Responses[0].Error.Message)
	}

	// Extract text from annotations
	if len(visionResp.Responses[0].TextAnnotations) == 0 {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
		}, nil
	}

	// First annotation contains the full text
	fullText := visionResp.Responses[0].TextAnnotations[0].Description
	confidence := visionResp.Responses[0].TextAnnotations[0].Score
	if confidence == 0 {
		confidence = 0.95 // Default confidence if not provided
	}

	return &OCRResult{
		Text:       fullText,
		Confidence: confidence,
	}, nil
}

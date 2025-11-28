// internal/core/whatsapp/cloud_api.go
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// CloudAPIProvider implements WhatsApp Cloud API (Official Business API)
// Documentation: https://developers.facebook.com/docs/whatsapp/cloud-api
type CloudAPIProvider struct {
	baseURL     string
	phoneID     string // WhatsApp Business Phone Number ID
	accessToken string // Meta Business Access Token
	apiVersion  string // API version (e.g., "v18.0")
	webhookURL  string // Webhook URL for receiving messages
	client      *http.Client
}

// CloudAPIConfig holds configuration for WhatsApp Cloud API
type CloudAPIConfig struct {
	PhoneID     string `json:"phone_id"`      // Your WhatsApp Business Phone Number ID
	AccessToken string `json:"access_token"`  // Meta Business Access Token
	APIVersion  string `json:"api_version"`   // API version (default: v18.0)
	WebhookURL  string `json:"webhook_url"`   // Your webhook URL
}

// CloudAPIMessage represents incoming message from webhook
type CloudAPIMessage struct {
	From      string                 `json:"from"`
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"` // text, image, document, etc.
	Text      *CloudAPITextMessage   `json:"text,omitempty"`
	Image     *CloudAPIMediaMessage  `json:"image,omitempty"`
	Document  *CloudAPIMediaMessage  `json:"document,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type CloudAPITextMessage struct {
	Body string `json:"body"`
}

type CloudAPIMediaMessage struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
}

// NewCloudAPIProvider creates a new WhatsApp Cloud API provider
func NewCloudAPIProvider(config CloudAPIConfig) (*CloudAPIProvider, error) {
	if config.PhoneID == "" {
		return nil, fmt.Errorf("phone_id is required")
	}
	if config.AccessToken == "" {
		return nil, fmt.Errorf("access_token is required")
	}

	// Default API version
	if config.APIVersion == "" {
		config.APIVersion = "v18.0"
	}

	baseURL := fmt.Sprintf("https://graph.facebook.com/%s/%s", config.APIVersion, config.PhoneID)

	return &CloudAPIProvider{
		baseURL:     baseURL,
		phoneID:     config.PhoneID,
		accessToken: config.AccessToken,
		apiVersion:  config.APIVersion,
		webhookURL:  config.WebhookURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Connect is a no-op for Cloud API (always connected via HTTP)
func (p *CloudAPIProvider) Connect() error {
	log.Printf("✅ WhatsApp Cloud API initialized (Phone ID: %s)", p.phoneID)
	return nil
}

// Disconnect is a no-op for Cloud API
func (p *CloudAPIProvider) Disconnect() {
	log.Printf("👋 WhatsApp Cloud API disconnected")
}

// SendMessage sends a text message via Cloud API
func (p *CloudAPIProvider) SendMessage(to, message string) error {
	// Remove @ suffix if present (Cloud API uses plain phone numbers)
	to = cleanPhoneNumber(to)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"preview_url": "false",
			"body":        message,
		},
	}

	return p.sendRequest("POST", "/messages", payload)
}

// SendMedia sends media (image, document, etc.) via Cloud API
func (p *CloudAPIProvider) SendMedia(to, mediaType, mediaID, caption string) error {
	to = cleanPhoneNumber(to)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                to,
		"type":              mediaType,
		mediaType: map[string]string{
			"id":      mediaID,
			"caption": caption,
		},
	}

	return p.sendRequest("POST", "/messages", payload)
}

// StartTyping sends typing indicator (Cloud API uses "composing" presence)
func (p *CloudAPIProvider) StartTyping(phoneNumber string) error {
	// Cloud API doesn't support typing indicators in the same way
	// This is a no-op, but we keep it for interface compatibility
	log.Printf("ℹ️ Typing indicator not supported in Cloud API")
	return nil
}

// StopTyping stops typing indicator
func (p *CloudAPIProvider) StopTyping(phoneNumber string) error {
	// No-op
	return nil
}

// StartListening is not used for Cloud API (webhook-based)
func (p *CloudAPIProvider) StartListening(handler func(evt interface{})) error {
	return fmt.Errorf("Cloud API uses webhooks, not polling. Configure webhook at: %s", p.webhookURL)
}

// GenerateQR is not applicable for Cloud API (uses phone number verification)
func (p *CloudAPIProvider) GenerateQR(sessionID string) ([]byte, error) {
	return nil, fmt.Errorf("Cloud API doesn't use QR codes. Use phone number verification instead")
}

// StartSession is not applicable for Cloud API
func (p *CloudAPIProvider) StartSession(sessionID string) error {
	return fmt.Errorf("Cloud API doesn't use sessions. It's always connected")
}

// GetSessionStatus returns true (Cloud API is always "connected")
func (p *CloudAPIProvider) GetSessionStatus(sessionID string) (bool, error) {
	return true, nil
}

// IsConnected always returns true for Cloud API
func (p *CloudAPIProvider) IsConnected() bool {
	return true
}

// StartKeepAlive is a no-op for Cloud API
func (p *CloudAPIProvider) StartKeepAlive(ctx context.Context) {
	// No-op - Cloud API doesn't need keep-alive
}

// GetProviderName returns the provider name
func (p *CloudAPIProvider) GetProviderName() string {
	return "WhatsApp Cloud API (Official)"
}

// MarkMessageAsRead marks a message as read
func (p *CloudAPIProvider) MarkMessageAsRead(messageID string) error {
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	return p.sendRequest("POST", "/messages", payload)
}

// GetMediaURL retrieves the URL for a media file
func (p *CloudAPIProvider) GetMediaURL(mediaID string) (string, error) {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s", p.apiVersion, mediaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get media info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get media URL: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.URL, nil
}

// DownloadMedia downloads media from Cloud API
func (p *CloudAPIProvider) DownloadMedia(mediaID string) ([]byte, error) {
	// Step 1: Get media URL
	mediaURL, err := p.GetMediaURL(mediaID)
	if err != nil {
		return nil, err
	}

	// Step 2: Download from URL
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download media: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// sendRequest is a helper to make API requests
func (p *CloudAPIProvider) sendRequest(method, endpoint string, payload interface{}) error {
	url := p.baseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("✅ Cloud API request successful: %s %s", method, endpoint)
	return nil
}

// cleanPhoneNumber removes WhatsApp JID suffix (@c.us)
func cleanPhoneNumber(phone string) string {
	if len(phone) > 5 && phone[len(phone)-5:] == "@c.us" {
		return phone[:len(phone)-5]
	}
	return phone
}

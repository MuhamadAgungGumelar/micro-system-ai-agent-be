// internal/core/whatsapp/greenapi.go
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

type GreenAPIProvider struct {
	instanceID  string
	token       string
	baseURL     string
	client      *http.Client
	connected   bool
	stopPolling chan bool
}

func NewGreenAPIProvider(instanceID, token, baseURL string) *GreenAPIProvider {
	return &GreenAPIProvider{
		instanceID: instanceID,
		token:      token,
		baseURL:    baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopPolling: make(chan bool),
	}
}

func (g *GreenAPIProvider) GetProviderName() string {
	return "GreenAPI"
}

func (g *GreenAPIProvider) Connect() error {
	// Check instance status
	endpoint := fmt.Sprintf("%s/waInstance%s/getStateInstance/%s", g.baseURL, g.instanceID, g.token)

	resp, err := g.client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to Green API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Green API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		StateInstance string `json:"stateInstance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.StateInstance != "authorized" {
		log.Printf("⚠️ Green API instance not authorized (state: %s)", result.StateInstance)
		log.Println("💡 Please scan QR code via Green API dashboard or use /whatsapp/qr endpoint")
	} else {
		log.Println("✅ Green API instance authorized")
	}

	g.connected = true
	return nil
}

func (g *GreenAPIProvider) Disconnect() {
	g.connected = false
	close(g.stopPolling)
	log.Println("🔌 Green API provider disconnected")
}

func (g *GreenAPIProvider) SendMessage(phoneNumber, message string) error {
	// Format nomor: 628123456789@c.us
	chatID := phoneNumber
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		chatID = phoneNumber[1:] + "@c.us"
	} else {
		chatID = phoneNumber + "@c.us"
	}

	endpoint := fmt.Sprintf("%s/waInstance%s/sendMessage/%s", g.baseURL, g.instanceID, g.token)

	payload := map[string]interface{}{
		"chatId":  chatID,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := g.client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Green API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (g *GreenAPIProvider) StartListening(handler func(evt interface{})) error {
	// Green API menggunakan webhook atau polling
	// Untuk simplicity, kita gunakan polling dengan receiveNotification

	log.Println("👂 Starting Green API message polling...")

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-g.stopPolling:
				log.Println("🛑 Stopped Green API polling")
				return
			case <-ticker.C:
				if g.connected {
					g.pollMessages(handler)
				}
			}
		}
	}()

	return nil
}

func (g *GreenAPIProvider) pollMessages(handler func(evt interface{})) {
	endpoint := fmt.Sprintf("%s/waInstance%s/receiveNotification/%s", g.baseURL, g.instanceID, g.token)

	resp, err := g.client.Get(endpoint)
	if err != nil {
		log.Printf("⚠️ Failed to poll messages: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var notification struct {
		ReceiptID int `json:"receiptId"`
		Body      struct {
			TypeWebhook  string      `json:"typeWebhook"`
			InstanceData interface{} `json:"instanceData"`
			Timestamp    int64       `json:"timestamp"`
			IDMessage    string      `json:"idMessage"`
			SenderData   struct {
				ChatID string `json:"chatId"`
				Sender string `json:"sender"`
			} `json:"senderData"`
			MessageData struct {
				TypeMessage     string `json:"typeMessage"`
				TextMessageData struct {
					TextMessage string `json:"textMessage"`
				} `json:"textMessageData"`
			} `json:"messageData"`
		} `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&notification); err != nil {
		return
	}

	// Hanya process incoming messages
	if notification.Body.TypeWebhook == "incomingMessageReceived" &&
		notification.Body.MessageData.TypeMessage == "textMessage" {

		// Extract phone number dari sender (format: 628xxx@c.us)
		sender := notification.Body.SenderData.Sender

		// Convert ke format yang compatible dengan existing handler
		evt := &GreenAPIMessage{
			From:    sender,
			Message: notification.Body.MessageData.TextMessageData.TextMessage,
		}

		handler(evt)

		// Delete notification after processing
		deleteEndpoint := fmt.Sprintf("%s/waInstance%s/deleteNotification/%s/%d",
			g.baseURL, g.instanceID, g.token, notification.ReceiptID)
		_, _ = g.client.Get(deleteEndpoint)
	}
}

func (g *GreenAPIProvider) GenerateQR(sessionID string) ([]byte, error) {
	// Green API doesn't support multiple sessions per instance
	// sessionID is ignored - each instance is tied to one WhatsApp account
	log.Printf("⚠️ Green API doesn't support dynamic sessions. Using instance: %s", g.instanceID)

	// Try to get QR code from Green API
	// Note: This endpoint might not exist in all Green API versions
	// Alternative: Scan QR via Green API Dashboard at https://console.green-api.com/
	endpoint := fmt.Sprintf("%s/waInstance%s/qr/%s", g.baseURL, g.instanceID, g.token)

	resp, err := g.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// If QR endpoint doesn't exist, return helpful message
	if resp.StatusCode == http.StatusNotFound {
		errorMsg := map[string]string{
			"error":   "QR code endpoint not available for Green API",
			"message": "Please scan QR code via Green API Dashboard: https://console.green-api.com/",
			"instance_id": g.instanceID,
		}
		return json.Marshal(errorMsg)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Green API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse as JSON first
	var result struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err == nil {
		// Green API returns base64 or URL to QR
		// Return as JSON with instructions
		response := map[string]string{
			"type":    result.Type,
			"message": result.Message,
			"note":    "Green API QR. Please scan via dashboard: https://console.green-api.com/",
		}
		return json.Marshal(response)
	}

	// If not JSON, might be image bytes directly
	return body, nil
}

// StartSession creates/starts a new session (Green API uses instance concept)
func (g *GreenAPIProvider) StartSession(sessionID string) error {
	// Green API doesn't support session creation - each instance is pre-configured
	// Just verify the instance is available
	log.Printf("ℹ️ Green API uses instance model. SessionID '%s' ignored, using instance: %s", sessionID, g.instanceID)
	return g.Connect()
}

// GetSessionStatus checks if a session is connected
func (g *GreenAPIProvider) GetSessionStatus(sessionID string) (bool, error) {
	// Green API checks instance status
	log.Printf("ℹ️ Checking Green API instance status (sessionID '%s' ignored)", sessionID)

	endpoint := fmt.Sprintf("%s/waInstance%s/getStateInstance/%s", g.baseURL, g.instanceID, g.token)

	resp, err := g.client.Get(endpoint)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		StateInstance string `json:"stateInstance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.StateInstance == "authorized", nil
}

func (g *GreenAPIProvider) IsConnected() bool {
	return g.connected
}

func (g *GreenAPIProvider) StartKeepAlive(ctx context.Context) {
	// Green API tidak butuh keep-alive manual
	log.Println("ℹ️ Green API doesn't require manual keep-alive")
}

// GreenAPIMessage adapter untuk compatibility dengan existing code
type GreenAPIMessage struct {
	From    string
	Message string
}

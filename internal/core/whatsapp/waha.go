// internal/core/whatsapp/waha.go
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

type WAHAProvider struct {
	baseURL      string
	apiKey       string
	sessionID    string
	client       *http.Client
	connected    bool
	stopPolling  chan bool
	processedIDs map[string]bool
}

func NewWAHAProvider(baseURL, apiKey, sessionID string) *WAHAProvider {
	return &WAHAProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		sessionID: sessionID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopPolling:  make(chan bool),
		processedIDs: make(map[string]bool),
	}
}

func (w *WAHAProvider) GetProviderName() string {
	return "WAHA"
}

func (w *WAHAProvider) Connect() error {
	// Start session
	endpoint := fmt.Sprintf("%s/api/sessions/%s/start", w.baseURL, w.sessionID)

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start WAHA session: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 200 = success, 201 = created, 409 = already exists, 422 = already started
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		log.Printf("✅ WAHA session '%s' started", w.sessionID)
	} else if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusUnprocessableEntity {
		log.Printf("ℹ️ WAHA session '%s' already exists/started", w.sessionID)
	} else {
		return fmt.Errorf("WAHA returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check status
	time.Sleep(2 * time.Second)
	status, err := w.getSessionStatus()
	if err != nil {
		log.Printf("⚠️ Failed to get session status: %v", err)
	} else {
		log.Printf("📱 WAHA session status: %s", status)

		if status == "SCAN_QR_CODE" || status == "STARTING" {
			log.Println("💡 Please scan QR code via /whatsapp/qr endpoint")
		}
	}

	w.connected = true
	return nil
}

func (w *WAHAProvider) getSessionStatus() (string, error) {
	endpoint := fmt.Sprintf("%s/api/sessions/%s", w.baseURL, w.sessionID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Status, nil
}

func (w *WAHAProvider) Disconnect() {
	w.connected = false
	close(w.stopPolling)

	endpoint := fmt.Sprintf("%s/api/sessions/%s/stop", w.baseURL, w.sessionID)

	req, _ := http.NewRequest("POST", endpoint, nil)
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	_, _ = w.client.Do(req)
	log.Println("🔌 WAHA provider disconnected")
}

func (w *WAHAProvider) SendMessage(phoneNumber, message string) error {
	// Format: 628123456789@c.us
	chatID := phoneNumber
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		chatID = phoneNumber[1:] + "@c.us"
	} else {
		chatID = phoneNumber + "@c.us"
	}

	endpoint := fmt.Sprintf("%s/api/sendText", w.baseURL)

	payload := map[string]interface{}{
		"session": w.sessionID,
		"chatId":  chatID,
		"text":    message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WAHA returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *WAHAProvider) StartListening(handler func(evt interface{})) error {
	log.Println("👂 Starting WAHA message polling...")
	log.Println("💡 For production, configure WAHA webhook to your /webhook endpoint")

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopPolling:
				log.Println("🛑 Stopped WAHA polling")
				return
			case <-ticker.C:
				if w.connected {
					w.pollMessages(handler)
				}
			}
		}
	}()

	return nil
}

func (w *WAHAProvider) pollMessages(handler func(evt interface{})) {
	endpoint := fmt.Sprintf("%s/api/%s/messages", w.baseURL, w.sessionID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return
	}

	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	// Add query params untuk filter
	q := req.URL.Query()
	q.Add("limit", "10")
	req.URL.RawQuery = q.Encode()

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("⚠️ Failed to poll messages: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var messages []struct {
		ID        string `json:"id"`
		From      string `json:"from"`
		Body      string `json:"body"`
		Type      string `json:"type"`
		FromMe    bool   `json:"fromMe"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return
	}

	for _, msg := range messages {
		// Skip jika sudah diproses
		if w.processedIDs[msg.ID] {
			continue
		}

		// Skip jika dari diri sendiri
		if msg.FromMe {
			continue
		}

		// Process hanya chat messages
		if msg.Type == "chat" && msg.Body != "" {
			evt := &WAHAMessage{
				From:    msg.From,
				Message: msg.Body,
			}
			handler(evt)

			// Mark as processed
			w.processedIDs[msg.ID] = true

			// Cleanup old IDs (keep last 100)
			if len(w.processedIDs) > 100 {
				// Simple cleanup: create new map
				newMap := make(map[string]bool)
				count := 0
				for id := range w.processedIDs {
					if count >= 50 {
						newMap[id] = true
					}
					count++
				}
				w.processedIDs = newMap
			}
		}
	}
}

func (w *WAHAProvider) GenerateQR(sessionID string) ([]byte, error) {
	// Use provided sessionID or fall back to default
	if sessionID == "" {
		sessionID = w.sessionID
	}

	log.Printf("🔍 Generating QR for session: %s", sessionID)

	// Ensure session exists first
	if err := w.StartSession(sessionID); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Wait for session to be ready for QR
	time.Sleep(2 * time.Second)

	endpoint := fmt.Sprintf("%s/api/%s/auth/qr", w.baseURL, sessionID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	// Set header untuk return image
	q := req.URL.Query()
	q.Add("format", "image")
	req.URL.RawQuery = q.Encode()

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WAHA returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ QR code generated successfully for session: %s", sessionID)

	// Return PNG image bytes
	return io.ReadAll(resp.Body)
}

// StartSession creates/starts a new WAHA session
func (w *WAHAProvider) StartSession(sessionID string) error {
	if sessionID == "" {
		sessionID = w.sessionID
	}

	log.Printf("🚀 Starting WAHA session: %s", sessionID)

	endpoint := fmt.Sprintf("%s/api/sessions/start", w.baseURL)

	payload := map[string]string{
		"name": sessionID,
	}
	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 200 = success, 409 = already exists (OK), 422 = already started (OK for WAHA Core)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		log.Printf("✅ Session created: %s", sessionID)
		return nil
	}

	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusUnprocessableEntity {
		// Session already exists or already started - this is OK
		log.Printf("ℹ️ Session '%s' already exists/started", sessionID)
		return nil
	}

	// Other errors
	return fmt.Errorf("WAHA returned status %d: %s", resp.StatusCode, string(body))
}

// GetSessionStatus checks if a session is connected
func (w *WAHAProvider) GetSessionStatus(sessionID string) (bool, error) {
	if sessionID == "" {
		sessionID = w.sessionID
	}

	endpoint := fmt.Sprintf("%s/api/%s", w.baseURL, sessionID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, err
	}

	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Status == "WORKING" || result.Status == "SCAN_QR_CODE", nil
}

func (w *WAHAProvider) IsConnected() bool {
	return w.connected
}

func (w *WAHAProvider) StartKeepAlive(ctx context.Context) {
	// WAHA handles connection maintenance internally
	log.Println("ℹ️ WAHA handles keep-alive internally")
}

// SetPresence sets presence status (typing, recording, paused, online, offline)
func (w *WAHAProvider) SetPresence(chatID, presence string) error {
	endpoint := fmt.Sprintf("%s/api/%s/presence", w.baseURL, w.sessionID)

	payload := map[string]string{
		"chatId":   chatID,
		"presence": presence,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set presence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WAHA returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StartTyping sets typing indicator for a chat
func (w *WAHAProvider) StartTyping(phoneNumber string) error {
	// Format: 628123456789@c.us
	chatID := phoneNumber
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		chatID = phoneNumber[1:] + "@c.us"
	} else if !contains(phoneNumber, "@c.us") {
		chatID = phoneNumber + "@c.us"
	}

	return w.SetPresence(chatID, "typing")
}

// StopTyping clears typing indicator for a chat
func (w *WAHAProvider) StopTyping(phoneNumber string) error {
	// Format: 628123456789@c.us
	chatID := phoneNumber
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		chatID = phoneNumber[1:] + "@c.us"
	} else if !contains(phoneNumber, "@c.us") {
		chatID = phoneNumber + "@c.us"
	}

	return w.SetPresence(chatID, "paused")
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
	       (len(s) > len(substr) && len(substr) > 0 && s[:len(substr)] == substr) ||
	       (len(s) >= len(substr) && len(substr) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// WAHAMessage adapter untuk compatibility
type WAHAMessage struct {
	From    string
	Message string
}

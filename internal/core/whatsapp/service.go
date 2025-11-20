// internal/core/whatsapp/service.go
package whatsapp

import (
	"context"
	"fmt"
	"log"
)

// Service adalah wrapper untuk WhatsApp provider
// Ini adalah layer yang digunakan oleh aplikasi
type Service struct {
	provider WhatsAppProvider
}

// NewService membuat service dengan provider dari environment
func NewService(storeURL string) *Service {
	cfg, err := LoadProviderFromEnv()
	if err != nil {
		log.Fatalf("❌ Failed to load provider config: %v", err)
	}

	// Override storeURL jika diberikan
	if storeURL != "" {
		cfg.StoreURL = storeURL
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create provider: %v", err)
	}

	log.Printf("✅ Using WhatsApp provider: %s", provider.GetProviderName())

	return &Service{
		provider: provider,
	}
}

// NewServiceWithProvider membuat service dengan provider spesifik (untuk testing)
func NewServiceWithProvider(provider WhatsAppProvider) *Service {
	return &Service{
		provider: provider,
	}
}

// Connect memulai koneksi WhatsApp
func (s *Service) Connect() error {
	return s.provider.Connect()
}

// Disconnect memutuskan koneksi
func (s *Service) Disconnect() {
	s.provider.Disconnect()
}

// SendMessage mengirim text message
func (s *Service) SendMessage(phoneNumber, message string) error {
	return s.provider.SendMessage(phoneNumber, message)
}

// StartListening mulai listen incoming messages
func (s *Service) StartListening(handler func(evt interface{})) error {
	// Wrap handler untuk normalize event dari berbagai provider
	normalizedHandler := func(evt interface{}) {
		// Convert provider-specific message ke format umum
		switch msg := evt.(type) {
		case *GreenAPIMessage:
			// Convert GreenAPI message ke whatsmeow-like format
			// Untuk compatibility dengan existing code
			handler(msg)

		case *WAHAMessage:
			// Convert WAHA message ke whatsmeow-like format
			handler(msg)

		default:
			// Whatsmeow native events atau unknown
			handler(evt)
		}
	}

	return s.provider.StartListening(normalizedHandler)
}

// GenerateQR generate QR code untuk pairing
// sessionID optional: if empty, use default session
func (s *Service) GenerateQR(sessionID string) ([]byte, error) {
	return s.provider.GenerateQR(sessionID)
}

// StartSession creates/starts a new session
func (s *Service) StartSession(sessionID string) error {
	return s.provider.StartSession(sessionID)
}

// GetSessionStatus checks if a session is connected
func (s *Service) GetSessionStatus(sessionID string) (bool, error) {
	return s.provider.GetSessionStatus(sessionID)
}

// IsConnected cek status koneksi (default session)
func (s *Service) IsConnected() bool {
	return s.provider.IsConnected()
}

// StartKeepAlive untuk maintain session
func (s *Service) StartKeepAlive(ctx context.Context) {
	s.provider.StartKeepAlive(ctx)
}

// GetProviderName return nama provider yang digunakan
func (s *Service) GetProviderName() string {
	return s.provider.GetProviderName()
}

// StartTyping shows typing indicator to the user
func (s *Service) StartTyping(phoneNumber string) error {
	return s.provider.StartTyping(phoneNumber)
}

// StopTyping stops/clears typing indicator
func (s *Service) StopTyping(phoneNumber string) error {
	return s.provider.StopTyping(phoneNumber)
}

// --- Backward compatibility helpers ---

// SendChatPresence untuk whatsmeow compatibility
func (s *Service) SendChatPresence(ctx context.Context, jid interface{}, state interface{}, media interface{}) error {
	// Only implemented for whatsmeow
	if w, ok := s.provider.(*WhatsmeowProvider); ok {
		if w.client != nil {
			// TODO: Implement proper JID and presence type conversion
			return nil
		}
	}
	return fmt.Errorf("SendChatPresence not supported for provider: %s", s.provider.GetProviderName())
}

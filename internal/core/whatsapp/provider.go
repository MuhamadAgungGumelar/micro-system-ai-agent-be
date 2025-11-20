// internal/core/whatsapp/provider.go
package whatsapp

import (
	"context"
	"fmt"
	"os"
)

// WhatsAppProvider adalah interface untuk semua WhatsApp integration providers
type WhatsAppProvider interface {
	// Connect menginisialisasi koneksi ke WhatsApp
	Connect() error

	// Disconnect memutuskan koneksi
	Disconnect()

	// SendMessage mengirim text message ke nomor tujuan
	SendMessage(phoneNumber, message string) error

	// StartListening mulai listen incoming messages
	StartListening(handler func(evt interface{})) error

	// GenerateQR generate QR code untuk pairing (return PNG bytes)
	// sessionID optional: if empty, use default session
	GenerateQR(sessionID string) ([]byte, error)

	// StartSession creates/starts a new session for a client
	StartSession(sessionID string) error

	// GetSessionStatus checks if a session is connected
	GetSessionStatus(sessionID string) (bool, error)

	// IsConnected cek apakah client masih terkoneksi (default session)
	IsConnected() bool

	// StartKeepAlive untuk maintain session (optional untuk beberapa provider)
	StartKeepAlive(ctx context.Context)

	// GetProviderName return nama provider untuk logging
	GetProviderName() string

	// StartTyping shows typing indicator to the user
	StartTyping(phoneNumber string) error

	// StopTyping stops/clears typing indicator
	StopTyping(phoneNumber string) error
}

// ProviderType untuk factory
type ProviderType string

const (
	ProviderWhatsmeow ProviderType = "whatsmeow"
	ProviderGreenAPI  ProviderType = "greenapi"
	ProviderWAHA      ProviderType = "waha"
)

// ProviderConfig konfigurasi untuk provider
type ProviderConfig struct {
	Type ProviderType

	// Common config
	StoreURL string

	// Green API specific
	GreenAPIInstanceID string
	GreenAPIToken      string
	GreenAPIURL        string

	// WAHA specific
	WAHABaseURL   string
	WAHAAPIKey    string
	WAHASessionID string
}

// NewProvider factory untuk create provider berdasarkan config
func NewProvider(cfg *ProviderConfig) (WhatsAppProvider, error) {
	switch cfg.Type {
	case ProviderWhatsmeow:
		return NewWhatsmeowProvider(cfg.StoreURL), nil

	case ProviderGreenAPI:
		if cfg.GreenAPIInstanceID == "" || cfg.GreenAPIToken == "" {
			return nil, fmt.Errorf("GREEN_API_INSTANCE_ID and GREEN_API_TOKEN are required")
		}
		return NewGreenAPIProvider(cfg.GreenAPIInstanceID, cfg.GreenAPIToken, cfg.GreenAPIURL), nil

	case ProviderWAHA:
		if cfg.WAHABaseURL == "" {
			return nil, fmt.Errorf("WAHA_BASE_URL is required")
		}
		return NewWAHAProvider(cfg.WAHABaseURL, cfg.WAHAAPIKey, cfg.WAHASessionID), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// LoadProviderFromEnv load config dari environment variables
func LoadProviderFromEnv() (*ProviderConfig, error) {
	providerType := os.Getenv("WHATSAPP_PROVIDER")
	if providerType == "" {
		providerType = "whatsmeow" // default
	}

	cfg := &ProviderConfig{
		Type:     ProviderType(providerType),
		StoreURL: os.Getenv("WHATSAPP_STORE_URL"),

		// Green API
		GreenAPIInstanceID: os.Getenv("GREEN_API_INSTANCE_ID"),
		GreenAPIToken:      os.Getenv("GREEN_API_TOKEN"),
		GreenAPIURL:        os.Getenv("GREEN_API_URL"),

		// WAHA
		WAHABaseURL:   os.Getenv("WAHA_BASE_URL"),
		WAHAAPIKey:    os.Getenv("WAHA_API_KEY"),
		WAHASessionID: os.Getenv("WAHA_SESSION_ID"),
	}

	// Set defaults
	if cfg.GreenAPIURL == "" {
		cfg.GreenAPIURL = "https://api.green-api.com"
	}
	if cfg.WAHASessionID == "" {
		cfg.WAHASessionID = "default"
	}

	return cfg, nil
}

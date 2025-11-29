package payment

import (
	"fmt"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/config"
	"gorm.io/gorm"
)

// NewGateway creates a payment gateway based on configuration
func NewGateway(cfg *config.Config, db *gorm.DB) (Gateway, error) {
	switch cfg.PaymentMode {
	case "manual":
		log.Println("💳 Using Manual Payment Gateway")
		return NewManualPaymentGateway(db), nil

	case "automated":
		if cfg.MidtransServerKey == "" {
			return nil, fmt.Errorf("MIDTRANS_SERVER_KEY is required for automated payment mode")
		}
		log.Println("💳 Using Midtrans Payment Gateway")
		return NewMidtransPaymentGateway(cfg.MidtransServerKey, cfg.MidtransIsProduction, db), nil

	default:
		// Default to manual
		log.Printf("⚠️  Unknown payment mode '%s', defaulting to manual", cfg.PaymentMode)
		return NewManualPaymentGateway(db), nil
	}
}

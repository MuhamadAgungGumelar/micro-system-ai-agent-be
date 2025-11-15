package whatsapp

import (
	"context"
	"log"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// StartKeepAlive mengirim presence update periodic untuk menjaga session tetap aktif
func (s *Service) StartKeepAlive(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second) // Ping tiap 60 detik
	defer ticker.Stop()

	log.Println("🔄 Keep-alive started (ping every 60s)")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Keep-alive stopped")
			return
		case <-ticker.C:
			if s.client != nil && s.client.IsConnected() {
				// Send "available" presence
				err := s.client.SendPresence(ctx, types.PresenceAvailable)
				if err != nil {
					log.Printf("⚠️ Keep-alive ping failed: %v", err)
				} else {
					log.Println("💓 Keep-alive ping sent")
				}
			}
		}
	}
}

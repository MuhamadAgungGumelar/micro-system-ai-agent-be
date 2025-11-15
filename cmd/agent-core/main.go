package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/agent"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/kb"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/tenant"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/config"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/database"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/utils"
)

func main() {
	// Init logger
	utils.InitLogger()

	// Load config
	cfg := config.LoadConfig()
	log.Info().Str("env", cfg.Env).Msg("🚀 Starting agent-core")

	// Init database
	db := database.NewDB(cfg.DatabaseURL)
	defer db.Close()

	// Init core services
	// TEMPORARY: Force SQLite for WhatsApp store (more stable)
	waService := whatsapp.NewService(cfg.WhatsAppStoreURL) // Empty string = use SQLite
	llmClient := llm.NewClient(cfg.OpenAIKey)
	kbRetriever := kb.NewRetriever(db.DB)
	tenantResolver := tenant.NewResolver(db.DB)

	// Init conversation logger (dari module saas)
	convRepo := repositories.NewConversationRepo(db.DB)

	// Init agent engine
	agentEngine := agent.NewEngine(
		waService,
		llmClient,
		kbRetriever,
		tenantResolver,
		convRepo,
	)

	// Connect WhatsApp
	log.Info().Msg("🔌 Connecting to WhatsApp...")
	if err := waService.Connect(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect WhatsApp")
	}

	// Start listening to messages
	log.Info().Msg("👂 Listening for WhatsApp messages...")
	err := waService.StartListening(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if !v.Info.IsFromMe {
				agentEngine.HandleMessage(v)
			}

		case *events.LoggedOut:
			log.Warn().Msg("⚠️ WhatsApp session logged out - device was removed by server")
			log.Warn().Msg("💡 This usually happens with personal WhatsApp accounts")
			log.Warn().Msg("💡 Consider using WhatsApp Business account or WhatsApp Business API")
			log.Warn().Msg("🔄 Attempting to reconnect in 10 seconds...")

			// Auto-reconnect after logout
			go func() {
				time.Sleep(10 * time.Second)
				log.Info().Msg("🔄 Reconnecting to WhatsApp...")
				if err := waService.Connect(); err != nil {
					log.Error().Err(err).Msg("Failed to reconnect")
				} else {
					log.Info().Msg("✅ Reconnected successfully!")
				}
			}()

		case *events.Connected:
			log.Info().Msg("✅ WhatsApp connected successfully")

		case *events.Disconnected:
			log.Warn().Msg("⚠️ WhatsApp disconnected - attempting reconnect...")
			go func() {
				time.Sleep(5 * time.Second)
				if err := waService.Connect(); err != nil {
					log.Error().Err(err).Msg("Failed to reconnect")
				}
			}()

		case *events.StreamError:
			log.Error().Interface("error", v).Msg("Stream error occurred")
		}
	})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start listening")
	}

	// Start keep-alive ping (helps prevent session timeout)
	keepAliveCtx, cancelKeepAlive := context.WithCancel(context.Background())
	defer cancelKeepAlive()
	go waService.StartKeepAlive(keepAliveCtx)

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info().Msg("🛑 Shutting down agent-core...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	waService.Disconnect()
	log.Info().Msg("👋 Goodbye!")
	_ = ctx // suppress unused warning
}

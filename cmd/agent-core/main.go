package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

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

	// Init core services (use GORM instance)
	waService := whatsapp.NewService(cfg.WhatsAppStoreURL)
	llmClient := llm.NewClient(cfg.OpenAIKey)
	kbRetriever := kb.NewRetriever(db.GORM)
	tenantResolver := tenant.NewResolver(db.DB) // Keep sql.DB for now (uses raw SQL)

	// Init conversation logger
	convRepo := repositories.NewConversationRepo(db.GORM)

	// Init agent engine
	agentEngine := agent.NewEngine(
		waService,
		llmClient,
		kbRetriever,
		tenantResolver,
		convRepo,
	)

	// Log provider yang digunakan
	log.Info().Str("provider", waService.GetProviderName()).Msg("📱 WhatsApp Provider")

	// Connect WhatsApp
	log.Info().Msg("🔌 Connecting to WhatsApp...")
	if err := waService.Connect(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect WhatsApp")
	}

	// Start listening to messages
	log.Info().Msg("👂 Listening for WhatsApp messages...")
	err := waService.StartListening(agentEngine.HandleMessage)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start listening")
	}

	// Start keep-alive (untuk whatsmeow, no-op untuk provider lain)
	keepAliveCtx, cancelKeepAlive := context.WithCancel(context.Background())
	defer cancelKeepAlive()
	go waService.StartKeepAlive(keepAliveCtx)

	log.Info().Msg("✅ Agent core is running. Press Ctrl+C to stop.")

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

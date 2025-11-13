package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/config"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/database"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/repositories"
	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/services"
)

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()
	log.Info().Str("env", cfg.Env).Msg("Starting whatsapp-bot-saas-be")

	db := database.NewDB(cfg.DatabaseURL)
	defer db.Close()

	clientRepo := repositories.NewClientRepo(db.DB)
	kbRepo := repositories.NewKBRepo(db.DB)
	convRepo := repositories.NewConversationRepo(db.DB)
	creditsRepo := repositories.NewCreditsRepo(db.DB)

	waService := services.NewWhatsAppService()
	aiService := services.NewAIService(cfg.OpenAIKey)
	msgService := services.NewMessageService(waService, aiService, clientRepo, kbRepo, convRepo, creditsRepo)

	clients, err := clientRepo.GetActiveClients()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch active clients")
	}
	if len(clients) == 0 {
		log.Warn().Msg("No active clients found")
		return
	}

	log.Info().Int("clients", len(clients)).Msg("Initializing bots...")

	if err := waService.ConnectClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect WhatsApp client")
	}

	err = waService.StartListening(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if !v.Info.IsFromMe {
				msgService.HandleIncomingMessage("default", v)
			}
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to start listening")
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info().Msg("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	waService.DisconnectAll(ctx)
	log.Info().Msg("Goodbye 👋")
}

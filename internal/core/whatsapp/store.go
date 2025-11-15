package whatsapp

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func initStore() (*sqlstore.Container, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("WhatsApp-Store", "ERROR", true)

	storeURL := os.Getenv("WHATSAPP_STORE_URL")
	if storeURL == "" {
		storeURL = os.Getenv("DATABASE_URL")
	}

	if storeURL == "" {
		return nil, fmt.Errorf("DATABASE_URL or WHATSAPP_STORE_URL is missing")
	}

	container, err := sqlstore.New(ctx, "postgres", storeURL, dbLog)
	if err != nil {
		return nil, fmt.Errorf("store init failed: %w", err)
	}

	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("store schema upgrade failed: %w", err)
	}

	log.Println("📦 WhatsApp store ready.")

	return container, nil
}

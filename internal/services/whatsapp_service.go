package services

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image/png"
	"log"
	"os"

	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

type WhatsAppService struct {
	client *whatsmeow.Client
}

// ✅ HELPER: Inisialisasi sqlstore + aktifkan foreign keys
func initWhatsAppStore() (*sqlstore.Container, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "ERROR", true)
	env := os.Getenv("ENV")

	switch env {
	case "production", "development":
		// 🟢 Mode production pakai PostgreSQL (Supabase)
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return nil, fmt.Errorf("DATABASE_URL not set for %s environment", env)
		}

		log.Printf("🌐 Using PostgreSQL database for %s environment", env)
		container, err := sqlstore.New(ctx, "postgres", dbURL, dbLog)
		if err != nil {
			return nil, fmt.Errorf("failed to init PostgreSQL store: %w", err)
		}

		if err := container.Upgrade(ctx); err != nil {
			return nil, fmt.Errorf("failed to upgrade PostgreSQL schema: %w", err)
		}

		return container, nil

	default:
		// 🧩 Default (local) pakai SQLite
		log.Println("💾 Using local SQLite store (store.db)")
		rawDB, err := sql.Open("sqlite", "file:store.db?_foreign_keys=on")
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite: %w", err)
		}

		// Aktifkan foreign keys
		_, err = rawDB.Exec("PRAGMA foreign_keys = ON;")
		if err != nil {
			log.Printf("⚠️ failed to enable foreign_keys pragma: %v", err)
		} else {
			log.Println("✅ SQLite foreign keys enabled")
		}

		container := sqlstore.NewWithDB(rawDB, "sqlite", dbLog)

		if err := container.Upgrade(ctx); err != nil {
			return nil, fmt.Errorf("failed to upgrade SQLite schema: %w", err)
		}

		return container, nil
	}
}

// ============================================================
// MAIN SERVICE
// ============================================================

func NewWhatsAppService() *WhatsAppService {
	container, err := initWhatsAppStore()
	if err != nil {
		log.Fatalf("❌ %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("failed to get device: %v", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	return &WhatsAppService{client: client}
}

func (s *WhatsAppService) ConnectClient() error {
	client := s.client

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("🔗 Scan QR ini di WhatsApp:", evt.Code)
			} else if evt.Event == "success" {
				fmt.Println("✅ Login berhasil!")
				break
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			return fmt.Errorf("failed reconnect: %w", err)
		}
		fmt.Println("✅ Reconnected to WhatsApp.")
	}

	return nil
}

func (s *WhatsAppService) StartListening(handler func(evt interface{})) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}
	s.client.AddEventHandler(handler)
	return nil
}

func (s *WhatsAppService) SendMessage(phoneNumber, message string) error {
	if s.client == nil {
		return fmt.Errorf("WhatsApp client not initialized")
	}

	jid := types.NewJID(phoneNumber, "s.whatsapp.net")
	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	_, err := s.client.SendMessage(context.Background(), jid, msg)
	return err
}

func (s *WhatsAppService) DisconnectAll(ctx context.Context) {
	if s.client != nil {
		s.client.Disconnect()
	}
	fmt.Println("🔌 Disconnected WhatsApp client.")
}

func (s *WhatsAppService) GenerateQRForClient(clientID string) ([]byte, error) {
	container, err := initWhatsAppStore()
	if err != nil {
		return nil, fmt.Errorf("failed to init store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	qrChan, _ := client.GetQRChannel(context.Background())
	defer client.Disconnect()

	go func() {
		_ = client.Connect()
	}()

	for evt := range qrChan {
		if evt.Event == "code" {
			// 🖼️ Convert QR text ke PNG bytes
			var buf bytes.Buffer
			img, _ := qrcode.New(evt.Code, qrcode.Medium)
			_ = png.Encode(&buf, img.Image(256))
			return buf.Bytes(), nil
		} else if evt.Event == "success" {
			break
		}
	}

	return nil, fmt.Errorf("no QR generated")
}

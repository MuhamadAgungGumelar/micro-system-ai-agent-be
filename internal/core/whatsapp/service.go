package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image/png"
	"log"
	"math/rand"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

type Service struct {
	client   *whatsmeow.Client
	storeURL string
}

func NewService(storeURL string) *Service {
	return &Service{
		storeURL: storeURL,
	}
}

// initStore inisialisasi sqlstore untuk WhatsApp session
func (s *Service) initStore() (*sqlstore.Container, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "ERROR", true)

	log.Println(s.storeURL)

	if s.storeURL != "" {
		log.Println("🌐 Using PostgreSQL database for WhatsApp store")
		container, err := sqlstore.New(ctx, "postgres", s.storeURL, dbLog)
		if err != nil {
			return nil, fmt.Errorf("failed to init PostgreSQL store: %w", err)
		}
		if err := container.Upgrade(ctx); err != nil {
			return nil, fmt.Errorf("failed to upgrade PostgreSQL schema: %w", err)
		}
		return container, nil
	}

	log.Println("💾 Using local SQLite store (store.db)")
	rawDB, err := sql.Open("sqlite", "file:store.db?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	if _, err = rawDB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		log.Printf("⚠️ Failed to enable foreign_keys pragma: %v", err)
	} else {
		log.Println("✅ SQLite foreign keys enabled")
	}

	container := sqlstore.NewWithDB(rawDB, "sqlite", dbLog)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("failed to upgrade SQLite schema: %w", err)
	}

	return container, nil
}

// Connect membuat koneksi WhatsApp client
func (s *Service) Connect() error {
	container, err := s.initStore()
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	s.client = whatsmeow.NewClient(deviceStore, clientLog)

	// Jika belum login, generate QR
	if s.client.Store.ID == nil {
		qrChan, _ := s.client.GetQRChannel(context.Background())
		if err := s.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("🔗 Scan QR ini di WhatsApp:", evt.Code)
				if err := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, "whatsapp-qr.png"); err != nil {
					log.Printf("Failed to generate QR image: %v", err)
				} else {
					fmt.Println("🖼️ QR code saved to whatsapp-qr.png")
					fmt.Println("")
					fmt.Println("⚠️  PENTING:")
					fmt.Println("   1. Gunakan nomor WhatsApp BUSINESS (bukan personal)")
					fmt.Println("   2. Jangan close WhatsApp di HP saat bot running")
					fmt.Println("   3. Pastikan HP tetap online & terhubung internet")
				}
			} else if evt.Event == "success" {
				fmt.Println("✅ Login berhasil!")
				break
			} else if evt.Event == "timeout" {
				return fmt.Errorf("QR code timeout, silakan coba lagi")
			}
		}
	} else {
		if err := s.client.Connect(); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
		fmt.Println("✅ Reconnected to WhatsApp.")
	}

	return nil
}

// StartListening mendengarkan event dari WhatsApp
func (s *Service) StartListening(handler func(evt interface{})) error {
	if s.client == nil {
		return fmt.Errorf("client not initialized")
	}
	s.client.AddEventHandler(handler)
	return nil
}

// SendMessage mengirim pesan text ke nomor tujuan
func (s *Service) SendMessage(phoneNumber, message string) error {
	if s.client == nil {
		return fmt.Errorf("WhatsApp client not initialized")
	}

	// Anti-ban: simulate typing
	typingDelay := time.Duration(1+rand.Intn(2)) * time.Second
	jid := types.NewJID(phoneNumber, "s.whatsapp.net")

	// Send typing presence
	_ = s.client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	time.Sleep(typingDelay)

	// Send message
	msg := &waProto.Message{
		Conversation: proto.String(message),
	}
	_, err := s.client.SendMessage(context.Background(), jid, msg)

	// Stop typing
	_ = s.client.SendChatPresence(context.Background(), jid, types.ChatPresencePaused, "")

	return err
}

// Disconnect memutuskan koneksi WhatsApp
func (s *Service) Disconnect() {
	if s.client != nil {
		s.client.Disconnect()
		log.Println("🔌 WhatsApp client disconnected")
	}
}

// GenerateQR menghasilkan QR code untuk client baru (untuk API endpoint)
func (s *Service) GenerateQR() ([]byte, error) {
	container, err := s.initStore()
	if err != nil {
		return nil, fmt.Errorf("failed to init store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	qrChan, _ := client.GetQRChannel(context.Background())

	go func() {
		_ = client.Connect()
	}()

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			var buf bytes.Buffer
			img, err := qrcode.New(evt.Code, qrcode.Medium)
			if err != nil {
				client.Disconnect()
				return nil, fmt.Errorf("failed to generate QR: %w", err)
			}

			if err := png.Encode(&buf, img.Image(256)); err != nil {
				client.Disconnect()
				return nil, fmt.Errorf("failed to encode QR png: %w", err)
			}

			// Auto-disconnect after 5 minutes
			go func(cli *whatsmeow.Client) {
				time.Sleep(5 * time.Minute)
				cli.Disconnect()
			}(client)

			return buf.Bytes(), nil

		case "timeout", "error":
			client.Disconnect()
			return nil, fmt.Errorf("QR generation failed: %s", evt.Event)
		}
	}

	return nil, fmt.Errorf("no QR generated")
}

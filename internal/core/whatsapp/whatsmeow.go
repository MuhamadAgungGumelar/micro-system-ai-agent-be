// internal/core/whatsapp/whatsmeow.go
package whatsapp

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image/png"
	"log"
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

type WhatsmeowProvider struct {
	client   *whatsmeow.Client
	storeURL string
}

func NewWhatsmeowProvider(storeURL string) *WhatsmeowProvider {
	return &WhatsmeowProvider{
		storeURL: storeURL,
	}
}

func (w *WhatsmeowProvider) GetProviderName() string {
	return "Whatsmeow"
}

func (w *WhatsmeowProvider) initStore() (*sqlstore.Container, error) {
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", "ERROR", true)

	if w.storeURL != "" {
		log.Println("🌐 Using PostgreSQL database for WhatsApp store")
		container, err := sqlstore.New(ctx, "postgres", w.storeURL, dbLog)
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

func (w *WhatsmeowProvider) Connect() error {
	container, err := w.initStore()
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	w.client = whatsmeow.NewClient(deviceStore, clientLog)

	if w.client.Store.ID == nil {
		qrChan, _ := w.client.GetQRChannel(context.Background())
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("🔗 Scan QR ini di WhatsApp:", evt.Code)
				if err := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, "whatsapp-qr.png"); err != nil {
					log.Printf("Failed to generate QR image: %v", err)
				} else {
					fmt.Println("🖼️ QR code saved to whatsapp-qr.png")
				}
			} else if evt.Event == "success" {
				fmt.Println("✅ Login berhasil!")
				break
			} else if evt.Event == "timeout" {
				return fmt.Errorf("QR code timeout")
			}
		}
	} else {
		if err := w.client.Connect(); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
		fmt.Println("✅ Reconnected to WhatsApp.")
	}

	return nil
}

func (w *WhatsmeowProvider) Disconnect() {
	if w.client != nil {
		w.client.Disconnect()
		log.Println("🔌 Whatsmeow client disconnected")
	}
}

func (w *WhatsmeowProvider) SendMessage(phoneNumber, message string) error {
	if w.client == nil {
		return fmt.Errorf("client not initialized")
	}

	jid := types.NewJID(phoneNumber, "s.whatsapp.net")
	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	_, err := w.client.SendMessage(context.Background(), jid, msg)
	return err
}

func (w *WhatsmeowProvider) StartListening(handler func(evt interface{})) error {
	if w.client == nil {
		return fmt.Errorf("client not initialized")
	}
	w.client.AddEventHandler(handler)
	return nil
}

func (w *WhatsmeowProvider) GenerateQR(sessionID string) ([]byte, error) {
	// Whatsmeow doesn't support multiple sessions per instance
	// sessionID is ignored for now
	container, err := w.initStore()
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
		if evt.Event == "code" {
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

			go func(cli *whatsmeow.Client) {
				time.Sleep(5 * time.Minute)
				cli.Disconnect()
			}(client)

			return buf.Bytes(), nil
		} else if evt.Event == "timeout" || evt.Event == "error" {
			client.Disconnect()
			return nil, fmt.Errorf("QR generation failed: %s", evt.Event)
		}
	}

	return nil, fmt.Errorf("no QR generated")
}

// StartSession creates/starts a new session (stub for Whatsmeow)
func (w *WhatsmeowProvider) StartSession(sessionID string) error {
	// Whatsmeow uses single session per instance
	return w.Connect()
}

// GetSessionStatus checks if a session is connected (stub for Whatsmeow)
func (w *WhatsmeowProvider) GetSessionStatus(sessionID string) (bool, error) {
	return w.IsConnected(), nil
}

func (w *WhatsmeowProvider) IsConnected() bool {
	return w.client != nil && w.client.IsConnected()
}

func (w *WhatsmeowProvider) StartKeepAlive(ctx context.Context) {
	if w.client == nil {
		return
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log.Println("🔄 Keep-alive started (ping every 60s)")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Keep-alive stopped")
			return
		case <-ticker.C:
			if w.client != nil && w.client.IsConnected() {
				err := w.client.SendPresence(ctx, types.PresenceAvailable)
				if err != nil {
					log.Printf("⚠️ Keep-alive ping failed: %v", err)
				} else {
					log.Println("💓 Keep-alive ping sent")
				}
			}
		}
	}
}

// StartTyping shows typing indicator (Whatsmeow implementation)
func (w *WhatsmeowProvider) StartTyping(phoneNumber string) error {
	if w.client == nil || !w.client.IsConnected() {
		return fmt.Errorf("whatsmeow client not connected")
	}

	// Parse JID from phone number
	jid, err := types.ParseJID(phoneNumber + "@s.whatsapp.net")
	if err != nil {
		return fmt.Errorf("invalid phone number: %w", err)
	}

	ctx := context.Background()
	return w.client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

// StopTyping clears typing indicator (Whatsmeow implementation)
func (w *WhatsmeowProvider) StopTyping(phoneNumber string) error {
	if w.client == nil || !w.client.IsConnected() {
		return fmt.Errorf("whatsmeow client not connected")
	}

	// Parse JID from phone number
	jid, err := types.ParseJID(phoneNumber + "@s.whatsapp.net")
	if err != nil {
		return fmt.Errorf("invalid phone number: %w", err)
	}

	ctx := context.Background()
	return w.client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
}

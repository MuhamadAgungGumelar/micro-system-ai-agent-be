package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/shared/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var module string
	var command string

	flag.StringVar(&module, "module", "saas", "Module to migrate (saas, umkm, farmasi)")
	flag.StringVar(&command, "cmd", "up", "Migration command (up, down, version, force)")
	flag.Parse()

	// Load config
	cfg := config.LoadConfig()

	// Migration path
	migrationPath := fmt.Sprintf("file://migrations/%s", module)

	log.Printf("🔄 Running migrations for module: %s", module)
	log.Printf("📂 Migration path: %s", migrationPath)
	log.Printf("💾 Database: %s", maskDatabaseURL(cfg.DatabaseURL))

	// Create migrate instance
	m, err := migrate.New(migrationPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		log.Println("⬆️  Running UP migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration UP failed: %v", err)
		}
		log.Println("✅ Migrations UP completed!")

	case "down":
		log.Println("⬇️  Running DOWN migrations...")
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration DOWN failed: %v", err)
		}
		log.Println("✅ Migrations DOWN completed!")

	case "version":
		version, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			log.Fatalf("❌ Failed to get version: %v", err)
		}
		log.Printf("📌 Current version: %d (dirty: %t)", version, dirty)

	case "force":
		if len(flag.Args()) < 1 {
			log.Fatal("❌ Please provide version number for force command")
		}
		var forceVersion int
		fmt.Sscanf(flag.Arg(0), "%d", &forceVersion)
		if err := m.Force(forceVersion); err != nil {
			log.Fatalf("❌ Force failed: %v", err)
		}
		log.Printf("✅ Forced version to: %d", forceVersion)

	default:
		log.Fatalf("❌ Unknown command: %s (use: up, down, version, force)", command)
	}
}

// maskDatabaseURL hides password in database URL for logging
func maskDatabaseURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:20] + "***" + url[len(url)-10:]
}

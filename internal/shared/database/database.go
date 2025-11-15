package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(connStr string) *DB {
	if connStr == "" {
		log.Fatal("❌ DATABASE_URL is empty")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(60 * time.Minute)

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	log.Println("✅ Database connected!")
	return &DB{DB: db}
}

func (db *DB) Close() error {
	log.Println("🔌 Closing database connection...")
	return db.DB.Close()
}

package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps *sql.DB and provides helper constructor
type DB struct {
	*sql.DB
}

// NewDB opens connection and returns *DB
func NewDB(connStr string) *DB {
	if connStr == "" {
		log.Fatal("DATABASE_URL is empty")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	// pool settings (tweak as needed)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(60 * time.Minute)

	// ping
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	log.Println("✅ Database connected!")
	return &DB{DB: db}
}

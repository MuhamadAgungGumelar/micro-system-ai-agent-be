package database

import (
	"database/sql"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps both GORM and sql.DB for backward compatibility
type DB struct {
	*sql.DB          // Keep for backward compatibility
	GORM    *gorm.DB // New GORM instance
}

// NewDB creates a new database connection using GORM
func NewDB(connStr string) *DB {
	if connStr == "" {
		log.Fatal("❌ DATABASE_URL is empty")
	}

	// Open GORM connection
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}

	// Get underlying sql.DB for backward compatibility
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("❌ Failed to get sql.DB: %v", err)
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(60 * 60) // 1 hour in seconds

	// Ping to verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	log.Println("✅ Database connected (GORM)!")
	return &DB{
		DB:   sqlDB,
		GORM: gormDB,
	}
}

func (db *DB) Close() error {
	log.Println("🔌 Closing database connection...")
	return db.DB.Close()
}

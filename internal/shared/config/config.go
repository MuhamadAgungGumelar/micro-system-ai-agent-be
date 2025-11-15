package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	WhatsAppStoreURL string
	OpenAIKey        string
	Port             string
	Env              string
	WameoAPIKey      string
	WameoAPIURL      string
	AgentCorePort    string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, using system environment variables")
	}

	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		WhatsAppStoreURL: os.Getenv("WHATSAPP_STORE_URL"),
		OpenAIKey:        os.Getenv("OPENAI_API_KEY"),
		Port:             os.Getenv("PORT"),
		Env:              os.Getenv("ENV"),
		WameoAPIKey:      os.Getenv("WAMEO_API_KEY"),
		WameoAPIURL:      os.Getenv("WAMEO_API_URL"),
		AgentCorePort:    os.Getenv("AGENT_CORE_PORT"),
	}

	// Default values
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.WhatsAppStoreURL == "" {
		// Default to main database if not specified
		cfg.WhatsAppStoreURL = cfg.DatabaseURL
	}
	if cfg.AgentCorePort == "" {
		cfg.AgentCorePort = "3000"
	}

	return cfg
}

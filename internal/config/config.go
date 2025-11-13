package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	OpenAIKey   string
	Port        string
	Env         string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  .env file not found, using system environment variables")
	}

	return Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		Port:        os.Getenv("PORT"),
		Env:         os.Getenv("ENV"),
	}
}

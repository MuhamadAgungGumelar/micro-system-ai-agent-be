package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	WhatsAppStoreURL    string
	OpenAIKey           string
	Port                string
	Env                 string
	WameoAPIKey         string
	WameoAPIURL         string
	AgentCorePort       string
	OCRProvider         string // "google_vision", "ocrspace", or "tesseract"
	GoogleVisionAPIKey  string
	OCRSpaceAPIKey      string
	TesseractLanguage   string // Language for Tesseract: "eng", "ind", or "eng+ind"

	// Payment Gateway Configuration
	PaymentMode         string // "manual" or "automated"
	MidtransServerKey   string
	MidtransIsProduction bool

	// Email Configuration
	EmailProvider string // "brevo" or "resend"
	BrevoAPIKey   string
	ResendAPIKey  string
	EmailFrom     string
	EmailFromName string

	// Notification Configuration
	AdminPhone string
	AdminEmail string

	// Authentication Configuration
	JWTSecret        string
	GoogleClientID   string
	GoogleClientSecret string

	// Upload Configuration
	UploadProvider     string // "local", "cloudinary", or "s3"
	UploadBasePath     string // Local storage: base directory path
	UploadBaseURL      string // Base URL for accessing files
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	S3AccessKeyID       string
	S3SecretAccessKey   string
	S3Region            string
	S3BucketName        string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, using system environment variables")
	}

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		WhatsAppStoreURL:   os.Getenv("WHATSAPP_STORE_URL"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		Port:               os.Getenv("PORT"),
		Env:                os.Getenv("ENV"),
		WameoAPIKey:        os.Getenv("WAMEO_API_KEY"),
		WameoAPIURL:        os.Getenv("WAMEO_API_URL"),
		AgentCorePort:      os.Getenv("AGENT_CORE_PORT"),
		OCRProvider:        os.Getenv("OCR_PROVIDER"),
		GoogleVisionAPIKey: os.Getenv("GOOGLE_VISION_API_KEY"),
		OCRSpaceAPIKey:     os.Getenv("OCRSPACE_API_KEY"),
		TesseractLanguage:  os.Getenv("TESSERACT_LANGUAGE"),

		// Payment Gateway
		PaymentMode:          os.Getenv("PAYMENT_MODE"),
		MidtransServerKey:    os.Getenv("MIDTRANS_SERVER_KEY"),
		MidtransIsProduction: os.Getenv("MIDTRANS_IS_PRODUCTION") == "true",

		// Email
		EmailProvider: os.Getenv("EMAIL_PROVIDER"),
		BrevoAPIKey:   os.Getenv("BREVO_API_KEY"),
		ResendAPIKey:  os.Getenv("RESEND_API_KEY"),
		EmailFrom:     os.Getenv("EMAIL_FROM"),
		EmailFromName: os.Getenv("EMAIL_FROM_NAME"),

		// Notification
		AdminPhone: os.Getenv("ADMIN_PHONE"),
		AdminEmail: os.Getenv("ADMIN_EMAIL"),

		// Authentication
		JWTSecret:          os.Getenv("JWT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),

		// Upload
		UploadProvider:      os.Getenv("UPLOAD_PROVIDER"),
		UploadBasePath:      os.Getenv("UPLOAD_BASE_PATH"),
		UploadBaseURL:       os.Getenv("UPLOAD_BASE_URL"),
		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),
		S3AccessKeyID:       os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:   os.Getenv("S3_SECRET_ACCESS_KEY"),
		S3Region:            os.Getenv("S3_REGION"),
		S3BucketName:        os.Getenv("S3_BUCKET_NAME"),
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
	if cfg.OCRProvider == "" {
		cfg.OCRProvider = "google_vision" // Default to Google Vision
	}
	if cfg.TesseractLanguage == "" {
		cfg.TesseractLanguage = "eng" // Default to English
	}
	if cfg.PaymentMode == "" {
		cfg.PaymentMode = "manual" // Default to manual for MVP
	}
	if cfg.EmailProvider == "" {
		cfg.EmailProvider = "brevo" // Default to Brevo
	}
	if cfg.EmailFromName == "" {
		cfg.EmailFromName = "WhatsApp Bot SaaS"
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "development-secret-key-change-in-production" // Default for development
		log.Println("⚠️ Using default JWT secret. Set JWT_SECRET in production!")
	}
	if cfg.UploadProvider == "" {
		cfg.UploadProvider = "local" // Default to local storage
	}
	if cfg.UploadBasePath == "" {
		cfg.UploadBasePath = "./uploads" // Default upload directory
	}
	if cfg.UploadBaseURL == "" {
		cfg.UploadBaseURL = "http://localhost:" + cfg.Port // Default base URL
	}

	return cfg
}

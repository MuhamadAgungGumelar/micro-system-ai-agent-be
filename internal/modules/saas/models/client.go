package models

import "time"

type Client struct {
	ID                 string    `json:"id"`
	WhatsAppNumber     string    `json:"whatsapp_number"`
	BusinessName       string    `json:"business_name"`
	SubscriptionPlan   string    `json:"subscription_plan"`
	SubscriptionStatus string    `json:"subscription_status"`
	Tone               string    `json:"tone"`
	Module             string    `json:"module"` // "saas", "farmasi", "umkm"
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type KnowledgeBase struct {
	BusinessName string    `json:"business_name"`
	Tone         string    `json:"tone"`
	FAQs         []FAQ     `json:"faqs"`
	Products     []Product `json:"products"`
}

type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Product struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Conversation struct {
	ID            string    `json:"id"`
	ClientID      string    `json:"client_id"`
	CustomerPhone string    `json:"customer_phone"`
	MessageType   string    `json:"message_type"`
	MessageText   string    `json:"message_text"`
	AIResponse    string    `json:"ai_response"`
	CreatedAt     time.Time `json:"created_at"`
}

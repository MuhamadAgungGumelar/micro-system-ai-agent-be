package models

import (
    "time"
)

type Client struct {
    ID                 string    `json:"id"`
    WhatsAppNumber     string    `json:"whatsapp_number"`
    BusinessName       string    `json:"business_name"`
    SubscriptionPlan   string    `json:"subscription_plan"`
    SubscriptionStatus string    `json:"subscription_status"`
    Tone               string    `json:"tone"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
}

type FAQ struct {
    Question string `json:"question"`
    Answer   string `json:"answer"`
}

type Product struct {
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

type KnowledgeBase struct {
    BusinessName string    `json:"business_name"`
    Tone         string    `json:"tone"`
    FAQs         []FAQ     `json:"faqs"`
    Products     []Product `json:"products"`
}
package database

import (
    "github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/models"
)

// GetActiveClients retrieves all active clients
func (db *DB) GetActiveClients() ([]models.Client, error) {
    query := `
        SELECT id, whatsapp_number, business_name, subscription_plan, 
               subscription_status, tone, created_at, updated_at
        FROM clients
        WHERE subscription_status = 'active'
    `

    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var clients []models.Client
    for rows.Next() {
        var c models.Client
        err := rows.Scan(
            &c.ID, &c.WhatsAppNumber, &c.BusinessName,
            &c.SubscriptionPlan, &c.SubscriptionStatus,
            &c.Tone, &c.CreatedAt, &c.UpdatedAt,
        )
        if err != nil {
            continue
        }
        clients = append(clients, c)
    }

    return clients, nil
}

// GetKnowledgeBase gets knowledge base for a client
func (db *DB) GetKnowledgeBase(clientID string) (*models.KnowledgeBase, error) {
    kb := &models.KnowledgeBase{}

    // Get client info
    err := db.QueryRow(`
        SELECT business_name, tone FROM clients WHERE id = $1
    `, clientID).Scan(&kb.BusinessName, &kb.Tone)
    if err != nil {
        return nil, err
    }

    // Get FAQs
    faqRows, err := db.Query(`
        SELECT question, answer FROM knowledge_base 
        WHERE client_id = $1 AND type = 'faq'
        LIMIT 50
    `, clientID)
    if err != nil {
        return nil, err
    }
    defer faqRows.Close()

    for faqRows.Next() {
        var faq models.FAQ
        faqRows.Scan(&faq.Question, &faq.Answer)
        kb.FAQs = append(kb.FAQs, faq)
    }

    // Get Products
    prodRows, err := db.Query(`
        SELECT product_name, product_price FROM knowledge_base 
        WHERE client_id = $1 AND type = 'product'
        LIMIT 100
    `, clientID)
    if err != nil {
        return nil, err
    }
    defer prodRows.Close()

    for prodRows.Next() {
        var prod models.Product
        prodRows.Scan(&prod.Name, &prod.Price)
        kb.Products = append(kb.Products, prod)
    }

    return kb, nil
}

// LogConversation logs a conversation to database
func (db *DB) LogConversation(clientID, customerPhone, message, response string) error {
    _, err := db.Exec(`
        INSERT INTO conversations 
        (client_id, customer_phone, message_type, message_text, ai_response)
        VALUES ($1, $2, 'incoming', $3, $4)
    `, clientID, customerPhone, message, response)

    if err == nil {
        // Update credits
        db.Exec(`
            UPDATE credits 
            SET credits_used = credits_used + 1
            WHERE client_id = $1 
            AND CURRENT_DATE BETWEEN period_start AND period_end
        `, clientID)
    }

    return err
}
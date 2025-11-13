package repositories

import "database/sql"

type ConversationRepo interface {
	LogConversation(clientID, customerPhone, message, response string) error
}

type conversationRepo struct {
	db *sql.DB
}

func NewConversationRepo(db *sql.DB) ConversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) LogConversation(clientID, customerPhone, message, response string) error {
	_, err := r.db.Exec(`
        INSERT INTO conversations (client_id, customer_phone, message_type, message_text, ai_response)
        VALUES ($1, $2, 'incoming', $3, $4)
    `, clientID, customerPhone, message, response)
	if err != nil {
		return err
	}
	// update credits (best effort)
	_, _ = r.db.Exec(`
        UPDATE credits SET credits_used = credits_used + 1
        WHERE client_id = $1 AND CURRENT_DATE BETWEEN period_start AND period_end
    `, clientID)
	return nil
}

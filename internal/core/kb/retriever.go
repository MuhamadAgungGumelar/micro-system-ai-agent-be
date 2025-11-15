package kb

import (
	"database/sql"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
)

type Retriever struct {
	db *sql.DB
}

func NewRetriever(db *sql.DB) *Retriever {
	return &Retriever{db: db}
}

// GetKnowledgeBase mengambil knowledge base untuk client tertentu
func (r *Retriever) GetKnowledgeBase(clientID string) (*llm.KnowledgeBase, error) {
	kb := &llm.KnowledgeBase{}

	// Get client info
	err := r.db.QueryRow(`
		SELECT business_name, tone 
		FROM clients 
		WHERE id = $1
	`, clientID).Scan(&kb.BusinessName, &kb.Tone)

	if err != nil {
		return nil, err
	}

	// Get FAQs
	faqRows, err := r.db.Query(`
		SELECT question, answer 
		FROM knowledge_base 
		WHERE client_id = $1 AND type = 'faq' 
		LIMIT 50
	`, clientID)

	if err != nil {
		return nil, err
	}
	defer faqRows.Close()

	for faqRows.Next() {
		var faq llm.FAQ
		if err := faqRows.Scan(&faq.Question, &faq.Answer); err == nil {
			kb.FAQs = append(kb.FAQs, faq)
		}
	}

	// Get Products
	prodRows, err := r.db.Query(`
		SELECT product_name, product_price 
		FROM knowledge_base 
		WHERE client_id = $1 AND type = 'product' 
		LIMIT 100
	`, clientID)

	if err != nil {
		return nil, err
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var prod llm.Product
		if err := prodRows.Scan(&prod.Name, &prod.Price); err == nil {
			kb.Products = append(kb.Products, prod)
		}
	}

	return kb, nil
}

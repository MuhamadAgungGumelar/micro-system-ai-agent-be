package repositories

import (
	"database/sql"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/models"
)

type KBRepo interface {
	GetKnowledgeBase(clientID string) (*models.KnowledgeBase, error)
}

type kbRepo struct {
	db *sql.DB
}

func NewKBRepo(db *sql.DB) KBRepo {
	return &kbRepo{db: db}
}

func (r *kbRepo) GetKnowledgeBase(clientID string) (*models.KnowledgeBase, error) {
	kb := &models.KnowledgeBase{}

	// client info
	if err := r.db.QueryRow(`SELECT business_name, tone FROM clients WHERE id = $1`, clientID).Scan(&kb.BusinessName, &kb.Tone); err != nil {
		return nil, err
	}

	// faqs
	faqRows, err := r.db.Query(`SELECT question, answer FROM knowledge_base WHERE client_id = $1 AND type = 'faq' LIMIT 50`, clientID)
	if err != nil {
		return nil, err
	}
	defer faqRows.Close()

	for faqRows.Next() {
		var f models.FAQ
		if err := faqRows.Scan(&f.Question, &f.Answer); err == nil {
			kb.FAQs = append(kb.FAQs, f)
		}
	}

	// products
	prodRows, err := r.db.Query(`SELECT product_name, product_price FROM knowledge_base WHERE client_id = $1 AND type = 'product' LIMIT 100`, clientID)
	if err != nil {
		return nil, err
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var p models.Product
		if err := prodRows.Scan(&p.Name, &p.Price); err == nil {
			kb.Products = append(kb.Products, p)
		}
	}

	return kb, nil
}

package repositories

import (
	"database/sql"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
)

type ClientRepo interface {
	GetActiveClients() ([]models.Client, error)
	GetByID(id string) (*models.Client, error)
}

type clientRepo struct {
	db *sql.DB
}

func NewClientRepo(db *sql.DB) ClientRepo {
	return &clientRepo{db: db}
}

func (r *clientRepo) GetActiveClients() ([]models.Client, error) {
	query := `
		SELECT id, whatsapp_number, business_name, subscription_plan, 
		       subscription_status, tone, COALESCE(module, 'saas'), created_at, updated_at
		FROM clients 
		WHERE subscription_status = 'active'
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Client
	for rows.Next() {
		var c models.Client
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&c.ID, &c.WhatsAppNumber, &c.BusinessName,
			&c.SubscriptionPlan, &c.SubscriptionStatus, &c.Tone,
			&c.Module, &createdAt, &updatedAt,
		)

		if err != nil {
			continue
		}

		if createdAt.Valid {
			c.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			c.UpdatedAt = updatedAt.Time
		}

		list = append(list, c)
	}

	return list, nil
}

func (r *clientRepo) GetByID(id string) (*models.Client, error) {
	query := `
		SELECT id, whatsapp_number, business_name, subscription_plan, 
		       subscription_status, tone, COALESCE(module, 'saas'), created_at, updated_at
		FROM clients 
		WHERE id = $1
	`

	var c models.Client
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.WhatsAppNumber, &c.BusinessName,
		&c.SubscriptionPlan, &c.SubscriptionStatus, &c.Tone,
		&c.Module, &createdAt, &updatedAt,
	)

	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		c.UpdatedAt = updatedAt.Time
	}

	return &c, nil
}

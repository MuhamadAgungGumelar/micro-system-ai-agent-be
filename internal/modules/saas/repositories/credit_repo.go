package repositories

import "database/sql"

type CreditsRepo struct {
	DB *sql.DB
}

func NewCreditsRepo(db *sql.DB) *CreditsRepo {
	return &CreditsRepo{DB: db}
}

// Contoh fungsi (dummy)
func (r *CreditsRepo) IncrementUsage(clientID string) error {
	_, err := r.DB.Exec(`UPDATE credits SET credits_used = credits_used + 1 WHERE client_id = $1`, clientID)
	return err
}

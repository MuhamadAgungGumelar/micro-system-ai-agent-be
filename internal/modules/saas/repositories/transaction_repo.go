package repositories

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"gorm.io/gorm"
)

// TransactionRepo interface defines transaction operations
type TransactionRepo interface {
	Create(transaction *models.Transaction) error
	GetByID(id string) (*models.Transaction, error)
	GetByClientID(clientID string, limit int) ([]models.Transaction, error)
}

type transactionRepo struct {
	db *gorm.DB
}

// NewTransactionRepo creates a new transaction repository
func NewTransactionRepo(db *gorm.DB) TransactionRepo {
	return &transactionRepo{db: db}
}

// Create inserts a new transaction
func (r *transactionRepo) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

// GetByID retrieves a transaction by ID
func (r *transactionRepo) GetByID(id string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.Where("id = ?", id).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByClientID retrieves transactions for a specific client
func (r *transactionRepo) GetByClientID(clientID string, limit int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	query := r.db.Where("client_id = ?", clientID).
		Order("transaction_date DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

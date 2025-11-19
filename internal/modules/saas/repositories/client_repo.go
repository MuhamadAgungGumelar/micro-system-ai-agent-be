package repositories

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientRepo interface {
	GetActiveClients() ([]models.Client, error)
	GetByID(id string) (*models.Client, error)
	Create(client *models.Client) error
	Update(client *models.Client) error
	Delete(id string) error
}

type clientRepo struct {
	db *gorm.DB
}

func NewClientRepo(db *gorm.DB) ClientRepo {
	return &clientRepo{db: db}
}

func (r *clientRepo) GetActiveClients() ([]models.Client, error) {
	var clients []models.Client
	err := r.db.Where("subscription_status = ?", "active").Find(&clients).Error
	return clients, err
}

func (r *clientRepo) GetByID(id string) (*models.Client, error) {
	// Parse UUID
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var client models.Client
	err = r.db.First(&client, "id = ?", uid).Error
	return &client, err
}

func (r *clientRepo) Create(client *models.Client) error {
	return r.db.Create(client).Error
}

func (r *clientRepo) Update(client *models.Client) error {
	return r.db.Save(client).Error
}

func (r *clientRepo) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.db.Delete(&models.Client{}, "id = ?", uid).Error
}

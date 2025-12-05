package repositories

import (
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepo interface {
	Create(cart *models.Cart) error
	GetByID(id string) (*models.Cart, error)
	GetActiveCart(clientID, customerPhone string) (*models.Cart, error)
	Update(cart *models.Cart) error
	Delete(id string) error
	ExpireCart(id string) error
	CleanupExpiredCarts() error
}

type cartRepo struct {
	db *gorm.DB
}

func NewCartRepo(db *gorm.DB) CartRepo {
	return &cartRepo{db: db}
}

func (r *cartRepo) Create(cart *models.Cart) error {
	return r.db.Create(cart).Error
}

func (r *cartRepo) GetByID(id string) (*models.Cart, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var cart models.Cart
	err = r.db.First(&cart, "id = ?", uid).Error
	return &cart, err
}

func (r *cartRepo) GetActiveCart(clientID, customerPhone string) (*models.Cart, error) {
	var cart models.Cart
	err := r.db.Where("client_id = ? AND customer_phone = ? AND status = ?", clientID, customerPhone, "active").
		First(&cart).Error
	return &cart, err
}

func (r *cartRepo) Update(cart *models.Cart) error {
	return r.db.Save(cart).Error
}

func (r *cartRepo) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.db.Delete(&models.Cart{}, "id = ?", uid).Error
}

func (r *cartRepo) ExpireCart(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.db.Model(&models.Cart{}).
		Where("id = ?", uid).
		Update("status", "expired").Error
}

func (r *cartRepo) CleanupExpiredCarts() error {
	// Update status to expired for carts that have passed their expiry time
	return r.db.Model(&models.Cart{}).
		Where("status = ? AND expires_at < ?", "active", time.Now()).
		Update("status", "expired").Error
}

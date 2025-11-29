package repositories

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepo interface {
	Create(order *models.Order) error
	GetByID(id string) (*models.Order, error)
	GetByOrderNumber(orderNumber string) (*models.Order, error)
	GetByClientID(clientID string, limit int) ([]models.Order, error)
	GetByCustomerPhone(clientID, customerPhone string, limit int) ([]models.Order, error)
	UpdatePaymentStatus(orderID, status string) error
	UpdateFulfillmentStatus(orderID, status string) error
	Update(order *models.Order) error
	Delete(id string) error
}

type orderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) OrderRepo {
	return &orderRepo{db: db}
}

func (r *orderRepo) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepo) GetByID(id string) (*models.Order, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var order models.Order
	err = r.db.First(&order, "id = ?", uid).Error
	return &order, err
}

func (r *orderRepo) GetByOrderNumber(orderNumber string) (*models.Order, error) {
	var order models.Order
	err := r.db.Where("order_number = ?", orderNumber).First(&order).Error
	return &order, err
}

func (r *orderRepo) GetByClientID(clientID string, limit int) ([]models.Order, error) {
	var orders []models.Order
	query := r.db.Where("client_id = ?", clientID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&orders).Error
	return orders, err
}

func (r *orderRepo) GetByCustomerPhone(clientID, customerPhone string, limit int) ([]models.Order, error) {
	var orders []models.Order
	query := r.db.Where("client_id = ? AND customer_phone = ?", clientID, customerPhone).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&orders).Error
	return orders, err
}

func (r *orderRepo) UpdatePaymentStatus(orderID, status string) error {
	return r.db.Model(&models.Order{}).
		Where("id = ?", orderID).
		Update("payment_status", status).Error
}

func (r *orderRepo) UpdateFulfillmentStatus(orderID, status string) error {
	return r.db.Model(&models.Order{}).
		Where("id = ?", orderID).
		Update("fulfillment_status", status).Error
}

func (r *orderRepo) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *orderRepo) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.db.Delete(&models.Order{}, "id = ?", uid).Error
}

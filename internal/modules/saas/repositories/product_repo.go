package repositories

import (
	"fmt"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepo interface {
	Create(product *models.Product) error
	GetByID(id string) (*models.Product, error)
	GetBySKU(clientID uuid.UUID, sku string) (*models.Product, error)
	List(filter models.ProductFilter) ([]models.Product, int64, error)
	Update(product *models.Product) error
	Delete(id string) error           // Soft delete
	HardDelete(id string) error       // Permanent delete
	UpdateStock(id string, quantity int) error
	BulkUpdateStock(updates map[string]int) error
}

type productRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) ProductRepo {
	return &productRepo{db: db}
}

func (r *productRepo) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *productRepo) GetByID(id string) (*models.Product, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID: %w", err)
	}

	var product models.Product
	err = r.db.First(&product, "id = ?", uid).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) GetBySKU(clientID uuid.UUID, sku string) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("client_id = ? AND sku = ?", clientID, sku).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) List(filter models.ProductFilter) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	query := r.db.Model(&models.Product{}).Where("client_id = ?", filter.ClientID)

	// Apply filters
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		query = query.Where("name ILIKE ? OR sku ILIKE ? OR description ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if filter.MinPrice != nil {
		query = query.Where("price >= ?", *filter.MinPrice)
	}

	if filter.MaxPrice != nil {
		query = query.Where("price <= ?", *filter.MaxPrice)
	}

	if filter.InStock != nil && *filter.InStock {
		query = query.Where("stock > 0")
	}

	// Count total
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	offset := (filter.Page - 1) * filter.PageSize
	err = query.Offset(offset).Limit(filter.PageSize).
		Order("created_at DESC").
		Find(&products).Error

	return products, total, err
}

func (r *productRepo) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *productRepo) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}

	// Soft delete
	return r.db.Delete(&models.Product{}, "id = ?", uid).Error
}

func (r *productRepo) HardDelete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}

	// Permanent delete
	return r.db.Unscoped().Delete(&models.Product{}, "id = ?", uid).Error
}

func (r *productRepo) UpdateStock(id string, quantity int) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %w", err)
	}

	return r.db.Model(&models.Product{}).
		Where("id = ?", uid).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).Error
}

func (r *productRepo) BulkUpdateStock(updates map[string]int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for productID, quantity := range updates {
			uid, err := uuid.Parse(productID)
			if err != nil {
				return fmt.Errorf("invalid product ID %s: %w", productID, err)
			}

			err = tx.Model(&models.Product{}).
				Where("id = ?", uid).
				UpdateColumn("stock", gorm.Expr("stock + ?", quantity)).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

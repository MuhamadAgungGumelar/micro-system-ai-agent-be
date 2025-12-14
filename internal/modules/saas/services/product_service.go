package services

import (
	"errors"
	"fmt"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService struct {
	productRepo repositories.ProductRepo
}

func NewProductService(productRepo repositories.ProductRepo) *ProductService {
	return &ProductService{
		productRepo: productRepo,
	}
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(clientID uuid.UUID, req *models.CreateProductRequest) (*models.Product, error) {
	// Validate request
	if req.Name == "" {
		return nil, errors.New("product name is required")
	}
	if req.Price < 0 {
		return nil, errors.New("price cannot be negative")
	}
	if req.Stock < 0 {
		return nil, errors.New("stock cannot be negative")
	}

	// Check if SKU already exists for this client
	if req.SKU != "" {
		existing, err := s.productRepo.GetBySKU(clientID, req.SKU)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("product with SKU '%s' already exists", req.SKU)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Create product
	product := &models.Product{
		ClientID:    clientID,
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Category:    req.Category,
		Price:       req.Price,
		Stock:       req.Stock,
		ImageURL:    req.ImageURL,
		IsActive:    true,
	}

	// Override IsActive if explicitly set
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	err := s.productRepo.Create(product)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return product, nil
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(productID string, clientID uuid.UUID) (*models.Product, error) {
	product, err := s.productRepo.GetByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	// Verify product belongs to client
	if product.ClientID != clientID {
		return nil, errors.New("product not found")
	}

	return product, nil
}

// ListProducts retrieves products with filtering and pagination
func (s *ProductService) ListProducts(filter models.ProductFilter) (*models.ProductListResponse, error) {
	products, total, err := s.productRepo.List(filter)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &models.ProductListResponse{
		Products:   products,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(productID string, clientID uuid.UUID, req *models.UpdateProductRequest) (*models.Product, error) {
	// Get existing product
	product, err := s.GetProduct(productID, clientID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.New("product name cannot be empty")
		}
		product.Name = *req.Name
	}

	if req.Description != nil {
		product.Description = *req.Description
	}

	if req.SKU != nil {
		// Check if new SKU already exists (for different product)
		if *req.SKU != product.SKU && *req.SKU != "" {
			existing, err := s.productRepo.GetBySKU(clientID, *req.SKU)
			if err == nil && existing != nil && existing.ID != product.ID {
				return nil, fmt.Errorf("product with SKU '%s' already exists", *req.SKU)
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		}
		product.SKU = *req.SKU
	}

	if req.Category != nil {
		product.Category = *req.Category
	}

	if req.Price != nil {
		if *req.Price < 0 {
			return nil, errors.New("price cannot be negative")
		}
		product.Price = *req.Price
	}

	if req.Stock != nil {
		if *req.Stock < 0 {
			return nil, errors.New("stock cannot be negative")
		}
		product.Stock = *req.Stock
	}

	if req.ImageURL != nil {
		product.ImageURL = *req.ImageURL
	}

	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	// Save updates
	err = s.productRepo.Update(product)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return product, nil
}

// DeleteProduct soft deletes a product
func (s *ProductService) DeleteProduct(productID string, clientID uuid.UUID) error {
	// Verify product belongs to client
	_, err := s.GetProduct(productID, clientID)
	if err != nil {
		return err
	}

	return s.productRepo.Delete(productID)
}

// UpdateStock updates product stock (can be positive or negative)
func (s *ProductService) UpdateStock(productID string, clientID uuid.UUID, quantity int) (*models.Product, error) {
	// Verify product belongs to client
	product, err := s.GetProduct(productID, clientID)
	if err != nil {
		return nil, err
	}

	// Check if deduction would result in negative stock
	if quantity < 0 && product.Stock+quantity < 0 {
		return nil, errors.New("insufficient stock")
	}

	err = s.productRepo.UpdateStock(productID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	// Get updated product
	product, err = s.productRepo.GetByID(productID)
	if err != nil {
		return nil, err
	}

	return product, nil
}

// BulkUpdateStock updates stock for multiple products
func (s *ProductService) BulkUpdateStock(clientID uuid.UUID, updates map[string]int) error {
	// Validate all products belong to client first
	for productID := range updates {
		_, err := s.GetProduct(productID, clientID)
		if err != nil {
			return fmt.Errorf("product %s: %w", productID, err)
		}
	}

	return s.productRepo.BulkUpdateStock(updates)
}

// GetProductBySKU retrieves a product by SKU
func (s *ProductService) GetProductBySKU(clientID uuid.UUID, sku string) (*models.Product, error) {
	if sku == "" {
		return nil, errors.New("SKU is required")
	}

	product, err := s.productRepo.GetBySKU(clientID, sku)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return product, nil
}

// ToggleProductStatus toggles product active status
func (s *ProductService) ToggleProductStatus(productID string, clientID uuid.UUID) (*models.Product, error) {
	product, err := s.GetProduct(productID, clientID)
	if err != nil {
		return nil, err
	}

	product.IsActive = !product.IsActive

	err = s.productRepo.Update(product)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle product status: %w", err)
	}

	return product, nil
}

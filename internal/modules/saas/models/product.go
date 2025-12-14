package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product represents a product in the catalog
type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID    uuid.UUID      `gorm:"type:uuid;not null" json:"client_id"`

	// Product Info
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	SKU         string `gorm:"type:text" json:"sku,omitempty"` // Stock Keeping Unit
	Category    string `gorm:"type:text" json:"category,omitempty"`

	// Pricing & Stock
	Price       float64 `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	Stock       int     `gorm:"type:integer;not null;default:0" json:"stock"`

	// Media
	ImageURL    string `gorm:"type:text" json:"image_url,omitempty"`

	// Status
	IsActive    bool `gorm:"type:boolean;default:true" json:"is_active"`

	// Timestamps
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name
func (Product) TableName() string {
	return "saas_products"
}

// BeforeCreate sets UUID before creating
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// IsAvailable checks if product is available for sale
func (p *Product) IsAvailable() bool {
	return p.IsActive && p.Stock > 0
}

// DeductStock reduces the stock by the specified quantity
func (p *Product) DeductStock(quantity int) bool {
	if p.Stock >= quantity {
		p.Stock -= quantity
		return true
	}
	return false
}

// AddStock increases the stock by the specified quantity
func (p *Product) AddStock(quantity int) {
	p.Stock += quantity
}

// CreateProductRequest represents product creation request
type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=200"`
	Description string  `json:"description,omitempty" validate:"max=1000"`
	SKU         string  `json:"sku,omitempty" validate:"max=100"`
	Category    string  `json:"category,omitempty" validate:"max=100"`
	Price       float64 `json:"price" validate:"required,gte=0"`
	Stock       int     `json:"stock" validate:"gte=0"`
	ImageURL    string  `json:"image_url,omitempty" validate:"omitempty,url"`
	IsActive    *bool   `json:"is_active,omitempty"` // Pointer to allow explicit false
}

// UpdateProductRequest represents product update request
type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=1000"`
	SKU         *string  `json:"sku,omitempty" validate:"omitempty,max=100"`
	Category    *string  `json:"category,omitempty" validate:"omitempty,max=100"`
	Price       *float64 `json:"price,omitempty" validate:"omitempty,gte=0"`
	Stock       *int     `json:"stock,omitempty" validate:"omitempty,gte=0"`
	ImageURL    *string  `json:"image_url,omitempty" validate:"omitempty,url"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// ProductListResponse represents paginated product list response
type ProductListResponse struct {
	Products   []Product `json:"products"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// ProductFilter represents product filtering options
type ProductFilter struct {
	ClientID   uuid.UUID
	Category   string
	IsActive   *bool
	SearchTerm string // Search in name, SKU, description
	MinPrice   *float64
	MaxPrice   *float64
	InStock    *bool // Only products with stock > 0
	Page       int
	PageSize   int
}

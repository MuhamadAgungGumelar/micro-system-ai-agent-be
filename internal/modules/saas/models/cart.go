package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CartItem represents a single item in the cart
type CartItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Subtotal    float64 `json:"subtotal"`
	Notes       string  `json:"notes,omitempty"`
}

// CartItems is a custom type for JSONB array
type CartItems []CartItem

// Scan implements sql.Scanner interface
func (c *CartItems) Scan(value interface{}) error {
	if value == nil {
		*c = []CartItem{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// Value implements driver.Valuer interface
func (c CartItems) Value() (driver.Value, error) {
	if c == nil {
		return json.Marshal([]CartItem{})
	}
	return json.Marshal(c)
}

// Cart represents a shopping cart
type Cart struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerPhone string         `json:"customer_phone" gorm:"not null"`
	ClientID      uuid.UUID      `json:"client_id" gorm:"type:uuid;not null"`
	Items         CartItems      `json:"items" gorm:"type:jsonb;not null"`
	TotalAmount   float64        `json:"total_amount" gorm:"type:decimal(12,2);default:0"`
	Status        string         `json:"status" gorm:"default:'active';check:status IN ('active', 'checked_out', 'expired', 'cancelled')"`
	CreatedAt     time.Time      `json:"created_at" gorm:"default:now()"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"default:now()"`
	ExpiresAt     time.Time      `json:"expires_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Cart) TableName() string {
	return "saas_carts"
}

// BeforeCreate hook to set expiry time
func (c *Cart) BeforeCreate(tx *gorm.DB) error {
	if c.ExpiresAt.IsZero() {
		c.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	return nil
}

// CalculateTotal recalculates the total amount based on items
func (c *Cart) CalculateTotal() {
	total := 0.0
	for _, item := range c.Items {
		total += item.Subtotal
	}
	c.TotalAmount = total
}

// AddItem adds or updates an item in the cart
func (c *Cart) AddItem(item CartItem) {
	// Calculate subtotal
	item.Subtotal = item.Price * float64(item.Quantity)

	// Check if item already exists
	for i, existingItem := range c.Items {
		if existingItem.ProductID == item.ProductID {
			// Update quantity and subtotal
			c.Items[i].Quantity += item.Quantity
			c.Items[i].Subtotal = c.Items[i].Price * float64(c.Items[i].Quantity)
			c.CalculateTotal()
			return
		}
	}

	// Add new item
	c.Items = append(c.Items, item)
	c.CalculateTotal()
}

// UpdateItem updates an existing item's quantity
func (c *Cart) UpdateItem(productID string, quantity int) bool {
	for i, item := range c.Items {
		if item.ProductID == productID {
			if quantity <= 0 {
				// Remove item if quantity is 0 or negative
				c.Items = append(c.Items[:i], c.Items[i+1:]...)
			} else {
				c.Items[i].Quantity = quantity
				c.Items[i].Subtotal = c.Items[i].Price * float64(quantity)
			}
			c.CalculateTotal()
			return true
		}
	}
	return false
}

// RemoveItem removes an item from the cart
func (c *Cart) RemoveItem(productID string) bool {
	for i, item := range c.Items {
		if item.ProductID == productID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.CalculateTotal()
			return true
		}
	}
	return false
}

// ClearItems removes all items from the cart
func (c *Cart) ClearItems() {
	c.Items = []CartItem{}
	c.TotalAmount = 0
}

// IsExpired checks if the cart has expired
func (c *Cart) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsEmpty checks if the cart has no items
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

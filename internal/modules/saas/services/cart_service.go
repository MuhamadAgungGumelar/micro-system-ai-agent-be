package services

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CartService struct {
	cartRepo  repositories.CartRepo
	orderRepo repositories.OrderRepo
}

func NewCartService(cartRepo repositories.CartRepo, orderRepo repositories.OrderRepo) *CartService {
	return &CartService{
		cartRepo:  cartRepo,
		orderRepo: orderRepo,
	}
}

type AddToCartRequest struct {
	ClientID      string  `json:"client_id"`
	CustomerPhone string  `json:"customer_phone"`
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	Quantity      int     `json:"quantity"`
	Price         float64 `json:"price"`
	Notes         string  `json:"notes,omitempty"`
}

type UpdateCartItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveFromCartRequest struct {
	ProductID string `json:"product_id"`
}

// AddToCart adds an item to the cart (creates cart if doesn't exist)
func (s *CartService) AddToCart(req *AddToCartRequest) (*models.Cart, error) {
	// Validate
	if req.Quantity <= 0 {
		return nil, errors.New("quantity must be greater than 0")
	}
	if req.Price < 0 {
		return nil, errors.New("price cannot be negative")
	}

	// Get or create active cart
	cart, err := s.cartRepo.GetActiveCart(req.ClientID, req.CustomerPhone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new cart
			clientID, err := uuid.Parse(req.ClientID)
			if err != nil {
				return nil, errors.New("invalid client_id")
			}

			cart = &models.Cart{
				ClientID:      clientID,
				CustomerPhone: req.CustomerPhone,
				Status:        "active",
				Items:         models.CartItems{},
			}

			if err := s.cartRepo.Create(cart); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Check if cart is expired
	if cart.IsExpired() {
		s.cartRepo.ExpireCart(cart.ID.String())
		return nil, errors.New("cart has expired, please create a new one")
	}

	// Add item to cart
	item := models.CartItem{
		ProductID:   req.ProductID,
		ProductName: req.ProductName,
		Quantity:    req.Quantity,
		Price:       req.Price,
		Notes:       req.Notes,
	}
	cart.AddItem(item)

	// Save cart
	if err := s.cartRepo.Update(cart); err != nil {
		return nil, err
	}

	log.Printf("🛒 Added %dx %s to cart for %s", req.Quantity, req.ProductName, req.CustomerPhone)
	return cart, nil
}

// UpdateCartItem updates the quantity of an item in the cart
func (s *CartService) UpdateCartItem(clientID, customerPhone string, req *UpdateCartItemRequest) (*models.Cart, error) {
	cart, err := s.cartRepo.GetActiveCart(clientID, customerPhone)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	if cart.IsExpired() {
		s.cartRepo.ExpireCart(cart.ID.String())
		return nil, errors.New("cart has expired")
	}

	// Update item (removes if quantity <= 0)
	if !cart.UpdateItem(req.ProductID, req.Quantity) {
		return nil, errors.New("product not found in cart")
	}

	if err := s.cartRepo.Update(cart); err != nil {
		return nil, err
	}

	log.Printf("🛒 Updated cart item %s quantity to %d for %s", req.ProductID, req.Quantity, customerPhone)
	return cart, nil
}

// RemoveFromCart removes an item from the cart
func (s *CartService) RemoveFromCart(clientID, customerPhone string, req *RemoveFromCartRequest) (*models.Cart, error) {
	cart, err := s.cartRepo.GetActiveCart(clientID, customerPhone)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	if cart.IsExpired() {
		s.cartRepo.ExpireCart(cart.ID.String())
		return nil, errors.New("cart has expired")
	}

	if !cart.RemoveItem(req.ProductID) {
		return nil, errors.New("product not found in cart")
	}

	if err := s.cartRepo.Update(cart); err != nil {
		return nil, err
	}

	log.Printf("🛒 Removed %s from cart for %s", req.ProductID, customerPhone)
	return cart, nil
}

// ViewCart retrieves the current active cart
func (s *CartService) ViewCart(clientID, customerPhone string) (*models.Cart, error) {
	cart, err := s.cartRepo.GetActiveCart(clientID, customerPhone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no active cart found")
		}
		return nil, err
	}

	if cart.IsExpired() {
		s.cartRepo.ExpireCart(cart.ID.String())
		return nil, errors.New("cart has expired")
	}

	return cart, nil
}

// ClearCart removes all items from the cart
func (s *CartService) ClearCart(clientID, customerPhone string) error {
	cart, err := s.cartRepo.GetActiveCart(clientID, customerPhone)
	if err != nil {
		return errors.New("cart not found")
	}

	cart.ClearItems()
	if err := s.cartRepo.Update(cart); err != nil {
		return err
	}

	log.Printf("🛒 Cleared cart for %s", customerPhone)
	return nil
}

// CheckoutCart converts the cart to an order
func (s *CartService) CheckoutCart(clientID, customerPhone string) (*models.Order, error) {
	cart, err := s.cartRepo.GetActiveCart(clientID, customerPhone)
	if err != nil {
		return nil, errors.New("cart not found")
	}

	if cart.IsExpired() {
		s.cartRepo.ExpireCart(cart.ID.String())
		return nil, errors.New("cart has expired")
	}

	if cart.IsEmpty() {
		return nil, errors.New("cart is empty, cannot checkout")
	}

	// Convert cart items to order items
	orderItems := make([]models.OrderItem, len(cart.Items))
	for i, item := range cart.Items {
		orderItems[i] = models.OrderItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
			Subtotal:    item.Subtotal,
		}
	}

	// Marshal to JSON
	itemsJSON, err := json.Marshal(orderItems)
	if err != nil {
		return nil, err
	}

	// Create order from cart
	order := &models.Order{
		ClientID:          cart.ClientID,
		CustomerPhone:     cart.CustomerPhone,
		Items:             datatypes.JSON(itemsJSON),
		TotalAmount:       cart.TotalAmount,
		PaymentStatus:     "pending",
		FulfillmentStatus: "pending",
	}

	if err := s.orderRepo.Create(order); err != nil {
		return nil, err
	}

	// Mark cart as checked out
	cart.Status = "checked_out"
	if err := s.cartRepo.Update(cart); err != nil {
		log.Printf("⚠️  Failed to mark cart as checked_out: %v", err)
	}

	log.Printf("✅ Checked out cart for %s - Order %s created", customerPhone, order.OrderNumber)
	return order, nil
}

// CleanupExpiredCarts marks expired carts as expired (should be run periodically)
func (s *CartService) CleanupExpiredCarts() error {
	return s.cartRepo.CleanupExpiredCarts()
}

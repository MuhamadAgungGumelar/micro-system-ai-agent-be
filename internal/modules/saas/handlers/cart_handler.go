package handlers

import (
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
)

type CartHandler struct {
	cartService *services.CartService
}

func NewCartHandler(cartService *services.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

// AddToCart godoc
// @Summary Add item to cart
// @Description Add a product to the shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param item body services.AddToCartRequest true "Item to add"
// @Success 200 {object} map[string]interface{}
// @Router /cart/add [post]
func (h *CartHandler) AddToCart(c *fiber.Ctx) error {
	var req services.AddToCartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// Validate required fields
	if req.ClientID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id is required"})
	}
	if req.CustomerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "customer_phone is required"})
	}
	if req.ProductID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "product_id is required"})
	}
	if req.ProductName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "product_name is required"})
	}
	if req.Quantity <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "quantity must be greater than 0"})
	}
	if req.Price < 0 {
		return c.Status(400).JSON(fiber.Map{"error": "price cannot be negative"})
	}

	cart, err := h.cartService.AddToCart(&req)
	if err != nil {
		log.Printf("❌ Failed to add to cart: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Item added to cart successfully",
		"cart":    cart,
	})
}

// UpdateCartItem godoc
// @Summary Update cart item quantity
// @Description Update the quantity of an item in the cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param client_id query string true "Client ID"
// @Param customer_phone query string true "Customer Phone"
// @Param item body services.UpdateCartItemRequest true "Update details"
// @Success 200 {object} map[string]interface{}
// @Router /cart/update [put]
func (h *CartHandler) UpdateCartItem(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	customerPhone := c.Query("customer_phone")

	if clientID == "" || customerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id and customer_phone are required"})
	}

	var req services.UpdateCartItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.ProductID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "product_id is required"})
	}

	cart, err := h.cartService.UpdateCartItem(clientID, customerPhone, &req)
	if err != nil {
		log.Printf("❌ Failed to update cart item: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Cart item updated successfully",
		"cart":    cart,
	})
}

// RemoveFromCart godoc
// @Summary Remove item from cart
// @Description Remove a product from the shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param client_id query string true "Client ID"
// @Param customer_phone query string true "Customer Phone"
// @Param item body services.RemoveFromCartRequest true "Product to remove"
// @Success 200 {object} map[string]interface{}
// @Router /cart/remove [delete]
func (h *CartHandler) RemoveFromCart(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	customerPhone := c.Query("customer_phone")

	if clientID == "" || customerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id and customer_phone are required"})
	}

	var req services.RemoveFromCartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.ProductID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "product_id is required"})
	}

	cart, err := h.cartService.RemoveFromCart(clientID, customerPhone, &req)
	if err != nil {
		log.Printf("❌ Failed to remove from cart: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Item removed from cart successfully",
		"cart":    cart,
	})
}

// ViewCart godoc
// @Summary View shopping cart
// @Description Get the current active shopping cart
// @Tags Cart
// @Produce json
// @Param client_id query string true "Client ID"
// @Param customer_phone query string true "Customer Phone"
// @Success 200 {object} map[string]interface{}
// @Router /cart [get]
func (h *CartHandler) ViewCart(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	customerPhone := c.Query("customer_phone")

	if clientID == "" || customerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id and customer_phone are required"})
	}

	cart, err := h.cartService.ViewCart(clientID, customerPhone)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"cart": cart,
	})
}

// ClearCart godoc
// @Summary Clear shopping cart
// @Description Remove all items from the cart
// @Tags Cart
// @Produce json
// @Param client_id query string true "Client ID"
// @Param customer_phone query string true "Customer Phone"
// @Success 200 {object} map[string]interface{}
// @Router /cart/clear [delete]
func (h *CartHandler) ClearCart(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	customerPhone := c.Query("customer_phone")

	if clientID == "" || customerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id and customer_phone are required"})
	}

	err := h.cartService.ClearCart(clientID, customerPhone)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Cart cleared successfully",
	})
}

// CheckoutCart godoc
// @Summary Checkout cart
// @Description Convert cart to order and initiate payment
// @Tags Cart
// @Produce json
// @Param client_id query string true "Client ID"
// @Param customer_phone query string true "Customer Phone"
// @Success 200 {object} map[string]interface{}
// @Router /cart/checkout [post]
func (h *CartHandler) CheckoutCart(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	customerPhone := c.Query("customer_phone")

	if clientID == "" || customerPhone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "client_id and customer_phone are required"})
	}

	order, err := h.cartService.CheckoutCart(clientID, customerPhone)
	if err != nil {
		log.Printf("❌ Failed to checkout cart: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Checkout successful",
		"order":   order,
	})
}

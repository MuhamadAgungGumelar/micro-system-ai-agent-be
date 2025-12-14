package handlers

import (
	"strconv"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ProductHandler struct {
	productService *services.ProductService
}

func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// CreateProduct godoc
// @Summary Create a new product
// @Description Create a new product in the catalog (requires authentication)
// @Tags Products
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param product body models.CreateProductRequest true "Product data"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	// Get client_id from auth context
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: client_id not found in context",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	var req models.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	product, err := h.productService.CreateProduct(clientID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(product)
}

// GetProduct godoc
// @Summary Get product by ID
// @Description Retrieve a product by its ID (requires authentication)
// @Tags Products
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Product ID"
// @Success 200 {object} models.Product
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID is required",
		})
	}

	product, err := h.productService.GetProduct(productID, clientID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

// ListProducts godoc
// @Summary List products
// @Description List products with filtering and pagination (requires authentication)
// @Tags Products
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param category query string false "Filter by category"
// @Param is_active query boolean false "Filter by active status"
// @Param search query string false "Search in name, SKU, description"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param in_stock query boolean false "Only products with stock > 0"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} models.ProductListResponse
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	// Build filter from query params
	filter := models.ProductFilter{
		ClientID:   clientID,
		Category:   c.Query("category"),
		SearchTerm: c.Query("search"),
		Page:       1,
		PageSize:   10,
	}

	// Parse is_active
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// Parse in_stock
	if inStockStr := c.Query("in_stock"); inStockStr != "" {
		inStock := inStockStr == "true"
		filter.InStock = &inStock
	}

	// Parse min_price
	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			filter.MinPrice = &minPrice
		}
	}

	// Parse max_price
	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			filter.MaxPrice = &maxPrice
		}
	}

	// Parse pagination
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			filter.PageSize = pageSize
		}
	}

	response, err := h.productService.ListProducts(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}

// UpdateProduct godoc
// @Summary Update product
// @Description Update an existing product (requires authentication)
// @Tags Products
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Product ID"
// @Param product body models.UpdateProductRequest true "Product updates"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID is required",
		})
	}

	var req models.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	product, err := h.productService.UpdateProduct(productID, clientID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

// DeleteProduct godoc
// @Summary Delete product
// @Description Soft delete a product (requires authentication)
// @Tags Products
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Product ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID is required",
		})
	}

	err = h.productService.DeleteProduct(productID, clientID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully",
	})
}

// UpdateStock godoc
// @Summary Update product stock
// @Description Update product stock (can add or deduct) (requires authentication)
// @Tags Products
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Product ID"
// @Param stock body map[string]int true "Stock update" example({"quantity": 10})
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]interface{}
// @Router /products/{id}/stock [patch]
func (h *ProductHandler) UpdateStock(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID is required",
		})
	}

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	product, err := h.productService.UpdateStock(productID, clientID, req.Quantity)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

// ToggleProductStatus godoc
// @Summary Toggle product active status
// @Description Toggle product active/inactive status (requires authentication)
// @Tags Products
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Product ID"
// @Success 200 {object} models.Product
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id}/toggle [patch]
func (h *ProductHandler) ToggleProductStatus(c *fiber.Ctx) error {
	clientIDStr, ok := c.Locals("clientID").(string)
	if !ok || clientIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client_id",
		})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Product ID is required",
		})
	}

	product, err := h.productService.ToggleProductStatus(productID, clientID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

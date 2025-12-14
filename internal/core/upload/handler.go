package upload

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// Handler handles file upload HTTP requests
type Handler struct {
	uploadService *Service
}

// NewHandler creates a new upload handler
func NewHandler(uploadService *Service) *Handler {
	return &Handler{
		uploadService: uploadService,
	}
}

// UploadFile godoc
// @Summary Upload a file
// @Description Upload a file to the configured storage provider (requires authentication)
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param file formData file true "File to upload"
// @Param folder formData string false "Folder to upload to" default("uploads")
// @Param public_id formData string false "Custom public ID"
// @Param overwrite formData boolean false "Overwrite existing file" default(false)
// @Success 200 {object} UploadResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /upload [post]
func (h *Handler) UploadFile(c *fiber.Ctx) error {
	// Get file from form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Get upload options from form
	options := &UploadOptions{
		Folder:    c.FormValue("folder", "uploads"),
		PublicID:  c.FormValue("public_id"),
		Overwrite: c.FormValue("overwrite") == "true",
	}

	// Parse width and height if provided
	if widthStr := c.FormValue("width"); widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil {
			options.Width = width
		}
	}
	if heightStr := c.FormValue("height"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			options.Height = height
		}
	}
	if c.FormValue("crop") == "true" {
		options.Crop = true
	}

	// Upload file
	result, err := h.uploadService.UploadMultipart(fileHeader, options)
	if err != nil {
		log.Printf("❌ Failed to upload file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("✅ File uploaded successfully: %s (%s)", result.FileName, result.URL)

	return c.JSON(result)
}

// UploadProductImage godoc
// @Summary Upload product image
// @Description Upload an image for a product (requires authentication)
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param file formData file true "Product image to upload"
// @Param product_id formData string false "Product ID (used as public_id)"
// @Success 200 {object} UploadResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /upload/product [post]
func (h *Handler) UploadProductImage(c *fiber.Ctx) error {
	// Get file from form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Get client ID from context (set by auth middleware)
	clientID, ok := c.Locals("clientID").(string)
	if !ok || clientID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: client_id not found",
		})
	}

	// Get product ID from form
	productID := c.FormValue("product_id")

	// Set upload options for product images
	options := &UploadOptions{
		Folder:       "products/" + clientID,
		PublicID:     productID,
		Overwrite:    true,
		ResourceType: "image",
		AllowedTypes: []string{"image/jpeg", "image/jpg", "image/png", "image/webp"},
		MaxSize:      5 * 1024 * 1024, // 5MB
		Width:        800,              // Resize to 800px width
		Height:       800,              // Resize to 800px height
		Crop:         true,             // Enable cropping
	}

	// Upload file
	result, err := h.uploadService.UploadMultipart(fileHeader, options)
	if err != nil {
		log.Printf("❌ Failed to upload product image: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("✅ Product image uploaded: %s (%s)", result.FileName, result.URL)

	return c.JSON(result)
}

// DeleteFile godoc
// @Summary Delete a file
// @Description Delete a file from storage (requires authentication)
// @Tags Upload
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param public_id query string true "Public ID of the file to delete"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /upload [delete]
func (h *Handler) DeleteFile(c *fiber.Ctx) error {
	publicID := c.Query("public_id")
	if publicID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "public_id is required",
		})
	}

	err := h.uploadService.Delete(publicID)
	if err != nil {
		log.Printf("❌ Failed to delete file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	log.Printf("✅ File deleted successfully: %s", publicID)

	return c.JSON(fiber.Map{
		"message":   "File deleted successfully",
		"public_id": publicID,
	})
}

// GetProviderInfo godoc
// @Summary Get upload provider info
// @Description Get information about the current upload provider
// @Tags Upload
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /upload/info [get]
func (h *Handler) GetProviderInfo(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"provider": h.uploadService.GetProviderName(),
	})
}

package handlers

import (
	"encoding/json"
	"io"
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/ocr"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OCRHandler handles OCR-related requests
type OCRHandler struct {
	ocrService        *ocr.Service
	llmService        *llm.Service
	transactionRepo   repositories.TransactionRepo
	workflowService   *services.WorkflowService
}

// NewOCRHandler creates a new OCR handler
func NewOCRHandler(ocrService *ocr.Service, llmService *llm.Service, transactionRepo repositories.TransactionRepo, workflowService *services.WorkflowService) *OCRHandler {
	return &OCRHandler{
		ocrService:      ocrService,
		llmService:      llmService,
		transactionRepo: transactionRepo,
		workflowService: workflowService,
	}
}

// ProcessReceiptRequest represents the request body for processing receipt
type ProcessReceiptRequest struct {
	ClientID string `form:"client_id" json:"client_id"`
}

// ProcessReceipt godoc
// @Summary Process receipt image and create transaction
// @Description Upload a receipt image, extract text using OCR, parse it, and create a transaction record
// @Tags OCR
// @Accept multipart/form-data
// @Produce json
// @Param client_id formData string true "Client ID"
// @Param image formData file true "Receipt image file"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ocr/process-receipt [post]
func (h *OCRHandler) ProcessReceipt(c *fiber.Ctx) error {
	// Get client_id from form
	clientID := c.FormValue("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	// Validate UUID
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid client_id format",
		})
	}

	// Get uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "image file is required",
		})
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/jpg" && contentType != "image/png" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "only JPEG and PNG images are supported",
		})
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file size must be less than 10MB",
		})
	}

	// Open and read file
	fileHandle, err := file.Open()
	if err != nil {
		log.Printf("❌ Failed to open file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to read image file",
		})
	}
	defer fileHandle.Close()

	imageData, err := io.ReadAll(fileHandle)
	if err != nil {
		log.Printf("❌ Failed to read file data: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to read image file",
		})
	}

	log.Printf("📸 Processing receipt image for client: %s (size: %.2f KB)", clientID, float64(file.Size)/1024)

	// Extract text using OCR
	log.Printf("🔍 Calling OCR service: %s", h.ocrService.GetProviderName())
	ocrResult, err := h.ocrService.ExtractText(c.Context(), imageData)
	if err != nil {
		log.Printf("❌ OCR extraction failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to extract text from image",
		})
	}

	log.Printf("✅ OCR extracted text (confidence: %.2f%%): %s", ocrResult.Confidence*100, ocrResult.Text[:min(100, len(ocrResult.Text))])

	// Parse receipt data using LLM
	log.Printf("🤖 Parsing receipt with LLM...")
	llmParser := ocr.NewLLMParser(h.llmService)
	receiptData, err := llmParser.ParseReceiptWithLLM(c.Context(), ocrResult.Text)
	if err != nil {
		log.Printf("❌ Failed to parse receipt: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to parse receipt data",
		})
	}

	log.Printf("📊 Parsed receipt: Total=%.2f, Items=%d, Store=%s", receiptData.TotalAmount, len(receiptData.Items), receiptData.StoreName)

	// Convert items to JSONB
	itemsJSON, err := json.Marshal(receiptData.Items)
	if err != nil {
		log.Printf("❌ Failed to marshal items: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to process receipt items",
		})
	}

	// Create transaction record
	transaction := &models.Transaction{
		ClientID:        clientUUID,
		TotalAmount:     receiptData.TotalAmount,
		TransactionDate: receiptData.TransactionDate,
		StoreName:       receiptData.StoreName,
		Items:           datatypes.JSON(itemsJSON),
		CreatedFrom:     "ocr",
		SourceType:      "receipt",
		OCRConfidence:   &ocrResult.Confidence,
		OCRRawText:      ocrResult.Text,
	}

	// Save to database
	if err := h.transactionRepo.Create(transaction); err != nil {
		log.Printf("❌ Failed to save transaction: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save transaction",
		})
	}

	log.Printf("💾 Transaction saved successfully: %s", transaction.ID.String())

	// Trigger workflow event: transaction_created
	if h.workflowService != nil {
		go func() {
			eventData := map[string]interface{}{
				"transaction_id":   transaction.ID.String(),
				"client_id":        transaction.ClientID.String(),
				"total_amount":     transaction.TotalAmount,
				"transaction_date": transaction.TransactionDate,
				"store_name":       transaction.StoreName,
				"items_count":      len(receiptData.Items),
				"created_from":     transaction.CreatedFrom,
				"source_type":      transaction.SourceType,
				"ocr_confidence":   transaction.OCRConfidence,
			}

			if err := h.workflowService.HandleEvent(c.Context(), "transaction_created", eventData); err != nil {
				log.Printf("⚠️ Failed to trigger workflows for transaction_created: %v", err)
			}
		}()
	}

	// Return success response
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"message": "Receipt processed successfully",
		"data": fiber.Map{
			"transaction_id":   transaction.ID.String(),
			"total_amount":     transaction.TotalAmount,
			"transaction_date": transaction.TransactionDate,
			"store_name":       transaction.StoreName,
			"items_count":      len(receiptData.Items),
			"items":            receiptData.Items,
			"ocr_confidence":   ocrResult.Confidence,
			"created_from":     "ocr",
			"source_type":      "receipt",
		},
	})
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetTransactions godoc
// @Summary Get transactions for a client
// @Description Retrieve transaction history for a specific client
// @Tags Transactions
// @Produce json
// @Param client_id query string true "Client ID"
// @Param limit query int false "Limit number of results" default(50)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /transactions [get]
func (h *OCRHandler) GetTransactions(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100 // Max limit
	}

	transactions, err := h.transactionRepo.GetByClientID(clientID, limit)
	if err != nil {
		log.Printf("❌ Failed to get transactions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve transactions",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(transactions),
		"data":   transactions,
	})
}

package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
)

// LLMParser uses LLM to parse receipt text into structured data
type LLMParser struct {
	llmService *llm.Service
}

// NewLLMParser creates a new LLM-based receipt parser
func NewLLMParser(llmService *llm.Service) *LLMParser {
	return &LLMParser{
		llmService: llmService,
	}
}

// ParseReceiptWithLLM parses receipt text using LLM (much more accurate than regex)
func (p *LLMParser) ParseReceiptWithLLM(ctx context.Context, ocrText string) (*ReceiptData, error) {
	log.Printf("🤖 Parsing receipt with LLM: %s", p.llmService.GetProviderName())

	// Build prompt for LLM
	systemPrompt := buildReceiptParserPrompt()
	userPrompt := fmt.Sprintf("Parse this Indonesian receipt OCR text:\n\n%s", ocrText)

	// Call LLM
	response, err := p.llmService.GenerateResponse(ctx, systemPrompt, userPrompt)
	if err != nil {
		log.Printf("❌ LLM parsing failed: %v", err)
		// Fallback to regex parser
		return ParseReceipt(ocrText)
	}

	log.Printf("🤖 Raw LLM response: %s", response)

	// Clean response - remove markdown code blocks if present
	cleanedResponse := strings.TrimSpace(response)
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	log.Printf("🧹 Cleaned LLM response: %s", cleanedResponse)

	// Parse LLM JSON response
	var receiptData ReceiptData
	if err := json.Unmarshal([]byte(cleanedResponse), &receiptData); err != nil {
		log.Printf("⚠️ Failed to parse LLM JSON response: %v", err)
		log.Printf("⚠️ Response was: %s", cleanedResponse)
		// Fallback to regex parser
		log.Printf("⬇️ Falling back to regex parser")
		return ParseReceipt(ocrText)
	}

	// Store raw text
	receiptData.RawText = ocrText

	// Validate parsed data
	if receiptData.TransactionDate.IsZero() {
		receiptData.TransactionDate = time.Now()
	}

	log.Printf("✅ LLM parsed: Total=%.2f, Date=%s, Items=%d, Store=%s",
		receiptData.TotalAmount, receiptData.TransactionDate.Format("2006-01-02"),
		len(receiptData.Items), receiptData.StoreName)

	return &receiptData, nil
}

// buildReceiptParserPrompt creates system prompt for receipt parsing
func buildReceiptParserPrompt() string {
	return `You are a receipt parser. Your task is to extract structured data from Indonesian receipts.

Parse the OCR text and return ONLY a valid JSON object with this exact structure:

{
  "store_name": "Name of the store/merchant",
  "total_amount": 0.0,
  "transaction_date": "2024-01-15T10:30:00Z",
  "items": [
    {
      "name": "Product name",
      "quantity": 1,
      "price": 0.0
    }
  ]
}

IMPORTANT RULES:
1. Return ONLY the JSON object, no markdown, no explanation, no code blocks
2. total_amount must be a number (not string), extract from "Total", "Grand Total", or "Jumlah"
3. transaction_date must be in ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
4. If date is not found, use current date/time
5. items array should contain all purchased products with their quantities and individual prices
6. Remove any spaces from numbers (e.g., "26 , 620" → 26620)
7. Handle various formats: "Rp 100,000", "100.000", "100,000", etc.
8. For item names, use the actual product name (e.g., "Indomie Goreng", not "lusin x")
9. quantity must be an integer
10. price is the unit price (not total price for that item)
11. If you cannot extract certain fields, use reasonable defaults:
    - store_name: "" (empty string)
    - total_amount: 0
    - items: [] (empty array)

Example OCR text:
"""
Karis Jaya Shop
1. Indomie Goreng
1 lusin x 36,000
Total: Rp 70.000
"""

Expected output:
{
  "store_name": "Karis Jaya Shop",
  "total_amount": 70000,
  "transaction_date": "2024-01-15T10:00:00Z",
  "items": [
    {
      "name": "Indomie Goreng",
      "quantity": 1,
      "price": 36000
    }
  ]
}

Now parse the receipt text provided by the user.`
}

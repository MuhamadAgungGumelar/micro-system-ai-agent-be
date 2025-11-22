package ocr

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ReceiptData represents parsed receipt information
type ReceiptData struct {
	TotalAmount     float64       `json:"total_amount"`
	TransactionDate time.Time     `json:"transaction_date"`
	Items           []ReceiptItem `json:"items"`
	StoreName       string        `json:"store_name,omitempty"`
	RawText         string        `json:"raw_text"`
}

// ReceiptItem represents an item in the receipt
type ReceiptItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// ParseReceipt attempts to parse receipt text into structured data
// This is a basic parser - can be improved with ML/AI for better accuracy
func ParseReceipt(text string) (*ReceiptData, error) {
	receipt := &ReceiptData{
		RawText: text,
		Items:   []ReceiptItem{},
	}

	lines := strings.Split(text, "\n")

	// Try to extract total amount
	receipt.TotalAmount = extractTotal(lines)

	// Try to extract date
	receipt.TransactionDate = extractDate(lines)

	// Try to extract store name (usually first few non-empty lines)
	receipt.StoreName = extractStoreName(lines)

	// Try to extract items
	receipt.Items = extractItems(lines)

	return receipt, nil
}

// extractTotal tries to find the total amount in the receipt
func extractTotal(lines []string) float64 {
	// Common patterns for total:
	// "Total: Rp 100,000"
	// "TOTAL 100000"
	// "Grand Total: Rp. 100.000"
	// "Jumlah: 100000"
	// "Total :       43,500" (BreadTalk format)

	totalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^[\s]*total[\s:]+(?:rp\.?\s*)?([0-9,.]+)`),
		regexp.MustCompile(`(?i)(grand\s*total|jumlah)[\s:]+(?:rp\.?\s*)?([0-9,.]+)`),
		regexp.MustCompile(`(?i)total[\s:]+([0-9,.]+)$`),
	}

	// Try to find "Total" line (prefer exact "Total" over "Subtotal")
	var totalAmount float64
	foundTotal := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip subtotal if we already found total
		if foundTotal && strings.Contains(strings.ToLower(line), "subtotal") {
			continue
		}

		for _, pattern := range totalPatterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				amountStr := ""
				if len(matches) >= 2 {
					// Get the last capture group (amount)
					amountStr = matches[len(matches)-1]
				}

				if amountStr != "" {
					// Remove dots and commas, parse as float
					amountStr = strings.ReplaceAll(amountStr, ".", "")
					amountStr = strings.ReplaceAll(amountStr, ",", "")
					if amount, err := strconv.ParseFloat(amountStr, 64); err == nil {
						// Prefer "Total" over "Subtotal"
						if strings.Contains(strings.ToLower(line), "total") &&
						   !strings.Contains(strings.ToLower(line), "subtotal") {
							totalAmount = amount
							foundTotal = true
							break
						} else if !foundTotal {
							totalAmount = amount
						}
					}
				}
			}
		}
	}

	return totalAmount
}

// extractDate tries to find the transaction date
func extractDate(lines []string) time.Time {
	// Common date patterns:
	// "21/11/2024"
	// "2024-11-21"
	// "21 Nov 2024"
	// "21 November 2024"
	// "10 May 19" (BreadTalk format)

	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d{1,2})[/-](\d{1,2})[/-](\d{2,4})`),           // 21/11/2024
		regexp.MustCompile(`(\d{4})[/-](\d{1,2})[/-](\d{1,2})`),             // 2024-11-21
		regexp.MustCompile(`(\d{1,2})\s+(Jan|Feb|Mar|Apr|Mei|May|Jun|Jul|Agt|Aug|Sep|Okt|Oct|Nov|Des|Dec)\w*\s+(\d{2,4})`), // 21 Nov 2024 or 10 May 19
	}

	for _, line := range lines {
		for _, pattern := range datePatterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				// Try to parse the date
				if parsedDate, err := parseDate(matches); err == nil {
					return parsedDate
				}
			}
		}
	}

	// Default to today if not found
	return time.Now()
}

// parseDate attempts to parse date from regex matches
func parseDate(matches []string) (time.Time, error) {
	if len(matches) < 3 {
		return time.Time{}, nil
	}

	// Try different formats
	dateStr := strings.Join(matches[1:], " ")

	formats := []string{
		"2/1/2006",
		"02/01/2006",
		"2006-1-2",
		"2006-01-02",
		"2 Jan 2006",
		"02 Jan 2006",
		"2 Jan 06",      // 10 May 19
		"02 Jan 06",
		"2 January 2006",
		"02 January 2006",
		"2 January 06",
		"02 January 06",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Now(), nil
}

// extractStoreName tries to extract the store name (usually in first few lines)
func extractStoreName(lines []string) string {
	// Take first non-empty line that's not a date or address
	for i := 0; i < len(lines) && i < 5; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Skip lines that look like addresses or dates
		if strings.Contains(strings.ToLower(line), "jl.") ||
			strings.Contains(strings.ToLower(line), "jalan") ||
			regexp.MustCompile(`\d{1,2}[/-]\d{1,2}[/-]\d{2,4}`).MatchString(line) {
			continue
		}

		return line
	}

	return ""
}

// extractItems tries to extract line items from the receipt
// This is a basic implementation - can be improved
func extractItems(lines []string) []ReceiptItem {
	items := []ReceiptItem{}

	// Patterns to match item lines:
	// "1 Bread Butter Pudding        11,500" (BreadTalk format - quantity first)
	// "Kopi Susu 2 x 15000 30000" (quantity after name)
	// "Nasi Goreng 1 15.000"

	itemPatterns := []*regexp.Regexp{
		// Pattern 1: "1 Item Name    11,500" (quantity at start)
		regexp.MustCompile(`^(\d+)\s+(.+?)\s+([0-9,.]+)$`),
		// Pattern 2: "Item Name 2 x 15000" or "Item Name 2 15000"
		regexp.MustCompile(`^(.+?)\s+(\d+)\s+(?:x\s+)?(?:rp\.?\s*)?([0-9,.]+)`),
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header/footer lines
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "total") ||
			strings.Contains(lowerLine, "subtotal") ||
			strings.Contains(lowerLine, "terima kasih") ||
			strings.Contains(lowerLine, "thank you") ||
			strings.Contains(lowerLine, "payment") ||
			strings.Contains(lowerLine, "payrrent") ||
			strings.Contains(lowerLine, "debit") ||
			strings.Contains(lowerLine, "cash") ||
			strings.Contains(lowerLine, "check no") ||
			strings.HasPrefix(lowerLine, "---") ||
			regexp.MustCompile(`^\d{2}\s+(jan|feb|mar|apr|mei|may|jun|jul|agt|aug|sep|okt|oct|nov|des|dec)`).MatchString(lowerLine) {
			continue
		}

		// Try each pattern
		for i, pattern := range itemPatterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				var name string
				var quantity int
				var price float64

				if i == 0 {
					// Pattern 1: quantity at start
					if len(matches) >= 4 {
						quantity, _ = strconv.Atoi(matches[1])
						name = strings.TrimSpace(matches[2])
						priceStr := strings.ReplaceAll(matches[3], ".", "")
						priceStr = strings.ReplaceAll(priceStr, ",", "")
						price, _ = strconv.ParseFloat(priceStr, 64)
					}
				} else {
					// Pattern 2: quantity after name
					if len(matches) >= 4 {
						name = strings.TrimSpace(matches[1])
						quantity, _ = strconv.Atoi(matches[2])
						priceStr := strings.ReplaceAll(matches[3], ".", "")
						priceStr = strings.ReplaceAll(priceStr, ",", "")
						price, _ = strconv.ParseFloat(priceStr, 64)
					}
				}

				// Validate and add item
				if name != "" && quantity > 0 && price > 0 {
					items = append(items, ReceiptItem{
						Name:     name,
						Quantity: quantity,
						Price:    price,
					})
					break // Don't try other patterns for this line
				}
			}
		}
	}

	return items
}

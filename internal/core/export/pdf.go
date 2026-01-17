package export

import (
	"fmt"
	"io"

	"github.com/jung-kurt/gofpdf"
)

// PDFExporter implements PDF export using gofpdf
type PDFExporter struct {
	orientation string
	pageSize    string
}

// NewPDFExporter creates a new PDF exporter
func NewPDFExporter() *PDFExporter {
	return &PDFExporter{
		orientation: "P", // Portrait
		pageSize:    "A4",
	}
}

// Export exports data to PDF format
func (p *PDFExporter) Export(data *ExportData, writer io.Writer) error {
	// Set orientation
	orientation := "P"
	if data.Style.Orientation == "landscape" {
		orientation = "L"
	}

	// Set page size
	pageSize := data.Style.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}

	// Create PDF
	pdf := gofpdf.New(orientation, "mm", pageSize, "")
	pdf.AddPage()

	// Set font
	fontFamily := data.Style.FontFamily
	if fontFamily == "" || fontFamily == "Arial" {
		fontFamily = "Arial"
		pdf.SetFont("Arial", "", data.Style.FontSize)
	} else {
		// For custom fonts, fallback to Arial
		pdf.SetFont("Arial", "", data.Style.FontSize)
	}

	// Add title
	if data.Title != "" {
		pdf.SetFont(fontFamily, "B", 16)
		pdf.Cell(0, 10, data.Title)
		pdf.Ln(12)
	}

	// Add description
	if data.Description != "" {
		pdf.SetFont(fontFamily, "", data.Style.FontSize)
		pdf.MultiCell(0, 5, data.Description, "", "", false)
		pdf.Ln(8)
	}

	// Add metadata (date, author)
	if !data.CreatedAt.IsZero() {
		pdf.SetFont(fontFamily, "I", 8)
		pdf.Cell(0, 5, fmt.Sprintf("Generated: %s", data.CreatedAt.Format("2006-01-02 15:04:05")))
		if data.Author != "" {
			pdf.Cell(0, 5, fmt.Sprintf(" | Author: %s", data.Author))
		}
		pdf.Ln(10)
	}

	// Calculate column widths
	pageWidth, _ := pdf.GetPageSize()
	leftMargin, _, rightMargin, _ := pdf.GetMargins()
	usableWidth := pageWidth - leftMargin - rightMargin

	numCols := len(data.Headers)
	if numCols == 0 {
		return fmt.Errorf("no headers provided")
	}

	colWidth := usableWidth / float64(numCols)

	// Draw header row
	pdf.SetFont(fontFamily, "B", data.Style.FontSize)
	if data.Style.HeaderBgColor != "" {
		r, g, b := hexToRGB(data.Style.HeaderBgColor)
		pdf.SetFillColor(r, g, b)
		pdf.SetTextColor(255, 255, 255) // White text
	}

	for _, header := range data.Headers {
		pdf.CellFormat(colWidth, 7, header, "1", 0, "C", data.Style.HeaderBgColor != "", 0, "")
	}
	pdf.Ln(-1)

	// Reset text color for data rows
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(fontFamily, "", data.Style.FontSize)

	// Draw data rows
	for rowIdx, row := range data.Rows {
		// Alternate row colors
		if data.Style.AlternateRows {
			if rowIdx%2 == 0 {
				r, g, b := hexToRGB(data.Style.RowBgColor1)
				pdf.SetFillColor(r, g, b)
			} else {
				r, g, b := hexToRGB(data.Style.RowBgColor2)
				pdf.SetFillColor(r, g, b)
			}
		}

		for colIdx, value := range row {
			// Convert value to string
			valueStr := fmt.Sprintf("%v", value)

			// Determine alignment
			align := "L"
			if colIdx == 0 {
				align = "L" // Left align first column
			}

			// Draw cell
			pdf.CellFormat(colWidth, 6, valueStr, "1", 0, align, data.Style.AlternateRows, 0, "")
		}
		pdf.Ln(-1)

		// Check if we need a new page
		if pdf.GetY() > 270 { // Near bottom of A4 page
			pdf.AddPage()
			// Redraw header on new page
			pdf.SetFont(fontFamily, "B", data.Style.FontSize)
			if data.Style.HeaderBgColor != "" {
				r, g, b := hexToRGB(data.Style.HeaderBgColor)
				pdf.SetFillColor(r, g, b)
				pdf.SetTextColor(255, 255, 255)
			}
			for _, header := range data.Headers {
				pdf.CellFormat(colWidth, 7, header, "1", 0, "C", data.Style.HeaderBgColor != "", 0, "")
			}
			pdf.Ln(-1)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont(fontFamily, "", data.Style.FontSize)
		}
	}

	// Write to output
	err := pdf.Output(writer)
	if err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// GetContentType returns the MIME type for PDF files
func (p *PDFExporter) GetContentType() string {
	return "application/pdf"
}

// GetFileExtension returns the file extension for PDF files
func (p *PDFExporter) GetFileExtension() string {
	return ".pdf"
}

// hexToRGB converts hex color to RGB values
func hexToRGB(hex string) (int, int, int) {
	// Remove # if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Default to white if invalid
	if len(hex) != 6 {
		return 255, 255, 255
	}

	// Parse hex
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

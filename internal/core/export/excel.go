package export

import (
	"fmt"
	"io"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// ExcelExporter implements Excel export using excelize
type ExcelExporter struct {
	sheetName string
}

// NewExcelExporter creates a new Excel exporter
func NewExcelExporter() *ExcelExporter {
	return &ExcelExporter{
		sheetName: "Sheet1",
	}
}

// Export exports data to Excel format
func (e *ExcelExporter) Export(data *ExportData, writer io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	// Set sheet name
	f.SetSheetName("Sheet1", e.sheetName)

	// Write title if provided
	rowIndex := 1
	if data.Title != "" {
		f.SetCellValue(e.sheetName, fmt.Sprintf("A%d", rowIndex), data.Title)
		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Size:   14,
				Family: data.Style.FontFamily,
			},
		})
		f.SetCellStyle(e.sheetName, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("A%d", rowIndex), titleStyle)
		rowIndex++

		// Add description if provided
		if data.Description != "" {
			f.SetCellValue(e.sheetName, fmt.Sprintf("A%d", rowIndex), data.Description)
			rowIndex++
		}
		rowIndex++ // Add blank row
	}

	// Create header style
	headerStyle, err := e.createHeaderStyle(f, data.Style)
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}

	// Write headers
	headerRow := rowIndex
	for colIndex, header := range data.Headers {
		cell := columnNumberToName(colIndex+1) + strconv.Itoa(rowIndex)
		f.SetCellValue(e.sheetName, cell, header)
		f.SetCellStyle(e.sheetName, cell, cell, headerStyle)

		// Set column width if specified
		if width, ok := data.Style.ColumnWidths[colIndex]; ok {
			colName := columnNumberToName(colIndex + 1)
			f.SetColWidth(e.sheetName, colName, colName, width)
		}
	}
	rowIndex++

	// Create alternating row styles if enabled
	var oddRowStyle, evenRowStyle int
	if data.Style.AlternateRows {
		oddRowStyle, _ = e.createRowStyle(f, data.Style, data.Style.RowBgColor1)
		evenRowStyle, _ = e.createRowStyle(f, data.Style, data.Style.RowBgColor2)
	} else {
		oddRowStyle, _ = e.createRowStyle(f, data.Style, data.Style.RowBgColor1)
		evenRowStyle = oddRowStyle
	}

	// Write data rows
	for rowIdx, row := range data.Rows {
		for colIndex, value := range row {
			cell := columnNumberToName(colIndex+1) + strconv.Itoa(rowIndex)
			f.SetCellValue(e.sheetName, cell, value)

			// Apply alternating row style
			if rowIdx%2 == 0 {
				f.SetCellStyle(e.sheetName, cell, cell, oddRowStyle)
			} else {
				f.SetCellStyle(e.sheetName, cell, cell, evenRowStyle)
			}
		}
		rowIndex++
	}

	// Freeze header row if enabled
	if data.Style.FreezeHeader {
		f.SetPanes(e.sheetName, &excelize.Panes{
			Freeze:      true,
			XSplit:      0,
			YSplit:      headerRow,
			TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
			ActivePane:  "bottomLeft",
		})
	}

	// Add auto-filter if enabled
	if data.Style.AutoFilter && len(data.Headers) > 0 {
		lastCol := columnNumberToName(len(data.Headers))
		lastRow := headerRow + len(data.Rows)
		f.AutoFilter(e.sheetName, fmt.Sprintf("A%d:%s%d", headerRow, lastCol, lastRow), nil)
	}

	// Write to output
	if err := f.Write(writer); err != nil {
		return fmt.Errorf("failed to write Excel file: %w", err)
	}

	return nil
}

// GetContentType returns the MIME type for Excel files
func (e *ExcelExporter) GetContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// GetFileExtension returns the file extension for Excel files
func (e *ExcelExporter) GetFileExtension() string {
	return ".xlsx"
}

// createHeaderStyle creates the header style
func (e *ExcelExporter) createHeaderStyle(f *excelize.File, style ExportStyle) (int, error) {
	headerStyle := &excelize.Style{
		Font: &excelize.Font{
			Bold:   style.HeaderBold,
			Size:   style.FontSize,
			Family: style.FontFamily,
			Color:  "FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{stripHashFromColor(style.HeaderBgColor)},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	return f.NewStyle(headerStyle)
}

// createRowStyle creates a row style with background color
func (e *ExcelExporter) createRowStyle(f *excelize.File, style ExportStyle, bgColor string) (int, error) {
	rowStyle := &excelize.Style{
		Font: &excelize.Font{
			Size:   style.FontSize,
			Family: style.FontFamily,
		},
	}

	// Only add fill if bgColor is not white
	if bgColor != "" && bgColor != "#FFFFFF" {
		rowStyle.Fill = excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{stripHashFromColor(bgColor)},
		}
	}

	return f.NewStyle(rowStyle)
}

// columnNumberToName converts column number to Excel column name (1 -> A, 27 -> AA)
func columnNumberToName(col int) string {
	name := ""
	for col > 0 {
		col--
		name = string(rune('A'+(col%26))) + name
		col /= 26
	}
	return name
}

// stripHashFromColor removes # from hex color codes
func stripHashFromColor(color string) string {
	if len(color) > 0 && color[0] == '#' {
		return color[1:]
	}
	return color
}

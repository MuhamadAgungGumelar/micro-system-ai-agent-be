package export

import (
	"io"
	"time"
)

// ExportFormat represents the export file format
type ExportFormat string

const (
	FormatPDF   ExportFormat = "pdf"
	FormatExcel ExportFormat = "excel"
	FormatCSV   ExportFormat = "csv"
)

// Exporter is the interface for all export formats
type Exporter interface {
	Export(data *ExportData, writer io.Writer) error
	GetContentType() string
	GetFileExtension() string
}

// ExportData represents the data to be exported
type ExportData struct {
	Title       string
	Description string
	Author      string
	CreatedAt   time.Time

	// Table data
	Headers []string
	Rows    [][]interface{}

	// Metadata
	Metadata map[string]interface{}

	// Styling options
	Style ExportStyle
}

// ExportStyle defines styling options for exports
type ExportStyle struct {
	// PDF specific
	Orientation string // "portrait" or "landscape"
	PageSize    string // "A4", "Letter", etc.

	// Common styling
	HeaderBold      bool
	HeaderBgColor   string // Hex color
	AlternateRows   bool
	RowBgColor1     string // Hex color for odd rows
	RowBgColor2     string // Hex color for even rows

	// Font settings
	FontFamily string
	FontSize   float64

	// Excel specific
	FreezeHeader bool
	AutoFilter   bool
	ColumnWidths map[int]float64 // Column index -> width
}

// DefaultStyle returns default export styling
func DefaultStyle() ExportStyle {
	return ExportStyle{
		Orientation:   "portrait",
		PageSize:      "A4",
		HeaderBold:    true,
		HeaderBgColor: "#4472C4",
		AlternateRows: true,
		RowBgColor1:   "#FFFFFF",
		RowBgColor2:   "#F2F2F2",
		FontFamily:    "Arial",
		FontSize:      10,
		FreezeHeader:  true,
		AutoFilter:    true,
		ColumnWidths:  make(map[int]float64),
	}
}

// TableData is a convenience struct for simple table exports
type TableData struct {
	Headers []string
	Rows    [][]interface{}
}

// ToExportData converts TableData to ExportData with defaults
func (t *TableData) ToExportData(title string) *ExportData {
	return &ExportData{
		Title:     title,
		CreatedAt: time.Now(),
		Headers:   t.Headers,
		Rows:      t.Rows,
		Style:     DefaultStyle(),
	}
}

// ChartData represents chart data for PDF exports
type ChartData struct {
	Type   string   // "line", "bar", "pie"
	Title  string
	Labels []string
	Series []ChartSeries
}

// ChartSeries represents a data series in a chart
type ChartSeries struct {
	Name   string
	Values []float64
	Color  string // Hex color
}

// ReportSection represents a section in a complex report
type ReportSection struct {
	Title   string
	Content string
	Table   *TableData
	Chart   *ChartData
}

// Report represents a multi-section document
type Report struct {
	Title       string
	Description string
	Author      string
	CreatedAt   time.Time
	Sections    []ReportSection
	Style       ExportStyle
}

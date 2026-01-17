package export

import (
	"bytes"
	"fmt"
	"io"
)

// Service provides high-level export functionality
type Service struct {
	pdfExporter   Exporter
	excelExporter Exporter
}

// NewService creates a new export service
func NewService() *Service {
	return &Service{
		pdfExporter:   NewPDFExporter(),
		excelExporter: NewExcelExporter(),
	}
}

// ExportToPDF exports data to PDF format
func (s *Service) ExportToPDF(data *ExportData) ([]byte, error) {
	var buf bytes.Buffer
	if err := s.pdfExporter.Export(data, &buf); err != nil {
		return nil, fmt.Errorf("PDF export failed: %w", err)
	}
	return buf.Bytes(), nil
}

// ExportToExcel exports data to Excel format
func (s *Service) ExportToExcel(data *ExportData) ([]byte, error) {
	var buf bytes.Buffer
	if err := s.excelExporter.Export(data, &buf); err != nil {
		return nil, fmt.Errorf("Excel export failed: %w", err)
	}
	return buf.Bytes(), nil
}

// Export exports data to the specified format
func (s *Service) Export(data *ExportData, format ExportFormat) ([]byte, string, error) {
	var exporter Exporter
	switch format {
	case FormatPDF:
		exporter = s.pdfExporter
	case FormatExcel:
		exporter = s.excelExporter
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}

	var buf bytes.Buffer
	if err := exporter.Export(data, &buf); err != nil {
		return nil, "", fmt.Errorf("export failed: %w", err)
	}

	return buf.Bytes(), exporter.GetContentType(), nil
}

// ExportToWriter exports data to a writer
func (s *Service) ExportToWriter(data *ExportData, format ExportFormat, writer io.Writer) error {
	var exporter Exporter
	switch format {
	case FormatPDF:
		exporter = s.pdfExporter
	case FormatExcel:
		exporter = s.excelExporter
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	return exporter.Export(data, writer)
}

// GetContentType returns the content type for the given format
func (s *Service) GetContentType(format ExportFormat) string {
	switch format {
	case FormatPDF:
		return s.pdfExporter.GetContentType()
	case FormatExcel:
		return s.excelExporter.GetContentType()
	default:
		return "application/octet-stream"
	}
}

// GetFileExtension returns the file extension for the given format
func (s *Service) GetFileExtension(format ExportFormat) string {
	switch format {
	case FormatPDF:
		return s.pdfExporter.GetFileExtension()
	case FormatExcel:
		return s.excelExporter.GetFileExtension()
	default:
		return ".bin"
	}
}

// ExportTableToPDF is a convenience method for simple table exports
func (s *Service) ExportTableToPDF(title string, headers []string, rows [][]interface{}) ([]byte, error) {
	data := &ExportData{
		Title:   title,
		Headers: headers,
		Rows:    rows,
		Style:   DefaultStyle(),
	}
	return s.ExportToPDF(data)
}

// ExportTableToExcel is a convenience method for simple table exports
func (s *Service) ExportTableToExcel(title string, headers []string, rows [][]interface{}) ([]byte, error) {
	data := &ExportData{
		Title:   title,
		Headers: headers,
		Rows:    rows,
		Style:   DefaultStyle(),
	}
	return s.ExportToExcel(data)
}

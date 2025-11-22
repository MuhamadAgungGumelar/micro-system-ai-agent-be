package ocr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TesseractProvider implements OCR using Tesseract OCR engine
type TesseractProvider struct {
	tesseractPath string
	language      string
}

// NewTesseractProvider creates a new Tesseract OCR provider
// language can be "eng", "ind" (Indonesian), or "eng+ind" for both
func NewTesseractProvider(language string) *TesseractProvider {
	if language == "" {
		language = "eng" // Default to English
	}

	return &TesseractProvider{
		tesseractPath: "tesseract", // Assumes tesseract is in PATH
		language:      language,
	}
}

// ExtractText extracts text from an image using Tesseract
func (p *TesseractProvider) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	// Create temporary file for image
	tempDir := os.TempDir()
	tempImagePath := filepath.Join(tempDir, fmt.Sprintf("ocr_image_%d.jpg", os.Getpid()))
	tempOutputPath := filepath.Join(tempDir, fmt.Sprintf("ocr_output_%d", os.Getpid()))

	// Write image data to temporary file
	if err := os.WriteFile(tempImagePath, imageData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp image: %w", err)
	}
	defer os.Remove(tempImagePath)

	// Run tesseract command
	// tesseract input.jpg output -l eng
	cmd := exec.CommandContext(ctx, p.tesseractPath, tempImagePath, tempOutputPath, "-l", p.language)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tesseract command failed: %w, output: %s", err, string(output))
	}

	// Read the output file (tesseract adds .txt extension automatically)
	outputFilePath := tempOutputPath + ".txt"
	defer os.Remove(outputFilePath)

	textBytes, err := os.ReadFile(outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tesseract output: %w", err)
	}

	text := strings.TrimSpace(string(textBytes))

	// Tesseract doesn't provide confidence per page via stdout by default
	// We'll use a fixed high confidence or parse it separately if needed
	confidence := 0.90 // Default confidence for Tesseract

	return &OCRResult{
		Text:       text,
		Confidence: confidence,
	}, nil
}

// GetProviderName returns the name of the provider
func (p *TesseractProvider) GetProviderName() string {
	return "Tesseract OCR"
}

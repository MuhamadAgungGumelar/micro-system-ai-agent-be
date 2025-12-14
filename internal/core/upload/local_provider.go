package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LocalProvider implements file upload to local filesystem
type LocalProvider struct {
	basePath   string // Base directory for uploads
	baseURL    string // Base URL to access files
	publicPath string // Public path for URL generation
}

// NewLocalProvider creates a new local file storage provider
func NewLocalProvider(basePath, baseURL string) *LocalProvider {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create upload directory: %v", err))
	}

	return &LocalProvider{
		basePath:   basePath,
		baseURL:    baseURL,
		publicPath: "/uploads/",
	}
}

// Upload uploads a file to local filesystem
func (p *LocalProvider) Upload(file io.Reader, filename string, options *UploadOptions) (*UploadResult, error) {
	options = MergeOptions(options)

	// Generate unique filename
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	var finalFilename string
	if options.PublicID != "" {
		finalFilename = options.PublicID + ext
	} else {
		uniqueID := uuid.New().String()[:8]
		finalFilename = fmt.Sprintf("%s_%d_%s%s", nameWithoutExt, time.Now().Unix(), uniqueID, ext)
	}

	// Create folder path
	folderPath := filepath.Join(p.basePath, options.Folder)
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	// Full file path
	filePath := filepath.Join(folderPath, finalFilename)

	// Check if file exists and overwrite is false
	if !options.Overwrite {
		if _, err := os.Stat(filePath); err == nil {
			return nil, fmt.Errorf("file already exists: %s", finalFilename)
		}
	}

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy the file content
	size, err := io.Copy(out, file)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Check file size
	if options.MaxSize > 0 && size > options.MaxSize {
		os.Remove(filePath) // Remove the file
		return nil, fmt.Errorf("file size exceeds maximum allowed size: %d bytes", options.MaxSize)
	}

	// Generate public URL
	publicURL := p.baseURL + p.publicPath + options.Folder + "/" + finalFilename
	publicID := options.Folder + "/" + finalFilename

	return &UploadResult{
		URL:          publicURL,
		SecureURL:    publicURL,
		FileName:     filename,
		Size:         size,
		Format:       strings.TrimPrefix(ext, "."),
		ResourceType: p.detectResourceType(ext),
		PublicID:     publicID,
	}, nil
}

// UploadMultipart uploads a file from multipart form
func (p *LocalProvider) UploadMultipart(fileHeader *multipart.FileHeader, options *UploadOptions) (*UploadResult, error) {
	options = MergeOptions(options)

	// Validate MIME type
	if len(options.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range options.AllowedTypes {
			if fileHeader.Header.Get("Content-Type") == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("file type not allowed: %s", fileHeader.Header.Get("Content-Type"))
		}
	}

	// Validate file size
	if options.MaxSize > 0 && fileHeader.Size > options.MaxSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size: %d bytes", options.MaxSize)
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Upload the file
	return p.Upload(file, fileHeader.Filename, options)
}

// Delete deletes a file from local filesystem
func (p *LocalProvider) Delete(publicID string) error {
	filePath := filepath.Join(p.basePath, publicID)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", publicID)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetURL gets the public URL for a file
func (p *LocalProvider) GetURL(publicID string) string {
	return p.baseURL + p.publicPath + publicID
}

// GetProviderName returns the provider name
func (p *LocalProvider) GetProviderName() string {
	return "Local Storage"
}

// detectResourceType detects the resource type based on file extension
func (p *LocalProvider) detectResourceType(ext string) string {
	ext = strings.ToLower(ext)

	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".bmp": true, ".svg": true,
	}

	videoExts := map[string]bool{
		".mp4": true, ".avi": true, ".mov": true, ".wmv": true,
		".flv": true, ".webm": true, ".mkv": true,
	}

	if imageExts[ext] {
		return "image"
	}
	if videoExts[ext] {
		return "video"
	}

	return "raw"
}

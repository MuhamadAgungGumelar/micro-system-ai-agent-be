package upload

import (
	"io"
	"mime/multipart"
)

// UploadResult represents the result of a file upload
type UploadResult struct {
	URL          string `json:"url"`           // Public URL to access the file
	SecureURL    string `json:"secure_url"`    // HTTPS URL (for Cloudinary)
	FileName     string `json:"file_name"`     // Original filename
	Size         int64  `json:"size"`          // File size in bytes
	Format       string `json:"format"`        // File extension/format
	ResourceType string `json:"resource_type"` // image, video, raw, etc.
	PublicID     string `json:"public_id"`     // Provider-specific identifier
}

// UploadOptions represents upload configuration options
type UploadOptions struct {
	Folder       string   `json:"folder"`        // Folder/directory to upload to
	PublicID     string   `json:"public_id"`     // Custom public ID
	Overwrite    bool     `json:"overwrite"`     // Overwrite existing file
	ResourceType string   `json:"resource_type"` // image, video, raw, auto
	AllowedTypes []string `json:"allowed_types"` // Allowed MIME types
	MaxSize      int64    `json:"max_size"`      // Max file size in bytes
	// Image-specific options
	Width  int  `json:"width"`  // Resize width
	Height int  `json:"height"` // Resize height
	Crop   bool `json:"crop"`   // Enable cropping
}

// Provider defines the interface for file upload providers
type Provider interface {
	// Upload uploads a file and returns the result
	Upload(file io.Reader, filename string, options *UploadOptions) (*UploadResult, error)

	// UploadMultipart uploads a file from multipart form
	UploadMultipart(fileHeader *multipart.FileHeader, options *UploadOptions) (*UploadResult, error)

	// Delete deletes a file by public ID
	Delete(publicID string) error

	// GetURL gets the public URL for a file
	GetURL(publicID string) string

	// GetProviderName returns the provider name
	GetProviderName() string
}

// DefaultUploadOptions returns default upload options
func DefaultUploadOptions() *UploadOptions {
	return &UploadOptions{
		Folder:       "uploads",
		Overwrite:    false,
		ResourceType: "auto",
		AllowedTypes: []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"},
		MaxSize:      10 * 1024 * 1024, // 10MB
		Width:        0,
		Height:       0,
		Crop:         false,
	}
}

// MergeOptions merges custom options with defaults
func MergeOptions(custom *UploadOptions) *UploadOptions {
	defaults := DefaultUploadOptions()

	if custom == nil {
		return defaults
	}

	if custom.Folder != "" {
		defaults.Folder = custom.Folder
	}
	if custom.PublicID != "" {
		defaults.PublicID = custom.PublicID
	}
	if custom.ResourceType != "" {
		defaults.ResourceType = custom.ResourceType
	}
	if len(custom.AllowedTypes) > 0 {
		defaults.AllowedTypes = custom.AllowedTypes
	}
	if custom.MaxSize > 0 {
		defaults.MaxSize = custom.MaxSize
	}
	if custom.Width > 0 {
		defaults.Width = custom.Width
	}
	if custom.Height > 0 {
		defaults.Height = custom.Height
	}

	defaults.Overwrite = custom.Overwrite
	defaults.Crop = custom.Crop

	return defaults
}

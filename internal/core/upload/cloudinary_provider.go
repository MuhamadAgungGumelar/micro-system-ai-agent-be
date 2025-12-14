package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryProvider implements file upload to Cloudinary
type CloudinaryProvider struct {
	cld      *cloudinary.Cloudinary
	cloudName string
}

// NewCloudinaryProvider creates a new Cloudinary provider
func NewCloudinaryProvider(cloudName, apiKey, apiSecret string) (*CloudinaryProvider, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	return &CloudinaryProvider{
		cld:      cld,
		cloudName: cloudName,
	}, nil
}

// Upload uploads a file to Cloudinary
func (p *CloudinaryProvider) Upload(file io.Reader, filename string, options *UploadOptions) (*UploadResult, error) {
	options = MergeOptions(options)

	ctx := context.Background()

	// Build upload parameters
	params := uploader.UploadParams{
		Folder:       options.Folder,
		ResourceType: options.ResourceType,
		Overwrite:    &options.Overwrite,
	}

	if options.PublicID != "" {
		params.PublicID = options.PublicID
	}

	// Add transformation parameters if specified
	if options.Width > 0 || options.Height > 0 {
		transformation := ""
		if options.Width > 0 {
			transformation += fmt.Sprintf("w_%d", options.Width)
		}
		if options.Height > 0 {
			if transformation != "" {
				transformation += ","
			}
			transformation += fmt.Sprintf("h_%d", options.Height)
		}
		if options.Crop {
			if transformation != "" {
				transformation += ","
			}
			transformation += "c_fill"
		}
		params.Transformation = transformation
	}

	// Upload to Cloudinary
	result, err := p.cld.Upload.Upload(ctx, file, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to Cloudinary: %w", err)
	}

	return &UploadResult{
		URL:          result.URL,
		SecureURL:    result.SecureURL,
		FileName:     filename,
		Size:         int64(result.Bytes),
		Format:       result.Format,
		ResourceType: result.ResourceType,
		PublicID:     result.PublicID,
	}, nil
}

// UploadMultipart uploads a file from multipart form to Cloudinary
func (p *CloudinaryProvider) UploadMultipart(fileHeader *multipart.FileHeader, options *UploadOptions) (*UploadResult, error) {
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

// Delete deletes a file from Cloudinary
func (p *CloudinaryProvider) Delete(publicID string) error {
	ctx := context.Background()

	params := uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image", // Default to image, could be made configurable
	}

	result, err := p.cld.Upload.Destroy(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to delete from Cloudinary: %w", err)
	}

	if result.Result != "ok" {
		return fmt.Errorf("Cloudinary delete failed: %s", result.Result)
	}

	return nil
}

// GetURL gets the public URL for a file from Cloudinary
func (p *CloudinaryProvider) GetURL(publicID string) string {
	// Generate Cloudinary URL
	url := fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s", p.cloudName, publicID)
	return url
}

// GetProviderName returns the provider name
func (p *CloudinaryProvider) GetProviderName() string {
	return "Cloudinary"
}

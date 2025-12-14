package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Provider implements file upload to AWS S3
type S3Provider struct {
	client     *s3.Client
	bucketName string
	region     string
	baseURL    string // Base URL for accessing files (e.g., CloudFront)
}

// NewS3Provider creates a new AWS S3 provider
func NewS3Provider(accessKeyID, secretAccessKey, region, bucketName string) (*S3Provider, error) {
	ctx := context.Background()

	// Load AWS config with credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Generate base URL
	baseURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, region)

	return &S3Provider{
		client:     client,
		bucketName: bucketName,
		region:     region,
		baseURL:    baseURL,
	}, nil
}

// Upload uploads a file to AWS S3
func (p *S3Provider) Upload(file io.Reader, filename string, options *UploadOptions) (*UploadResult, error) {
	options = MergeOptions(options)

	ctx := context.Background()

	// Generate unique key
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	var key string
	if options.PublicID != "" {
		key = filepath.Join(options.Folder, options.PublicID+ext)
	} else {
		uniqueID := uuid.New().String()[:8]
		finalFilename := fmt.Sprintf("%s_%d_%s%s", nameWithoutExt, time.Now().Unix(), uniqueID, ext)
		key = filepath.Join(options.Folder, finalFilename)
	}

	// Convert Windows path separators to Unix style for S3
	key = strings.ReplaceAll(key, "\\", "/")

	// Detect content type
	contentType := p.detectContentType(ext)

	// Upload to S3
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucketName),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		ACL:         "public-read", // Make file publicly accessible
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate public URL
	publicURL := fmt.Sprintf("%s/%s", p.baseURL, key)

	// Get file info (we can't get size from PutObject, so we'll estimate)
	// In production, you might want to read the file size before uploading
	size := int64(0) // Placeholder

	return &UploadResult{
		URL:          publicURL,
		SecureURL:    publicURL,
		FileName:     filename,
		Size:         size,
		Format:       strings.TrimPrefix(ext, "."),
		ResourceType: p.detectResourceType(ext),
		PublicID:     key,
	}, nil
}

// UploadMultipart uploads a file from multipart form to S3
func (p *S3Provider) UploadMultipart(fileHeader *multipart.FileHeader, options *UploadOptions) (*UploadResult, error) {
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
	result, err := p.Upload(file, fileHeader.Filename, options)
	if err != nil {
		return nil, err
	}

	// Update size with actual file size
	result.Size = fileHeader.Size

	return result, nil
}

// Delete deletes a file from AWS S3
func (p *S3Provider) Delete(publicID string) error {
	ctx := context.Background()

	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucketName),
		Key:    aws.String(publicID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GetURL gets the public URL for a file from S3
func (p *S3Provider) GetURL(publicID string) string {
	return fmt.Sprintf("%s/%s", p.baseURL, publicID)
}

// GetProviderName returns the provider name
func (p *S3Provider) GetProviderName() string {
	return "AWS S3"
}

// detectContentType detects the content type based on file extension
func (p *S3Provider) detectContentType(ext string) string {
	ext = strings.ToLower(ext)

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}

	return "application/octet-stream"
}

// detectResourceType detects the resource type based on file extension
func (p *S3Provider) detectResourceType(ext string) string {
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

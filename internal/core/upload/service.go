package upload

import (
	"fmt"
	"io"
	"mime/multipart"
)

// Service provides file upload functionality with provider switching
type Service struct {
	provider     Provider
	providerName string
}

// NewService creates a new upload service
func NewService(provider Provider) *Service {
	return &Service{
		provider:     provider,
		providerName: provider.GetProviderName(),
	}
}

// Upload uploads a file using the configured provider
func (s *Service) Upload(file io.Reader, filename string, options *UploadOptions) (*UploadResult, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("upload provider not configured")
	}

	return s.provider.Upload(file, filename, options)
}

// UploadMultipart uploads a file from multipart form
func (s *Service) UploadMultipart(fileHeader *multipart.FileHeader, options *UploadOptions) (*UploadResult, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("upload provider not configured")
	}

	return s.provider.UploadMultipart(fileHeader, options)
}

// Delete deletes a file by public ID
func (s *Service) Delete(publicID string) error {
	if s.provider == nil {
		return fmt.Errorf("upload provider not configured")
	}

	return s.provider.Delete(publicID)
}

// GetURL gets the public URL for a file
func (s *Service) GetURL(publicID string) string {
	if s.provider == nil {
		return ""
	}

	return s.provider.GetURL(publicID)
}

// GetProviderName returns the current provider name
func (s *Service) GetProviderName() string {
	return s.providerName
}

// SetProvider changes the upload provider
func (s *Service) SetProvider(provider Provider) {
	s.provider = provider
	s.providerName = provider.GetProviderName()
}

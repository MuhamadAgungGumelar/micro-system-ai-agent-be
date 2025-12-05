package email

import (
	"fmt"
)

// Provider defines the interface for email providers
type Provider interface {
	SendEmail(to, subject, body string) error
	SendTemplateEmail(to, subject string, templateData map[string]interface{}) error
	GetProviderName() string
}

// Service wraps the email provider
type Service struct {
	provider Provider
}

// NewService creates a new email service with the specified provider
func NewService(provider Provider) *Service {
	return &Service{
		provider: provider,
	}
}

// SendEmail sends a plain text or HTML email
func (s *Service) SendEmail(to, subject, body string) error {
	if s.provider == nil {
		return fmt.Errorf("no email provider configured")
	}
	return s.provider.SendEmail(to, subject, body)
}

// SendTemplateEmail sends an email using a template
func (s *Service) SendTemplateEmail(to, subject string, templateData map[string]interface{}) error {
	if s.provider == nil {
		return fmt.Errorf("no email provider configured")
	}
	return s.provider.SendTemplateEmail(to, subject, templateData)
}

// GetProviderName returns the name of the current provider
func (s *Service) GetProviderName() string {
	if s.provider == nil {
		return "none"
	}
	return s.provider.GetProviderName()
}

// EmailMessage represents a structured email message
type EmailMessage struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

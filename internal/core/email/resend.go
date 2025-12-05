package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ResendProvider implements email sending via Resend API
type ResendProvider struct {
	apiKey     string
	fromEmail  string
	fromName   string
	httpClient *http.Client
}

// NewResendProvider creates a new Resend email provider
func NewResendProvider(apiKey, fromEmail, fromName string) *ResendProvider {
	return &ResendProvider{
		apiKey:     apiKey,
		fromEmail:  fromEmail,
		fromName:   fromName,
		httpClient: &http.Client{},
	}
}

type resendEmailRequest struct {
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html,omitempty"`
	Text    string `json:"text,omitempty"`
}

// SendEmail sends an email via Resend API
func (p *ResendProvider) SendEmail(to, subject, body string) error {
	fromAddress := p.fromEmail
	if p.fromName != "" {
		fromAddress = fmt.Sprintf("%s <%s>", p.fromName, p.fromEmail)
	}

	reqBody := resendEmailRequest{
		From:    fromAddress,
		To:      []string{to},
		Subject: subject,
		HTML:    body,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendTemplateEmail sends an email using template data
func (p *ResendProvider) SendTemplateEmail(to, subject string, templateData map[string]interface{}) error {
	// Build HTML from template data
	htmlContent := buildHTMLFromTemplate(templateData)
	return p.SendEmail(to, subject, htmlContent)
}

// GetProviderName returns the provider name
func (p *ResendProvider) GetProviderName() string {
	return "resend"
}

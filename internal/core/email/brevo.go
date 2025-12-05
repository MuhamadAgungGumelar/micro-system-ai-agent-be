package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BrevoProvider implements email sending via Brevo (formerly Sendinblue)
type BrevoProvider struct {
	apiKey     string
	fromEmail  string
	fromName   string
	httpClient *http.Client
}

// NewBrevoProvider creates a new Brevo email provider
func NewBrevoProvider(apiKey, fromEmail, fromName string) *BrevoProvider {
	return &BrevoProvider{
		apiKey:     apiKey,
		fromEmail:  fromEmail,
		fromName:   fromName,
		httpClient: &http.Client{},
	}
}

type brevoEmailRequest struct {
	Sender  brevoContact   `json:"sender"`
	To      []brevoContact `json:"to"`
	Subject string         `json:"subject"`
	HTMLContent string     `json:"htmlContent,omitempty"`
	TextContent string     `json:"textContent,omitempty"`
}

type brevoContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendEmail sends an email via Brevo API
func (p *BrevoProvider) SendEmail(to, subject, body string) error {
	reqBody := brevoEmailRequest{
		Sender: brevoContact{
			Email: p.fromEmail,
			Name:  p.fromName,
		},
		To: []brevoContact{
			{Email: to},
		},
		Subject:     subject,
		HTMLContent: body,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendTemplateEmail sends an email using template data
func (p *BrevoProvider) SendTemplateEmail(to, subject string, templateData map[string]interface{}) error {
	// Build HTML from template data
	htmlContent := buildHTMLFromTemplate(templateData)
	return p.SendEmail(to, subject, htmlContent)
}

// GetProviderName returns the provider name
func (p *BrevoProvider) GetProviderName() string {
	return "brevo"
}

// buildHTMLFromTemplate creates a simple HTML email from template data
func buildHTMLFromTemplate(data map[string]interface{}) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { padding: 10px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + getStringValue(data, "title", "Notification") + `</h1>
        </div>
        <div class="content">
            <p>` + getStringValue(data, "message", "") + `</p>
        </div>
        <div class="footer">
            <p>Sent from WhatsApp Bot SaaS</p>
        </div>
    </div>
</body>
</html>`
	return html
}

func getStringValue(data map[string]interface{}, key, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"gorm.io/gorm"
)

// ActionExecutor executes workflow actions
type ActionExecutor struct {
	db         *gorm.DB
	waService  *whatsapp.Service
	llmService *llm.Service
	httpClient *http.Client
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(db *gorm.DB, waService *whatsapp.Service, llmService *llm.Service) *ActionExecutor {
	return &ActionExecutor{
		db:         db,
		waService:  waService,
		llmService: llmService,
		httpClient: &http.Client{},
	}
}

// Execute executes a single action with the given context data
func (e *ActionExecutor) Execute(ctx context.Context, action Action, contextData map[string]interface{}) error {
	log.Printf("🔧 Executing action: %s", action.Type)

	switch action.Type {
	case "send_whatsapp":
		return e.executeSendWhatsApp(ctx, action, contextData)

	case "update_database":
		return e.executeUpdateDatabase(ctx, action, contextData)

	case "call_api":
		return e.executeCallAPI(ctx, action, contextData)

	case "call_llm":
		return e.executeCallLLM(ctx, action, contextData)

	case "log_message":
		return e.executeLogMessage(action, contextData)

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeSendWhatsApp sends a WhatsApp message
func (e *ActionExecutor) executeSendWhatsApp(ctx context.Context, action Action, contextData map[string]interface{}) error {
	// Get session ID from config or context
	sessionID, ok := action.Config["session_id"].(string)
	if !ok || sessionID == "" {
		// Try to get from context
		if sid, exists := contextData["session_id"]; exists {
			sessionID, _ = sid.(string)
		}
	}

	if sessionID == "" {
		return fmt.Errorf("session_id is required for send_whatsapp action")
	}

	// Get recipient
	recipient, ok := action.Config["recipient"].(string)
	if !ok || recipient == "" {
		// Try to get from context (e.g., sender phone number)
		if rec, exists := contextData["from"]; exists {
			recipient, _ = rec.(string)
		}
	}

	if recipient == "" {
		return fmt.Errorf("recipient is required for send_whatsapp action")
	}

	// Get message template
	messageTemplate, ok := action.Config["message"].(string)
	if !ok {
		messageTemplate, ok = action.Config["template"].(string)
		if !ok {
			return fmt.Errorf("message or template is required for send_whatsapp action")
		}
	}

	// Replace variables in template with context data
	message := e.replaceVariables(messageTemplate, contextData)

	// Send WhatsApp message
	log.Printf("📤 Sending WhatsApp to %s: %s", recipient, message)

	err := e.waService.SendMessage(recipient, message)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	log.Printf("✅ WhatsApp message sent successfully")
	return nil
}

// executeUpdateDatabase updates a database record
func (e *ActionExecutor) executeUpdateDatabase(ctx context.Context, action Action, contextData map[string]interface{}) error {
	table, ok := action.Config["table"].(string)
	if !ok || table == "" {
		return fmt.Errorf("table is required for update_database action")
	}

	updates, ok := action.Config["updates"].(map[string]interface{})
	if !ok || len(updates) == 0 {
		return fmt.Errorf("updates is required for update_database action")
	}

	// Get WHERE conditions
	where, ok := action.Config["where"].(map[string]interface{})
	if !ok || len(where) == 0 {
		return fmt.Errorf("where is required for update_database action")
	}

	// Build query
	query := e.db.Table(table)

	// Add WHERE clauses
	for field, value := range where {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	// Execute update
	result := query.Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("database update failed: %w", result.Error)
	}

	log.Printf("✅ Updated %d rows in table %s", result.RowsAffected, table)
	return nil
}

// executeCallAPI calls an external API
func (e *ActionExecutor) executeCallAPI(ctx context.Context, action Action, contextData map[string]interface{}) error {
	url, ok := action.Config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required for call_api action")
	}

	method, ok := action.Config["method"].(string)
	if !ok || method == "" {
		method = "POST" // Default to POST
	}

	// Get body
	body := action.Config["body"]
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	if headers, ok := action.Config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Set default Content-Type if not specified
	if req.Header.Get("Content-Type") == "" && bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	log.Printf("🌐 Calling API: %s %s", method, url)
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("✅ API call successful: %d", resp.StatusCode)
	return nil
}

// executeCallLLM calls the LLM service
func (e *ActionExecutor) executeCallLLM(ctx context.Context, action Action, contextData map[string]interface{}) error {
	systemPrompt, _ := action.Config["system_prompt"].(string)
	userPrompt, ok := action.Config["user_prompt"].(string)
	if !ok || userPrompt == "" {
		return fmt.Errorf("user_prompt is required for call_llm action")
	}

	// Replace variables in prompts
	systemPrompt = e.replaceVariables(systemPrompt, contextData)
	userPrompt = e.replaceVariables(userPrompt, contextData)

	// Call LLM
	log.Printf("🤖 Calling LLM with prompt: %s", userPrompt[:min(100, len(userPrompt))])
	response, err := e.llmService.GenerateResponse(ctx, systemPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Store response in context for next actions (if needed)
	contextData["llm_response"] = response

	log.Printf("✅ LLM call successful")
	return nil
}

// executeLogMessage logs a message
func (e *ActionExecutor) executeLogMessage(action Action, contextData map[string]interface{}) error {
	message, ok := action.Config["message"].(string)
	if !ok || message == "" {
		return fmt.Errorf("message is required for log_message action")
	}

	// Replace variables
	message = e.replaceVariables(message, contextData)

	log.Printf("📝 Workflow Log: %s", message)
	return nil
}

// replaceVariables replaces {variable} placeholders with actual values from context
func (e *ActionExecutor) replaceVariables(template string, contextData map[string]interface{}) string {
	// Find all {variable} patterns
	re := regexp.MustCompile(`\{([^}]+)\}`)

	result := re.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name (remove { and })
		varName := strings.Trim(match, "{}")

		// Look up value in context data
		if value, exists := contextData[varName]; exists {
			return fmt.Sprintf("%v", value)
		}

		// Return original if not found
		return match
	})

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

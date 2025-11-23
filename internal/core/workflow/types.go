package workflow

import "time"

// TriggerConfig represents the configuration for a workflow trigger
type TriggerConfig struct {
	EventName string `json:"event_name,omitempty"` // For event triggers: "transaction_created", "message_received", etc.
	Schedule  string `json:"schedule,omitempty"`   // For scheduled triggers: cron expression "0 18 * * *"
}

// Condition represents a single condition to evaluate
type Condition struct {
	Field    string      `json:"field"`           // Field to check (e.g., "total_amount", "customer_type")
	Operator string      `json:"operator"`        // Operator: "equals", "greater_than", "less_than", "contains", etc.
	Value    interface{} `json:"value"`           // Value to compare against
	Logic    string      `json:"logic,omitempty"` // "AND" or "OR" (default: "AND")
}

// Action represents a single action to execute
type Action struct {
	Type   string                 `json:"type"`   // Action type: "send_whatsapp", "update_database", "call_api", etc.
	Config map[string]interface{} `json:"config"` // Action-specific configuration
}

// ExecutionLogEntry represents a single log entry during workflow execution
type ExecutionLogEntry struct {
	Timestamp  time.Time   `json:"timestamp"`
	Step       string      `json:"step"` // "condition_check", "action_execute", etc.
	ActionType string      `json:"action_type,omitempty"`
	Status     string      `json:"status"` // "success", "failed", "skipped"
	Message    string      `json:"message"`
	Error      string      `json:"error,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

// CreateWorkflowRequest represents the request body for creating a workflow
type CreateWorkflowRequest struct {
	Name          string        `json:"name" validate:"required"`
	Description   string        `json:"description"`
	TriggerType   string        `json:"trigger_type" validate:"required,oneof=event scheduled manual"`
	TriggerConfig TriggerConfig `json:"trigger_config" validate:"required"`
	Conditions    []Condition   `json:"conditions"`
	Actions       []Action      `json:"actions" validate:"required,min=1"`
	IsActive      *bool         `json:"is_active"` // Pointer to allow explicit false
}

// UpdateWorkflowRequest represents the request body for updating a workflow
type UpdateWorkflowRequest struct {
	Name          *string        `json:"name"`
	Description   *string        `json:"description"`
	TriggerType   *string        `json:"trigger_type" validate:"omitempty,oneof=event scheduled manual"`
	TriggerConfig *TriggerConfig `json:"trigger_config"`
	Conditions    []Condition    `json:"conditions"`
	Actions       []Action       `json:"actions" validate:"omitempty,min=1"`
	IsActive      *bool          `json:"is_active"`
}

// WorkflowExecutionRequest represents the request to manually execute a workflow
type WorkflowExecutionRequest struct {
	TriggerData map[string]interface{} `json:"trigger_data"`
}

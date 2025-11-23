package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Workflow represents an automation rule for a SaaS client
type Workflow struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID      uuid.UUID      `json:"client_id" gorm:"type:uuid;not null;index"`
	Name          string         `json:"name" gorm:"type:varchar(255);not null"`
	Description   string         `json:"description" gorm:"type:text"`
	TriggerType   string         `json:"trigger_type" gorm:"type:varchar(50);not null;index"` // 'event', 'scheduled', 'manual'
	TriggerConfig datatypes.JSON `json:"trigger_config" gorm:"type:jsonb;not null;default:'{}'"`
	Conditions    datatypes.JSON `json:"conditions" gorm:"type:jsonb;default:'[]'"`
	Actions       datatypes.JSON `json:"actions" gorm:"type:jsonb;not null;default:'[]'"`
	IsActive      bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt     time.Time      `json:"created_at" gorm:"autoCreateTime;index:,sort:desc"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for Workflow
func (Workflow) TableName() string {
	return "saas_workflows"
}

// WorkflowExecution represents a single execution of a workflow
type WorkflowExecution struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkflowID       uuid.UUID      `json:"workflow_id" gorm:"type:uuid;not null;index"`
	TriggerData      datatypes.JSON `json:"trigger_data" gorm:"type:jsonb"`
	Status           string         `json:"status" gorm:"type:varchar(50);not null;default:'pending';index"` // 'pending', 'running', 'completed', 'failed'
	ActionsCompleted int            `json:"actions_completed" gorm:"default:0"`
	ActionsFailed    int            `json:"actions_failed" gorm:"default:0"`
	ExecutionLog     datatypes.JSON `json:"execution_log" gorm:"type:jsonb;default:'[]'"`
	ErrorMessage     string         `json:"error_message,omitempty" gorm:"type:text"`
	StartedAt        time.Time      `json:"started_at" gorm:"autoCreateTime;index:,sort:desc"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	DurationMs       int            `json:"duration_ms,omitempty"` // Duration in milliseconds
}

// TableName specifies the table name for WorkflowExecution
func (WorkflowExecution) TableName() string {
	return "saas_workflow_executions"
}

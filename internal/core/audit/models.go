package audit

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AuditLog represents a system audit log entry
type AuditLog struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Context
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	ClientID uuid.UUID `json:"client_id" gorm:"type:uuid;index"`

	// Action details
	Action   string `json:"action" gorm:"type:text;not null;index"` // create, update, delete, view, login, logout
	Entity   string `json:"entity" gorm:"type:text;not null;index"` // product, user, order, etc.
	EntityID string `json:"entity_id" gorm:"type:text;index"`       // ID of the affected entity

	// Change tracking
	OldValue datatypes.JSON `json:"old_value,omitempty" gorm:"type:jsonb"` // Previous state
	NewValue datatypes.JSON `json:"new_value,omitempty" gorm:"type:jsonb"` // New state

	// Request metadata
	IPAddress string `json:"ip_address,omitempty" gorm:"type:text"`
	UserAgent string `json:"user_agent,omitempty" gorm:"type:text"`
	Method    string `json:"method,omitempty" gorm:"type:text"`     // HTTP method (GET, POST, etc.)
	Endpoint  string `json:"endpoint,omitempty" gorm:"type:text"`   // API endpoint
	Duration  int64  `json:"duration,omitempty" gorm:"type:bigint"` // Request duration in ms

	// Additional metadata
	Description string         `json:"description,omitempty" gorm:"type:text"` // Human-readable description
	Metadata    datatypes.JSON `json:"metadata,omitempty" gorm:"type:jsonb"`   // Additional context

	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

// TableName specifies the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditFilter represents filters for querying audit logs
type AuditFilter struct {
	ClientID  *uuid.UUID
	UserID    *uuid.UUID
	Action    string
	Entity    string
	EntityID  string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PageSize  int
}

// AuditLogResponse represents paginated audit log response
type AuditLogResponse struct {
	Logs       []AuditLog `json:"logs"`
	TotalCount int64      `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

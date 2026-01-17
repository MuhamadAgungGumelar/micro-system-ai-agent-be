package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
	StatusCancelled  JobStatus = "cancelled"
)

// JobPriority represents the priority of a job
type JobPriority int

const (
	PriorityLow    JobPriority = 0
	PriorityNormal JobPriority = 5
	PriorityHigh   JobPriority = 10
	PriorityCritical JobPriority = 20
)

// Job represents a background job in the database
type Job struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ClientID  uuid.UUID      `gorm:"type:uuid;not null;index"`
	Queue     string         `gorm:"type:varchar(100);not null;index"`
	Type      string         `gorm:"type:varchar(100);not null"`
	Payload   datatypes.JSON `gorm:"type:jsonb"`

	Status    JobStatus   `gorm:"type:varchar(20);not null;default:'pending';index"`
	Priority  JobPriority `gorm:"type:int;not null;default:5;index"`

	Attempts  int       `gorm:"not null;default:0"`
	MaxRetries int      `gorm:"not null;default:3"`

	ScheduledAt *time.Time `gorm:"index"` // For delayed jobs
	StartedAt   *time.Time
	CompletedAt *time.Time
	FailedAt    *time.Time

	Error     string         `gorm:"type:text"`
	Result    datatypes.JSON `gorm:"type:jsonb"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName specifies the table name for Job model
func (Job) TableName() string {
	return "jobs"
}

// JobHandler is the interface that job handlers must implement
type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
	GetType() string
}

// JobPayload is a convenience interface for job payloads
type JobPayload interface {
	Validate() error
}

// EnqueueOptions contains options for enqueueing a job
type EnqueueOptions struct {
	Queue      string
	Priority   JobPriority
	MaxRetries int
	ScheduleAt *time.Time
	Metadata   map[string]interface{}
}

// DefaultEnqueueOptions returns default enqueue options
func DefaultEnqueueOptions() EnqueueOptions {
	return EnqueueOptions{
		Queue:      "default",
		Priority:   PriorityNormal,
		MaxRetries: 3,
	}
}

// JobFilter contains options for filtering jobs
type JobFilter struct {
	ClientID *uuid.UUID
	Queue    string
	Type     string
	Status   JobStatus
	Priority *JobPriority
	Limit    int
}

// JobStats represents statistics about jobs
type JobStats struct {
	TotalJobs       int64            `json:"total_jobs"`
	PendingJobs     int64            `json:"pending_jobs"`
	ProcessingJobs  int64            `json:"processing_jobs"`
	CompletedJobs   int64            `json:"completed_jobs"`
	FailedJobs      int64            `json:"failed_jobs"`
	JobsByQueue     map[string]int64 `json:"jobs_by_queue"`
	JobsByType      map[string]int64 `json:"jobs_by_type"`
	AverageWaitTime float64          `json:"average_wait_time_seconds"`
}

// WorkerConfig contains configuration for job workers
type WorkerConfig struct {
	Queue       string
	Concurrency int           // Number of concurrent workers
	PollInterval time.Duration // How often to poll for new jobs
	Timeout     time.Duration // Maximum time for job execution
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Queue:       "default",
		Concurrency: 5,
		PollInterval: 1 * time.Second,
		Timeout:     5 * time.Minute,
	}
}

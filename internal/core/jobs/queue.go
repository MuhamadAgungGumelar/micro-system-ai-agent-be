package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Queue manages job queue operations
type Queue struct {
	db *gorm.DB
}

// NewQueue creates a new job queue
func NewQueue(db *gorm.DB) *Queue {
	return &Queue{db: db}
}

// Enqueue adds a new job to the queue
func (q *Queue) Enqueue(ctx context.Context, clientID uuid.UUID, jobType string, payload interface{}, opts EnqueueOptions) (*Job, error) {
	// Set defaults
	if opts.Queue == "" {
		opts.Queue = "default"
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}

	// Serialize payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Serialize metadata
	var metadataJSON datatypes.JSON
	if opts.Metadata != nil {
		metadataBytes, err := json.Marshal(opts.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	// Create job
	job := &Job{
		ClientID:    clientID,
		Queue:       opts.Queue,
		Type:        jobType,
		Payload:     payloadJSON,
		Status:      StatusPending,
		Priority:    opts.Priority,
		MaxRetries:  opts.MaxRetries,
		ScheduledAt: opts.ScheduleAt,
		Metadata:    metadataJSON,
	}

	if err := q.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// Dequeue retrieves the next job to process from the queue
func (q *Queue) Dequeue(ctx context.Context, queueName string) (*Job, error) {
	var job Job

	// Transaction to ensure atomic dequeue
	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find next pending job with highest priority
		// - Must be pending status
		// - If scheduled, must be past scheduled time
		// - Order by priority DESC, created_at ASC
		query := tx.Where("queue = ? AND status = ?", queueName, StatusPending)

		// Check if job is ready to run (not scheduled or scheduled time has passed)
		query = query.Where("scheduled_at IS NULL OR scheduled_at <= ?", time.Now())

		query = query.Order("priority DESC, created_at ASC").Limit(1)

		if err := query.First(&job).Error; err != nil {
			return err
		}

		// Mark as processing
		now := time.Now()
		job.Status = StatusProcessing
		job.StartedAt = &now
		job.Attempts++

		return tx.Save(&job).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	return &job, nil
}

// MarkCompleted marks a job as completed
func (q *Queue) MarkCompleted(ctx context.Context, jobID uuid.UUID, result interface{}) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       StatusCompleted,
		"completed_at": now,
	}

	// Serialize result if provided
	if result != nil {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to serialize result: %w", err)
		}
		updates["result"] = resultJSON
	}

	return q.db.WithContext(ctx).Model(&Job{}).Where("id = ?", jobID).Updates(updates).Error
}

// MarkFailed marks a job as failed
func (q *Queue) MarkFailed(ctx context.Context, jobID uuid.UUID, err error) error {
	var job Job
	if err := q.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	now := time.Now()
	job.Error = err.Error()
	job.FailedAt = &now

	// Check if we should retry
	if job.Attempts < job.MaxRetries {
		// Calculate exponential backoff
		backoffSeconds := calculateBackoff(job.Attempts)
		scheduleAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

		job.Status = StatusRetrying
		job.ScheduledAt = &scheduleAt
	} else {
		job.Status = StatusFailed
	}

	return q.db.WithContext(ctx).Save(&job).Error
}

// Cancel cancels a pending job
func (q *Queue) Cancel(ctx context.Context, jobID uuid.UUID) error {
	result := q.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND status IN ?", jobID, []JobStatus{StatusPending, StatusRetrying}).
		Update("status", StatusCancelled)

	if result.Error != nil {
		return fmt.Errorf("failed to cancel job: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("job not found or not in cancellable state")
	}

	return nil
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	var job Job
	if err := q.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListJobs lists jobs with optional filters
func (q *Queue) ListJobs(ctx context.Context, filter JobFilter) ([]Job, error) {
	query := q.db.WithContext(ctx).Model(&Job{})

	if filter.ClientID != nil {
		query = query.Where("client_id = ?", *filter.ClientID)
	}
	if filter.Queue != "" {
		query = query.Where("queue = ?", filter.Queue)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	query = query.Order("created_at DESC")

	var jobs []Job
	if err := query.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

// GetStats retrieves statistics about jobs
func (q *Queue) GetStats(ctx context.Context, clientID *uuid.UUID) (*JobStats, error) {
	stats := &JobStats{
		JobsByQueue: make(map[string]int64),
		JobsByType:  make(map[string]int64),
	}

	query := q.db.WithContext(ctx).Model(&Job{})
	if clientID != nil {
		query = query.Where("client_id = ?", *clientID)
	}

	// Total jobs
	query.Count(&stats.TotalJobs)

	// Jobs by status
	query.Where("status = ?", StatusPending).Count(&stats.PendingJobs)
	query.Where("status = ?", StatusProcessing).Count(&stats.ProcessingJobs)
	query.Where("status = ?", StatusCompleted).Count(&stats.CompletedJobs)
	query.Where("status = ?", StatusFailed).Count(&stats.FailedJobs)

	// Jobs by queue
	var queueStats []struct {
		Queue string
		Count int64
	}
	q.db.WithContext(ctx).Model(&Job{}).Select("queue, COUNT(*) as count").Group("queue").Find(&queueStats)
	for _, qs := range queueStats {
		stats.JobsByQueue[qs.Queue] = qs.Count
	}

	// Jobs by type
	var typeStats []struct {
		Type  string
		Count int64
	}
	q.db.WithContext(ctx).Model(&Job{}).Select("type, COUNT(*) as count").Group("type").Find(&typeStats)
	for _, ts := range typeStats {
		stats.JobsByType[ts.Type] = ts.Count
	}

	// Average wait time (time from creation to start)
	var avgWait float64
	q.db.WithContext(ctx).Model(&Job{}).
		Select("AVG(EXTRACT(EPOCH FROM (started_at - created_at)))").
		Where("started_at IS NOT NULL").
		Scan(&avgWait)
	stats.AverageWaitTime = avgWait

	return stats, nil
}

// DeleteOldJobs deletes completed/failed jobs older than the specified duration
func (q *Queue) DeleteOldJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	result := q.db.WithContext(ctx).
		Where("status IN ? AND completed_at < ?", []JobStatus{StatusCompleted, StatusFailed}, cutoff).
		Delete(&Job{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old jobs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// calculateBackoff calculates exponential backoff time in seconds
func calculateBackoff(attempt int) int {
	// Exponential backoff: 2^attempt seconds, max 1 hour
	backoff := 1 << attempt // 2^attempt
	if backoff > 3600 {
		backoff = 3600 // Max 1 hour
	}
	return backoff
}

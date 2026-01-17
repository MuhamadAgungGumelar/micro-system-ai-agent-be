package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service provides high-level job queue functionality
type Service struct {
	queue      *Queue
	workerPool *WorkerPool
}

// NewService creates a new job service
func NewService(db *gorm.DB) *Service {
	return &Service{
		queue:      NewQueue(db),
		workerPool: NewWorkerPool(),
	}
}

// Enqueue adds a new job to the queue
func (s *Service) Enqueue(ctx context.Context, clientID uuid.UUID, jobType string, payload interface{}, opts ...EnqueueOptions) (*Job, error) {
	options := DefaultEnqueueOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	return s.queue.Enqueue(ctx, clientID, jobType, payload, options)
}

// EnqueueDelayed adds a delayed job to the queue
func (s *Service) EnqueueDelayed(ctx context.Context, clientID uuid.UUID, jobType string, payload interface{}, delay time.Duration, opts ...EnqueueOptions) (*Job, error) {
	options := DefaultEnqueueOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	scheduleAt := time.Now().Add(delay)
	options.ScheduleAt = &scheduleAt

	return s.queue.Enqueue(ctx, clientID, jobType, payload, options)
}

// EnqueueAt adds a scheduled job to the queue
func (s *Service) EnqueueAt(ctx context.Context, clientID uuid.UUID, jobType string, payload interface{}, scheduleAt time.Time, opts ...EnqueueOptions) (*Job, error) {
	options := DefaultEnqueueOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	options.ScheduleAt = &scheduleAt

	return s.queue.Enqueue(ctx, clientID, jobType, payload, options)
}

// Cancel cancels a pending job
func (s *Service) Cancel(ctx context.Context, jobID uuid.UUID) error {
	return s.queue.Cancel(ctx, jobID)
}

// GetJob retrieves a job by ID
func (s *Service) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	return s.queue.GetJob(ctx, jobID)
}

// ListJobs lists jobs with filters
func (s *Service) ListJobs(ctx context.Context, filter JobFilter) ([]Job, error) {
	return s.queue.ListJobs(ctx, filter)
}

// GetStats retrieves job statistics
func (s *Service) GetStats(ctx context.Context, clientID *uuid.UUID) (*JobStats, error) {
	return s.queue.GetStats(ctx, clientID)
}

// RegisterWorker creates and registers a worker for a queue
func (s *Service) RegisterWorker(config WorkerConfig, handlers ...JobHandler) *Worker {
	worker := NewWorker(s.queue, config)

	// Register all handlers
	for _, handler := range handlers {
		worker.RegisterHandler(handler)
	}

	s.workerPool.AddWorker(worker)
	return worker
}

// StartWorkers starts all registered workers
func (s *Service) StartWorkers(ctx context.Context) error {
	return s.workerPool.Start(ctx)
}

// StopWorkers stops all workers
func (s *Service) StopWorkers() {
	s.workerPool.Stop()
}

// WaitForWorkers waits for all workers to finish
func (s *Service) WaitForWorkers() {
	s.workerPool.Wait()
}

// Cleanup deletes old completed/failed jobs
func (s *Service) Cleanup(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.queue.DeleteOldJobs(ctx, olderThan)
}

// --- Convenience methods for common job operations ---

// EnqueueEmailJob is a convenience method for email jobs
func (s *Service) EnqueueEmailJob(ctx context.Context, clientID uuid.UUID, payload interface{}) (*Job, error) {
	return s.Enqueue(ctx, clientID, "send_email", payload, EnqueueOptions{
		Queue:    "emails",
		Priority: PriorityNormal,
	})
}

// EnqueueNotificationJob is a convenience method for notification jobs
func (s *Service) EnqueueNotificationJob(ctx context.Context, clientID uuid.UUID, payload interface{}) (*Job, error) {
	return s.Enqueue(ctx, clientID, "send_notification", payload, EnqueueOptions{
		Queue:    "notifications",
		Priority: PriorityHigh,
	})
}

// EnqueueReportJob is a convenience method for report generation jobs
func (s *Service) EnqueueReportJob(ctx context.Context, clientID uuid.UUID, payload interface{}) (*Job, error) {
	return s.Enqueue(ctx, clientID, "generate_report", payload, EnqueueOptions{
		Queue:    "reports",
		Priority: PriorityLow,
	})
}

// EnqueueDataExportJob is a convenience method for data export jobs
func (s *Service) EnqueueDataExportJob(ctx context.Context, clientID uuid.UUID, payload interface{}) (*Job, error) {
	return s.Enqueue(ctx, clientID, "export_data", payload, EnqueueOptions{
		Queue:      "exports",
		Priority:   PriorityNormal,
		MaxRetries: 1, // Exports usually shouldn't be retried too many times
	})
}

// GetQueueStats returns statistics for a specific queue
func (s *Service) GetQueueStats(ctx context.Context, queueName string) (map[JobStatus]int64, error) {
	stats := make(map[JobStatus]int64)

	statuses := []JobStatus{StatusPending, StatusProcessing, StatusCompleted, StatusFailed, StatusRetrying}

	for _, status := range statuses {
		var count int64
		filter := JobFilter{
			Queue:  queueName,
			Status: status,
		}

		jobs, err := s.queue.ListJobs(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get queue stats: %w", err)
		}

		count = int64(len(jobs))
		stats[status] = count
	}

	return stats, nil
}

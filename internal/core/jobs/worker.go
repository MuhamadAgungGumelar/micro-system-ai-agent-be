package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Worker processes jobs from a queue
type Worker struct {
	queue    *Queue
	config   WorkerConfig
	handlers map[string]JobHandler
	mu       sync.RWMutex
	stopped  bool
	wg       sync.WaitGroup
}

// NewWorker creates a new job worker
func NewWorker(queue *Queue, config WorkerConfig) *Worker {
	return &Worker{
		queue:    queue,
		config:   config,
		handlers: make(map[string]JobHandler),
		stopped:  false,
	}
}

// RegisterHandler registers a job handler for a specific job type
func (w *Worker) RegisterHandler(handler JobHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[handler.GetType()] = handler
	log.Printf("✅ Registered job handler: %s", handler.GetType())
}

// Start starts the worker pool
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return fmt.Errorf("worker is stopped, cannot restart")
	}
	w.mu.Unlock()

	log.Printf("🚀 Starting job worker for queue '%s' with %d workers", w.config.Queue, w.config.Concurrency)

	// Start worker goroutines
	for i := 0; i < w.config.Concurrency; i++ {
		w.wg.Add(1)
		go w.runWorker(ctx, i+1)
	}

	log.Printf("✅ Job worker started successfully")
	return nil
}

// Stop gracefully stops the worker pool
func (w *Worker) Stop() {
	w.mu.Lock()
	w.stopped = true
	w.mu.Unlock()

	log.Printf("🛑 Stopping job worker for queue '%s'...", w.config.Queue)

	// Wait for all workers to finish
	w.wg.Wait()

	log.Printf("✅ Job worker stopped")
}

// Wait waits for all workers to finish
func (w *Worker) Wait() {
	w.wg.Wait()
}

// runWorker runs a single worker goroutine
func (w *Worker) runWorker(ctx context.Context, workerID int) {
	defer w.wg.Done()

	log.Printf("Worker #%d started for queue '%s'", workerID, w.config.Queue)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker #%d stopping due to context cancellation", workerID)
			return

		case <-ticker.C:
			w.mu.RLock()
			if w.stopped {
				w.mu.RUnlock()
				log.Printf("Worker #%d stopping", workerID)
				return
			}
			w.mu.RUnlock()

			// Try to dequeue a job
			if err := w.processNextJob(ctx, workerID); err != nil {
				// Log error but continue processing
				if err != ErrNoJobsAvailable {
					log.Printf("⚠️  Worker #%d error: %v", workerID, err)
				}
			}
		}
	}
}

// ErrNoJobsAvailable is returned when no jobs are available
var ErrNoJobsAvailable = fmt.Errorf("no jobs available")

// processNextJob processes the next available job
func (w *Worker) processNextJob(ctx context.Context, workerID int) error {
	// Dequeue next job
	job, err := w.queue.Dequeue(ctx, w.config.Queue)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	if job == nil {
		return ErrNoJobsAvailable
	}

	log.Printf("🔨 Worker #%d processing job %s (type: %s, attempt: %d)", workerID, job.ID, job.Type, job.Attempts)

	// Find handler
	w.mu.RLock()
	handler, exists := w.handlers[job.Type]
	w.mu.RUnlock()

	if !exists {
		log.Printf("❌ Worker #%d: no handler registered for job type '%s'", workerID, job.Type)
		w.queue.MarkFailed(ctx, job.ID, fmt.Errorf("no handler registered for job type: %s", job.Type))
		return nil
	}

	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	// Execute job handler
	startTime := time.Now()
	err = handler.Handle(jobCtx, job)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Worker #%d: job %s failed after %v: %v", workerID, job.ID, duration, err)
		if markErr := w.queue.MarkFailed(ctx, job.ID, err); markErr != nil {
			log.Printf("⚠️  Worker #%d: failed to mark job as failed: %v", workerID, markErr)
		}
		return nil
	}

	log.Printf("✅ Worker #%d: job %s completed in %v", workerID, job.ID, duration)
	if err := w.queue.MarkCompleted(ctx, job.ID, nil); err != nil {
		log.Printf("⚠️  Worker #%d: failed to mark job as completed: %v", workerID, err)
	}

	return nil
}

// WorkerPool manages multiple workers across different queues
type WorkerPool struct {
	workers []*Worker
	mu      sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make([]*Worker, 0),
	}
}

// AddWorker adds a worker to the pool
func (p *WorkerPool) AddWorker(worker *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers = append(p.workers, worker)
}

// Start starts all workers in the pool
func (p *WorkerPool) Start(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		if err := worker.Start(ctx); err != nil {
			return fmt.Errorf("failed to start worker: %w", err)
		}
	}

	return nil
}

// Stop stops all workers in the pool
func (p *WorkerPool) Stop() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var wg sync.WaitGroup
	for _, worker := range p.workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			w.Stop()
		}(worker)
	}

	wg.Wait()
}

// Wait waits for all workers to finish
func (p *WorkerPool) Wait() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, worker := range p.workers {
		worker.Wait()
	}
}

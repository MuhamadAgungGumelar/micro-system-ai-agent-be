package workflow

import (
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

// Scheduler handles scheduled (cron-based) workflow triggers
type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]cron.EntryID // workflow_id -> entry_id
	jobsMux sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()), // Support seconds in cron expressions
		jobs: make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	log.Println("⏰ Starting workflow scheduler...")
	s.cron.Start()
	log.Println("✅ Workflow scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("⏰ Stopping workflow scheduler...")
	s.cron.Stop()
	log.Println("✅ Workflow scheduler stopped")
}

// AddWorkflow adds a workflow to the scheduler
// schedule should be a cron expression (e.g., "0 18 * * *" for daily at 6 PM)
func (s *Scheduler) AddWorkflow(workflowID string, schedule string, job func()) error {
	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	// Remove existing job if any
	if entryID, exists := s.jobs[workflowID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, workflowID)
	}

	// Add new job
	entryID, err := s.cron.AddFunc(schedule, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobs[workflowID] = entryID
	log.Printf("   ✅ Scheduled workflow %s: %s", workflowID, schedule)

	return nil
}

// RemoveWorkflow removes a workflow from the scheduler
func (s *Scheduler) RemoveWorkflow(workflowID string) {
	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	if entryID, exists := s.jobs[workflowID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, workflowID)
		log.Printf("   ✅ Removed scheduled workflow: %s", workflowID)
	}
}

// GetScheduledWorkflows returns all currently scheduled workflow IDs
func (s *Scheduler) GetScheduledWorkflows() []string {
	s.jobsMux.RLock()
	defer s.jobsMux.RUnlock()

	workflowIDs := make([]string, 0, len(s.jobs))
	for workflowID := range s.jobs {
		workflowIDs = append(workflowIDs, workflowID)
	}

	return workflowIDs
}

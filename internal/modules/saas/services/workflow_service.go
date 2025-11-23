package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/llm"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/whatsapp"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/workflow"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/repositories"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WorkflowService handles workflow operations for SaaS module
type WorkflowService struct {
	workflowRepo       repositories.WorkflowRepo
	db                 *gorm.DB
	conditionEvaluator *workflow.ConditionEvaluator
	actionExecutor     *workflow.ActionExecutor
	scheduler          *workflow.Scheduler
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	workflowRepo repositories.WorkflowRepo,
	db *gorm.DB,
	waService *whatsapp.Service,
	llmService *llm.Service,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo:       workflowRepo,
		db:                 db,
		conditionEvaluator: workflow.NewConditionEvaluator(),
		actionExecutor:     workflow.NewActionExecutor(db, waService, llmService),
		scheduler:          workflow.NewScheduler(),
	}
}

// Initialize starts the workflow service (scheduler, etc.)
func (s *WorkflowService) Initialize() error {
	log.Println("🔧 Initializing Workflow Service...")

	// Load and schedule all active scheduled workflows
	if err := s.loadScheduledWorkflows(); err != nil {
		return fmt.Errorf("failed to load scheduled workflows: %w", err)
	}

	// Start the scheduler
	s.scheduler.Start()

	log.Println("✅ Workflow Service initialized successfully")
	return nil
}

// Shutdown stops the workflow service
func (s *WorkflowService) Shutdown() {
	log.Println("🛑 Shutting down Workflow Service...")
	s.scheduler.Stop()
	log.Println("✅ Workflow Service stopped")
}

// CreateWorkflow creates a new workflow
func (s *WorkflowService) CreateWorkflow(clientID uuid.UUID, req workflow.CreateWorkflowRequest) (*models.Workflow, error) {
	// Marshal trigger config
	triggerConfigJSON, err := json.Marshal(req.TriggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trigger config: %w", err)
	}

	// Marshal conditions
	conditionsJSON, err := json.Marshal(req.Conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conditions: %w", err)
	}

	// Marshal actions
	actionsJSON, err := json.Marshal(req.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actions: %w", err)
	}

	// Set default for IsActive
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Create workflow
	wf := &models.Workflow{
		ClientID:      clientID,
		Name:          req.Name,
		Description:   req.Description,
		TriggerType:   req.TriggerType,
		TriggerConfig: datatypes.JSON(triggerConfigJSON),
		Conditions:    datatypes.JSON(conditionsJSON),
		Actions:       datatypes.JSON(actionsJSON),
		IsActive:      isActive,
	}

	if err := s.workflowRepo.Create(wf); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// If it's a scheduled workflow and active, add to scheduler
	if wf.TriggerType == "scheduled" && wf.IsActive {
		if err := s.addWorkflowToScheduler(wf); err != nil {
			log.Printf("⚠️ Failed to schedule workflow: %v", err)
		}
	}

	log.Printf("✅ Workflow created: %s (ID: %s)", wf.Name, wf.ID)
	return wf, nil
}

// ListWorkflows lists all workflows for a client
func (s *WorkflowService) ListWorkflows(clientID uuid.UUID) ([]models.Workflow, error) {
	return s.workflowRepo.FindByClientID(clientID)
}

// GetWorkflow retrieves a workflow by ID
func (s *WorkflowService) GetWorkflow(workflowID uuid.UUID) (*models.Workflow, error) {
	return s.workflowRepo.FindByID(workflowID)
}

// UpdateWorkflow updates an existing workflow
func (s *WorkflowService) UpdateWorkflow(workflowID uuid.UUID, req workflow.UpdateWorkflowRequest) (*models.Workflow, error) {
	// Get existing workflow
	wf, err := s.workflowRepo.FindByID(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		wf.Name = *req.Name
	}
	if req.Description != nil {
		wf.Description = *req.Description
	}
	if req.TriggerType != nil {
		wf.TriggerType = *req.TriggerType
	}
	if req.TriggerConfig != nil {
		configJSON, err := json.Marshal(req.TriggerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal trigger config: %w", err)
		}
		wf.TriggerConfig = datatypes.JSON(configJSON)
	}
	if req.Conditions != nil {
		conditionsJSON, err := json.Marshal(req.Conditions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal conditions: %w", err)
		}
		wf.Conditions = datatypes.JSON(conditionsJSON)
	}
	if req.Actions != nil {
		actionsJSON, err := json.Marshal(req.Actions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal actions: %w", err)
		}
		wf.Actions = datatypes.JSON(actionsJSON)
	}
	if req.IsActive != nil {
		wasActive := wf.IsActive
		wf.IsActive = *req.IsActive

		// Handle scheduler updates
		if wf.TriggerType == "scheduled" {
			if !wasActive && wf.IsActive {
				// Workflow was inactive, now active - add to scheduler
				if err := s.addWorkflowToScheduler(wf); err != nil {
					log.Printf("⚠️ Failed to schedule workflow: %v", err)
				}
			} else if wasActive && !wf.IsActive {
				// Workflow was active, now inactive - remove from scheduler
				s.scheduler.RemoveWorkflow(wf.ID.String())
			}
		}
	}

	// Save updates
	if err := s.workflowRepo.Update(wf); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	log.Printf("✅ Workflow updated: %s (ID: %s)", wf.Name, wf.ID)
	return wf, nil
}

// DeleteWorkflow deletes a workflow
func (s *WorkflowService) DeleteWorkflow(workflowID uuid.UUID) error {
	// Get workflow to check if it's scheduled
	wf, err := s.workflowRepo.FindByID(workflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// Remove from scheduler if it's a scheduled workflow
	if wf.TriggerType == "scheduled" {
		s.scheduler.RemoveWorkflow(workflowID.String())
	}

	// Delete workflow
	if err := s.workflowRepo.Delete(workflowID); err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	log.Printf("✅ Workflow deleted: %s (ID: %s)", wf.Name, wf.ID)
	return nil
}

// ExecuteWorkflow manually executes a workflow
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, triggerData map[string]interface{}) error {
	// Get workflow
	wf, err := s.workflowRepo.FindByID(workflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// Check if workflow is active
	if !wf.IsActive {
		return fmt.Errorf("workflow is not active")
	}

	// Execute workflow
	return s.executeWorkflowInternal(ctx, wf, triggerData)
}

// HandleEvent handles an event-based workflow trigger
func (s *WorkflowService) HandleEvent(ctx context.Context, eventName string, eventData map[string]interface{}) error {
	log.Printf("📬 Event received: %s", eventName)

	// Find all active workflows with this event trigger
	var workflows []models.Workflow
	err := s.db.Where("trigger_type = ? AND is_active = ?", "event", true).Find(&workflows).Error
	if err != nil {
		return fmt.Errorf("failed to query workflows: %w", err)
	}

	log.Printf("   Found %d active workflows to check", len(workflows))

	// Execute workflows asynchronously
	for _, wf := range workflows {
		// Parse trigger config
		var triggerConfig workflow.TriggerConfig
		if err := json.Unmarshal(wf.TriggerConfig, &triggerConfig); err != nil {
			log.Printf("⚠️ Failed to unmarshal trigger config for workflow %s: %v", wf.ID, err)
			continue
		}

		// Check if event name matches
		if triggerConfig.EventName == eventName {
			log.Printf("   ✅ Workflow '%s' matches event '%s', executing...", wf.Name, eventName)

			// Execute workflow in background
			go func(workflow models.Workflow) {
				if err := s.executeWorkflowInternal(ctx, &workflow, eventData); err != nil {
					log.Printf("⚠️ Workflow execution failed for %s: %v", workflow.Name, err)
				}
			}(wf)
		}
	}

	return nil
}

// GetExecutions retrieves execution history for a workflow
func (s *WorkflowService) GetExecutions(workflowID uuid.UUID, limit int) ([]models.WorkflowExecution, error) {
	return s.workflowRepo.FindExecutionsByWorkflowID(workflowID, limit)
}

// executeWorkflowInternal executes a workflow with the given trigger data
func (s *WorkflowService) executeWorkflowInternal(ctx context.Context, wf *models.Workflow, triggerData map[string]interface{}) error {
	startTime := time.Now()

	// Create execution record
	execution := &models.WorkflowExecution{
		WorkflowID: wf.ID,
		Status:     "running",
		StartedAt:  startTime,
	}

	// Marshal trigger data
	triggerDataJSON, _ := json.Marshal(triggerData)
	execution.TriggerData = datatypes.JSON(triggerDataJSON)

	if err := s.workflowRepo.CreateExecution(execution); err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}

	log.Printf("🚀 Executing workflow: %s (ID: %s)", wf.Name, wf.ID)

	// Initialize execution log
	var executionLog []workflow.ExecutionLogEntry

	// Parse conditions
	var conditions []workflow.Condition
	if len(wf.Conditions) > 0 {
		if err := json.Unmarshal(wf.Conditions, &conditions); err != nil {
			return s.failExecution(execution, fmt.Errorf("failed to parse conditions: %w", err), executionLog)
		}
	}

	// Evaluate conditions
	conditionsPassed, err := s.conditionEvaluator.Evaluate(conditions, triggerData)
	if err != nil {
		return s.failExecution(execution, fmt.Errorf("condition evaluation error: %w", err), executionLog)
	}

	executionLog = append(executionLog, workflow.ExecutionLogEntry{
		Timestamp: time.Now(),
		Step:      "condition_check",
		Status:    map[bool]string{true: "success", false: "failed"}[conditionsPassed],
		Message:   fmt.Sprintf("Conditions %s", map[bool]string{true: "passed", false: "failed"}[conditionsPassed]),
	})

	if !conditionsPassed {
		log.Printf("⏭️  Conditions not met, skipping workflow execution")
		execution.Status = "completed"
		execution.ActionsCompleted = 0
		completedAt := time.Now()
		execution.CompletedAt = &completedAt
		execution.DurationMs = int(time.Since(startTime).Milliseconds())
		logJSON, _ := json.Marshal(executionLog)
		execution.ExecutionLog = datatypes.JSON(logJSON)
		s.workflowRepo.UpdateExecution(execution)
		return nil
	}

	// Parse actions
	var actions []workflow.Action
	if err := json.Unmarshal(wf.Actions, &actions); err != nil {
		return s.failExecution(execution, fmt.Errorf("failed to parse actions: %w", err), executionLog)
	}

	// Execute actions sequentially
	actionsCompleted := 0
	actionsFailed := 0

	for i, action := range actions {
		log.Printf("   🔧 Executing action %d/%d: %s", i+1, len(actions), action.Type)

		err := s.actionExecutor.Execute(ctx, action, triggerData)
		if err != nil {
			log.Printf("   ❌ Action failed: %v", err)
			actionsFailed++
			executionLog = append(executionLog, workflow.ExecutionLogEntry{
				Timestamp:  time.Now(),
				Step:       "action_execute",
				ActionType: action.Type,
				Status:     "failed",
				Message:    fmt.Sprintf("Action %d failed", i+1),
				Error:      err.Error(),
			})
		} else {
			log.Printf("   ✅ Action completed successfully")
			actionsCompleted++
			executionLog = append(executionLog, workflow.ExecutionLogEntry{
				Timestamp:  time.Now(),
				Step:       "action_execute",
				ActionType: action.Type,
				Status:     "success",
				Message:    fmt.Sprintf("Action %d completed", i+1),
			})
		}
	}

	// Update execution record
	execution.Status = "completed"
	execution.ActionsCompleted = actionsCompleted
	execution.ActionsFailed = actionsFailed
	completedAt := time.Now()
	execution.CompletedAt = &completedAt
	execution.DurationMs = int(time.Since(startTime).Milliseconds())

	logJSON, _ := json.Marshal(executionLog)
	execution.ExecutionLog = datatypes.JSON(logJSON)

	if err := s.workflowRepo.UpdateExecution(execution); err != nil {
		log.Printf("⚠️ Failed to update execution record: %v", err)
	}

	log.Printf("✅ Workflow execution completed: %d/%d actions succeeded", actionsCompleted, len(actions))
	return nil
}

// failExecution marks execution as failed
func (s *WorkflowService) failExecution(execution *models.WorkflowExecution, err error, executionLog []workflow.ExecutionLogEntry) error {
	execution.Status = "failed"
	execution.ErrorMessage = err.Error()
	completedAt := time.Now()
	execution.CompletedAt = &completedAt
	execution.DurationMs = int(time.Since(execution.StartedAt).Milliseconds())

	logJSON, _ := json.Marshal(executionLog)
	execution.ExecutionLog = datatypes.JSON(logJSON)

	s.workflowRepo.UpdateExecution(execution)
	return err
}

// loadScheduledWorkflows loads all active scheduled workflows into the scheduler
func (s *WorkflowService) loadScheduledWorkflows() error {
	workflows, err := s.workflowRepo.FindScheduledActive()
	if err != nil {
		return fmt.Errorf("failed to load scheduled workflows: %w", err)
	}

	log.Printf("   Loading %d scheduled workflow(s)...", len(workflows))

	for _, wf := range workflows {
		if err := s.addWorkflowToScheduler(&wf); err != nil {
			log.Printf("⚠️ Failed to schedule workflow %s: %v", wf.Name, err)
		}
	}

	return nil
}

// addWorkflowToScheduler adds a workflow to the cron scheduler
func (s *WorkflowService) addWorkflowToScheduler(wf *models.Workflow) error {
	// Parse trigger config to get schedule
	var triggerConfig workflow.TriggerConfig
	if err := json.Unmarshal(wf.TriggerConfig, &triggerConfig); err != nil {
		return fmt.Errorf("failed to unmarshal trigger config: %w", err)
	}

	if triggerConfig.Schedule == "" {
		return fmt.Errorf("schedule is empty")
	}

	// Create job function
	workflowID := wf.ID
	job := func() {
		log.Printf("⏰ Scheduled workflow triggered: %s", wf.Name)
		ctx := context.Background()
		triggerData := map[string]interface{}{
			"triggered_by": "schedule",
			"schedule":     triggerConfig.Schedule,
			"timestamp":    time.Now(),
		}

		// Get fresh workflow data
		freshWf, err := s.workflowRepo.FindByID(workflowID)
		if err != nil {
			log.Printf("❌ Failed to get workflow %s: %v", workflowID, err)
			return
		}

		if err := s.executeWorkflowInternal(ctx, freshWf, triggerData); err != nil {
			log.Printf("❌ Scheduled workflow execution failed: %v", err)
		}
	}

	// Add to scheduler
	return s.scheduler.AddWorkflow(wf.ID.String(), triggerConfig.Schedule, job)
}

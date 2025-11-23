package handlers

import (
	"log"

	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/core/workflow"
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WorkflowHandler handles workflow-related requests
type WorkflowHandler struct {
	workflowService *services.WorkflowService
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(workflowService *services.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: workflowService,
	}
}

// CreateWorkflow godoc
// @Summary Create a new workflow
// @Description Create a new automation workflow for a client
// @Tags Workflows
// @Accept json
// @Produce json
// @Param workflow body workflow.CreateWorkflowRequest true "Workflow details"
// @Param client_id query string true "Client ID"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows [post]
func (h *WorkflowHandler) CreateWorkflow(c *fiber.Ctx) error {
	// Get client_id from query or body
	clientIDStr := c.Query("client_id")
	if clientIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid client_id format",
		})
	}

	// Parse request body
	var req workflow.CreateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	if req.TriggerType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "trigger_type is required",
		})
	}

	if len(req.Actions) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least one action is required",
		})
	}

	// Create workflow
	createdWorkflow, err := h.workflowService.CreateWorkflow(clientID, req)
	if err != nil {
		log.Printf("❌ Failed to create workflow: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create workflow",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Workflow created successfully",
		"data":    createdWorkflow,
	})
}

// ListWorkflows godoc
// @Summary List workflows for a client
// @Description Retrieve all workflows for a specific client
// @Tags Workflows
// @Produce json
// @Param client_id query string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows [get]
func (h *WorkflowHandler) ListWorkflows(c *fiber.Ctx) error {
	clientIDStr := c.Query("client_id")
	if clientIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "client_id is required",
		})
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid client_id format",
		})
	}

	workflows, err := h.workflowService.ListWorkflows(clientID)
	if err != nil {
		log.Printf("❌ Failed to list workflows: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve workflows",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(workflows),
		"data":   workflows,
	})
}

// GetWorkflow godoc
// @Summary Get workflow by ID
// @Description Retrieve a specific workflow by its ID
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows/{id} [get]
func (h *WorkflowHandler) GetWorkflow(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow id is required",
		})
	}

	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid workflow id format",
		})
	}

	wf, err := h.workflowService.GetWorkflow(workflowID)
	if err != nil {
		log.Printf("❌ Failed to get workflow: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "workflow not found",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   wf,
	})
}

// UpdateWorkflow godoc
// @Summary Update a workflow
// @Description Update an existing workflow
// @Tags Workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param workflow body workflow.UpdateWorkflowRequest true "Updated workflow details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows/{id} [put]
func (h *WorkflowHandler) UpdateWorkflow(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow id is required",
		})
	}

	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid workflow id format",
		})
	}

	var req workflow.UpdateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	updatedWorkflow, err := h.workflowService.UpdateWorkflow(workflowID, req)
	if err != nil {
		log.Printf("❌ Failed to update workflow: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update workflow",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Workflow updated successfully",
		"data":    updatedWorkflow,
	})
}

// DeleteWorkflow godoc
// @Summary Delete a workflow
// @Description Delete a workflow by ID
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows/{id} [delete]
func (h *WorkflowHandler) DeleteWorkflow(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow id is required",
		})
	}

	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid workflow id format",
		})
	}

	if err := h.workflowService.DeleteWorkflow(workflowID); err != nil {
		log.Printf("❌ Failed to delete workflow: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete workflow",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Workflow deleted successfully",
	})
}

// ExecuteWorkflow godoc
// @Summary Manually execute a workflow
// @Description Trigger a workflow execution manually
// @Tags Workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body workflow.WorkflowExecutionRequest false "Trigger data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows/{id}/execute [post]
func (h *WorkflowHandler) ExecuteWorkflow(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow id is required",
		})
	}

	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid workflow id format",
		})
	}

	var req workflow.WorkflowExecutionRequest
	if err := c.BodyParser(&req); err != nil {
		// If no body provided, use empty trigger data
		req.TriggerData = make(map[string]interface{})
	}

	// Add triggered_by to context
	req.TriggerData["triggered_by"] = "manual"

	// Execute workflow
	err = h.workflowService.ExecuteWorkflow(c.Context(), workflowID, req.TriggerData)
	if err != nil {
		log.Printf("❌ Failed to execute workflow: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to execute workflow",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Workflow execution started",
	})
}

// GetWorkflowExecutions godoc
// @Summary Get workflow execution history
// @Description Retrieve execution history for a specific workflow
// @Tags Workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param limit query int false "Limit number of results" default(50)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /workflows/{id}/executions [get]
func (h *WorkflowHandler) GetWorkflowExecutions(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "workflow id is required",
		})
	}

	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid workflow id format",
		})
	}

	limit := c.QueryInt("limit", 50)

	executions, err := h.workflowService.GetExecutions(workflowID, limit)
	if err != nil {
		log.Printf("❌ Failed to get executions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve executions",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"count":  len(executions),
		"data":   executions,
	})
}

package repositories

import (
	"github.com/MuhamadAgungGumelar/micro-system-ai-agent-be/internal/modules/saas/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowRepo interface for workflow database operations
type WorkflowRepo interface {
	Create(workflow *models.Workflow) error
	FindByID(id uuid.UUID) (*models.Workflow, error)
	FindByClientID(clientID uuid.UUID) ([]models.Workflow, error)
	FindScheduledActive() ([]models.Workflow, error)
	Update(workflow *models.Workflow) error
	Delete(id uuid.UUID) error
	CreateExecution(execution *models.WorkflowExecution) error
	FindExecutionsByWorkflowID(workflowID uuid.UUID, limit int) ([]models.WorkflowExecution, error)
	UpdateExecution(execution *models.WorkflowExecution) error
}

type workflowRepo struct {
	db *gorm.DB
}

// NewWorkflowRepo creates a new workflow repository
func NewWorkflowRepo(db *gorm.DB) WorkflowRepo {
	return &workflowRepo{db: db}
}

func (r *workflowRepo) Create(workflow *models.Workflow) error {
	return r.db.Create(workflow).Error
}

func (r *workflowRepo) FindByID(id uuid.UUID) (*models.Workflow, error) {
	var workflow models.Workflow
	err := r.db.Where("id = ?", id).First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (r *workflowRepo) FindByClientID(clientID uuid.UUID) ([]models.Workflow, error) {
	var workflows []models.Workflow
	err := r.db.Where("client_id = ?", clientID).Order("created_at DESC").Find(&workflows).Error
	return workflows, err
}

func (r *workflowRepo) FindScheduledActive() ([]models.Workflow, error) {
	var workflows []models.Workflow
	err := r.db.Where("trigger_type = ? AND is_active = ?", "scheduled", true).Find(&workflows).Error
	return workflows, err
}

func (r *workflowRepo) Update(workflow *models.Workflow) error {
	return r.db.Save(workflow).Error
}

func (r *workflowRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Workflow{}).Error
}

func (r *workflowRepo) CreateExecution(execution *models.WorkflowExecution) error {
	return r.db.Create(execution).Error
}

func (r *workflowRepo) FindExecutionsByWorkflowID(workflowID uuid.UUID, limit int) ([]models.WorkflowExecution, error) {
	var executions []models.WorkflowExecution
	query := r.db.Where("workflow_id = ?", workflowID).Order("started_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&executions).Error
	return executions, err
}

func (r *workflowRepo) UpdateExecution(execution *models.WorkflowExecution) error {
	return r.db.Save(execution).Error
}

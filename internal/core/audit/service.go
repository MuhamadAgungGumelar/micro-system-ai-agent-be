package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service provides audit logging functionality
type Service struct {
	db *gorm.DB
}

// NewService creates a new audit service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Log creates a new audit log entry
func (s *Service) Log(ctx context.Context, log *AuditLog) error {
	if err := s.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// LogAction creates an audit log with basic action information
func (s *Service) LogAction(ctx context.Context, userID, clientID uuid.UUID, action, entity, entityID string) error {
	return s.Log(ctx, &AuditLog{
		UserID:   userID,
		ClientID: clientID,
		Action:   action,
		Entity:   entity,
		EntityID: entityID,
	})
}

// LogChange creates an audit log tracking a change (create, update, delete)
func (s *Service) LogChange(ctx context.Context, userID, clientID uuid.UUID, action, entity, entityID string, oldValue, newValue interface{}) error {
	oldJSON, err := toJSON(oldValue)
	if err != nil {
		log.Printf("Warning: failed to serialize old value: %v", err)
	}

	newJSON, err := toJSON(newValue)
	if err != nil {
		log.Printf("Warning: failed to serialize new value: %v", err)
	}

	return s.Log(ctx, &AuditLog{
		UserID:   userID,
		ClientID: clientID,
		Action:   action,
		Entity:   entity,
		EntityID: entityID,
		OldValue: oldJSON,
		NewValue: newJSON,
	})
}

// GetLogs retrieves audit logs with filtering
func (s *Service) GetLogs(filter AuditFilter) (*AuditLogResponse, error) {
	query := s.db.Model(&AuditLog{})

	// Apply filters
	if filter.ClientID != nil {
		query = query.Where("client_id = ?", *filter.ClientID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Entity != "" {
		query = query.Where("entity = ?", filter.Entity)
	}
	if filter.EntityID != "" {
		query = query.Where("entity_id = ?", filter.EntityID)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	// Count total
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Apply pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 50
	}

	offset := (filter.Page - 1) * filter.PageSize

	// Get logs
	var logs []AuditLog
	if err := query.
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	totalPages := int(totalCount) / filter.PageSize
	if int(totalCount)%filter.PageSize > 0 {
		totalPages++
	}

	return &AuditLogResponse{
		Logs:       logs,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetEntityHistory retrieves all changes for a specific entity
func (s *Service) GetEntityHistory(clientID uuid.UUID, entity, entityID string) ([]AuditLog, error) {
	var logs []AuditLog
	err := s.db.Where("client_id = ? AND entity = ? AND entity_id = ?", clientID, entity, entityID).
		Order("created_at DESC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get entity history: %w", err)
	}

	return logs, nil
}

// GetUserActivity retrieves all actions performed by a user
func (s *Service) GetUserActivity(userID uuid.UUID, limit int) ([]AuditLog, error) {
	if limit < 1 {
		limit = 100
	}

	var logs []AuditLog
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}

	return logs, nil
}

// GetActionStats returns statistics about audit log actions
func (s *Service) GetActionStats(clientID uuid.UUID, startDate, endDate *uuid.Time) (map[string]int64, error) {
	query := s.db.Model(&AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("client_id = ?", clientID)

	if startDate != nil {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", endDate)
	}

	var results []struct {
		Action string
		Count  int64
	}

	if err := query.Group("action").Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get action stats: %w", err)
	}

	stats := make(map[string]int64)
	for _, result := range results {
		stats[result.Action] = result.Count
	}

	return stats, nil
}

// DeleteOldLogs deletes audit logs older than a certain number of days
// This is useful for data retention compliance
func (s *Service) DeleteOldLogs(daysToKeep int) error {
	if daysToKeep < 1 {
		return fmt.Errorf("daysToKeep must be at least 1")
	}

	cutoffDate := s.db.NowFunc().AddDate(0, 0, -daysToKeep)

	result := s.db.Where("created_at < ?", cutoffDate).Delete(&AuditLog{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", result.Error)
	}

	log.Printf("Deleted %d old audit logs (older than %d days)", result.RowsAffected, daysToKeep)
	return nil
}

// Helper function to convert value to JSON
func toJSON(value interface{}) (datatypes.JSON, error) {
	if value == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(bytes), nil
}

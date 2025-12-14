package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(user *CompanyUser) error {
	return r.db.Create(user).Error
}

// GetUserByEmail retrieves user by email
func (r *Repository) GetUserByEmail(email string) (*CompanyUser, error) {
	var user CompanyUser
	err := r.db.Where("email = ? AND is_active = ?", email, true).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves user by ID
func (r *Repository) GetUserByID(id string) (*CompanyUser, error) {
	var user CompanyUser
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByGoogleID retrieves user by Google OAuth ID
func (r *Repository) GetUserByGoogleID(googleID string) (*CompanyUser, error) {
	var user CompanyUser
	err := r.db.Where("google_id = ? AND is_active = ?", googleID, true).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByRefreshToken retrieves user by refresh token
func (r *Repository) GetUserByRefreshToken(refreshToken string) (*CompanyUser, error) {
	var user CompanyUser
	err := r.db.Where("refresh_token = ? AND is_active = ?", refreshToken, true).First(&user).Error
	if err != nil {
		return nil, err
	}

	// Check if refresh token is expired
	if user.RefreshTokenExpiresAt != nil && user.RefreshTokenExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("refresh token expired")
	}

	return &user, nil
}

// UpdateUser updates user information
func (r *Repository) UpdateUser(user *CompanyUser) error {
	return r.db.Save(user).Error
}

// UpdateRefreshToken updates user's refresh token
func (r *Repository) UpdateRefreshToken(userID string, refreshToken string, expiresAt time.Time) error {
	return r.db.Model(&CompanyUser{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token":            refreshToken,
			"refresh_token_expires_at": expiresAt,
		}).Error
}

// UpdateLastLogin updates user's last login timestamp
func (r *Repository) UpdateLastLogin(userID string) error {
	now := time.Now()
	return r.db.Model(&CompanyUser{}).
		Where("id = ?", userID).
		Update("last_login_at", now).Error
}

// RevokeRefreshToken revokes (clears) user's refresh token
func (r *Repository) RevokeRefreshToken(userID string) error {
	return r.db.Model(&CompanyUser{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token":            nil,
			"refresh_token_expires_at": nil,
		}).Error
}

// EmailExists checks if email already exists
func (r *Repository) EmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&CompanyUser{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserWithClient retrieves user with associated client information
func (r *Repository) GetUserWithClient(userID string) (*CompanyUser, *ClientInfo, error) {
	var user CompanyUser
	var client struct {
		ID             uuid.UUID
		BusinessName   string
		Module         string
		WhatsAppNumber string
	}

	// Get user
	err := r.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error
	if err != nil {
		return nil, nil, err
	}

	// Get client info
	err = r.db.Table("clients").
		Select("id, business_name, module, whatsapp_number").
		Where("id = ?", user.ClientID).
		First(&client).Error
	if err != nil {
		return &user, nil, nil // User exists but client not found
	}

	clientInfo := &ClientInfo{
		ID:             client.ID.String(),
		BusinessName:   client.BusinessName,
		Module:         client.Module,
		WhatsAppNumber: client.WhatsAppNumber,
	}

	return &user, clientInfo, nil
}

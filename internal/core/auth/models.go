package auth

import (
	"time"

	"github.com/google/uuid"
)

// CompanyUser represents a user that can login to the CMS
// Can be tenant admin, staff, or super admin
type CompanyUser struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClientID uuid.UUID `gorm:"type:uuid;not null" json:"client_id"`

	// Basic Info
	PhoneNumber string `gorm:"type:text" json:"phone_number"`
	Email       string `gorm:"type:text;unique" json:"email"`
	Name        string `gorm:"type:text" json:"name"`
	Role        string `gorm:"type:text;not null" json:"role"` // super_admin, admin_tenant, staff_tenant

	// Authentication
	PasswordHash string `gorm:"type:text" json:"-"` // Hidden from JSON

	// OAuth
	GoogleID      string `gorm:"type:text;unique;column:google_id" json:"google_id,omitempty"`
	OAuthProvider string `gorm:"type:text;default:'email';column:oauth_provider" json:"oauth_provider"`

	// Profile
	AvatarURL string `gorm:"type:text" json:"avatar_url,omitempty"`

	// Status
	IsActive      bool `gorm:"type:boolean;default:true" json:"is_active"`
	EmailVerified bool `gorm:"type:boolean;default:false" json:"email_verified"`

	// JWT Refresh Token
	RefreshToken          string     `gorm:"type:text" json:"-"`
	RefreshTokenExpiresAt *time.Time `json:"-"`

	// Timestamps
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (CompanyUser) TableName() string {
	return "company_users"
}

// LoginRequest represents login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest represents registration request payload
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	Name        string `json:"name" validate:"required"`
	PhoneNumber string `json:"phone_number,omitempty"`
	ClientID    string `json:"client_id" validate:"required,uuid"`
	Role        string `json:"role" validate:"required,oneof=admin_tenant staff_tenant"`
}

// GoogleLoginRequest represents Google OAuth login request
type GoogleLoginRequest struct {
	GoogleIDToken string `json:"google_id_token" validate:"required"`
	ClientID      string `json:"client_id,omitempty"` // Optional for first-time login
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // seconds
	User         *UserInfo    `json:"user"`
	Client       *ClientInfo  `json:"client,omitempty"`
}

// UserInfo represents user information in auth response
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	OAuthProvider string `json:"oauth_provider"`
}

// ClientInfo represents client/tenant information
type ClientInfo struct {
	ID             string `json:"id"`
	BusinessName   string `json:"business_name"`
	Module         string `json:"module"` // saas, farmasi, umkm
	WhatsAppNumber string `json:"whatsapp_number,omitempty"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	ClientID string `json:"client_id"`
	Module   string `json:"module"`
}

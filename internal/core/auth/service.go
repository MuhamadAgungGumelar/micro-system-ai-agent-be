package auth

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo       *Repository
	jwtService *JWTService
}

// NewService creates a new auth service
func NewService(db *gorm.DB, jwtSecret string) *Service {
	return &Service{
		repo:       NewRepository(db),
		jwtService: NewJWTService(jwtSecret),
	}
}

// Register creates a new user account
func (s *Service) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Check if email already exists
	exists, err := s.repo.EmailExists(req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Parse client ID
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client_id: %w", err)
	}

	// Create user
	user := &CompanyUser{
		ClientID:      clientID,
		Email:         req.Email,
		Name:          req.Name,
		PhoneNumber:   req.PhoneNumber,
		Role:          req.Role,
		PasswordHash:  passwordHash,
		OAuthProvider: "email",
		IsActive:      true,
		EmailVerified: false,
	}

	err = s.repo.CreateUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("✅ User registered: %s (%s)", user.Email, user.ID.String())

	// Generate tokens and return auth response
	return s.generateAuthResponse(user)
}

// Login authenticates user with email and password
func (s *Service) Login(req *LoginRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	err = VerifyPassword(user.PasswordHash, req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(user.ID.String())

	log.Printf("✅ User logged in: %s (%s)", user.Email, user.ID.String())

	// Generate tokens and return auth response
	return s.generateAuthResponse(user)
}

// LoginWithGoogle authenticates user with Google OAuth
func (s *Service) LoginWithGoogle(googleID, email, name, avatarURL string, clientIDStr string) (*AuthResponse, error) {
	// Try to find existing user by Google ID
	user, err := s.repo.GetUserByGoogleID(googleID)

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// If user doesn't exist, create new user
	if err == gorm.ErrRecordNotFound {
		// For Google login, we need a client ID
		if clientIDStr == "" {
			return nil, fmt.Errorf("client_id required for first-time Google login")
		}

		clientID, err := uuid.Parse(clientIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid client_id: %w", err)
		}

		user = &CompanyUser{
			ClientID:      clientID,
			Email:         email,
			Name:          name,
			GoogleID:      googleID,
			AvatarURL:     avatarURL,
			OAuthProvider: "google",
			Role:          "staff_tenant", // Default role for OAuth users
			IsActive:      true,
			EmailVerified: true, // Google accounts are pre-verified
		}

		err = s.repo.CreateUser(user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		log.Printf("✅ New Google user registered: %s (%s)", user.Email, user.ID.String())
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(user.ID.String())

	log.Printf("✅ User logged in via Google: %s (%s)", user.Email, user.ID.String())

	// Generate tokens and return auth response
	return s.generateAuthResponse(user)
}

// RefreshToken generates new access token from refresh token
func (s *Service) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	userID, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user by refresh token (verify it matches DB)
	user, err := s.repo.GetUserByRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Verify user ID matches
	if user.ID.String() != userID {
		return nil, fmt.Errorf("refresh token user mismatch")
	}

	log.Printf("✅ Token refreshed for user: %s (%s)", user.Email, user.ID.String())

	// Generate new tokens
	return s.generateAuthResponse(user)
}

// Logout revokes user's refresh token
func (s *Service) Logout(userID string) error {
	err := s.repo.RevokeRefreshToken(userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	log.Printf("✅ User logged out: %s", userID)
	return nil
}

// ValidateToken validates an access token and returns user info
func (s *Service) ValidateToken(accessToken string) (*TokenClaims, error) {
	claims, err := s.jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}
	return claims, nil
}

// generateAuthResponse generates auth response with tokens and user info
func (s *Service) generateAuthResponse(user *CompanyUser) (*AuthResponse, error) {
	// Get client info
	_, clientInfo, err := s.repo.GetUserWithClient(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get client info: %w", err)
	}

	// Prepare token claims
	module := ""
	if clientInfo != nil {
		module = clientInfo.Module
	}

	claims := &TokenClaims{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Role:     user.Role,
		ClientID: user.ClientID.String(),
		Module:   module,
	}

	// Generate access token
	accessToken, expiresIn, err := s.jwtService.GenerateAccessToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, expiresAt, err := s.jwtService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database
	err = s.repo.UpdateRefreshToken(user.ID.String(), refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Prepare user info
	userInfo := &UserInfo{
		ID:            user.ID.String(),
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		PhoneNumber:   user.PhoneNumber,
		AvatarURL:     user.AvatarURL,
		OAuthProvider: user.OAuthProvider,
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         userInfo,
		Client:       clientInfo,
	}, nil
}

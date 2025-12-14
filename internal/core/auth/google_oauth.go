package auth

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleOAuthService handles Google OAuth operations
type GoogleOAuthService struct {
	clientID string
}

// NewGoogleOAuthService creates a new Google OAuth service
func NewGoogleOAuthService(clientID string) *GoogleOAuthService {
	return &GoogleOAuthService{
		clientID: clientID,
	}
}

// GoogleUserInfo represents user information from Google
type GoogleUserInfo struct {
	GoogleID  string
	Email     string
	Name      string
	AvatarURL string
}

// VerifyIDToken verifies Google ID token and returns user information
func (s *GoogleOAuthService) VerifyIDToken(ctx context.Context, idToken string) (*GoogleUserInfo, error) {
	// Verify the ID token
	payload, err := idtoken.Validate(ctx, idToken, s.clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Google ID token: %w", err)
	}

	// Extract user information
	googleID, ok := payload.Claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("missing sub claim in token")
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	avatarURL, _ := payload.Claims["picture"].(string)

	// Verify email
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if !emailVerified {
		return nil, fmt.Errorf("email not verified by Google")
	}

	return &GoogleUserInfo{
		GoogleID:  googleID,
		Email:     email,
		Name:      name,
		AvatarURL: avatarURL,
	}, nil
}

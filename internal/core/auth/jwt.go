package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey             string
	accessTokenDuration   time.Duration
	refreshTokenDuration  time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey:            secretKey,
		accessTokenDuration:  15 * time.Minute,      // Short-lived access token
		refreshTokenDuration: 7 * 24 * time.Hour,    // 7 days refresh token
	}
}

// GenerateAccessToken generates a new access token
func (s *JWTService) GenerateAccessToken(claims *TokenClaims) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessTokenDuration)

	jwtClaims := jwt.MapClaims{
		"user_id":   claims.UserID,
		"email":     claims.Email,
		"role":      claims.Role,
		"client_id": claims.ClientID,
		"module":    claims.Module,
		"exp":       expiresAt.Unix(),
		"iat":       now.Unix(),
		"nbf":       now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, int64(s.accessTokenDuration.Seconds()), nil
}

// GenerateRefreshToken generates a new refresh token
func (s *JWTService) GenerateRefreshToken(userID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.refreshTokenDuration)

	jwtClaims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     expiresAt.Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateAccessToken validates an access token and returns claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Extract claims
	userID, _ := claims["user_id"].(string)
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)
	clientID, _ := claims["client_id"].(string)
	module, _ := claims["module"].(string)

	return &TokenClaims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		ClientID: clientID,
		Module:   module,
	}, nil
}

// ValidateRefreshToken validates a refresh token and returns user ID
func (s *JWTService) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Validate token type
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return "", fmt.Errorf("not a refresh token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id in token")
	}

	return userID, nil
}

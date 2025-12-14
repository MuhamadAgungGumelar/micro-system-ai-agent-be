package auth

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	authService  *Service
	googleOAuth  *GoogleOAuthService
}

// NewHandler creates a new auth handler
func NewHandler(authService *Service, googleClientID string) *Handler {
	return &Handler{
		authService: authService,
		googleOAuth: NewGoogleOAuthService(googleClientID),
	}
}

// Register godoc
// @Summary Register new user
// @Description Create a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Email == "" || req.Password == "" || req.Name == "" || req.ClientID == "" || req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: email, password, name, client_id, role",
		})
	}

	// Validate role
	if req.Role != "admin_tenant" && req.Role != "staff_tenant" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role must be 'admin_tenant' or 'staff_tenant'",
		})
	}

	// Register user
	authResponse, err := h.authService.Register(&req)
	if err != nil {
		log.Printf("❌ Registration failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(authResponse)
}

// Login godoc
// @Summary Login with email and password
// @Description Authenticate user and return JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	// Login
	authResponse, err := h.authService.Login(&req)
	if err != nil {
		log.Printf("❌ Login failed for %s: %v", req.Email, err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	return c.JSON(authResponse)
}

// LoginWithGoogle godoc
// @Summary Login with Google OAuth
// @Description Authenticate user with Google ID token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body GoogleLoginRequest true "Google ID token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/google [post]
func (h *Handler) LoginWithGoogle(c *fiber.Ctx) error {
	var req GoogleLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.GoogleIDToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "google_id_token is required",
		})
	}

	// Verify Google ID token
	ctx := context.Background()
	googleUser, err := h.googleOAuth.VerifyIDToken(ctx, req.GoogleIDToken)
	if err != nil {
		log.Printf("❌ Google token verification failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Google ID token",
		})
	}

	// Login with Google
	authResponse, err := h.authService.LoginWithGoogle(
		googleUser.GoogleID,
		googleUser.Email,
		googleUser.Name,
		googleUser.AvatarURL,
		req.ClientID,
	)
	if err != nil {
		log.Printf("❌ Google login failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(authResponse)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token is required",
		})
	}

	// Refresh token
	authResponse, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("❌ Token refresh failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	return c.JSON(authResponse)
}

// Logout godoc
// @Summary Logout user
// @Description Revoke user's refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/logout [post]
func (h *Handler) Logout(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Logout
	err := h.authService.Logout(userID.(string))
	if err != nil {
		log.Printf("❌ Logout failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to logout",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me godoc
// @Summary Get current user
// @Description Get authenticated user information
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserInfo
// @Failure 401 {object} map[string]interface{}
// @Router /auth/me [get]
func (h *Handler) Me(c *fiber.Ctx) error {
	// Get user info from context (set by auth middleware)
	userInfo := c.Locals("user")
	if userInfo == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.JSON(userInfo)
}

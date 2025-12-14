package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(authService *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format. Use: Bearer <token>",
			})
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store user information in context
		c.Locals("userID", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("clientID", claims.ClientID)
		c.Locals("module", claims.Module)

		// Store full user info for /auth/me endpoint
		c.Locals("user", &UserInfo{
			ID:       claims.UserID,
			Email:    claims.Email,
			Role:     claims.Role,
		})

		// Continue to next handler
		return c.Next()
	}
}

// RequireRole creates a middleware that checks if user has required role
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role")
		if userRole == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Check if user has required role
		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// RequireModule creates a middleware that checks if user belongs to required module
func RequireModule(modules ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userModule := c.Locals("module")
		if userModule == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Check if user has required module
		moduleStr := userModule.(string)
		for _, module := range modules {
			if moduleStr == module {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied for this module",
		})
	}
}

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/waffles/mcp-gateway/internal/repository"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// UserContext keys for storing user information in the request context
const (
	ContextKeyUserID    = "user_id"
	ContextKeyUserEmail = "user_email"
	ContextKeyUserRoles = "user_roles"
	ContextKeyAuthType  = "auth_type"
)

// AuthType represents the type of authentication used
type AuthType string

const (
	AuthTypeSession AuthType = "session"
	AuthTypeAPIKey  AuthType = "apikey"
)

// AuthConfig contains configuration for authentication middleware
type AuthConfig struct {
	Logger       logger.Logger
	UserRepo     *repository.UserRepository
	APIKeyRepo   *repository.APIKeyRepository
	SessionName  string
}

// SessionAuth creates a middleware that validates session-based authentication
func SessionAuth(cfg *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(ContextKeyUserID)

		if userID == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Please log in to access this resource",
			})
			return
		}

		// Set user context from session
		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyUserEmail, session.Get(ContextKeyUserEmail))
		c.Set(ContextKeyUserRoles, session.Get(ContextKeyUserRoles))
		c.Set(ContextKeyAuthType, AuthTypeSession)

		c.Next()
	}
}

// APIKeyAuth creates a middleware that validates API key authentication
func APIKeyAuth(cfg *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "API key required",
			})
			return
		}

		// Validate the API key
		keyHash := repository.HashAPIKey(apiKey)
		key, err := cfg.APIKeyRepo.GetByHash(c.Request.Context(), keyHash)
		if err != nil {
			cfg.Logger.Warn().Err(err).Msg("Invalid API key attempt")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired API key",
			})
			return
		}

		// Get user and roles
		user, err := cfg.UserRepo.GetByID(c.Request.Context(), key.UserID)
		if err != nil {
			cfg.Logger.Error().Err(err).Str("user_id", key.UserID).Msg("Failed to get user for API key")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not found",
			})
			return
		}

		if !user.IsActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User account is inactive",
			})
			return
		}

		// Get user roles
		roles, err := cfg.UserRepo.GetUserRoles(c.Request.Context(), user.ID)
		if err != nil {
			cfg.Logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
			roles = []string{} // Default to no roles
		}

		// Update last used timestamp (async to not block request)
		go func() {
			ctx := context.Background()
			if err := cfg.APIKeyRepo.UpdateLastUsed(ctx, key.ID); err != nil {
				cfg.Logger.Error().Err(err).Str("key_id", key.ID).Msg("Failed to update API key last_used_at")
			}
		}()

		// Set user context
		c.Set(ContextKeyUserID, user.ID)
		c.Set(ContextKeyUserEmail, user.Email)
		c.Set(ContextKeyUserRoles, roles)
		c.Set(ContextKeyAuthType, AuthTypeAPIKey)

		c.Next()
	}
}

// CombinedAuth creates a middleware that accepts either session or API key authentication
// This is useful for endpoints that should work for both browser and programmatic access
func CombinedAuth(cfg *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First, check for API key in header (takes precedence for programmatic access)
		apiKey := extractAPIKey(c)
		if apiKey != "" {
			// Validate API key
			keyHash := repository.HashAPIKey(apiKey)
			key, err := cfg.APIKeyRepo.GetByHash(c.Request.Context(), keyHash)
			if err == nil {
				// Valid API key - get user info
				user, err := cfg.UserRepo.GetByID(c.Request.Context(), key.UserID)
				if err == nil && user.IsActive {
					roles, _ := cfg.UserRepo.GetUserRoles(c.Request.Context(), user.ID)

					// Update last used (async)
					go func() {
						ctx := context.Background()
						cfg.APIKeyRepo.UpdateLastUsed(ctx, key.ID)
					}()

					c.Set(ContextKeyUserID, user.ID)
					c.Set(ContextKeyUserEmail, user.Email)
					c.Set(ContextKeyUserRoles, roles)
					c.Set(ContextKeyAuthType, AuthTypeAPIKey)
					c.Next()
					return
				}
			}
			// Invalid API key - don't fall through to session, return error
			cfg.Logger.Warn().Msg("Invalid API key attempt")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired API key",
			})
			return
		}

		// No API key - check session
		session := sessions.Default(c)
		userID := session.Get(ContextKeyUserID)

		if userID == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			return
		}

		// Set user context from session
		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyUserEmail, session.Get(ContextKeyUserEmail))
		c.Set(ContextKeyUserRoles, session.Get(ContextKeyUserRoles))
		c.Set(ContextKeyAuthType, AuthTypeSession)

		c.Next()
	}
}

// OptionalAuth creates a middleware that extracts user info if authenticated, but allows anonymous access
func OptionalAuth(cfg *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key
		apiKey := extractAPIKey(c)
		if apiKey != "" {
			keyHash := repository.HashAPIKey(apiKey)
			key, err := cfg.APIKeyRepo.GetByHash(c.Request.Context(), keyHash)
			if err == nil {
				user, err := cfg.UserRepo.GetByID(c.Request.Context(), key.UserID)
				if err == nil && user.IsActive {
					roles, _ := cfg.UserRepo.GetUserRoles(c.Request.Context(), user.ID)
					c.Set(ContextKeyUserID, user.ID)
					c.Set(ContextKeyUserEmail, user.Email)
					c.Set(ContextKeyUserRoles, roles)
					c.Set(ContextKeyAuthType, AuthTypeAPIKey)
				}
			}
		} else {
			// Check session
			session := sessions.Default(c)
			if userID := session.Get(ContextKeyUserID); userID != nil {
				c.Set(ContextKeyUserID, userID)
				c.Set(ContextKeyUserEmail, session.Get(ContextKeyUserEmail))
				c.Set(ContextKeyUserRoles, session.Get(ContextKeyUserRoles))
				c.Set(ContextKeyAuthType, AuthTypeSession)
			}
		}

		c.Next()
	}
}

// extractAPIKey extracts the API key from the request
// Supports: Authorization: Bearer mcpgw_xxx or X-API-Key: mcpgw_xxx
func extractAPIKey(c *gin.Context) string {
	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token := strings.TrimSpace(parts[1])
			if strings.HasPrefix(token, "mcpgw_") {
				return token
			}
		}
	}

	// Check X-API-Key header
	apiKeyHeader := c.GetHeader("X-API-Key")
	if apiKeyHeader != "" && strings.HasPrefix(apiKeyHeader, "mcpgw_") {
		return apiKeyHeader
	}

	return ""
}

// GetUserID retrieves the user ID from the context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(ContextKeyUserID); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserEmail retrieves the user email from the context
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get(ContextKeyUserEmail); exists {
		if e, ok := email.(string); ok {
			return e
		}
	}
	return ""
}

// GetUserRoles retrieves the user roles from the context
func GetUserRoles(c *gin.Context) []string {
	if roles, exists := c.Get(ContextKeyUserRoles); exists {
		if r, ok := roles.([]string); ok {
			return r
		}
	}
	return []string{}
}

// GetAuthType retrieves the authentication type from the context
func GetAuthType(c *gin.Context) AuthType {
	if authType, exists := c.Get(ContextKeyAuthType); exists {
		if t, ok := authType.(AuthType); ok {
			return t
		}
	}
	return ""
}

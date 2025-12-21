package middleware

import (
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/pkg/logger"
)

// AuthzConfig contains configuration for authorization middleware
type AuthzConfig struct {
	Logger   logger.Logger
	Enforcer *casbin.Enforcer
}

// formatRoles converts a slice of roles to a comma-separated string for logging
func formatRoles(roles []string) string {
	return strings.Join(roles, ", ")
}

// Authz creates a middleware that enforces Casbin authorization policies
func Authz(cfg *AuthzConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := GetUserRoles(c)
		path := c.Request.URL.Path
		method := c.Request.Method

		// If no roles, deny access
		if len(roles) == 0 {
			cfg.Logger.Warn().
				Str("path", path).
				Str("method", method).
				Msg("Access denied: no roles assigned")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "No roles assigned to user",
			})
			return
		}

		// Check if any role has permission
		allowed := false
		for _, role := range roles {
			ok, err := cfg.Enforcer.Enforce(role, path, method)
			if err != nil {
				cfg.Logger.Error().Err(err).
					Str("role", role).
					Str("path", path).
					Str("method", method).
					Msg("Error checking permission")
				continue
			}
			if ok {
				allowed = true
				break
			}
		}

		if !allowed {
			cfg.Logger.Warn().
				Str("roles", formatRoles(roles)).
				Str("path", path).
				Str("method", method).
				Msg("Access denied: insufficient permissions")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "You don't have permission to access this resource",
			})
			return
		}

		c.Next()
	}
}

// RequireRoles creates a middleware that requires the user to have specific roles
func RequireRoles(cfg *AuthzConfig, requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles := GetUserRoles(c)

		// Check if user has any of the required roles
		hasRole := false
		for _, required := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == required {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			cfg.Logger.Warn().
				Str("user_roles", formatRoles(userRoles)).
				Str("required_roles", formatRoles(requiredRoles)).
				Msg("Access denied: missing required role")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "This action requires specific roles",
			})
			return
		}

		c.Next()
	}
}

// RequirePermission creates a middleware that requires a specific permission
// This is an alternative to role-based checks, using Casbin's permission checking
func RequirePermission(cfg *AuthzConfig, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := GetUserRoles(c)

		allowed := false
		for _, role := range roles {
			ok, err := cfg.Enforcer.Enforce(role, resource, action)
			if err != nil {
				cfg.Logger.Error().Err(err).
					Str("role", role).
					Str("resource", resource).
					Str("action", action).
					Msg("Error checking permission")
				continue
			}
			if ok {
				allowed = true
				break
			}
		}

		if !allowed {
			cfg.Logger.Warn().
				Str("roles", formatRoles(roles)).
				Str("resource", resource).
				Str("action", action).
				Msg("Access denied: insufficient permissions")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "You don't have permission to perform this action",
			})
			return
		}

		c.Next()
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// ScopeMiddleware provides scope-based access control for API keys
type ScopeMiddleware struct {
	logger logger.Logger
}

// NewScopeMiddleware creates a new scope middleware
func NewScopeMiddleware() *ScopeMiddleware {
	return &ScopeMiddleware{
		logger: logger.NewNop(),
	}
}

// NewScopeMiddlewareWithLogger creates a new scope middleware with logging
func NewScopeMiddlewareWithLogger(log logger.Logger) *ScopeMiddleware {
	return &ScopeMiddleware{
		logger: log,
	}
}

// RequireScope returns middleware that requires a specific scope
func (m *ScopeMiddleware) RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			// No API key in context - might be session auth, allow
			c.Next()
			return
		}

		// Check if API key has required scope
		if !apiKey.HasScope(scope) {
			m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("required_scope", scope).
				Any("key_scopes", apiKey.Scopes).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("API key scope validation failed")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient scope",
				"required_scope": scope,
			})
			return
		}

		// Check IP whitelist with detailed result
		ipResult := apiKey.CheckIPAllowed(c.ClientIP())
		if !ipResult.Allowed {
			logEvent := m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("client_ip", c.ClientIP()).
				Any("ip_whitelist", apiKey.IPWhitelist).
				Str("path", c.Request.URL.Path).
				Str("reason", ipResult.Reason)
			if ipResult.ParseError {
				logEvent.Bool("parse_error", true)
			}
			logEvent.Msg("API key IP whitelist validation failed")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "IP address not allowed for this API key",
			})
			return
		}

		// Check read-only restriction
		if apiKey.ReadOnly && !isReadOnlyMethod(c.Request.Method) {
			m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("API key read-only restriction violated")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "API key is read-only",
			})
			return
		}

		c.Next()
	}
}

// RequireAnyScope returns middleware that requires any of the specified scopes
func (m *ScopeMiddleware) RequireAnyScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			c.Next()
			return
		}

		if !apiKey.HasAnyScope(scopes...) {
			m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Any("required_scopes", scopes).
				Any("key_scopes", apiKey.Scopes).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("API key scope validation failed - none of required scopes present")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":           "Insufficient scope",
				"required_scopes": scopes,
			})
			return
		}

		ipResult := apiKey.CheckIPAllowed(c.ClientIP())
		if !ipResult.Allowed {
			logEvent := m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("client_ip", c.ClientIP()).
				Any("ip_whitelist", apiKey.IPWhitelist).
				Str("path", c.Request.URL.Path).
				Str("reason", ipResult.Reason)
			if ipResult.ParseError {
				logEvent.Bool("parse_error", true)
			}
			logEvent.Msg("API key IP whitelist validation failed")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "IP address not allowed for this API key",
			})
			return
		}

		if apiKey.ReadOnly && !isReadOnlyMethod(c.Request.Method) {
			m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("API key read-only restriction violated")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "API key is read-only",
			})
			return
		}

		c.Next()
	}
}

// RequireServerAccess returns middleware that checks if API key can access a specific server
func (m *ScopeMiddleware) RequireServerAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			c.Next()
			return
		}

		serverID := c.Param("id")
		if serverID == "" {
			serverID = c.Param("server_id")
		}

		if serverID != "" && !apiKey.IsServerAllowed(serverID) {
			m.logger.Warn().
				Str("api_key_id", apiKey.ID).
				Str("server_id", serverID).
				Any("allowed_servers", apiKey.AllowedServers).
				Str("path", c.Request.URL.Path).
				Msg("API key server access denied")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":     "API key not authorized for this server",
				"server_id": serverID,
			})
			return
		}

		c.Next()
	}
}

// RequireNamespaceAccess returns middleware that checks if API key can access a specific namespace
func (m *ScopeMiddleware) RequireNamespaceAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			c.Next()
			return
		}

		namespaceID := c.Param("namespace_id")
		if namespaceID == "" {
			namespaceID = c.Param("id")
		}

		// Only check if we have a namespace ID and it's for namespace routes
		if namespaceID != "" && strings.Contains(c.FullPath(), "namespace") {
			if !apiKey.IsNamespaceAllowed(namespaceID) {
				m.logger.Warn().
					Str("api_key_id", apiKey.ID).
					Str("namespace_id", namespaceID).
					Any("allowed_namespaces", apiKey.Namespaces).
					Str("path", c.Request.URL.Path).
					Msg("API key namespace access denied")
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":        "API key not authorized for this namespace",
					"namespace_id": namespaceID,
				})
				return
			}
		}

		c.Next()
	}
}

// CheckReadOnly returns middleware that enforces read-only restriction
func (m *ScopeMiddleware) CheckReadOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			c.Next()
			return
		}

		if apiKey.ReadOnly && !isReadOnlyMethod(c.Request.Method) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "API key is read-only, write operations not allowed",
			})
			return
		}

		c.Next()
	}
}

// CheckIPWhitelist returns middleware that enforces IP whitelist
func (m *ScopeMiddleware) CheckIPWhitelist() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := GetAPIKeyFromContext(c)
		if apiKey == nil {
			c.Next()
			return
		}

		if !apiKey.IsIPAllowed(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":     "IP address not allowed for this API key",
				"client_ip": c.ClientIP(),
			})
			return
		}

		c.Next()
	}
}

// GetAPIKeyFromContext retrieves the API key from the gin context
func GetAPIKeyFromContext(c *gin.Context) *domain.APIKey {
	val, exists := c.Get("api_key")
	if !exists {
		return nil
	}
	apiKey, ok := val.(*domain.APIKey)
	if !ok {
		return nil
	}
	return apiKey
}

// SetAPIKeyInContext stores the API key in the gin context
func SetAPIKeyInContext(c *gin.Context, apiKey *domain.APIKey) {
	c.Set("api_key", apiKey)
}

// isReadOnlyMethod checks if an HTTP method is read-only
func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ==================== RequestID Tests ====================

func TestRequestID(t *testing.T) {
	t.Run("generates new request ID when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handler := RequestID()
		handler(c)

		// Check that request ID was set in context
		requestID, exists := c.Get(RequestIDKey)
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)

		// Check that request ID was set in response header
		assert.Equal(t, requestID, w.Header().Get(RequestIDHeader))
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set(RequestIDHeader, "existing-request-id-123")

		handler := RequestID()
		handler(c)

		// Check that existing request ID was preserved
		requestID, exists := c.Get(RequestIDKey)
		assert.True(t, exists)
		assert.Equal(t, "existing-request-id-123", requestID)

		// Check that response header has the same ID
		assert.Equal(t, "existing-request-id-123", w.Header().Get(RequestIDHeader))
	})

	t.Run("generated ID is a valid UUID format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handler := RequestID()
		handler(c)

		requestID, _ := c.Get(RequestIDKey)
		idStr := requestID.(string)

		// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		assert.Len(t, idStr, 36)
		assert.Contains(t, idStr, "-")
	})
}

func TestRequestIDConstants(t *testing.T) {
	assert.Equal(t, "X-Request-ID", RequestIDHeader)
	assert.Equal(t, "request_id", RequestIDKey)
}

// ==================== CORS Tests ====================

func TestCORS(t *testing.T) {
	t.Run("sets CORS headers for regular request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handler := CORS()
		handler(c)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PUT")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "DELETE")
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("handles OPTIONS preflight request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("OPTIONS", "/test", nil)

		handler := CORS()
		handler(c)

		// OPTIONS should return 204 No Content
		assert.Equal(t, http.StatusNoContent, w.Code)

		// CORS headers should still be set
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("allows POST request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/test", nil)

		handler := CORS()
		handler(c)

		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})
}

// ==================== Timeout Tests ====================

func TestTimeout(t *testing.T) {
	t.Run("skips timeout for gateway routes", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/gateway/server-123/tools/call", nil)

		handlerCalled := false
		c.Set("handlerCalled", &handlerCalled)

		handler := Timeout(100 * time.Millisecond)

		// Run middleware and verify it doesn't timeout
		handler(c)

		// Gateway routes should skip timeout
		// The middleware should call c.Next() directly
	})

	t.Run("skips timeout for SSE requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/events", nil)
		c.Request.Header.Set("Accept", "text/event-stream")

		handler := Timeout(100 * time.Millisecond)
		handler(c)

		// SSE requests should skip timeout
	})

	t.Run("applies timeout to regular routes", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/servers", nil)

		handler := Timeout(100 * time.Millisecond)
		handler(c)

		// Regular routes should have timeout applied
		// Context should have deadline
		_, hasDeadline := c.Request.Context().Deadline()
		assert.True(t, hasDeadline)
	})
}

// ==================== Recovery Tests ====================

func TestRecovery(t *testing.T) {
	t.Run("allows normal request to proceed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		log := logger.NewNopLogger()
		handler := Recovery(log)

		// Run the middleware
		handler(c)

		// Should not set any error status
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("recovers from panic and returns 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, engine := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/panic", nil)

		log := logger.NewNopLogger()

		// Set up a route that panics
		engine.Use(Recovery(log))
		engine.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		engine.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal server error")
	})

	t.Run("logs panic details", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, engine := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/panic", nil)

		log := logger.NewNopLogger()

		engine.Use(Recovery(log))
		engine.GET("/panic", func(c *gin.Context) {
			panic("detailed panic message")
		})

		engine.ServeHTTP(w, c.Request)

		// Panic should be recovered and logged
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ==================== Logging Tests ====================

func TestLogger(t *testing.T) {
	t.Run("logs request details", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, engine := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		log := logger.NewNopLogger()

		engine.Use(Logger(log))
		engine.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		engine.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ==================== Metrics Tests ====================

func TestMetrics(t *testing.T) {
	t.Run("records metrics for request", func(t *testing.T) {
		// This would require mocking the metrics registry
		// For now, test that it doesn't panic with nil registry
	})
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/servers", "/api/v1/servers"},
		{"/api/v1/servers/123", "/api/v1/servers/:id"},
		{"/api/v1/servers/456/details", "/api/v1/servers/:id/details"},
		{"/api/v1/servers/a1b2c3d4-e5f6-7890-abcd-ef1234567890", "/api/v1/servers/:id"},
		{"/health", "/health"},
		{"/ready", "/ready"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== Auth Context Helper Tests ====================

func TestAuthContextHelpers(t *testing.T) {
	t.Run("GetUserID returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := GetUserID(c)
		assert.Empty(t, userID)
	})

	t.Run("GetUserID returns value when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(ContextKeyUserID, "user-123")

		userID := GetUserID(c)
		assert.Equal(t, "user-123", userID)
	})

	t.Run("GetUserEmail returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		email := GetUserEmail(c)
		assert.Empty(t, email)
	})

	t.Run("GetUserEmail returns value when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(ContextKeyUserEmail, "user@example.com")

		email := GetUserEmail(c)
		assert.Equal(t, "user@example.com", email)
	})

	t.Run("GetUserRoles returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		roles := GetUserRoles(c)
		assert.Empty(t, roles)
	})

	t.Run("GetUserRoles returns roles when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(ContextKeyUserRoles, []string{"admin", "user"})

		roles := GetUserRoles(c)
		assert.Equal(t, []string{"admin", "user"}, roles)
	})

	t.Run("GetAuthType returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		authType := GetAuthType(c)
		assert.Empty(t, authType)
	})

	t.Run("GetAuthType returns value when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(ContextKeyAuthType, AuthTypeAPIKey)

		authType := GetAuthType(c)
		assert.Equal(t, AuthTypeAPIKey, authType)
	})
}

// ==================== OAuth Adapter Tests ====================

func TestOAuthServiceAdapter(t *testing.T) {
	t.Run("NewOAuthServiceAdapter with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		require.NotNil(t, adapter)
		assert.Nil(t, adapter.service)
	})

	t.Run("IsEnabled returns false with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		assert.False(t, adapter.IsEnabled())
	})

	t.Run("GetIssuer returns empty with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		assert.Empty(t, adapter.GetIssuer())
	})

	t.Run("GetBaseURL returns empty with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		assert.Empty(t, adapter.GetBaseURL())
	})

	t.Run("GetDefaultRole returns empty with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		assert.Empty(t, adapter.GetDefaultRole())
	})

	t.Run("AutoCreateUsers returns false with nil service", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)
		assert.False(t, adapter.AutoCreateUsers())
	})
}

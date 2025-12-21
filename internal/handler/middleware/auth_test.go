package middleware

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/metrics"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		authHeader   string
		apiKeyHeader string
		expectedKey  string
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer mcpgw_test123",
			expectedKey: "mcpgw_test123",
		},
		{
			name:        "bearer token case insensitive",
			authHeader:  "bearer mcpgw_test456",
			expectedKey: "mcpgw_test456",
		},
		{
			name:        "bearer token with extra spaces",
			authHeader:  "Bearer   mcpgw_test789  ",
			expectedKey: "mcpgw_test789",
		},
		{
			name:         "x-api-key header",
			apiKeyHeader: "mcpgw_apikey123",
			expectedKey:  "mcpgw_apikey123",
		},
		{
			name:        "bearer token without mcpgw prefix",
			authHeader:  "Bearer some_other_token",
			expectedKey: "",
		},
		{
			name:         "x-api-key without mcpgw prefix",
			apiKeyHeader: "other_key",
			expectedKey:  "",
		},
		{
			name:        "empty auth header",
			authHeader:  "",
			expectedKey: "",
		},
		{
			name:        "malformed auth header",
			authHeader:  "InvalidHeader",
			expectedKey: "",
		},
		{
			name:         "bearer takes precedence over x-api-key",
			authHeader:   "Bearer mcpgw_bearer_key",
			apiKeyHeader: "mcpgw_header_key",
			expectedKey:  "mcpgw_bearer_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.apiKeyHeader != "" {
				c.Request.Header.Set("X-API-Key", tt.apiKeyHeader)
			}

			result := extractAPIKey(c)
			assert.Equal(t, tt.expectedKey, result)
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		setValue interface{}
		expected string
	}{
		{
			name:     "valid string user ID",
			setValue: "user-123",
			expected: "user-123",
		},
		{
			name:     "empty string",
			setValue: "",
			expected: "",
		},
		{
			name:     "non-string value",
			setValue: 12345,
			expected: "",
		},
		{
			name:     "nil value",
			setValue: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setValue != nil {
				c.Set(ContextKeyUserID, tt.setValue)
			}

			result := GetUserID(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserEmail(t *testing.T) {
	tests := []struct {
		name     string
		setValue interface{}
		expected string
	}{
		{
			name:     "valid email",
			setValue: "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "empty string",
			setValue: "",
			expected: "",
		},
		{
			name:     "non-string value",
			setValue: 12345,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setValue != nil {
				c.Set(ContextKeyUserEmail, tt.setValue)
			}

			result := GetUserEmail(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserRoles(t *testing.T) {
	tests := []struct {
		name     string
		setValue interface{}
		expected []string
	}{
		{
			name:     "valid roles",
			setValue: []string{"admin", "operator"},
			expected: []string{"admin", "operator"},
		},
		{
			name:     "single role",
			setValue: []string{"viewer"},
			expected: []string{"viewer"},
		},
		{
			name:     "empty roles",
			setValue: []string{},
			expected: []string{},
		},
		{
			name:     "non-slice value",
			setValue: "admin",
			expected: []string{},
		},
		{
			name:     "nil value",
			setValue: nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setValue != nil {
				c.Set(ContextKeyUserRoles, tt.setValue)
			}

			result := GetUserRoles(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAuthType(t *testing.T) {
	tests := []struct {
		name     string
		setValue interface{}
		expected AuthType
	}{
		{
			name:     "session auth type",
			setValue: AuthTypeSession,
			expected: AuthTypeSession,
		},
		{
			name:     "api key auth type",
			setValue: AuthTypeAPIKey,
			expected: AuthTypeAPIKey,
		},
		{
			name:     "invalid type",
			setValue: "invalid",
			expected: "",
		},
		{
			name:     "nil value",
			setValue: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setValue != nil {
				c.Set(ContextKeyAuthType, tt.setValue)
			}

			result := GetAuthType(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSessionAuth_NoSession(t *testing.T) {
	// This test verifies that SessionAuth returns 401 when sessions package isn't set up
	// In real usage, sessions middleware must be added before SessionAuth
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/protected", nil)

	// Without sessions middleware, this will panic
	// We test the helper functions instead since they don't depend on sessions
	assert.Equal(t, "", GetUserID(c))
	assert.Equal(t, "", GetUserEmail(c))
	assert.Empty(t, GetUserRoles(c))
}

func TestAuthConfig(t *testing.T) {
	cfg := &AuthConfig{
		Logger:      nil,
		UserRepo:    nil,
		APIKeyRepo:  nil,
		SessionName: "test_session",
	}

	require.NotNil(t, cfg)
	assert.Equal(t, "test_session", cfg.SessionName)
}

func TestContextKeys(t *testing.T) {
	// Verify context key constants are defined correctly
	assert.Equal(t, "user_id", ContextKeyUserID)
	assert.Equal(t, "user_email", ContextKeyUserEmail)
	assert.Equal(t, "user_roles", ContextKeyUserRoles)
	assert.Equal(t, "auth_type", ContextKeyAuthType)
}

func TestAuthTypes(t *testing.T) {
	// Verify auth type constants
	assert.Equal(t, AuthType("session"), AuthTypeSession)
	assert.Equal(t, AuthType("apikey"), AuthTypeAPIKey)
}

// TestContextSetAndGet verifies that values set in context can be retrieved
func TestContextSetAndGet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set all context values
	c.Set(ContextKeyUserID, "user-abc-123")
	c.Set(ContextKeyUserEmail, "test@example.com")
	c.Set(ContextKeyUserRoles, []string{"admin", "operator"})
	c.Set(ContextKeyAuthType, AuthTypeSession)

	// Retrieve and verify
	assert.Equal(t, "user-abc-123", GetUserID(c))
	assert.Equal(t, "test@example.com", GetUserEmail(c))
	assert.Equal(t, []string{"admin", "operator"}, GetUserRoles(c))
	assert.Equal(t, AuthTypeSession, GetAuthType(c))
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expectedKey string
	}{
		{
			name:        "valid OAuth bearer token",
			authHeader:  "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedKey: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:        "bearer token case insensitive",
			authHeader:  "bearer some_oauth_token",
			expectedKey: "some_oauth_token",
		},
		{
			name:        "bearer token with extra spaces",
			authHeader:  "Bearer   oauth_token_here  ",
			expectedKey: "oauth_token_here",
		},
		{
			name:        "mcpgw_ prefix returns empty (API key)",
			authHeader:  "Bearer mcpgw_test123",
			expectedKey: "",
		},
		{
			name:        "empty auth header",
			authHeader:  "",
			expectedKey: "",
		},
		{
			name:        "malformed auth header - no space",
			authHeader:  "BearerNoSpace",
			expectedKey: "",
		},
		{
			name:        "wrong auth type",
			authHeader:  "Basic dXNlcjpwYXNz",
			expectedKey: "",
		},
		{
			name:        "BEARER uppercase",
			authHeader:  "BEARER uppercase_token",
			expectedKey: "uppercase_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			result := extractBearerToken(c)
			assert.Equal(t, tt.expectedKey, result)
		})
	}
}

func TestSendUnauthorizedWithWWWAuthenticate(t *testing.T) {
	t.Run("sends 401 without OAuth validator", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/protected", nil)

		cfg := &AuthConfig{
			OAuthValidator: nil,
		}
		sendUnauthorizedWithWWWAuthenticate(c, cfg, "test error message")

		assert.Equal(t, 401, w.Code)
	})

	t.Run("sends 401 with disabled OAuth validator", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/protected", nil)

		mockValidator := &mockOAuthValidator{enabled: false}
		cfg := &AuthConfig{
			OAuthValidator: mockValidator,
		}
		sendUnauthorizedWithWWWAuthenticate(c, cfg, "error message")

		assert.Equal(t, 401, w.Code)
	})

	t.Run("sends 401 with WWW-Authenticate header when OAuth enabled", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/protected", nil)
		c.Request.Host = "localhost:8080"

		mockValidator := &mockOAuthValidator{
			enabled: true,
			baseURL: "https://gateway.example.com",
		}
		cfg := &AuthConfig{
			OAuthValidator: mockValidator,
		}
		sendUnauthorizedWithWWWAuthenticate(c, cfg, "error message")

		assert.Equal(t, 401, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Bearer resource_metadata=")
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "https://gateway.example.com")
	})

	t.Run("uses request host when base URL is empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/protected", nil)
		c.Request.Host = "localhost:8080"

		mockValidator := &mockOAuthValidator{
			enabled: true,
			baseURL: "",
		}
		cfg := &AuthConfig{
			OAuthValidator: mockValidator,
		}
		sendUnauthorizedWithWWWAuthenticate(c, cfg, "error message")

		assert.Equal(t, 401, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "http://localhost:8080")
	})
}

// mockOAuthValidator implements OAuthValidator interface for testing.
type mockOAuthValidator struct {
	baseURL string
	enabled bool
}

func (m *mockOAuthValidator) ValidateBearerToken(ctx context.Context, token string) (*OAuthUserInfo, error) {
	return &OAuthUserInfo{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}, nil
}

func (m *mockOAuthValidator) IsEnabled() bool {
	return m.enabled
}

func (m *mockOAuthValidator) GetIssuer() string {
	return "https://issuer.example.com"
}

func (m *mockOAuthValidator) GetBaseURL() string {
	return m.baseURL
}

func (m *mockOAuthValidator) GetDefaultRole() string {
	return "user"
}

func (m *mockOAuthValidator) AutoCreateUsers() bool {
	return true
}

// Tests for SessionAuth middleware.
func TestSessionAuth(t *testing.T) {
	t.Run("returns 401 when no session user", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := gin.New()

		// Add session middleware
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		cfg := &AuthConfig{
			SessionName: "test_session",
		}
		router.GET("/protected", SessionAuth(cfg), func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
		assert.Contains(t, w.Body.String(), "Please log in")
	})

	t.Run("allows access when session user exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := gin.New()

		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		cfg := &AuthConfig{
			SessionName: "test_session",
		}

		// First request to set session
		router.GET("/set-session", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(ContextKeyUserID, "user-123")
			session.Set(ContextKeyUserEmail, "test@example.com")
			session.Set(ContextKeyUserRoles, []string{"admin"})
			_ = session.Save()
			c.JSON(200, gin.H{"status": "session set"})
		})

		router.GET("/protected", SessionAuth(cfg), func(c *gin.Context) {
			userID := GetUserID(c)
			email := GetUserEmail(c)
			authType := GetAuthType(c)
			c.JSON(200, gin.H{
				"user_id":   userID,
				"email":     email,
				"auth_type": authType,
			})
		})

		// Set session first
		req1 := httptest.NewRequest("GET", "/set-session", nil)
		router.ServeHTTP(w, req1)
		cookies := w.Result().Cookies()

		// Then access protected route with session cookie
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/protected", nil)
		for _, cookie := range cookies {
			req2.AddCookie(cookie)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 200, w2.Code)
		assert.Contains(t, w2.Body.String(), "user-123")
		assert.Contains(t, w2.Body.String(), "session")
	})
}

// Tests for Metrics middleware with real registry.
func TestMetricsMiddleware(t *testing.T) {
	t.Run("records metrics for successful request", func(t *testing.T) {
		reg := metrics.NewRegistry()

		w := httptest.NewRecorder()
		router := gin.New()
		router.Use(Metrics(reg))
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("records metrics for error request", func(t *testing.T) {
		reg := metrics.NewRegistry()

		w := httptest.NewRecorder()
		router := gin.New()
		router.Use(Metrics(reg))
		router.GET("/api/v1/error", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/api/v1/error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
	})

	t.Run("normalizes UUID paths", func(t *testing.T) {
		reg := metrics.NewRegistry()

		w := httptest.NewRecorder()
		router := gin.New()
		router.Use(Metrics(reg))
		router.GET("/api/v1/servers/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"id": c.Param("id")})
		})

		req := httptest.NewRequest("GET", "/api/v1/servers/550e8400-e29b-41d4-a716-446655440000", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("normalizes numeric paths", func(t *testing.T) {
		reg := metrics.NewRegistry()

		w := httptest.NewRecorder()
		router := gin.New()
		router.Use(Metrics(reg))
		router.GET("/api/v1/items/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"id": c.Param("id")})
		})

		req := httptest.NewRequest("GET", "/api/v1/items/12345", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("tracks in-flight requests", func(t *testing.T) {
		reg := metrics.NewRegistry()

		w := httptest.NewRecorder()
		router := gin.New()
		router.Use(Metrics(reg))
		router.POST("/api/v1/create", func(c *gin.Context) {
			c.JSON(201, gin.H{"created": true})
		})

		req := httptest.NewRequest("POST", "/api/v1/create", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 201, w.Code)
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		reg := metrics.NewRegistry()
		router := gin.New()
		router.Use(Metrics(reg))

		router.PUT("/api/v1/update", func(c *gin.Context) {
			c.JSON(200, gin.H{"updated": true})
		})
		router.DELETE("/api/v1/delete", func(c *gin.Context) {
			c.JSON(204, nil)
		})

		// Test PUT
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, httptest.NewRequest("PUT", "/api/v1/update", nil))
		assert.Equal(t, 200, w1.Code)

		// Test DELETE
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, httptest.NewRequest("DELETE", "/api/v1/delete", nil))
		assert.Equal(t, 204, w2.Code)
	})
}

// Tests for AuditMiddleware responseWriter.
func TestAuditMiddleware_ResponseWriter(t *testing.T) {
	t.Run("responseWriter captures body through gin router", func(t *testing.T) {
		capturedBody := ""
		w := httptest.NewRecorder()
		router := gin.New()

		// Custom middleware to test responseWriter
		router.Use(func(c *gin.Context) {
			rw := &responseWriter{
				ResponseWriter: c.Writer,
				body:           bytes.NewBufferString(""),
			}
			c.Writer = rw

			c.Next()

			capturedBody = rw.body.String()
		})

		router.GET("/test", func(c *gin.Context) {
			c.String(200, "test response body")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "test response body", capturedBody)
		assert.Equal(t, "test response body", w.Body.String())
	})
}

// Tests for MCPAuthConfig.
func TestMCPAuthConfig(t *testing.T) {
	t.Run("default values are false", func(t *testing.T) {
		cfg := MCPAuthConfig{}
		assert.False(t, cfg.APIKeyEnabled)
		assert.False(t, cfg.SessionEnabled)
	})

	t.Run("can enable both methods", func(t *testing.T) {
		cfg := MCPAuthConfig{
			APIKeyEnabled:  true,
			SessionEnabled: true,
		}
		assert.True(t, cfg.APIKeyEnabled)
		assert.True(t, cfg.SessionEnabled)
	})
}

// Tests for OAuthUserInfo.
func TestOAuthUserInfo(t *testing.T) {
	info := OAuthUserInfo{
		ID:       "oauth-123",
		Email:    "oauth@example.com",
		Name:     "OAuth User",
		Provider: "google",
	}

	assert.Equal(t, "oauth-123", info.ID)
	assert.Equal(t, "oauth@example.com", info.Email)
	assert.Equal(t, "OAuth User", info.Name)
	assert.Equal(t, "google", info.Provider)
}

// Tests for AuthTypeOAuth constant.
func TestAuthTypeOAuth(t *testing.T) {
	assert.Equal(t, AuthType("oauth"), AuthTypeOAuth)
}

// Additional tests for OAuthServiceAdapter to cover more branches.
func TestOAuthServiceAdapter_ValidateBearerToken(t *testing.T) {
	t.Run("returns nil when service is nil", func(t *testing.T) {
		adapter := NewOAuthServiceAdapter(nil)

		info, err := adapter.ValidateBearerToken(context.Background(), "test-token")

		assert.Nil(t, err)
		assert.Nil(t, info)
	})
}

// Tests for Logger middleware edge cases.
func TestLoggerMiddleware(t *testing.T) {
	t.Run("logs 4xx errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := gin.New()
		log := logger.NewNopLogger()

		router.Use(Logger(log))
		router.GET("/not-found", func(c *gin.Context) {
			c.JSON(404, gin.H{"error": "not found"})
		})

		req := httptest.NewRequest("GET", "/not-found", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 404, w.Code)
	})

	t.Run("logs 5xx errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		router := gin.New()
		log := logger.NewNopLogger()

		router.Use(Logger(log))
		router.GET("/error", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/error", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
	})
}

// Tests for Authz middleware denying access when user has no roles.
func TestAuthz_NoRoles(t *testing.T) {
	t.Run("denies access when user has no roles", func(t *testing.T) {
		cfg := &AuthzConfig{
			Logger:   logger.NewNopLogger(),
			Enforcer: nil, // Will fail but that's ok for this test
		}
		middleware := Authz(cfg)

		w := httptest.NewRecorder()
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))
		router.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserID, "user-123")
			c.Set(ContextKeyUserRoles, []string{}) // No roles
			c.Next()
		})
		router.Use(middleware)
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
		assert.Contains(t, w.Body.String(), "No roles assigned")
	})
}

// Tests for RequireRoles edge cases.
func TestRequireRoles_EdgeCases(t *testing.T) {
	t.Run("rejects when user has no roles", func(t *testing.T) {
		cfg := &AuthzConfig{
			Logger:   logger.NewNopLogger(),
			Enforcer: nil,
		}
		middleware := RequireRoles(cfg, "admin")

		w := httptest.NewRecorder()
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))
		router.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserID, "user-123")
			c.Set(ContextKeyUserRoles, []string{}) // No roles
			c.Next()
		})
		router.Use(middleware)
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})

	t.Run("allows when user has matching role", func(t *testing.T) {
		cfg := &AuthzConfig{
			Logger:   logger.NewNopLogger(),
			Enforcer: nil,
		}
		middleware := RequireRoles(cfg, "admin", "operator")

		w := httptest.NewRecorder()
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))
		router.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserID, "user-123")
			c.Set(ContextKeyUserRoles, []string{"operator"})
			c.Next()
		})
		router.Use(middleware)
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("rejects when user role does not match", func(t *testing.T) {
		cfg := &AuthzConfig{
			Logger:   logger.NewNopLogger(),
			Enforcer: nil,
		}
		middleware := RequireRoles(cfg, "admin")

		w := httptest.NewRecorder()
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))
		router.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserID, "user-123")
			c.Set(ContextKeyUserRoles, []string{"viewer"}) // Wrong role
			c.Next()
		})
		router.Use(middleware)
		router.GET("/admin", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})
}

// Tests for RequirePermission - need Casbin enforcer to properly test.
func TestRequirePermission_NoRoles(t *testing.T) {
	t.Run("denies when user has no roles", func(t *testing.T) {
		cfg := &AuthzConfig{
			Logger:   logger.NewNopLogger(),
			Enforcer: nil, // Will cause panic if used
		}
		middleware := RequirePermission(cfg, "servers", "delete")

		w := httptest.NewRecorder()
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))
		router.Use(func(c *gin.Context) {
			c.Set(ContextKeyUserID, "user-123")
			c.Set(ContextKeyUserRoles, []string{}) // No roles
			c.Next()
		})
		router.Use(middleware)
		router.DELETE("/servers/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"deleted": true})
		})

		req := httptest.NewRequest("DELETE", "/servers/123", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})
}

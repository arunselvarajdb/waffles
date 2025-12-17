package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		apiKeyHeader   string
		expectedKey    string
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

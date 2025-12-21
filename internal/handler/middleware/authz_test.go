package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// createTestEnforcer creates a Casbin enforcer with test policies
func createTestEnforcer(t *testing.T) *casbin.Enforcer {
	modelText := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`
	m, err := model.NewModelFromString(modelText)
	require.NoError(t, err)

	enforcer, err := casbin.NewEnforcer(m)
	require.NoError(t, err)

	// Add test policies
	policies := [][]string{
		{"admin", "/api/v1/*", "*"},
		{"operator", "/api/v1/servers", "*"},
		{"operator", "/api/v1/servers/*", "*"},
		{"viewer", "/api/v1/servers", "GET"},
		{"viewer", "/api/v1/servers/*", "GET"},
		{"user", "/api/v1/me", "GET"},
	}

	for _, p := range policies {
		_, err := enforcer.AddPolicy(p)
		require.NoError(t, err)
	}

	// Add role hierarchy
	_, err = enforcer.AddGroupingPolicy("admin", "operator")
	require.NoError(t, err)
	_, err = enforcer.AddGroupingPolicy("operator", "viewer")
	require.NoError(t, err)
	_, err = enforcer.AddGroupingPolicy("viewer", "user")
	require.NoError(t, err)

	return enforcer
}

func TestFormatRoles(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		expected string
	}{
		{
			name:     "multiple roles",
			roles:    []string{"admin", "operator", "viewer"},
			expected: "admin, operator, viewer",
		},
		{
			name:     "single role",
			roles:    []string{"admin"},
			expected: "admin",
		},
		{
			name:     "empty roles",
			roles:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRoles(tt.roles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthz(t *testing.T) {
	enforcer := createTestEnforcer(t)
	log := logger.NewNopLogger()

	cfg := &AuthzConfig{
		Logger:   log,
		Enforcer: enforcer,
	}

	tests := []struct {
		name           string
		roles          []string
		path           string
		method         string
		expectedStatus int
		shouldAllow    bool
	}{
		{
			name:           "admin can access anything",
			roles:          []string{"admin"},
			path:           "/api/v1/servers",
			method:         "DELETE",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
		{
			name:           "admin can access user endpoints",
			roles:          []string{"admin"},
			path:           "/api/v1/me",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
		{
			name:           "operator can manage servers",
			roles:          []string{"operator"},
			path:           "/api/v1/servers",
			method:         "POST",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
		{
			name:           "viewer can read servers",
			roles:          []string{"viewer"},
			path:           "/api/v1/servers",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
		{
			name:           "viewer cannot delete servers",
			roles:          []string{"viewer"},
			path:           "/api/v1/servers/123",
			method:         "DELETE",
			expectedStatus: http.StatusForbidden,
			shouldAllow:    false,
		},
		{
			name:           "user can access me endpoint",
			roles:          []string{"user"},
			path:           "/api/v1/me",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
		{
			name:           "user cannot access servers",
			roles:          []string{"user"},
			path:           "/api/v1/servers",
			method:         "GET",
			expectedStatus: http.StatusForbidden,
			shouldAllow:    false,
		},
		{
			name:           "no roles - forbidden",
			roles:          []string{},
			path:           "/api/v1/servers",
			method:         "GET",
			expectedStatus: http.StatusForbidden,
			shouldAllow:    false,
		},
		{
			name:           "multiple roles - one allowed",
			roles:          []string{"user", "viewer"},
			path:           "/api/v1/servers",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldAllow:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			// Set up the route with Authz middleware
			router.Use(func(c *gin.Context) {
				// Set roles before authz check
				c.Set(ContextKeyUserRoles, tt.roles)
				c.Next()
			})
			router.Use(Authz(cfg))
			router.Any("/*path", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request = httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(w, c.Request)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.shouldAllow {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "forbidden", response["error"])
			}
		})
	}
}

func TestRequireRoles(t *testing.T) {
	log := logger.NewNopLogger()
	enforcer := createTestEnforcer(t)

	cfg := &AuthzConfig{
		Logger:   log,
		Enforcer: enforcer,
	}

	tests := []struct {
		name           string
		userRoles      []string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "user has required role",
			userRoles:      []string{"admin"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has one of required roles",
			userRoles:      []string{"operator"},
			requiredRoles:  []string{"admin", "operator"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required role",
			userRoles:      []string{"viewer"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has multiple roles including required",
			userRoles:      []string{"viewer", "admin"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty user roles",
			userRoles:      []string{},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			router.Use(func(c *gin.Context) {
				c.Set(ContextKeyUserRoles, tt.userRoles)
				c.Next()
			})
			router.Use(RequireRoles(cfg, tt.requiredRoles...))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request = httptest.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, c.Request)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	enforcer := createTestEnforcer(t)
	log := logger.NewNopLogger()

	cfg := &AuthzConfig{
		Logger:   log,
		Enforcer: enforcer,
	}

	tests := []struct {
		name           string
		userRoles      []string
		resource       string
		action         string
		expectedStatus int
	}{
		{
			name:           "admin can do anything on servers",
			userRoles:      []string{"admin"},
			resource:       "/api/v1/servers",
			action:         "DELETE",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "viewer can only read servers",
			userRoles:      []string{"viewer"},
			resource:       "/api/v1/servers",
			action:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "viewer cannot delete servers",
			userRoles:      []string{"viewer"},
			resource:       "/api/v1/servers",
			action:         "DELETE",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user without permission",
			userRoles:      []string{"user"},
			resource:       "/api/v1/servers",
			action:         "GET",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			router.Use(func(c *gin.Context) {
				c.Set(ContextKeyUserRoles, tt.userRoles)
				c.Next()
			})
			router.Use(RequirePermission(cfg, tt.resource, tt.action))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			c.Request = httptest.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, c.Request)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAuthzConfig(t *testing.T) {
	enforcer := createTestEnforcer(t)
	log := logger.NewNopLogger()

	cfg := &AuthzConfig{
		Logger:   log,
		Enforcer: enforcer,
	}

	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Enforcer)
}

func TestRoleHierarchy(t *testing.T) {
	enforcer := createTestEnforcer(t)

	// Test that admin inherits from operator
	allowed, err := enforcer.Enforce("admin", "/api/v1/servers", "POST")
	assert.NoError(t, err)
	assert.True(t, allowed, "admin should have operator permissions")

	// Test that operator inherits from viewer
	allowed, err = enforcer.Enforce("operator", "/api/v1/servers", "GET")
	assert.NoError(t, err)
	assert.True(t, allowed, "operator should have viewer permissions")

	// Test that viewer inherits from user
	allowed, err = enforcer.Enforce("viewer", "/api/v1/me", "GET")
	assert.NoError(t, err)
	assert.True(t, allowed, "viewer should have user permissions")
}

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/handler/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestScopeMiddleware_RequireScope(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		scope          string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			scope:          "servers:read",
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:  "API key with required scope allows request",
			scope: "servers:read",
			apiKey: &domain.APIKey{
				ID:     "key-1",
				Scopes: []string{"servers:read", "servers:write"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "API key without required scope denies request",
			scope: "servers:write",
			apiKey: &domain.APIKey{
				ID:     "key-1",
				Scopes: []string{"servers:read"},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:  "API key with empty scopes allows any scope (backward compat)",
			scope: "servers:write",
			apiKey: &domain.APIKey{
				ID:     "key-1",
				Scopes: []string{},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			if tt.apiKey != nil {
				middleware.SetAPIKeyInContext(c, tt.apiKey)
			}

			handler := scopeMiddleware.RequireScope(tt.scope)
			handler(c)

			if tt.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestScopeMiddleware_RequireAnyScope(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		scopes         []string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			scopes:         []string{"servers:read", "servers:write"},
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "API key with one of required scopes allows request",
			scopes: []string{"servers:read", "servers:write"},
			apiKey: &domain.APIKey{
				ID:     "key-1",
				Scopes: []string{"servers:read"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "API key with none of required scopes denies request",
			scopes: []string{"servers:read", "servers:write"},
			apiKey: &domain.APIKey{
				ID:     "key-1",
				Scopes: []string{"gateway:execute"},
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			if tt.apiKey != nil {
				middleware.SetAPIKeyInContext(c, tt.apiKey)
			}

			handler := scopeMiddleware.RequireAnyScope(tt.scopes...)
			handler(c)

			if tt.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestScopeMiddleware_RequireServerAccess(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		serverID       string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			serverID:       "server-1",
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key with allowed server allows request",
			serverID: "server-1",
			apiKey: &domain.APIKey{
				ID:             "key-1",
				AllowedServers: []string{"server-1", "server-2"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key without allowed server denies request",
			serverID: "server-3",
			apiKey: &domain.APIKey{
				ID:             "key-1",
				AllowedServers: []string{"server-1", "server-2"},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:     "API key with empty allowed servers allows any server",
			serverID: "server-any",
			apiKey: &domain.APIKey{
				ID:             "key-1",
				AllowedServers: []string{},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.serverID}}

			if tt.apiKey != nil {
				middleware.SetAPIKeyInContext(c, tt.apiKey)
			}

			handler := scopeMiddleware.RequireServerAccess()
			handler(c)

			if tt.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestScopeMiddleware_CheckReadOnly(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		method         string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			method:         "POST",
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "read-only API key allows GET",
			method: "GET",
			apiKey: &domain.APIKey{
				ID:       "key-1",
				ReadOnly: true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "read-only API key denies POST",
			method: "POST",
			apiKey: &domain.APIKey{
				ID:       "key-1",
				ReadOnly: true,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "read-only API key denies DELETE",
			method: "DELETE",
			apiKey: &domain.APIKey{
				ID:       "key-1",
				ReadOnly: true,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "non-read-only API key allows POST",
			method: "POST",
			apiKey: &domain.APIKey{
				ID:       "key-1",
				ReadOnly: false,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, "/test", nil)

			if tt.apiKey != nil {
				middleware.SetAPIKeyInContext(c, tt.apiKey)
			}

			handler := scopeMiddleware.CheckReadOnly()
			handler(c)

			if tt.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestScopeMiddleware_CheckIPWhitelist(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		clientIP       string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			clientIP:       "192.168.1.100",
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key with empty IP whitelist allows any IP",
			clientIP: "192.168.1.100",
			apiKey: &domain.APIKey{
				ID:          "key-1",
				IPWhitelist: []string{},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key with matching IP allows request",
			clientIP: "192.168.1.100",
			apiKey: &domain.APIKey{
				ID:          "key-1",
				IPWhitelist: []string{"192.168.1.100"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key with matching CIDR allows request",
			clientIP: "192.168.1.100",
			apiKey: &domain.APIKey{
				ID:          "key-1",
				IPWhitelist: []string{"192.168.1.0/24"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "API key with non-matching IP denies request",
			clientIP: "10.0.0.1",
			apiKey: &domain.APIKey{
				ID:          "key-1",
				IPWhitelist: []string{"192.168.1.0/24"},
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.RemoteAddr = tt.clientIP + ":12345"

			if tt.apiKey != nil {
				middleware.SetAPIKeyInContext(c, tt.apiKey)
			}

			handler := scopeMiddleware.CheckIPWhitelist()
			handler(c)

			if tt.expectedStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestScopeMiddleware_RequireNamespaceAccess(t *testing.T) {
	scopeMiddleware := middleware.NewScopeMiddleware()

	tests := []struct {
		name           string
		namespaceID    string
		apiKey         *domain.APIKey
		expectedStatus int
	}{
		{
			name:           "no API key allows request",
			namespaceID:    "ns-1",
			apiKey:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "API key with allowed namespace allows request",
			namespaceID: "ns-1",
			apiKey: &domain.APIKey{
				ID:         "key-1",
				Namespaces: []string{"ns-1", "ns-2"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "API key without allowed namespace denies request",
			namespaceID: "ns-3",
			apiKey: &domain.APIKey{
				ID:         "key-1",
				Namespaces: []string{"ns-1", "ns-2"},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "API key with empty namespaces allows any namespace",
			namespaceID: "ns-any",
			apiKey: &domain.APIKey{
				ID:         "key-1",
				Namespaces: []string{},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			engine := gin.New()

			// Set up the API key in context and run the middleware
			engine.GET("/api/v1/namespaces/:id", func(c *gin.Context) {
				if tt.apiKey != nil {
					middleware.SetAPIKeyInContext(c, tt.apiKey)
				}

				handler := scopeMiddleware.RequireNamespaceAccess()
				handler(c)

				if !c.IsAborted() {
					c.Status(http.StatusOK)
				}
			})

			req := httptest.NewRequest("GET", "/api/v1/namespaces/"+tt.namespaceID, nil)
			engine.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetAPIKeyFromContext(t *testing.T) {
	t.Run("returns nil when no API key in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		key := middleware.GetAPIKeyFromContext(c)
		assert.Nil(t, key)
	})

	t.Run("returns API key when set in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		expectedKey := &domain.APIKey{
			ID:   "test-key",
			Name: "Test Key",
		}
		middleware.SetAPIKeyInContext(c, expectedKey)

		key := middleware.GetAPIKeyFromContext(c)
		assert.NotNil(t, key)
		assert.Equal(t, expectedKey.ID, key.ID)
		assert.Equal(t, expectedKey.Name, key.Name)
	})
}

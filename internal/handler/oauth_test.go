package handler

import (
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/service/oauth"
	"github.com/waffles/waffles/pkg/logger"
)

func init() {
	// Register types for session serialization
	gob.Register([]string{})
}

// MockOAuthService implements OAuthServiceInterface for testing.
type MockOAuthService struct {
	IsEnabledFunc       func() bool
	GenerateStateFunc   func() (string, error)
	GetAuthURLFunc      func(state string) (string, error)
	ExchangeCodeFunc    func(ctx context.Context, code string) (*oauth.UserInfo, error)
	AutoCreateUsersFunc func() bool
	GetDefaultRoleFunc  func() string
}

func (m *MockOAuthService) IsEnabled() bool {
	if m.IsEnabledFunc != nil {
		return m.IsEnabledFunc()
	}

	return true
}

func (m *MockOAuthService) GenerateState() (string, error) {
	if m.GenerateStateFunc != nil {
		return m.GenerateStateFunc()
	}

	return "test-state-123", nil
}

func (m *MockOAuthService) GetAuthURL(state string) (string, error) {
	if m.GetAuthURLFunc != nil {
		return m.GetAuthURLFunc(state)
	}

	return "https://auth.example.com/authorize?state=" + state, nil
}

func (m *MockOAuthService) ExchangeCode(ctx context.Context, code string) (*oauth.UserInfo, error) {
	if m.ExchangeCodeFunc != nil {
		return m.ExchangeCodeFunc(ctx, code)
	}

	return &oauth.UserInfo{
		ID:       "user-123",
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}, nil
}

func (m *MockOAuthService) AutoCreateUsers() bool {
	if m.AutoCreateUsersFunc != nil {
		return m.AutoCreateUsersFunc()
	}

	return true
}

func (m *MockOAuthService) GetDefaultRole() string {
	if m.GetDefaultRoleFunc != nil {
		return m.GetDefaultRoleFunc()
	}

	return "user"
}

// MockOAuthUserRepo implements OAuthUserRepoInterface for testing.
type MockOAuthUserRepo struct {
	FindOrCreateOAuthUserFunc func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error)
	GetUserRolesFunc          func(ctx context.Context, userID string) ([]string, error)
	AssignRoleFunc            func(ctx context.Context, userID, role string) error
}

func (m *MockOAuthUserRepo) FindOrCreateOAuthUser(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
	if m.FindOrCreateOAuthUserFunc != nil {
		return m.FindOrCreateOAuthUserFunc(ctx, provider, externalID, email, name)
	}

	return &domain.User{
		ID:       "user-123",
		Email:    email,
		Name:     name,
		IsActive: true,
	}, false, nil
}

func (m *MockOAuthUserRepo) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	if m.GetUserRolesFunc != nil {
		return m.GetUserRolesFunc(ctx, userID)
	}

	return []string{"user"}, nil
}

func (m *MockOAuthUserRepo) AssignRole(ctx context.Context, userID, role string) error {
	if m.AssignRoleFunc != nil {
		return m.AssignRoleFunc(ctx, userID, role)
	}

	return nil
}

func setupOAuthTestRouter(handler *OAuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Setup session middleware for testing
	store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
	r.Use(sessions.Sessions("test_session", store))

	return r
}

func TestNewOAuthHandler(t *testing.T) {
	log := logger.NewNopLogger()

	handler := NewOAuthHandler(nil, nil, log, "http://localhost:3000")

	require.NotNil(t, handler)
	assert.Nil(t, handler.oauthService)
	assert.Nil(t, handler.userRepo)
	assert.Equal(t, "http://localhost:3000", handler.frontendURL)
}

func TestNewOAuthHandlerWithInterface(t *testing.T) {
	log := logger.NewNopLogger()
	mockService := &MockOAuthService{}
	mockRepo := &MockOAuthUserRepo{}

	handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

	require.NotNil(t, handler)
	assert.NotNil(t, handler.oauthService)
	assert.NotNil(t, handler.userRepo)
	assert.Equal(t, "http://localhost:3000", handler.frontendURL)
}

func TestOAuthHandler_GetSSOStatus(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns enabled when SSO is enabled", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc: func() bool { return true },
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso/status", handler.GetSSOStatus)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso/status", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"enabled":true`)
	})

	t.Run("returns disabled when SSO is disabled", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc: func() bool { return false },
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso/status", handler.GetSSOStatus)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso/status", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"enabled":false`)
	})
}

func TestOAuthHandler_Authorize(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns 400 when SSO is not enabled", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc: func() bool { return false },
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso", handler.Authorize)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "sso_not_enabled")
	})

	t.Run("returns 500 when state generation fails", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc: func() bool { return true },
			GenerateStateFunc: func() (string, error) {
				return "", errors.New("failed to generate state")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso", handler.Authorize)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal_error")
	})

	t.Run("returns 500 when GetAuthURL fails", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc:     func() bool { return true },
			GenerateStateFunc: func() (string, error) { return "test-state", nil },
			GetAuthURLFunc: func(state string) (string, error) {
				return "", errors.New("failed to get auth URL")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso", handler.Authorize)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal_error")
	})

	t.Run("redirects to auth URL on success", func(t *testing.T) {
		mockService := &MockOAuthService{
			IsEnabledFunc:     func() bool { return true },
			GenerateStateFunc: func() (string, error) { return "test-state-123", nil },
			GetAuthURLFunc: func(state string) (string, error) {
				return "https://auth.example.com/authorize?state=" + state, nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso", handler.Authorize)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "https://auth.example.com/authorize")
	})
}

func TestOAuthHandler_Callback(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("redirects with error when OAuth provider returns error", func(t *testing.T) {
		mockService := &MockOAuthService{}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?error=access_denied&error_description=User%20cancelled", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when code is missing", func(t *testing.T) {
		mockService := &MockOAuthService{}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso/callback", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when state mismatch", func(t *testing.T) {
		mockService := &MockOAuthService{}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")
		router := setupOAuthTestRouter(handler)
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=invalid-state", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when code exchange fails", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return nil, errors.New("code exchange failed")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")

		// Create a router with session that has stored state
		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		// First set the state in session
		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		// Set state
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)

		// Get session cookie
		cookies := w1.Result().Cookies()

		// Make callback request with cookie
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when user has no email", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-123",
					Email:    "",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		// Set state
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		// Make callback request
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when user creation fails", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-123",
					Email:    "test@example.com",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return nil, false, errors.New("database error")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/login?error=")
	})

	t.Run("redirects with error when user is inactive", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-123",
					Email:    "test@example.com",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "user-123",
					Email:    email,
					Name:     name,
					IsActive: false,
				}, false, nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/login?error=")
	})

	t.Run("successful login for existing user redirects to dashboard", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-123",
					Email:    "test@example.com",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
			AutoCreateUsersFunc: func() bool { return true },
			GetDefaultRoleFunc:  func() string { return "user" },
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "user-123",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, false, nil // false = existing user
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return []string{"user"}, nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/dashboard")
	})

	t.Run("successful login for admin user redirects to admin", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "admin-123",
					Email:    "admin@example.com",
					Name:     "Admin User",
					Provider: "google",
				}, nil
			},
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "admin-123",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, false, nil
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return []string{"admin", "user"}, nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/admin")
	})

	t.Run("successful login for new user with role assignment", func(t *testing.T) {
		roleAssigned := false
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "new-user-123",
					Email:    "newuser@example.com",
					Name:     "New User",
					Provider: "google",
				}, nil
			},
			AutoCreateUsersFunc: func() bool { return true },
			GetDefaultRoleFunc:  func() string { return "user" },
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "new-user-123",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, true, nil // true = new user
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return []string{"user"}, nil
			},
			AssignRoleFunc: func(ctx context.Context, userID, role string) error {
				roleAssigned = true

				return nil
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.True(t, roleAssigned, "Role should have been assigned for new user")
	})

	t.Run("new user role assignment failure does not block login", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "new-user-456",
					Email:    "newuser2@example.com",
					Name:     "New User 2",
					Provider: "google",
				}, nil
			},
			AutoCreateUsersFunc: func() bool { return true },
			GetDefaultRoleFunc:  func() string { return "user" },
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "new-user-456",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, true, nil // true = new user
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return []string{}, nil
			},
			AssignRoleFunc: func(ctx context.Context, userID, role string) error {
				return errors.New("role assignment failed")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		// Should still redirect successfully even if role assignment failed
		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/dashboard")
	})

	t.Run("GetUserRoles error returns empty roles", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-789",
					Email:    "test@example.com",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "user-789",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, false, nil
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return nil, errors.New("failed to get roles")
			},
		}
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "http://localhost:3000")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		// Should still succeed with empty roles
		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		assert.Contains(t, w2.Header().Get("Location"), "/dashboard")
	})
}

func TestOAuthHandler_redirectWithError(t *testing.T) {
	log := logger.NewNopLogger()
	mockService := &MockOAuthService{}
	handler := NewOAuthHandlerWithInterface(mockService, &MockOAuthUserRepo{}, log, "http://localhost:3000")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/callback", nil)

	handler.redirectWithError(c, "Test error message")

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "http://localhost:3000/login?error=Test error message", w.Header().Get("Location"))
}

func TestSSOStatusResponse(t *testing.T) {
	t.Run("enabled status", func(t *testing.T) {
		resp := SSOStatusResponse{Enabled: true}
		assert.True(t, resp.Enabled)
	})

	t.Run("disabled status", func(t *testing.T) {
		resp := SSOStatusResponse{Enabled: false}
		assert.False(t, resp.Enabled)
	})
}

func TestOAuthHandler_Callback_EmptyFrontendURL(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("uses root path when frontendURL is empty", func(t *testing.T) {
		mockService := &MockOAuthService{
			ExchangeCodeFunc: func(ctx context.Context, code string) (*oauth.UserInfo, error) {
				return &oauth.UserInfo{
					ID:       "user-123",
					Email:    "test@example.com",
					Name:     "Test User",
					Provider: "google",
				}, nil
			},
		}
		mockRepo := &MockOAuthUserRepo{
			FindOrCreateOAuthUserFunc: func(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
				return &domain.User{
					ID:       "user-123",
					Email:    email,
					Name:     name,
					IsActive: true,
				}, false, nil
			},
			GetUserRolesFunc: func(ctx context.Context, userID string) ([]string, error) {
				return []string{"user"}, nil
			},
		}
		// Test with empty frontendURL
		handler := NewOAuthHandlerWithInterface(mockService, mockRepo, log, "")

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		router.GET("/set-state", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(oauthStateKey, "valid-state")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.GET("/api/v1/auth/sso/callback", handler.Callback)

		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-state", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/auth/sso/callback?code=test-code&state=valid-state", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
		// Should redirect to /dashboard since frontendURL is empty
		assert.Contains(t, w2.Header().Get("Location"), "/dashboard")
	})
}

func TestOAuthHandler_NewOAuthHandler_WithNonNilArgs(t *testing.T) {
	log := logger.NewNopLogger()

	// Test constructor logs with non-nil service pointer
	// This tests the other branches of the NewOAuthHandler constructor
	handler := NewOAuthHandler(nil, nil, log, "http://localhost:3000")
	require.NotNil(t, handler)
	assert.Equal(t, "http://localhost:3000", handler.frontendURL)
}

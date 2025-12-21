package handler

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

func init() {
	// Register types for session serialization
	gob.Register([]string{})
}

// ======================== Mock implementations ========================

type mockUserRepo struct {
	users              map[string]*domain.User
	usersByEmail       map[string]*domain.User
	roles              map[string][]string
	getByIDFunc        func(ctx context.Context, id string) (*domain.User, error)
	getByEmailFunc     func(ctx context.Context, email string) (*domain.User, error)
	getUserRolesFunc   func(ctx context.Context, userID string) ([]string, error)
	updatePasswordFunc func(ctx context.Context, userID string, passwordHash string) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:        make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
		roles:        make(map[string][]string),
	}
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user

	return nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user

	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	user, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	delete(m.users, id)
	delete(m.usersByEmail, user.Email)

	return nil
}

func (m *mockUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	var result []*domain.User
	for _, user := range m.users {
		result = append(result, user)
	}

	return result, len(result), nil
}

func (m *mockUserRepo) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	if m.getUserRolesFunc != nil {
		return m.getUserRolesFunc(ctx, userID)
	}
	roles, ok := m.roles[userID]
	if !ok {
		return []string{}, nil
	}

	return roles, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, userID, passwordHash)
	}
	user, ok := m.users[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	user.PasswordHash = passwordHash

	return nil
}

// Helper to create a test user with hashed password.
func createTestUser(id, email, password string, isActive bool) *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	return &domain.User{
		ID:           id,
		Email:        email,
		Name:         "Test User",
		PasswordHash: string(hash),
		IsActive:     isActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// ======================== Tests ========================

func TestNewAuthHandler(t *testing.T) {
	t.Run("creates handler with nil repository", func(t *testing.T) {
		log := logger.NewNopLogger()
		handler := NewAuthHandler(nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.userRepo)
		assert.NotNil(t, handler.logger)
	})
}

func setupAuthTestRouter(handler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Setup session middleware for testing
	store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
	r.Use(sessions.Sessions("test_session", store))

	return r
}

func TestAuthHandler_Login(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("invalid request body", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(`{invalid`)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation_error")
	})

	t.Run("missing email", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "nonexistent@example.com", "password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_credentials")
	})

	t.Run("database error", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		mockRepo.getByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("database error")
		}
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "test@example.com", "password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal_error")
	})

	t.Run("inactive user", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "password123", false)
		mockRepo.usersByEmail["test@example.com"] = user
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "test@example.com", "password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "user_inactive")
	})

	t.Run("wrong password", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "correctpassword", true)
		mockRepo.usersByEmail["test@example.com"] = user
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "test@example.com", "password": "wrongpassword"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_credentials")
	})

	t.Run("successful login", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "password123", true)
		mockRepo.users["user-123"] = user
		mockRepo.usersByEmail["test@example.com"] = user
		mockRepo.roles["user-123"] = []string{"admin", "user"}
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "test@example.com", "password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "user-123", response.User.ID)
		assert.Equal(t, "test@example.com", response.User.Email)
		assert.Contains(t, response.User.Roles, "admin")
	})

	t.Run("successful login with role fetch error", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "password123", true)
		mockRepo.users["user-123"] = user
		mockRepo.usersByEmail["test@example.com"] = user
		mockRepo.getUserRolesFunc = func(ctx context.Context, userID string) ([]string, error) {
			return nil, errors.New("role fetch error")
		}
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/login", handler.Login)

		w := httptest.NewRecorder()
		body := `{"email": "test@example.com", "password": "password123"}`
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Should still succeed, just with empty roles
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("successful logout without session", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)
		router := setupAuthTestRouter(handler)
		router.POST("/api/v1/auth/logout", handler.Logout)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Logged out successfully")
	})

	t.Run("successful logout with session", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		store := cookie.NewStore([]byte("test-secret-key-32-bytes-long!!!"))
		router.Use(sessions.Sessions("test_session", store))

		// First set session values
		router.GET("/set-session", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set(middleware.ContextKeyUserID, "user-123")
			session.Set(middleware.ContextKeyUserEmail, "test@example.com")
			session.Save()
			c.Status(http.StatusOK)
		})
		router.POST("/api/v1/auth/logout", handler.Logout)

		// Set session
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/set-session", nil)
		router.ServeHTTP(w1, req1)
		cookies := w1.Result().Cookies()

		// Logout
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		for _, c := range cookies {
			req2.AddCookie(c)
		}
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Contains(t, w2.Body.String(), "Logged out successfully")
	})
}

func TestAuthHandler_GetCurrentUser(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)

		handler.GetCurrentUser(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response["error"])
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		c.Set(middleware.ContextKeyUserID, "nonexistent")

		handler.GetCurrentUser(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "password", true)
		mockRepo.users["user-123"] = user
		mockRepo.roles["user-123"] = []string{"admin", "user"}
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		c.Set(middleware.ContextKeyUserID, "user-123")
		c.Set(middleware.ContextKeyUserRoles, []string{"admin", "user"})

		handler.GetCurrentUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response UserInfo
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "user-123", response.ID)
		assert.Equal(t, "test@example.com", response.Email)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		mockRepo.getByIDFunc = func(ctx context.Context, id string) (*domain.User, error) {
			return nil, errors.New("database error")
		}
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetCurrentUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "old", "new_password": "newpassword123"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("password too short", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "old", "new_password": "short"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		mockRepo.getByIDFunc = func(ctx context.Context, id string) (*domain.User, error) {
			return nil, errors.New("database error")
		}
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "oldpassword", "new_password": "newpassword123"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("wrong current password", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "correctpassword", true)
		mockRepo.users["user-123"] = user
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "wrongpassword", "new_password": "newpassword123"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "invalid_password", response["error"])
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "oldpassword", true)
		mockRepo.users["user-123"] = user
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "oldpassword", "new_password": "newpassword123"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Password changed successfully", response["message"])
	})

	t.Run("update password error", func(t *testing.T) {
		mockRepo := newMockUserRepo()
		user := createTestUser("user-123", "test@example.com", "oldpassword", true)
		mockRepo.users["user-123"] = user
		mockRepo.updatePasswordFunc = func(ctx context.Context, userID string, passwordHash string) error {
			return errors.New("database error")
		}
		handler := NewAuthHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"current_password": "oldpassword", "new_password": "newpassword123"}`
		c.Request = httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Test request/response types.
func TestAuthTypes(t *testing.T) {
	t.Run("LoginRequest JSON", func(t *testing.T) {
		jsonStr := `{"email": "test@example.com", "password": "secret"}`
		var req LoginRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", req.Email)
		assert.Equal(t, "secret", req.Password)
	})

	t.Run("LoginResponse JSON", func(t *testing.T) {
		resp := LoginResponse{
			User: UserInfo{
				ID:       "user-123",
				Email:    "test@example.com",
				Name:     "Test User",
				Roles:    []string{"admin"},
				IsActive: true,
			},
		}
		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(data), "user-123")
		assert.Contains(t, string(data), "test@example.com")
	})

	t.Run("UserInfo JSON", func(t *testing.T) {
		info := UserInfo{
			ID:       "user-123",
			Email:    "test@example.com",
			Name:     "Test User",
			Roles:    []string{"admin", "user"},
			IsActive: true,
		}
		data, err := json.Marshal(info)
		require.NoError(t, err)
		assert.Contains(t, string(data), "user-123")
		assert.Contains(t, string(data), "admin")
	})

	t.Run("ChangePasswordRequest JSON", func(t *testing.T) {
		jsonStr := `{"current_password": "old", "new_password": "newpassword"}`
		var req ChangePasswordRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		require.NoError(t, err)
		assert.Equal(t, "old", req.CurrentPassword)
		assert.Equal(t, "newpassword", req.NewPassword)
	})
}

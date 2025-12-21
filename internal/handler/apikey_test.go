package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/handler/middleware"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// ======================== Mock implementations ========================

type mockAPIKeyRepo struct {
	keys           map[string]*APIKey
	createFunc     func(ctx context.Context, userID, name string, expiresAt *time.Time) (*APIKey, string, error)
	getByIDFunc    func(ctx context.Context, keyID string) (*APIKey, error)
	listByUserFunc func(ctx context.Context, userID string) ([]*APIKey, error)
	deleteFunc     func(ctx context.Context, keyID, userID string) error
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{
		keys: make(map[string]*APIKey),
	}
}

func (m *mockAPIKeyRepo) Create(ctx context.Context, userID, name string, expiresAt *time.Time) (*APIKey, string, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, name, expiresAt)
	}
	key := &APIKey{
		ID:        "key-" + name,
		UserID:    userID,
		Name:      name,
		KeyPrefix: "mcpgw_abc",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	m.keys[key.ID] = key

	return key, "mcpgw_plainkey123", nil
}

func (m *mockAPIKeyRepo) GetByID(ctx context.Context, keyID string) (*APIKey, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, keyID)
	}
	key, ok := m.keys[keyID]
	if !ok {
		return nil, domain.ErrAPIKeyNotFound
	}

	return key, nil
}

func (m *mockAPIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	return nil, domain.ErrAPIKeyNotFound
}

func (m *mockAPIKeyRepo) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID)
	}
	var result []*APIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			result = append(result, key)
		}
	}

	return result, nil
}

func (m *mockAPIKeyRepo) Delete(ctx context.Context, keyID, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, keyID, userID)
	}
	key, ok := m.keys[keyID]
	if !ok {
		return domain.ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return domain.ErrAPIKeyNotFound
	}
	delete(m.keys, keyID)

	return nil
}

func (m *mockAPIKeyRepo) UpdateLastUsed(ctx context.Context, keyID string) error {
	return nil
}

// ======================== Tests ========================

func TestNewAPIKeyHandler(t *testing.T) {
	t.Run("creates handler with nil repository", func(t *testing.T) {
		log := logger.NewNopLogger()
		handler := NewAPIKeyHandler(nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.apiKeyRepo)
		assert.NotNil(t, handler.logger)
	})
}

func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"name": "My API Key"}`
		c.Request = httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response["error"])
	})

	t.Run("invalid body", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"name": "My API Key"}`
		c.Request = httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response CreateAPIKeyResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "My API Key", response.Name)
		assert.Contains(t, response.Key, "mcpgw_")
		assert.Contains(t, response.Message, "Save this key")
	})

	t.Run("success with expiry", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"name": "My API Key", "expires_in_days": 30}`
		c.Request = httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.createFunc = func(ctx context.Context, userID, name string, expiresAt *time.Time) (*APIKey, string, error) {
			return nil, "", errors.New("database error")
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"name": "My API Key"}`
		c.Request = httptest.NewRequest("POST", "/api/v1/api-keys", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAPIKeyHandler_ListAPIKeys(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys", nil)

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success - empty list", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys", nil)
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(0), response["total"])
	})

	t.Run("success - with keys", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.keys["key-1"] = &APIKey{ID: "key-1", UserID: "user-123", Name: "Key 1", KeyPrefix: "mcpgw_abc"}
		mockRepo.keys["key-2"] = &APIKey{ID: "key-2", UserID: "user-123", Name: "Key 2", KeyPrefix: "mcpgw_xyz"}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys", nil)
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.listByUserFunc = func(ctx context.Context, userID string) ([]*APIKey, error) {
			return nil, errors.New("database error")
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys", nil)
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAPIKeyHandler_DeleteAPIKey(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/api-keys/", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/api-keys/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.keys["key-123"] = &APIKey{ID: "key-123", UserID: "user-123", Name: "My Key"}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "API key deleted successfully", response["message"])
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.deleteFunc = func(ctx context.Context, keyID, userID string) error {
			return errors.New("database error")
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAPIKeyHandler_GetAPIKey(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("unauthorized - no user ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.keys["key-123"] = &APIKey{
			ID:        "key-123",
			UserID:    "user-123",
			Name:      "My Key",
			KeyPrefix: "mcpgw_abc",
			CreatedAt: time.Now(),
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response APIKeyInfo
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "key-123", response.ID)
		assert.Equal(t, "My Key", response.Name)
	})

	t.Run("access denied - different user", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.keys["key-123"] = &APIKey{
			ID:     "key-123",
			UserID: "other-user",
			Name:   "Other User's Key",
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code) // Returns 404 for security
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := newMockAPIKeyRepo()
		mockRepo.getByIDFunc = func(ctx context.Context, keyID string) (*APIKey, error) {
			return nil, errors.New("database error")
		}
		handler := NewAPIKeyHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/api-keys/key-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "key-123"}}
		c.Set(middleware.ContextKeyUserID, "user-123")

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Test request/response types.
func TestAPIKeyTypes(t *testing.T) {
	t.Run("CreateAPIKeyRequest JSON", func(t *testing.T) {
		jsonStr := `{"name": "Test Key", "expires_in_days": 30}`
		var req CreateAPIKeyRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		require.NoError(t, err)
		assert.Equal(t, "Test Key", req.Name)
		assert.Equal(t, 30, *req.ExpiresIn)
	})

	t.Run("CreateAPIKeyResponse JSON", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		resp := CreateAPIKeyResponse{
			ID:        "key-123",
			Name:      "My Key",
			Key:       "mcpgw_abc123",
			ExpiresAt: &expiresAt,
			CreatedAt: time.Now(),
			Message:   "Save this key",
		}
		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(data), "key-123")
		assert.Contains(t, string(data), "mcpgw_abc123")
	})

	t.Run("APIKeyInfo JSON", func(t *testing.T) {
		info := APIKeyInfo{
			ID:        "key-123",
			Name:      "My Key",
			KeyPrefix: "mcpgw_abc",
			CreatedAt: time.Now(),
		}
		data, err := json.Marshal(info)
		require.NoError(t, err)
		assert.Contains(t, string(data), "key-123")
		assert.Contains(t, string(data), "mcpgw_abc")
	})
}

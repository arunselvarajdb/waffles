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

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// ======================== Mock implementations ========================

type mockNamespaceRepo struct {
	namespaces                 map[string]*domain.Namespace
	members                    map[string][]string                      // namespaceID -> []serverID
	roleAccess                 map[string]map[string]domain.AccessLevel // namespaceID -> roleID -> level
	roleIDs                    map[string]string                        // roleName -> roleID
	createFunc                 func(ctx context.Context, req *domain.NamespaceCreate) (*domain.Namespace, error)
	getFunc                    func(ctx context.Context, id string) (*domain.Namespace, error)
	listFunc                   func(ctx context.Context) ([]*domain.Namespace, error)
	updateFunc                 func(ctx context.Context, id string, req *domain.NamespaceUpdate) (*domain.Namespace, error)
	deleteFunc                 func(ctx context.Context, id string) error
	addServerFunc              func(ctx context.Context, serverID, namespaceID string) error
	removeServerFunc           func(ctx context.Context, serverID, namespaceID string) error
	getNamespaceServersFunc    func(ctx context.Context, namespaceID string) ([]*domain.NamespaceMember, error)
	setRoleAccessFunc          func(ctx context.Context, roleID, namespaceID string, level domain.AccessLevel) error
	removeRoleAccessFunc       func(ctx context.Context, roleID, namespaceID string) error
	getNamespaceRoleAccessFunc func(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error)
	getRoleIDByNameFunc        func(ctx context.Context, roleName string) (string, error)
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{
		namespaces: make(map[string]*domain.Namespace),
		members:    make(map[string][]string),
		roleAccess: make(map[string]map[string]domain.AccessLevel),
		roleIDs:    make(map[string]string),
	}
}

func (m *mockNamespaceRepo) Create(ctx context.Context, req *domain.NamespaceCreate) (*domain.Namespace, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	ns := &domain.Namespace{
		ID:          "ns-123",
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.namespaces[ns.ID] = ns

	return ns, nil
}

func (m *mockNamespaceRepo) Get(ctx context.Context, id string) (*domain.Namespace, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	ns, ok := m.namespaces[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return ns, nil
}

func (m *mockNamespaceRepo) List(ctx context.Context) ([]*domain.Namespace, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	var result []*domain.Namespace
	for _, ns := range m.namespaces {
		result = append(result, ns)
	}

	return result, nil
}

func (m *mockNamespaceRepo) Update(ctx context.Context, id string, req *domain.NamespaceUpdate) (*domain.Namespace, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	ns, ok := m.namespaces[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if req.Name != nil {
		ns.Name = *req.Name
	}
	if req.Description != nil {
		ns.Description = *req.Description
	}
	ns.UpdatedAt = time.Now()

	return ns, nil
}

func (m *mockNamespaceRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	if _, ok := m.namespaces[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.namespaces, id)

	return nil
}

func (m *mockNamespaceRepo) AddServerToNamespace(ctx context.Context, serverID, namespaceID string) error {
	if m.addServerFunc != nil {
		return m.addServerFunc(ctx, serverID, namespaceID)
	}
	if m.members[namespaceID] == nil {
		m.members[namespaceID] = []string{}
	}
	m.members[namespaceID] = append(m.members[namespaceID], serverID)

	return nil
}

func (m *mockNamespaceRepo) RemoveServerFromNamespace(ctx context.Context, serverID, namespaceID string) error {
	if m.removeServerFunc != nil {
		return m.removeServerFunc(ctx, serverID, namespaceID)
	}
	servers, ok := m.members[namespaceID]
	if !ok {
		return domain.ErrNotFound
	}
	for i, s := range servers {
		if s == serverID {
			m.members[namespaceID] = append(servers[:i], servers[i+1:]...)

			return nil
		}
	}

	return domain.ErrNotFound
}

func (m *mockNamespaceRepo) GetNamespaceServers(ctx context.Context, namespaceID string) ([]*domain.NamespaceMember, error) {
	if m.getNamespaceServersFunc != nil {
		return m.getNamespaceServersFunc(ctx, namespaceID)
	}
	var result []*domain.NamespaceMember
	for _, serverID := range m.members[namespaceID] {
		result = append(result, &domain.NamespaceMember{
			ServerID:      serverID,
			ServerName:    "server-" + serverID,
			NamespaceID:   namespaceID,
			NamespaceName: "namespace",
		})
	}

	return result, nil
}

func (m *mockNamespaceRepo) SetRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string, level domain.AccessLevel) error {
	if m.setRoleAccessFunc != nil {
		return m.setRoleAccessFunc(ctx, roleID, namespaceID, level)
	}
	if m.roleAccess[namespaceID] == nil {
		m.roleAccess[namespaceID] = make(map[string]domain.AccessLevel)
	}
	m.roleAccess[namespaceID][roleID] = level

	return nil
}

func (m *mockNamespaceRepo) RemoveRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string) error {
	if m.removeRoleAccessFunc != nil {
		return m.removeRoleAccessFunc(ctx, roleID, namespaceID)
	}
	if _, ok := m.roleAccess[namespaceID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.roleAccess[namespaceID][roleID]; !ok {
		return domain.ErrNotFound
	}
	delete(m.roleAccess[namespaceID], roleID)

	return nil
}

func (m *mockNamespaceRepo) GetNamespaceRoleAccess(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error) {
	if m.getNamespaceRoleAccessFunc != nil {
		return m.getNamespaceRoleAccessFunc(ctx, namespaceID)
	}
	var result []*domain.RoleNamespaceAccess
	for roleID, level := range m.roleAccess[namespaceID] {
		result = append(result, &domain.RoleNamespaceAccess{
			RoleID:        roleID,
			RoleName:      "role-" + roleID,
			NamespaceID:   namespaceID,
			NamespaceName: "namespace",
			AccessLevel:   level,
		})
	}

	return result, nil
}

func (m *mockNamespaceRepo) GetRoleIDByName(ctx context.Context, roleName string) (string, error) {
	if m.getRoleIDByNameFunc != nil {
		return m.getRoleIDByNameFunc(ctx, roleName)
	}
	roleID, ok := m.roleIDs[roleName]
	if !ok {
		return "", domain.ErrNotFound
	}

	return roleID, nil
}

// ======================== Tests ========================

func TestNewNamespaceHandler(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("creates handler with nil repo", func(t *testing.T) {
		handler := NewNamespaceHandler(nil, log)
		require.NotNil(t, handler)
		assert.Nil(t, handler.namespaceRepo)
		assert.NotNil(t, handler.logger)
	})

	t.Run("creates handler with interface", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)
		require.NotNil(t, handler)
		assert.NotNil(t, handler.namespaceRepo)
		assert.NotNil(t, handler.logger)
	})
}

func TestNewServerGroupHandler_Alias(t *testing.T) {
	t.Run("legacy alias creates same handler", func(t *testing.T) {
		log := logger.NewNopLogger()
		handler := NewServerGroupHandler(nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.namespaceRepo)
	})
}

func TestNamespaceHandler_ListNamespaces(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces", nil)

		handler.ListNamespaces(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(0), response["count"])
	})

	t.Run("returns namespaces", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-1"] = &domain.Namespace{ID: "ns-1", Name: "namespace1"}
		mockRepo.namespaces["ns-2"] = &domain.Namespace{ID: "ns-2", Name: "namespace2"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces", nil)

		handler.ListNamespaces(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.listFunc = func(ctx context.Context) ([]*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces", nil)

		handler.ListNamespaces(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_CreateNamespace(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("creates namespace successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "test-namespace", "description": "A test namespace"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateNamespace(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response domain.Namespace
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test-namespace", response.Name)
	})

	t.Run("returns bad request for invalid body", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateNamespace(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.createFunc = func(ctx context.Context, req *domain.NamespaceCreate) (*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "test-namespace"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateNamespace(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_GetNamespace(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns namespace", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test-namespace"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.GetNamespace(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response domain.Namespace
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ns-123", response.ID)
	})

	t.Run("returns not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.GetNamespace(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.getFunc = func(ctx context.Context, id string) (*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.GetNamespace(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_UpdateNamespace(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("updates namespace successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "old-name"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "new-name"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/namespaces/ns-123", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.UpdateNamespace(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response domain.Namespace
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "new-name", response.Name)
	})

	t.Run("returns bad request for invalid body", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/namespaces/ns-123", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.UpdateNamespace(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "new-name"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/namespaces/nonexistent", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.UpdateNamespace(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.updateFunc = func(ctx context.Context, id string, req *domain.NamespaceUpdate) (*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "new-name"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/namespaces/ns-123", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.UpdateNamespace(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_DeleteNamespace(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("deletes namespace successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.DeleteNamespace(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Namespace deleted", response["message"])
	})

	t.Run("returns not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.DeleteNamespace(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.deleteFunc = func(ctx context.Context, id string) error {
			return errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.DeleteNamespace(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_AddServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("adds server successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"server_id": "server-123"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/servers", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.AddServer(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Server added to namespace", response["message"])
	})

	t.Run("returns bad request for invalid body", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/servers", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.AddServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns not found when namespace doesn't exist", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"server_id": "server-123"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/nonexistent/servers", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.AddServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles get namespace error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.getFunc = func(ctx context.Context, id string) (*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"server_id": "server-123"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/servers", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.AddServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles add server error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		mockRepo.addServerFunc = func(ctx context.Context, serverID, namespaceID string) error {
			return errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"server_id": "server-123"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/servers", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.AddServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_RemoveServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("removes server successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.members["ns-123"] = []string{"server-123"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/servers/server-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "server_id", Value: "server-123"},
		}

		handler.RemoveServer(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Server removed from namespace", response["message"])
	})

	t.Run("returns not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/servers/server-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "server_id", Value: "server-123"},
		}

		handler.RemoveServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.removeServerFunc = func(ctx context.Context, serverID, namespaceID string) error {
			return errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/servers/server-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "server_id", Value: "server-123"},
		}

		handler.RemoveServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_ListServers(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/servers", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(0), response["count"])
	})

	t.Run("returns servers", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.members["ns-123"] = []string{"server-1", "server-2"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/servers", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.getNamespaceServersFunc = func(ctx context.Context, namespaceID string) ([]*domain.NamespaceMember, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/servers", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListServers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_SetRoleAccess(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("sets role access successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		mockRepo.roleIDs["admin"] = "role-123"
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "execute"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Role access set", response["message"])
		assert.Equal(t, "admin", response["role"])
	})

	t.Run("returns bad request for invalid body", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(`{invalid`)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns bad request for invalid access level", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "invalid"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns bad request when role not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "nonexistent", "access_level": "view"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("handles get role error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.getRoleIDByNameFunc = func(ctx context.Context, roleName string) (string, error) {
			return "", errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "view"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns not found when namespace doesn't exist", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.roleIDs["admin"] = "role-123"
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "view"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/nonexistent/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles get namespace error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.roleIDs["admin"] = "role-123"
		mockRepo.getFunc = func(ctx context.Context, id string) (*domain.Namespace, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "view"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("handles set role access error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		mockRepo.roleIDs["admin"] = "role-123"
		mockRepo.setRoleAccessFunc = func(ctx context.Context, roleID, namespaceID string, level domain.AccessLevel) error {
			return errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"role_name": "admin", "access_level": "view"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/namespaces/ns-123/access", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.SetRoleAccess(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_RemoveRoleAccess(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("removes role access successfully", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.roleAccess["ns-123"] = map[string]domain.AccessLevel{
			"role-123": domain.AccessLevelView,
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/access/role-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "role_id", Value: "role-123"},
		}

		handler.RemoveRoleAccess(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Role access removed", response["message"])
	})

	t.Run("returns not found", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/access/role-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "role_id", Value: "role-123"},
		}

		handler.RemoveRoleAccess(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.removeRoleAccessFunc = func(ctx context.Context, roleID, namespaceID string) error {
			return errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/namespaces/ns-123/access/role-123", nil)
		c.Params = gin.Params{
			{Key: "id", Value: "ns-123"},
			{Key: "role_id", Value: "role-123"},
		}

		handler.RemoveRoleAccess(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_ListRoleAccess(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/access", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListRoleAccess(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(0), response["count"])
	})

	t.Run("returns role access entries", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.roleAccess["ns-123"] = map[string]domain.AccessLevel{
			"role-1": domain.AccessLevelView,
			"role-2": domain.AccessLevelExecute,
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/access", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListRoleAccess(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.getNamespaceRoleAccessFunc = func(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error) {
			return nil, errors.New("database error")
		}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/namespaces/ns-123/access", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.ListRoleAccess(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestNamespaceHandler_LegacyAliases(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("ListGroups calls ListNamespaces", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-1"] = &domain.Namespace{ID: "ns-1", Name: "namespace1"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/groups", nil)

		handler.ListGroups(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("CreateGroup calls CreateNamespace", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "test-group"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/groups", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateGroup(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("GetGroup calls GetNamespace", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/v1/groups/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.GetGroup(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("UpdateGroup calls UpdateNamespace", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "old-name"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		body := `{"name": "new-name"}`
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/v1/groups/ns-123", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.UpdateGroup(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("DeleteGroup calls DeleteNamespace", func(t *testing.T) {
		mockRepo := newMockNamespaceRepo()
		mockRepo.namespaces["ns-123"] = &domain.Namespace{ID: "ns-123", Name: "test"}
		handler := NewNamespaceHandlerWithInterface(mockRepo, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/api/v1/groups/ns-123", nil)
		c.Params = gin.Params{{Key: "id", Value: "ns-123"}}

		handler.DeleteGroup(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ServerGroupHandler is an alias for NamespaceHandler", func(t *testing.T) {
		var handler = NewNamespaceHandlerWithInterface(newMockNamespaceRepo(), log)
		assert.NotNil(t, handler)
	})
}

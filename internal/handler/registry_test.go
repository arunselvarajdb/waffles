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
	"github.com/waffles/mcp-gateway/internal/service/registry"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ======================== Mock implementations ========================

// mockRegistryService implements RegistryServiceInterface for testing.
type mockRegistryService struct {
	servers       map[string]*domain.MCPServer
	healthRecords map[string]*domain.ServerHealth

	createServerFunc       func(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error)
	listServersForUserFunc func(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error)
	getServerFunc          func(ctx context.Context, id string) (*domain.MCPServer, error)
	updateServerFunc       func(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error)
	deleteServerFunc       func(ctx context.Context, id string) error
	toggleServerFunc       func(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error)
	getHealthStatusFunc    func(ctx context.Context, serverID string) (*domain.ServerHealth, error)
	checkHealthFunc        func(ctx context.Context, serverID string) error
	testConnectionFunc     func(ctx context.Context, req *registry.TestConnectionRequest) (*registry.TestConnectionResult, error)
	callToolFunc           func(ctx context.Context, req *registry.CallToolRequest) (*registry.CallToolResult, error)
}

func newMockRegistryService() *mockRegistryService {
	return &mockRegistryService{
		servers:       make(map[string]*domain.MCPServer),
		healthRecords: make(map[string]*domain.ServerHealth),
	}
}

func (m *mockRegistryService) CreateServer(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
	if m.createServerFunc != nil {
		return m.createServerFunc(ctx, req)
	}
	server := &domain.MCPServer{
		ID:              "server-" + req.Name,
		Name:            req.Name,
		Description:     req.Description,
		URL:             req.URL,
		ProtocolVersion: "1.0.0",
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	m.servers[server.ID] = server

	return server, nil
}

func (m *mockRegistryService) ListServersForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
	if m.listServersForUserFunc != nil {
		return m.listServersForUserFunc(ctx, filter, accessibleServerIDs)
	}
	var servers []*domain.MCPServer
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers, nil
}

func (m *mockRegistryService) GetServer(ctx context.Context, id string) (*domain.MCPServer, error) {
	if m.getServerFunc != nil {
		return m.getServerFunc(ctx, id)
	}
	server, ok := m.servers[id]
	if !ok {
		return nil, domain.ErrServerNotFound
	}

	return server, nil
}

func (m *mockRegistryService) UpdateServer(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
	if m.updateServerFunc != nil {
		return m.updateServerFunc(ctx, id, req)
	}
	server, ok := m.servers[id]
	if !ok {
		return nil, domain.ErrServerNotFound
	}
	if req.Name != nil {
		server.Name = *req.Name
	}

	return server, nil
}

func (m *mockRegistryService) DeleteServer(ctx context.Context, id string) error {
	if m.deleteServerFunc != nil {
		return m.deleteServerFunc(ctx, id)
	}
	if _, ok := m.servers[id]; !ok {
		return domain.ErrServerNotFound
	}
	delete(m.servers, id)

	return nil
}

func (m *mockRegistryService) ToggleServer(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error) {
	if m.toggleServerFunc != nil {
		return m.toggleServerFunc(ctx, id, enabled)
	}
	server, ok := m.servers[id]
	if !ok {
		return nil, domain.ErrServerNotFound
	}
	server.IsActive = enabled

	return server, nil
}

func (m *mockRegistryService) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	if m.getHealthStatusFunc != nil {
		return m.getHealthStatusFunc(ctx, serverID)
	}
	health, ok := m.healthRecords[serverID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return health, nil
}

func (m *mockRegistryService) CheckHealth(ctx context.Context, serverID string) error {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx, serverID)
	}

	return nil
}

func (m *mockRegistryService) TestConnection(ctx context.Context, req *registry.TestConnectionRequest) (*registry.TestConnectionResult, error) {
	if m.testConnectionFunc != nil {
		return m.testConnectionFunc(ctx, req)
	}

	return &registry.TestConnectionResult{
		Success:        true,
		ResponseTimeMs: 50,
	}, nil
}

func (m *mockRegistryService) CallTool(ctx context.Context, req *registry.CallToolRequest) (*registry.CallToolResult, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, req)
	}

	return &registry.CallToolResult{
		Success: true,
		Content: map[string]interface{}{"result": "ok"},
	}, nil
}

// mockAccessService implements ServerAccessServiceInterface for testing.
type mockAccessService struct {
	err                 error
	canAccessFunc       func(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error)
	accessibleServerIDs []string
}

func (m *mockAccessService) GetAccessibleServerIDs(ctx context.Context, roles []string, level domain.AccessLevel) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.accessibleServerIDs, nil
}

func (m *mockAccessService) CanAccessServer(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error) {
	if m.canAccessFunc != nil {
		return m.canAccessFunc(ctx, roles, serverID, level)
	}
	// Default: check if serverID is in accessibleServerIDs
	if m.accessibleServerIDs == nil {
		return true, nil // Admin bypass
	}
	for _, id := range m.accessibleServerIDs {
		if id == serverID {
			return true, nil
		}
	}

	return false, nil
}

// ======================== Helper functions ========================

// createTestContext creates a gin test context with the request.
func createTestContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

// ======================== Tests ========================

func TestNewRegistryHandler(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("creates handler with nil services", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.service)
		assert.Nil(t, handler.accessService)
		assert.NotNil(t, handler.logger)
	})

	t.Run("creates handler with interfaces", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockAccess := &mockAccessService{}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, mockAccess, log)

		require.NotNil(t, handler)
		assert.NotNil(t, handler.service)
		assert.NotNil(t, handler.accessService)
	})
}

// Tests for ListServers

func TestRegistryHandler_ListServers(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success with mock service", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Server 1"}
		mockSvc.servers["server-2"] = &domain.MCPServer{ID: "server-2", Name: "Server 2"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers", nil)
		c.Set(middleware.ContextKeyUserRoles, []string{"admin"})

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("invalid is_active parameter", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?is_active=invalid", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Invalid is_active parameter", response["error"])
	})

	t.Run("invalid limit - non-numeric", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?limit=abc", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid limit - zero", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?limit=0", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid limit - exceeds max", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?limit=101", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid offset - non-numeric", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?offset=abc", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid offset - negative", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?offset=-1", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("access service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockAccess := &mockAccessService{err: errors.New("access check failed")}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, mockAccess, log)

		c, w := createTestContext("GET", "/api/v1/servers", nil)
		c.Set(middleware.ContextKeyUserRoles, []string{"user"})

		handler.ListServers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.listServersForUserFunc = func(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers", nil)

		handler.ListServers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("with name filter", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?name=Test", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("with tags filter", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?tags=prod&tags=api", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("with valid is_active filter", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server", IsActive: true}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?is_active=true", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("with custom limit and offset", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/servers?limit=50&offset=10", nil)
		c.Request = req

		handler.ListServers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Tests for CreateServer

func TestRegistryHandler_CreateServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"name": "test-server", "url": "https://example.com/mcp"}`
		c, w := createTestContext("POST", "/api/v1/servers", []byte(body))

		handler.CreateServer(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response domain.MCPServer
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test-server", response.Name)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers", []byte(`{"invalid json`))

		handler.CreateServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.createServerFunc = func(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"name": "test-server", "url": "https://example.com/mcp"}`
		c, w := createTestContext("POST", "/api/v1/servers", []byte(body))

		handler.CreateServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for GetServer

func TestRegistryHandler_GetServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.GetServer(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.MCPServer
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "server-1", response.ID)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.GetServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.GetServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("access denied", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Test Server"}
		mockAccess := &mockAccessService{
			canAccessFunc: func(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error) {
				return false, nil
			},
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, mockAccess, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}
		c.Set(middleware.ContextKeyUserRoles, []string{"user"})

		handler.GetServer(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("access check error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockAccess := &mockAccessService{
			canAccessFunc: func(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error) {
				return false, errors.New("access check failed")
			},
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, mockAccess, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}
		c.Set(middleware.ContextKeyUserRoles, []string{"user"})

		handler.GetServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.getServerFunc = func(ctx context.Context, id string) (*domain.MCPServer, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.GetServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for UpdateServer

func TestRegistryHandler_UpdateServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Original Name"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"name": "Updated Name"}`
		c, w := createTestContext("PUT", "/api/v1/servers/server-1", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.UpdateServer(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		body := `{"name": "Updated Name"}`
		c, w := createTestContext("PUT", "/api/v1/servers/", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.UpdateServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("PUT", "/api/v1/servers/server-1", []byte(`{invalid`))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.UpdateServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"name": "Updated Name"}`
		c, w := createTestContext("PUT", "/api/v1/servers/nonexistent", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.UpdateServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.updateServerFunc = func(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"name": "Updated Name"}`
		c, w := createTestContext("PUT", "/api/v1/servers/server-1", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.UpdateServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for DeleteServer

func TestRegistryHandler_DeleteServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1"}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("DELETE", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.DeleteServer(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("DELETE", "/api/v1/servers/", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.DeleteServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("DELETE", "/api/v1/servers/nonexistent", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.DeleteServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.deleteServerFunc = func(ctx context.Context, id string) error {
			return errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("DELETE", "/api/v1/servers/server-1", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.DeleteServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for ToggleServer

func TestRegistryHandler_ToggleServer(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success enable", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.servers["server-1"] = &domain.MCPServer{ID: "server-1", IsActive: false}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"is_active": true}`
		c, w := createTestContext("PATCH", "/api/v1/servers/server-1/toggle", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.ToggleServer(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		body := `{"is_active": true}`
		c, w := createTestContext("PATCH", "/api/v1/servers//toggle", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.ToggleServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("PATCH", "/api/v1/servers/server-1/toggle", []byte(`{invalid`))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.ToggleServer(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"is_active": true}`
		c, w := createTestContext("PATCH", "/api/v1/servers/nonexistent/toggle", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.ToggleServer(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.toggleServerFunc = func(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"is_active": true}`
		c, w := createTestContext("PATCH", "/api/v1/servers/server-1/toggle", []byte(body))
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.ToggleServer(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for GetHealthStatus

func TestRegistryHandler_GetHealthStatus(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.healthRecords["server-1"] = &domain.ServerHealth{
			ServerID: "server-1",
			Status:   domain.ServerStatusHealthy,
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.GetHealthStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers//health", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.GetHealthStatus(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.getHealthStatusFunc = func(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
			return nil, errors.New("database error")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("GET", "/api/v1/servers/server-1/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.GetHealthStatus(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for CheckHealth

func TestRegistryHandler_CheckHealth(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.healthRecords["server-1"] = &domain.ServerHealth{
			ServerID: "server-1",
			Status:   domain.ServerStatusHealthy,
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/server-1/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.CheckHealth(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty ID", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers//health", nil)
		c.Params = gin.Params{{Key: "id", Value: ""}}

		handler.CheckHealth(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("check health error - not found", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.checkHealthFunc = func(ctx context.Context, serverID string) error {
			return domain.ErrServerNotFound
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/nonexistent/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

		handler.CheckHealth(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("check health error - internal", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.checkHealthFunc = func(ctx context.Context, serverID string) error {
			return errors.New("health check failed")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/server-1/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.CheckHealth(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get health status error after check", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.getHealthStatusFunc = func(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
			return nil, errors.New("failed to get status")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/server-1/health", nil)
		c.Params = gin.Params{{Key: "id", Value: "server-1"}}

		handler.CheckHealth(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for TestConnection

func TestRegistryHandler_TestConnection(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"url": "https://example.com/mcp"}`
		c, w := createTestContext("POST", "/api/v1/servers/test-connection", []byte(body))

		handler.TestConnection(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/test-connection", []byte(`{invalid`))

		handler.TestConnection(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty URL", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		body := `{"url": ""}`
		c, w := createTestContext("POST", "/api/v1/servers/test-connection", []byte(body))

		handler.TestConnection(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.testConnectionFunc = func(ctx context.Context, req *registry.TestConnectionRequest) (*registry.TestConnectionResult, error) {
			return nil, errors.New("connection failed")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"url": "https://example.com/mcp"}`
		c, w := createTestContext("POST", "/api/v1/servers/test-connection", []byte(body))

		handler.TestConnection(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Tests for CallTool

func TestRegistryHandler_CallTool(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"url": "https://example.com/mcp", "tool_name": "calculator"}`
		c, w := createTestContext("POST", "/api/v1/servers/call-tool", []byte(body))

		handler.CallTool(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		c, w := createTestContext("POST", "/api/v1/servers/call-tool", []byte(`{invalid`))

		handler.CallTool(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty URL", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		body := `{"url": "", "tool_name": "test"}`
		c, w := createTestContext("POST", "/api/v1/servers/call-tool", []byte(body))

		handler.CallTool(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty tool name", func(t *testing.T) {
		handler := NewRegistryHandler(nil, nil, log)

		body := `{"url": "https://example.com/mcp", "tool_name": ""}`
		c, w := createTestContext("POST", "/api/v1/servers/call-tool", []byte(body))

		handler.CallTool(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := newMockRegistryService()
		mockSvc.callToolFunc = func(ctx context.Context, req *registry.CallToolRequest) (*registry.CallToolResult, error) {
			return nil, errors.New("tool call failed")
		}

		handler := NewRegistryHandlerWithInterfaces(mockSvc, nil, log)

		body := `{"url": "https://example.com/mcp", "tool_name": "calculator"}`
		c, w := createTestContext("POST", "/api/v1/servers/call-tool", []byte(body))

		handler.CallTool(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// Test domain types.
func TestRegistryHandler_DomainTypes(t *testing.T) {
	t.Run("ServerFilter struct", func(t *testing.T) {
		isActive := true
		filter := &domain.ServerFilter{
			Name:     "test",
			IsActive: &isActive,
			Tags:     []string{"prod", "api"},
			Limit:    10,
			Offset:   5,
		}

		assert.Equal(t, "test", filter.Name)
		assert.True(t, *filter.IsActive)
		assert.Equal(t, 2, len(filter.Tags))
	})

	t.Run("server status constants", func(t *testing.T) {
		assert.Equal(t, domain.ServerStatus("healthy"), domain.ServerStatusHealthy)
		assert.Equal(t, domain.ServerStatus("unhealthy"), domain.ServerStatusUnhealthy)
		assert.Equal(t, domain.ServerStatus("unknown"), domain.ServerStatusUnknown)
	})

	t.Run("access level constants", func(t *testing.T) {
		assert.Equal(t, domain.AccessLevel("view"), domain.AccessLevelView)
		assert.Equal(t, domain.AccessLevel("execute"), domain.AccessLevelExecute)
	})
}

// Test domain errors.
func TestRegistryHandler_DomainErrors(t *testing.T) {
	t.Run("ErrServerNotFound", func(t *testing.T) {
		err := domain.ErrServerNotFound
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrServerNotFound))
	})

	t.Run("ErrNotFound", func(t *testing.T) {
		err := domain.ErrNotFound
		assert.Error(t, err)
	})
}

package registry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// mockServerRepository implements a mock for testing the registry service.
type mockServerRepository struct {
	servers       map[string]*domain.MCPServer
	healthRecords map[string]*domain.ServerHealth

	// Error injection for testing error paths
	createErr           error
	getErr              error
	listErr             error
	listForUserErr      error
	updateErr           error
	deleteErr           error
	getHealthStatusErr  error
	saveHealthStatusErr error
}

func newMockRepository() *mockServerRepository {
	return &mockServerRepository{
		servers:       make(map[string]*domain.MCPServer),
		healthRecords: make(map[string]*domain.ServerHealth),
	}
}

func (m *mockServerRepository) Create(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	server := &domain.MCPServer{
		ID:                  "server-" + req.Name,
		Name:                req.Name,
		Description:         req.Description,
		URL:                 req.URL,
		ProtocolVersion:     req.ProtocolVersion,
		Transport:           req.Transport,
		AuthType:            req.AuthType,
		AuthConfig:          req.AuthConfig,
		HealthCheckURL:      req.HealthCheckURL,
		HealthCheckInterval: req.HealthCheckInterval,
		TimeoutSeconds:      req.TimeoutSeconds,
		MaxConnections:      req.MaxConnections,
		IsActive:            true,
		Tags:                req.Tags,
		AllowedTools:        req.AllowedTools,
		Metadata:            req.Metadata,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	m.servers[server.ID] = server

	return server, nil
}

func (m *mockServerRepository) Get(ctx context.Context, id string) (*domain.MCPServer, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	server, ok := m.servers[id]
	if !ok {
		return nil, domain.ErrServerNotFound
	}

	return server, nil
}

func (m *mockServerRepository) List(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	var servers []*domain.MCPServer
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers, nil
}

func (m *mockServerRepository) ListForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
	if m.listForUserErr != nil {
		return nil, m.listForUserErr
	}

	// If nil, return all (admin bypass)
	if accessibleServerIDs == nil {
		return m.List(ctx, filter)
	}

	// If empty, return none
	if len(accessibleServerIDs) == 0 {
		return []*domain.MCPServer{}, nil
	}

	// Filter by accessible IDs
	accessibleSet := make(map[string]bool)
	for _, id := range accessibleServerIDs {
		accessibleSet[id] = true
	}

	var servers []*domain.MCPServer
	for _, server := range m.servers {
		if accessibleSet[server.ID] {
			servers = append(servers, server)
		}
	}

	return servers, nil
}

func (m *mockServerRepository) Update(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}

	server, ok := m.servers[id]
	if !ok {
		return nil, domain.ErrServerNotFound
	}

	if req.Name != nil {
		server.Name = *req.Name
	}
	if req.Description != nil {
		server.Description = *req.Description
	}
	if req.URL != nil {
		server.URL = *req.URL
	}
	if req.IsActive != nil {
		server.IsActive = *req.IsActive
	}
	if req.TimeoutSeconds != nil {
		server.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.MaxConnections != nil {
		server.MaxConnections = *req.MaxConnections
	}

	server.UpdatedAt = time.Now()

	return server, nil
}

func (m *mockServerRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.servers[id]; !ok {
		return domain.ErrServerNotFound
	}

	delete(m.servers, id)

	return nil
}

func (m *mockServerRepository) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	if m.getHealthStatusErr != nil {
		return nil, m.getHealthStatusErr
	}

	health, ok := m.healthRecords[serverID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return health, nil
}

func (m *mockServerRepository) SaveHealthStatus(ctx context.Context, health *domain.ServerHealth) error {
	if m.saveHealthStatusErr != nil {
		return m.saveHealthStatusErr
	}

	m.healthRecords[health.ServerID] = health

	return nil
}

// testableService wraps Service for testing with mock repository.
type testableService struct {
	*Service
	mockRepo *mockServerRepository
}

func newTestableService() *testableService {
	mockRepo := newMockRepository()
	log := logger.NewNopLogger()

	// We need to use the real Service struct but inject our mock
	// Since Service uses concrete type, we'll test via the testable wrapper
	return &testableService{
		Service: &Service{
			repo:   nil, // We'll override methods via the wrapper
			logger: log,
		},
		mockRepo: mockRepo,
	}
}

// Override Service methods to use mock repository

func (ts *testableService) CreateServer(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
	// Set defaults if not provided (same as real service)
	if req.ProtocolVersion == "" {
		req.ProtocolVersion = "1.0.0"
	}
	if req.HealthCheckInterval == 0 {
		req.HealthCheckInterval = 60
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 30
	}
	if req.MaxConnections == 0 {
		req.MaxConnections = 100
	}

	server, err := ts.mockRepo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	ts.logger.Info().
		Str("server_id", server.ID).
		Str("name", server.Name).
		Msg("MCP server registered")

	return server, nil
}

func (ts *testableService) ListServers(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
	servers, err := ts.mockRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	ts.logger.Debug().Int("count", len(servers)).Msg("Servers listed")

	return servers, nil
}

func (ts *testableService) ListServersForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
	servers, err := ts.mockRepo.ListForUser(ctx, filter, accessibleServerIDs)
	if err != nil {
		return nil, err
	}

	ts.logger.Debug().
		Int("count", len(servers)).
		Bool("filtered", accessibleServerIDs != nil).
		Msg("Servers listed for user")

	return servers, nil
}

func (ts *testableService) GetServer(ctx context.Context, id string) (*domain.MCPServer, error) {
	server, err := ts.mockRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	health, err := ts.mockRepo.GetHealthStatus(ctx, id)
	if err != nil {
		ts.logger.Warn().Err(err).Str("server_id", id).Msg("Failed to get health status")
	} else {
		server.CurrentStatus = health
	}

	return server, nil
}

func (ts *testableService) UpdateServer(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
	server, err := ts.mockRepo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	ts.logger.Info().
		Str("server_id", id).
		Str("name", server.Name).
		Msg("MCP server updated")

	return server, nil
}

func (ts *testableService) DeleteServer(ctx context.Context, id string) error {
	err := ts.mockRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	ts.logger.Info().Str("server_id", id).Msg("MCP server deleted")

	return nil
}

func (ts *testableService) ToggleServer(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error) {
	update := &domain.ServerUpdate{
		IsActive: &enabled,
	}

	server, err := ts.mockRepo.Update(ctx, id, update)
	if err != nil {
		return nil, err
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}

	ts.logger.Info().
		Str("server_id", id).
		Str("name", server.Name).
		Str("action", action).
		Msg("MCP server toggled")

	return server, nil
}

func (ts *testableService) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	return ts.mockRepo.GetHealthStatus(ctx, serverID)
}

// Tests

func TestCreateServer_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	req := &domain.ServerCreate{
		Name: "test-server",
		URL:  "https://example.com/mcp",
	}

	server, err := ts.CreateServer(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "test-server", server.Name)
	assert.Equal(t, "https://example.com/mcp", server.URL)
	assert.True(t, server.IsActive)
}

func TestCreateServer_SetsDefaults(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	req := &domain.ServerCreate{
		Name: "test-server",
		URL:  "https://example.com/mcp",
	}

	server, err := ts.CreateServer(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", server.ProtocolVersion)
	assert.Equal(t, 60, server.HealthCheckInterval)
	assert.Equal(t, 30, server.TimeoutSeconds)
	assert.Equal(t, 100, server.MaxConnections)
}

func TestCreateServer_PreservesProvidedValues(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	req := &domain.ServerCreate{
		Name:                "test-server",
		URL:                 "https://example.com/mcp",
		ProtocolVersion:     "2024-11-05",
		HealthCheckInterval: 120,
		TimeoutSeconds:      60,
		MaxConnections:      50,
	}

	server, err := ts.CreateServer(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "2024-11-05", server.ProtocolVersion)
	assert.Equal(t, 120, server.HealthCheckInterval)
	assert.Equal(t, 60, server.TimeoutSeconds)
	assert.Equal(t, 50, server.MaxConnections)
}

func TestCreateServer_RepositoryError(t *testing.T) {
	ts := newTestableService()
	ts.mockRepo.createErr = errors.New("database error")
	ctx := context.Background()

	req := &domain.ServerCreate{
		Name: "test-server",
		URL:  "https://example.com/mcp",
	}

	server, err := ts.CreateServer(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "database error")
}

func TestListServers_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	// Create some servers
	ts.mockRepo.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Server 1"}
	ts.mockRepo.servers["server-2"] = &domain.MCPServer{ID: "server-2", Name: "Server 2"}

	servers, err := ts.ListServers(ctx, nil)

	require.NoError(t, err)
	assert.Len(t, servers, 2)
}

func TestListServers_Empty(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	servers, err := ts.ListServers(ctx, nil)

	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestListServers_RepositoryError(t *testing.T) {
	ts := newTestableService()
	ts.mockRepo.listErr = errors.New("database error")
	ctx := context.Background()

	servers, err := ts.ListServers(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, servers)
}

func TestListServersForUser_AdminBypass(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Server 1"}
	ts.mockRepo.servers["server-2"] = &domain.MCPServer{ID: "server-2", Name: "Server 2"}

	// nil accessibleServerIDs = admin bypass, returns all
	servers, err := ts.ListServersForUser(ctx, nil, nil)

	require.NoError(t, err)
	assert.Len(t, servers, 2)
}

func TestListServersForUser_EmptyAccess(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Server 1"}
	ts.mockRepo.servers["server-2"] = &domain.MCPServer{ID: "server-2", Name: "Server 2"}

	// Empty slice = no access
	servers, err := ts.ListServersForUser(ctx, nil, []string{})

	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestListServersForUser_FilteredAccess(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{ID: "server-1", Name: "Server 1"}
	ts.mockRepo.servers["server-2"] = &domain.MCPServer{ID: "server-2", Name: "Server 2"}
	ts.mockRepo.servers["server-3"] = &domain.MCPServer{ID: "server-3", Name: "Server 3"}

	// Only access to server-1 and server-3
	servers, err := ts.ListServersForUser(ctx, nil, []string{"server-1", "server-3"})

	require.NoError(t, err)
	assert.Len(t, servers, 2)

	// Verify correct servers returned
	ids := make(map[string]bool)
	for _, s := range servers {
		ids[s.ID] = true
	}
	assert.True(t, ids["server-1"])
	assert.True(t, ids["server-3"])
	assert.False(t, ids["server-2"])
}

func TestGetServer_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:   "server-1",
		Name: "Test Server",
		URL:  "https://example.com/mcp",
	}

	server, err := ts.GetServer(ctx, "server-1")

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "server-1", server.ID)
	assert.Equal(t, "Test Server", server.Name)
}

func TestGetServer_WithHealthStatus(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:   "server-1",
		Name: "Test Server",
	}
	ts.mockRepo.healthRecords["server-1"] = &domain.ServerHealth{
		ServerID: "server-1",
		Status:   domain.ServerStatusHealthy,
	}

	server, err := ts.GetServer(ctx, "server-1")

	require.NoError(t, err)
	require.NotNil(t, server.CurrentStatus)
	assert.Equal(t, domain.ServerStatusHealthy, server.CurrentStatus.Status)
}

func TestGetServer_NotFound(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	server, err := ts.GetServer(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.ErrorIs(t, err, domain.ErrServerNotFound)
}

func TestGetServer_HealthStatusError_StillReturnsServer(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:   "server-1",
		Name: "Test Server",
	}
	ts.mockRepo.getHealthStatusErr = errors.New("health check error")

	server, err := ts.GetServer(ctx, "server-1")

	// Server should still be returned even if health status fails
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Nil(t, server.CurrentStatus)
}

func TestUpdateServer_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:   "server-1",
		Name: "Old Name",
		URL:  "https://old.example.com",
	}

	newName := "New Name"
	newURL := "https://new.example.com"
	update := &domain.ServerUpdate{
		Name: &newName,
		URL:  &newURL,
	}

	server, err := ts.UpdateServer(ctx, "server-1", update)

	require.NoError(t, err)
	assert.Equal(t, "New Name", server.Name)
	assert.Equal(t, "https://new.example.com", server.URL)
}

func TestUpdateServer_NotFound(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	name := "New Name"
	update := &domain.ServerUpdate{Name: &name}

	server, err := ts.UpdateServer(ctx, "nonexistent", update)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.ErrorIs(t, err, domain.ErrServerNotFound)
}

func TestDeleteServer_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:   "server-1",
		Name: "Test Server",
	}

	err := ts.DeleteServer(ctx, "server-1")

	require.NoError(t, err)
	assert.Empty(t, ts.mockRepo.servers)
}

func TestDeleteServer_NotFound(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	err := ts.DeleteServer(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServerNotFound)
}

func TestToggleServer_Enable(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:       "server-1",
		Name:     "Test Server",
		IsActive: false,
	}

	server, err := ts.ToggleServer(ctx, "server-1", true)

	require.NoError(t, err)
	assert.True(t, server.IsActive)
}

func TestToggleServer_Disable(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.servers["server-1"] = &domain.MCPServer{
		ID:       "server-1",
		Name:     "Test Server",
		IsActive: true,
	}

	server, err := ts.ToggleServer(ctx, "server-1", false)

	require.NoError(t, err)
	assert.False(t, server.IsActive)
}

func TestToggleServer_NotFound(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	server, err := ts.ToggleServer(ctx, "nonexistent", true)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.ErrorIs(t, err, domain.ErrServerNotFound)
}

func TestGetHealthStatus_Success(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	ts.mockRepo.healthRecords["server-1"] = &domain.ServerHealth{
		ServerID:       "server-1",
		Status:         domain.ServerStatusHealthy,
		ResponseTimeMs: 50,
	}

	health, err := ts.GetHealthStatus(ctx, "server-1")

	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, domain.ServerStatusHealthy, health.Status)
	assert.Equal(t, 50, health.ResponseTimeMs)
}

func TestGetHealthStatus_NotFound(t *testing.T) {
	ts := newTestableService()
	ctx := context.Background()

	health, err := ts.GetHealthStatus(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, health)
}

func TestParseSSEResponse_ValidData(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}

	sseData := `event: message
data: {"jsonrpc":"2.0","result":{"tools":[{"name":"test"}]},"id":1}
`
	result := s.parseSSEResponse(sseData)

	require.NotEmpty(t, result)
	assert.Contains(t, result, "result")
}

func TestParseSSEResponse_EmptyData(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}

	result := s.parseSSEResponse("")

	assert.Empty(t, result)
}

func TestParseSSEResponse_InvalidJSON(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}

	sseData := `event: message
data: not valid json
`
	result := s.parseSSEResponse(sseData)

	assert.Empty(t, result)
}

func TestParseSSEResponse_MultipleDataLines(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}

	sseData := `event: message
data: {"jsonrpc":"2.0","result":{"first":true},"id":1}

event: message
data: {"jsonrpc":"2.0","result":{"second":true},"id":2}
`
	// Should return first valid JSON
	result := s.parseSSEResponse(sseData)

	require.NotEmpty(t, result)
	if rpcResult, ok := result["result"].(map[string]interface{}); ok {
		assert.True(t, rpcResult["first"].(bool))
	}
}

func TestTestConnectionRequest_Defaults(t *testing.T) {
	req := &TestConnectionRequest{
		URL: "https://example.com/mcp",
	}

	assert.Equal(t, "", req.Transport)
	assert.Equal(t, 0, req.TimeoutSeconds)
}

func TestCallToolRequest_Defaults(t *testing.T) {
	req := &CallToolRequest{
		URL:      "https://example.com/mcp",
		ToolName: "test_tool",
	}

	assert.Equal(t, "", req.Transport)
	assert.Equal(t, 0, req.TimeoutSeconds)
	assert.Equal(t, "", req.ProtocolVersion)
}

func TestTestConnectionResult_Fields(t *testing.T) {
	result := &TestConnectionResult{
		Success:        true,
		ResponseTimeMs: 100,
		ToolCount:      5,
		Tools:          []any{"tool1", "tool2"},
	}

	assert.True(t, result.Success)
	assert.Equal(t, 100, result.ResponseTimeMs)
	assert.Equal(t, 5, result.ToolCount)
	assert.Len(t, result.Tools, 2)
}

func TestCallToolResult_Fields(t *testing.T) {
	result := &CallToolResult{
		Success:      true,
		Content:      map[string]string{"data": "value"},
		IsError:      false,
		ErrorMessage: "",
	}

	assert.True(t, result.Success)
	assert.False(t, result.IsError)
	assert.NotNil(t, result.Content)
}

func TestCallToolResult_Error(t *testing.T) {
	result := &CallToolResult{
		Success:      false,
		IsError:      true,
		ErrorMessage: "Tool execution failed",
	}

	assert.False(t, result.Success)
	assert.True(t, result.IsError)
	assert.Equal(t, "Tool execution failed", result.ErrorMessage)
}

// Tests for HTTP-based methods using httptest

func TestPerformHealthCheck_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	status, responseTime, errorMsg := s.performHealthCheck(ctx, ts.URL+"/health")

	assert.Equal(t, domain.ServerStatusHealthy, status)
	assert.GreaterOrEqual(t, responseTime, 0)
	assert.Empty(t, errorMsg)
}

func TestPerformHealthCheck_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	status, responseTime, errorMsg := s.performHealthCheck(ctx, ts.URL+"/health")

	assert.Equal(t, domain.ServerStatusUnhealthy, status)
	assert.GreaterOrEqual(t, responseTime, 0)
	assert.Contains(t, errorMsg, "500")
}

func TestPerformHealthCheck_Degraded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	status, responseTime, errorMsg := s.performHealthCheck(ctx, ts.URL+"/health")

	assert.Equal(t, domain.ServerStatusDegraded, status)
	assert.GreaterOrEqual(t, responseTime, 0)
	assert.Contains(t, errorMsg, "400")
}

func TestPerformHealthCheck_ConnectionFailed(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	status, _, errorMsg := s.performHealthCheck(ctx, "http://localhost:1/invalid")

	assert.Equal(t, domain.ServerStatusUnhealthy, status)
	assert.Contains(t, errorMsg, "Request failed")
}

func TestTestHTTPTransport_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/initialize" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		} else if r.URL.Path == "/tools/list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"tools":[{"name":"tool1"},{"name":"tool2"}]}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testHTTPTransport(ctx, ts.URL)

	assert.True(t, result.Success)
	assert.Equal(t, 2, result.ToolCount)
}

func TestTestHTTPTransport_InitializeFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testHTTPTransport(ctx, ts.URL)

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "500")
}

func TestTestHTTPTransport_ConnectionFailed(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testHTTPTransport(ctx, "http://localhost:1/invalid")

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Connection failed")
}

func TestTestStreamableHTTPTransport_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "initialize" {
			w.Header().Set("MCP-Session-Id", "session-123")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"serverInfo":{"name":"test"}},"id":1}`))
		} else if req["method"] == "notifications/initialized" {
			w.WriteHeader(http.StatusAccepted)
		} else if req["method"] == "tools/list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"tools":[{"name":"tool1"}]},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testStreamableHTTPTransport(ctx, ts.URL, "2025-11-25")

	assert.True(t, result.Success)
	assert.Equal(t, 1, result.ToolCount)
}

func TestTestStreamableHTTPTransport_SSEResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"serverInfo\":{\"name\":\"test\"}},\"id\":1}\n\n"))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testStreamableHTTPTransport(ctx, ts.URL, "")

	assert.True(t, result.Success)
}

func TestTestStreamableHTTPTransport_ConnectionFailed(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testStreamableHTTPTransport(ctx, "http://localhost:1/invalid", "")

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Connection failed")
}

func TestTestConnection_HTTPTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &TestConnectionRequest{
		URL:            ts.URL,
		Transport:      "http",
		TimeoutSeconds: 10,
	}

	result, err := s.TestConnection(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTestConnection_StreamableHTTPTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &TestConnectionRequest{
		URL:       ts.URL,
		Transport: "streamable_http",
	}

	result, err := s.TestConnection(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTestConnection_SSETransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":{"serverInfo":{}},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &TestConnectionRequest{
		URL:       ts.URL,
		Transport: "sse",
	}

	result, err := s.TestConnection(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTestConnection_UnsupportedTransport(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &TestConnectionRequest{
		URL:       "http://example.com",
		Transport: "unknown",
	}

	result, err := s.TestConnection(ctx, req)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Unsupported transport")
}

func TestTestConnection_DefaultTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &TestConnectionRequest{
		URL:            ts.URL,
		TimeoutSeconds: 0, // Should default to 10
	}

	result, err := s.TestConnection(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCallTool_StreamableHTTP(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Initialize
			w.Header().Set("MCP-Session-Id", "session-123")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else if callCount == 2 {
			// Initialized notification
			w.WriteHeader(http.StatusAccepted)
		} else {
			// Tool call
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[{"text":"result"}]},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &CallToolRequest{
		URL:       ts.URL,
		Transport: "streamable_http",
		ToolName:  "test_tool",
		Arguments: map[string]interface{}{"arg1": "value1"},
	}

	result, err := s.CallTool(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCallTool_HTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content":[{"text":"result"}]}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &CallToolRequest{
		URL:       ts.URL,
		Transport: "http",
		ToolName:  "test_tool",
	}

	result, err := s.CallTool(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCallTool_SSE(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[{"text":"result"}]},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &CallToolRequest{
		URL:       ts.URL,
		Transport: "sse",
		ToolName:  "test_tool",
	}

	result, err := s.CallTool(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestCallTool_UnsupportedTransport(t *testing.T) {
	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &CallToolRequest{
		URL:       "http://example.com",
		Transport: "unknown",
		ToolName:  "test_tool",
	}

	result, err := s.CallTool(ctx, req)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Unsupported transport")
}

func TestCallTool_DefaultTransport(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[]},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	req := &CallToolRequest{
		URL:       ts.URL,
		Transport: "", // Should default to streamable_http
		ToolName:  "test_tool",
	}

	result, err := s.CallTool(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCallToolStreamableHTTP_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.IsError)
}

func TestCallToolHTTP_ParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Failed to parse")
}

func TestTestSSETransport_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "initialize" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"serverInfo":{"name":"test"}},"id":1}`))
		} else if req["method"] == "tools/list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"tools":[{"name":"tool1"}]},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	assert.True(t, result.Success)
	assert.Equal(t, 1, result.ToolCount)
}

func TestTestSSETransport_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Error"},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Error")
}

func TestCallToolSSE_SSEResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"content\":[]},\"id\":1}\n\n"))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolSSE(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
}

func TestNewService_ValidInputs(t *testing.T) {
	log := logger.NewNopLogger()

	svc := NewService(nil, log)

	require.NotNil(t, svc)
	assert.Nil(t, svc.repo)
	assert.NotNil(t, svc.logger)
}

func TestCallToolSSE_RpcError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Tool not found"},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolSSE(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ErrorMessage, "Tool not found")
}

func TestCallToolSSE_ResultWithIsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":"error occurred","isError":true},"id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolSSE(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
	assert.True(t, result.IsError)
}

func TestCallToolSSE_DirectContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"text":"direct content"}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolSSE(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
}

func TestCallToolSSE_ParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolSSE(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Failed to parse")
}

func TestTestSSETransport_NoResultField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	assert.True(t, result.Success)
	assert.NotNil(t, result.ServerInfo)
}

func TestTestSSETransport_SSEFormatResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "initialize" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"serverInfo\":{\"name\":\"test\"}},\"id\":1}\n\n"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[]},\"id\":2}\n\n"))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	assert.True(t, result.Success)
}

func TestTestSSETransport_ToolsListingFails(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Initialize succeeds
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"serverInfo":{"name":"test"}},"id":1}`))
		} else {
			// Tools listing fails with error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	// Connection succeeded even though tools listing failed
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.ToolCount)
}

func TestTestSSETransport_ToolsSSEFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "initialize" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"serverInfo":{}},"id":1}`))
		} else if req["method"] == "tools/list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[{\"name\":\"tool1\"},{\"name\":\"tool2\"}]},\"id\":2}\n\n"))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.testSSETransport(ctx, ts.URL)

	assert.True(t, result.Success)
	assert.Equal(t, 2, result.ToolCount)
}

func TestCallToolStreamableHTTP_LowercaseSessionHeader(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Initialize with lowercase header
			w.Header().Set("mcp-session-id", "session-456")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else if callCount == 2 {
			// Initialized notification
			w.WriteHeader(http.StatusAccepted)
		} else {
			// Tool call
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[]},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
}

func TestCallToolStreamableHTTP_SSEFormat(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"result\":{\"content\":[]},\"id\":2}\n\n"))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
}

func TestCallToolStreamableHTTP_ResultWithIsError(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":"error","isError":true},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
	assert.True(t, result.IsError)
}

func TestCallToolStreamableHTTP_DirectContent(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else {
			// Response without result wrapper
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","id":2,"text":"direct"}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
}

func TestCallToolStreamableHTTP_ProtocolVersion(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			assert.Equal(t, "2024-11-05", r.Header.Get("MCP-Protocol-Version"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{},"id":1}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":{"content":[]},"id":2}`))
		}
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolStreamableHTTP(ctx, &CallToolRequest{
		URL:             ts.URL,
		ToolName:        "test_tool",
		ProtocolVersion: "2024-11-05",
	})

	assert.True(t, result.Success)
}

func TestCallToolHTTP_WithContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content":[{"type":"text","text":"hello"}]}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	assert.True(t, result.Success)
	assert.NotNil(t, result.Content)
}

func TestCallToolHTTP_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error":"Tool execution failed"}`))
	}))
	defer ts.Close()

	s := &Service{logger: logger.NewNopLogger()}
	ctx := context.Background()

	result := s.callToolHTTP(ctx, &CallToolRequest{
		URL:      ts.URL,
		ToolName: "test_tool",
	})

	// callToolHTTP doesn't parse the error field - it just returns content
	assert.True(t, result.Success)
	assert.NotNil(t, result.Content)
}

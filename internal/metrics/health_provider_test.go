package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
)

// mockServerRepository implements ServerRepository for testing
type mockServerRepository struct {
	listFunc            func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error)
	getHealthStatusFunc func(ctx context.Context, serverID string) (*domain.ServerHealth, error)
}

func (m *mockServerRepository) List(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return []*domain.MCPServer{}, nil
}

func (m *mockServerRepository) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	if m.getHealthStatusFunc != nil {
		return m.getHealthStatusFunc(ctx, serverID)
	}
	return nil, nil
}

func TestNewHealthProviderAdapter(t *testing.T) {
	repo := &mockServerRepository{}
	adapter := NewHealthProviderAdapter(repo)

	assert.NotNil(t, adapter)
	assert.Equal(t, repo, adapter.repo)
}

func TestHealthProviderAdapter_GetAllServersHealth_Empty(t *testing.T) {
	repo := &mockServerRepository{
		listFunc: func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
			return []*domain.MCPServer{}, nil
		},
	}
	adapter := NewHealthProviderAdapter(repo)

	health, err := adapter.GetAllServersHealth(context.Background())

	require.NoError(t, err)
	assert.Empty(t, health)
}

func TestHealthProviderAdapter_GetAllServersHealth_WithServers(t *testing.T) {
	servers := []*domain.MCPServer{
		{ID: "server-1", Name: "Test Server 1", IsActive: true},
		{ID: "server-2", Name: "Test Server 2", IsActive: false},
	}

	repo := &mockServerRepository{
		listFunc: func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
			return servers, nil
		},
		getHealthStatusFunc: func(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
			if serverID == "server-1" {
				return &domain.ServerHealth{
					ServerID: "server-1",
					Status:   domain.ServerStatusHealthy,
				}, nil
			}
			return nil, errors.New("health check failed")
		},
	}
	adapter := NewHealthProviderAdapter(repo)

	health, err := adapter.GetAllServersHealth(context.Background())

	require.NoError(t, err)
	assert.Len(t, health, 2)

	// Check server-1 health
	server1Health, ok := health["server-1"]
	require.True(t, ok)
	assert.Equal(t, "server-1", server1Health.ServerID)
	assert.Equal(t, "Test Server 1", server1Health.ServerName)
	assert.Equal(t, "healthy", server1Health.Status)
	assert.True(t, server1Health.IsActive)

	// Check server-2 health (health check failed, so status should be "unknown")
	server2Health, ok := health["server-2"]
	require.True(t, ok)
	assert.Equal(t, "server-2", server2Health.ServerID)
	assert.Equal(t, "Test Server 2", server2Health.ServerName)
	assert.Equal(t, "unknown", server2Health.Status)
	assert.False(t, server2Health.IsActive)
}

func TestHealthProviderAdapter_GetAllServersHealth_ListError(t *testing.T) {
	repo := &mockServerRepository{
		listFunc: func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
			return nil, errors.New("database error")
		},
	}
	adapter := NewHealthProviderAdapter(repo)

	health, err := adapter.GetAllServersHealth(context.Background())

	require.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "database error")
}

func TestHealthProviderAdapter_GetAllServersHealth_HealthStatusError(t *testing.T) {
	servers := []*domain.MCPServer{
		{ID: "server-1", Name: "Test Server 1", IsActive: true},
	}

	repo := &mockServerRepository{
		listFunc: func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
			return servers, nil
		},
		getHealthStatusFunc: func(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
			return nil, errors.New("health check failed")
		},
	}
	adapter := NewHealthProviderAdapter(repo)

	health, err := adapter.GetAllServersHealth(context.Background())

	// Should not return error - just set status to "unknown"
	require.NoError(t, err)
	assert.Len(t, health, 1)

	serverHealth := health["server-1"]
	assert.Equal(t, "unknown", serverHealth.Status)
}

func TestHealthProviderAdapter_GetAllServersHealth_AllStatuses(t *testing.T) {
	servers := []*domain.MCPServer{
		{ID: "healthy-server", Name: "Healthy", IsActive: true},
		{ID: "degraded-server", Name: "Degraded", IsActive: true},
		{ID: "unhealthy-server", Name: "Unhealthy", IsActive: true},
		{ID: "unknown-server", Name: "Unknown", IsActive: false},
	}

	repo := &mockServerRepository{
		listFunc: func(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
			return servers, nil
		},
		getHealthStatusFunc: func(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
			switch serverID {
			case "healthy-server":
				return &domain.ServerHealth{Status: domain.ServerStatusHealthy}, nil
			case "degraded-server":
				return &domain.ServerHealth{Status: domain.ServerStatusDegraded}, nil
			case "unhealthy-server":
				return &domain.ServerHealth{Status: domain.ServerStatusUnhealthy}, nil
			default:
				return nil, errors.New("not found")
			}
		},
	}
	adapter := NewHealthProviderAdapter(repo)

	health, err := adapter.GetAllServersHealth(context.Background())

	require.NoError(t, err)
	assert.Len(t, health, 4)

	assert.Equal(t, "healthy", health["healthy-server"].Status)
	assert.Equal(t, "degraded", health["degraded-server"].Status)
	assert.Equal(t, "unhealthy", health["unhealthy-server"].Status)
	assert.Equal(t, "unknown", health["unknown-server"].Status)
}

func TestServerHealth_Structure(t *testing.T) {
	health := ServerHealth{
		ServerID:   "server-123",
		ServerName: "Test Server",
		Status:     "healthy",
		IsActive:   true,
	}

	assert.Equal(t, "server-123", health.ServerID)
	assert.Equal(t, "Test Server", health.ServerName)
	assert.Equal(t, "healthy", health.Status)
	assert.True(t, health.IsActive)
}

package metrics

import (
	"context"

	"github.com/waffles/waffles/internal/domain"
)

// ServerRepository defines the interface for getting server data
type ServerRepository interface {
	List(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error)
	GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error)
}

// HealthProviderAdapter adapts the server repository to implement ServerHealthProvider
type HealthProviderAdapter struct {
	repo ServerRepository
}

// NewHealthProviderAdapter creates a new health provider adapter
func NewHealthProviderAdapter(repo ServerRepository) *HealthProviderAdapter {
	return &HealthProviderAdapter{repo: repo}
}

// GetAllServersHealth returns health status for all servers
func (h *HealthProviderAdapter) GetAllServersHealth(ctx context.Context) (map[string]ServerHealth, error) {
	// Get all servers
	servers, err := h.repo.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	healthMap := make(map[string]ServerHealth)

	for _, server := range servers {
		health := ServerHealth{
			ServerID:   server.ID,
			ServerName: server.Name,
			Status:     "unknown",
			IsActive:   server.IsActive,
		}

		// Get latest health status
		serverHealth, err := h.repo.GetHealthStatus(ctx, server.ID)
		if err == nil && serverHealth != nil {
			health.Status = string(serverHealth.Status)
		}

		healthMap[server.ID] = health
	}

	return healthMap, nil
}

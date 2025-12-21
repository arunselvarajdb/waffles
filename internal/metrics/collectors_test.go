package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDBStatsProvider is a mock implementation of DBStatsProvider for testing.
type mockDBStatsProvider struct {
	stats *pgxpool.Stat
}

func (m *mockDBStatsProvider) Stats() *pgxpool.Stat {
	return m.stats
}

// Since pgxpool.Stat doesn't have a public constructor, we'll test with nil handling.
type mockStat struct {
	totalConns    int32
	acquiredConns int32
	idleConns     int32
	acquireCount  int64
	acquireDur    time.Duration
}

func TestNewDBStatsCollector(t *testing.T) {
	reg := NewRegistry()

	t.Run("creates collector with provider", func(t *testing.T) {
		provider := &mockDBStatsProvider{}
		collector := NewDBStatsCollector(reg, provider)

		require.NotNil(t, collector)
		assert.NotNil(t, collector.registry)
		assert.NotNil(t, collector.dbProvider)
	})

	t.Run("creates collector with nil provider", func(t *testing.T) {
		collector := NewDBStatsCollector(reg, nil)

		require.NotNil(t, collector)
		assert.NotNil(t, collector.registry)
		assert.Nil(t, collector.dbProvider)
	})
}

func TestDBStatsCollector_Collect(t *testing.T) {
	t.Run("handles nil provider gracefully", func(t *testing.T) {
		reg := NewRegistry()
		collector := NewDBStatsCollector(reg, nil)

		// Should not panic
		collector.Collect()
	})

	t.Run("handles nil stats gracefully", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockDBStatsProvider{stats: nil}
		collector := NewDBStatsCollector(reg, provider)

		// Should not panic
		collector.Collect()
	})
}

// mockServerHealthProvider is a mock implementation of ServerHealthProvider for testing.
type mockServerHealthProvider struct {
	healthMap map[string]ServerHealth
	err       error
}

func (m *mockServerHealthProvider) GetAllServersHealth(ctx context.Context) (map[string]ServerHealth, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.healthMap, nil
}

func TestNewServerHealthCollector(t *testing.T) {
	reg := NewRegistry()

	t.Run("creates collector with provider", func(t *testing.T) {
		provider := &mockServerHealthProvider{}
		collector := NewServerHealthCollector(reg, provider)

		require.NotNil(t, collector)
		assert.NotNil(t, collector.registry)
		assert.NotNil(t, collector.healthProvider)
	})

	t.Run("creates collector with nil provider", func(t *testing.T) {
		collector := NewServerHealthCollector(reg, nil)

		require.NotNil(t, collector)
		assert.NotNil(t, collector.registry)
		assert.Nil(t, collector.healthProvider)
	})
}

func TestServerHealthCollector_Collect(t *testing.T) {
	t.Run("handles nil provider gracefully", func(t *testing.T) {
		reg := NewRegistry()
		collector := NewServerHealthCollector(reg, nil)

		err := collector.Collect(context.Background())

		assert.NoError(t, err)
	})

	t.Run("returns error from provider", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			err: errors.New("failed to get health"),
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get health")
	})

	t.Run("collects health status for healthy servers", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			healthMap: map[string]ServerHealth{
				"server-1": {
					ServerID:   "server-1",
					ServerName: "Server One",
					Status:     "healthy",
					IsActive:   true,
				},
				"server-2": {
					ServerID:   "server-2",
					ServerName: "Server Two",
					Status:     "healthy",
					IsActive:   true,
				},
			},
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		require.NoError(t, err)

		// Verify metrics were collected
		families, err := reg.GetRegistry().Gather()
		require.NoError(t, err)

		var foundHealthStatus bool
		var foundServersTotal bool
		for _, mf := range families {
			if mf.GetName() == "gateway_server_health_status" {
				foundHealthStatus = true
				assert.GreaterOrEqual(t, len(mf.GetMetric()), 1)
			}
			if mf.GetName() == "registry_servers_total" {
				foundServersTotal = true
			}
		}
		assert.True(t, foundHealthStatus, "gateway_server_health_status metric should be found")
		assert.True(t, foundServersTotal, "registry_servers_total metric should be found")
	})

	t.Run("collects health status for unhealthy servers", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			healthMap: map[string]ServerHealth{
				"server-1": {
					ServerID:   "server-1",
					ServerName: "Server One",
					Status:     "unhealthy",
					IsActive:   true,
				},
			},
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		require.NoError(t, err)
	})

	t.Run("counts active and inactive servers", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			healthMap: map[string]ServerHealth{
				"server-1": {
					ServerID:   "server-1",
					ServerName: "Active Server",
					Status:     "healthy",
					IsActive:   true,
				},
				"server-2": {
					ServerID:   "server-2",
					ServerName: "Inactive Server",
					Status:     "unhealthy",
					IsActive:   false,
				},
				"server-3": {
					ServerID:   "server-3",
					ServerName: "Another Active",
					Status:     "healthy",
					IsActive:   true,
				},
			},
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		require.NoError(t, err)

		// Verify active/inactive counts
		families, err := reg.GetRegistry().Gather()
		require.NoError(t, err)

		for _, mf := range families {
			if mf.GetName() == "registry_servers_total" {
				for _, m := range mf.GetMetric() {
					for _, label := range m.GetLabel() {
						if label.GetName() == "status" {
							if label.GetValue() == "active" {
								assert.Equal(t, float64(2), m.GetGauge().GetValue())
							} else if label.GetValue() == "inactive" {
								assert.Equal(t, float64(1), m.GetGauge().GetValue())
							}
						}
					}
				}
			}
		}
	})

	t.Run("handles empty health map", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			healthMap: map[string]ServerHealth{},
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		require.NoError(t, err)
	})

	t.Run("sets different values for different health statuses", func(t *testing.T) {
		reg := NewRegistry()
		provider := &mockServerHealthProvider{
			healthMap: map[string]ServerHealth{
				"server-1": {
					ServerID:   "server-1",
					ServerName: "Healthy Server",
					Status:     "healthy",
					IsActive:   true,
				},
				"server-2": {
					ServerID:   "server-2",
					ServerName: "Degraded Server",
					Status:     "degraded",
					IsActive:   true,
				},
				"server-3": {
					ServerID:   "server-3",
					ServerName: "Unhealthy Server",
					Status:     "unhealthy",
					IsActive:   true,
				},
			},
		}
		collector := NewServerHealthCollector(reg, provider)

		err := collector.Collect(context.Background())

		require.NoError(t, err)

		// Verify health status metrics
		families, err := reg.GetRegistry().Gather()
		require.NoError(t, err)

		healthyCount := 0
		unhealthyCount := 0
		for _, mf := range families {
			if mf.GetName() == "gateway_server_health_status" {
				for _, m := range mf.GetMetric() {
					if m.GetGauge().GetValue() == 1 {
						healthyCount++
					} else {
						unhealthyCount++
					}
				}
			}
		}
		assert.Equal(t, 1, healthyCount, "should have 1 healthy server (value=1)")
		assert.Equal(t, 2, unhealthyCount, "should have 2 non-healthy servers (value=0)")
	})
}

func TestServerHealth(t *testing.T) {
	t.Run("struct fields are accessible", func(t *testing.T) {
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
	})
}

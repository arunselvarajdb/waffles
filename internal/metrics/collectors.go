package metrics

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBStatsProvider is an interface for getting database pool stats
// This allows us to decouple the metrics package from the database package
type DBStatsProvider interface {
	Stats() *pgxpool.Stat
}

// DBStatsCollector collects database connection pool statistics
type DBStatsCollector struct {
	registry   *Registry
	dbProvider DBStatsProvider
}

// NewDBStatsCollector creates a new database stats collector
func NewDBStatsCollector(registry *Registry, dbProvider DBStatsProvider) *DBStatsCollector {
	return &DBStatsCollector{
		registry:   registry,
		dbProvider: dbProvider,
	}
}

// Collect updates the database metrics with current stats
// This should be called periodically (e.g., every 10 seconds) or on-demand
func (c *DBStatsCollector) Collect() {
	if c.dbProvider == nil {
		return
	}

	stats := c.dbProvider.Stats()
	if stats == nil {
		return
	}

	// Update gauges using pgxpool.Stat fields
	c.registry.DBConnectionsOpen.Set(float64(stats.TotalConns()))
	c.registry.DBConnectionsInUse.Set(float64(stats.AcquiredConns()))
	c.registry.DBConnectionsIdle.Set(float64(stats.IdleConns()))

	// pgxpool tracks acquire count and duration (similar to wait stats)
	c.registry.DBConnectionWaitCount.Add(float64(stats.AcquireCount()))
	c.registry.DBConnectionWaitDuration.Observe(stats.AcquireDuration().Seconds())
}

// ServerHealthProvider is an interface for getting server health status
// This allows us to decouple the metrics package from the registry package
type ServerHealthProvider interface {
	// GetAllServersHealth returns a map of server_id -> health status
	GetAllServersHealth(ctx context.Context) (map[string]ServerHealth, error)
}

// ServerHealth represents the health status of a server
type ServerHealth struct {
	ServerID   string
	ServerName string
	Status     string // "healthy", "degraded", "unhealthy"
	IsActive   bool
}

// ServerHealthCollector collects server health status
type ServerHealthCollector struct {
	registry       *Registry
	healthProvider ServerHealthProvider
}

// NewServerHealthCollector creates a new server health collector
func NewServerHealthCollector(registry *Registry, healthProvider ServerHealthProvider) *ServerHealthCollector {
	return &ServerHealthCollector{
		registry:       registry,
		healthProvider: healthProvider,
	}
}

// Collect updates the server health metrics with current status
// This should be called periodically (e.g., every 30 seconds) or after health checks
func (c *ServerHealthCollector) Collect(ctx context.Context) error {
	if c.healthProvider == nil {
		return nil
	}

	healthMap, err := c.healthProvider.GetAllServersHealth(ctx)
	if err != nil {
		return err
	}

	// Reset all health status gauges first
	c.registry.GatewayServerHealthStatus.Reset()

	// Update health status for each server
	for _, health := range healthMap {
		// Set gauge to 1 if healthy, 0 otherwise
		var value float64
		if health.Status == "healthy" {
			value = 1
		} else {
			value = 0
		}

		c.registry.GatewayServerHealthStatus.WithLabelValues(
			health.ServerID,
			health.ServerName,
			health.Status,
		).Set(value)
	}

	// Count active and inactive servers
	activeCount := 0
	inactiveCount := 0
	for _, health := range healthMap {
		if health.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	c.registry.RegistryServersTotal.WithLabelValues("active").Set(float64(activeCount))
	c.registry.RegistryServersTotal.WithLabelValues("inactive").Set(float64(inactiveCount))

	return nil
}

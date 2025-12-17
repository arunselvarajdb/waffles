package database

import (
	"context"
	"time"
)

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Healthy          bool   `json:"healthy"`
	TotalConnections int32  `json:"total_connections"`
	IdleConnections  int32  `json:"idle_connections"`
	MaxConnections   int32  `json:"max_connections"`
	Message          string `json:"message,omitempty"`
}

// Health checks the health of the database connection
func (db *DB) Health(ctx context.Context) HealthStatus {
	// Set timeout for health check
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	status := HealthStatus{
		Healthy: false,
	}

	// Try to ping the database
	if err := db.Pool.Ping(ctx); err != nil {
		status.Message = err.Error()
		db.logger.Error().Err(err).Msg("Database health check failed")
		return status
	}

	// Get pool statistics
	stats := db.Pool.Stat()
	status.Healthy = true
	status.TotalConnections = stats.TotalConns()
	status.IdleConnections = stats.IdleConns()
	status.MaxConnections = stats.MaxConns()

	return status
}

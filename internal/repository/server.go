package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// ServerRepository handles MCP server data persistence
type ServerRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

// NewServerRepository creates a new server repository
func NewServerRepository(pool *pgxpool.Pool, log logger.Logger) *ServerRepository {
	return &ServerRepository{
		pool:   pool,
		logger: log,
	}
}

// Create creates a new MCP server
func (r *ServerRepository) Create(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error) {
	query := `
		INSERT INTO mcp_servers (
			name, description, url, protocol_version, transport,
			auth_type, auth_config, health_check_url, health_check_interval,
			timeout_seconds, max_connections, is_active, tags, allowed_tools, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	// Default transport to http if not specified
	transport := req.Transport
	if transport == "" {
		transport = domain.TransportHTTP
	}

	var server domain.MCPServer
	err := r.pool.QueryRow(ctx, query,
		req.Name,
		req.Description,
		req.URL,
		req.ProtocolVersion,
		transport,
		req.AuthType,
		req.AuthConfig,
		req.HealthCheckURL,
		req.HealthCheckInterval,
		req.TimeoutSeconds,
		req.MaxConnections,
		true, // is_active defaults to true
		req.Tags,
		req.AllowedTools,
		req.Metadata,
	).Scan(&server.ID, &server.CreatedAt, &server.UpdatedAt)

	if err != nil {
		r.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create server")
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Populate rest of fields
	server.Name = req.Name
	server.Description = req.Description
	server.URL = req.URL
	server.ProtocolVersion = req.ProtocolVersion
	server.Transport = transport
	server.AuthType = req.AuthType
	server.AuthConfig = req.AuthConfig
	server.HealthCheckURL = req.HealthCheckURL
	server.HealthCheckInterval = req.HealthCheckInterval
	server.TimeoutSeconds = req.TimeoutSeconds
	server.MaxConnections = req.MaxConnections
	server.IsActive = true // defaults to true
	server.Tags = req.Tags
	server.AllowedTools = req.AllowedTools
	server.Metadata = req.Metadata

	r.logger.Info().
		Str("server_id", server.ID).
		Str("name", server.Name).
		Str("transport", string(transport)).
		Msg("Server created successfully")

	return &server, nil
}

// List retrieves all MCP servers with optional filtering
func (r *ServerRepository) List(ctx context.Context, filter *domain.ServerFilter) ([]*domain.MCPServer, error) {
	query := `
		SELECT
			id, name, description, url, protocol_version, transport,
			auth_type, auth_config, health_check_url, health_check_interval,
			timeout_seconds, max_connections, is_active, tags, allowed_tools, metadata,
			created_at, updated_at
		FROM mcp_servers
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.Name != "" {
			query += fmt.Sprintf(" AND name ILIKE $%d", argPos)
			args = append(args, "%"+filter.Name+"%")
			argPos++
		}
		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argPos)
			args = append(args, *filter.IsActive)
			argPos++
		}
		if len(filter.Tags) > 0 {
			query += fmt.Sprintf(" AND tags && $%d", argPos)
			args = append(args, filter.Tags)
			argPos++
		}
	}

	// Default ordering
	query += " ORDER BY created_at DESC"

	// Apply limit and offset
	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list servers")
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	var servers []*domain.MCPServer
	for rows.Next() {
		var s domain.MCPServer
		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.URL, &s.ProtocolVersion, &s.Transport,
			&s.AuthType, &s.AuthConfig, &s.HealthCheckURL, &s.HealthCheckInterval,
			&s.TimeoutSeconds, &s.MaxConnections, &s.IsActive, &s.Tags, &s.AllowedTools, &s.Metadata,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan server row")
			continue
		}
		servers = append(servers, &s)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating server rows")
		return nil, fmt.Errorf("error iterating servers: %w", err)
	}

	r.logger.Debug().Int("count", len(servers)).Msg("Servers listed")
	return servers, nil
}

// Get retrieves a single MCP server by ID
func (r *ServerRepository) Get(ctx context.Context, id string) (*domain.MCPServer, error) {
	query := `
		SELECT
			id, name, description, url, protocol_version, transport,
			auth_type, auth_config, health_check_url, health_check_interval,
			timeout_seconds, max_connections, is_active, tags, allowed_tools, metadata,
			created_at, updated_at
		FROM mcp_servers
		WHERE id = $1
	`

	var server domain.MCPServer
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&server.ID, &server.Name, &server.Description, &server.URL, &server.ProtocolVersion, &server.Transport,
		&server.AuthType, &server.AuthConfig, &server.HealthCheckURL, &server.HealthCheckInterval,
		&server.TimeoutSeconds, &server.MaxConnections, &server.IsActive, &server.Tags, &server.AllowedTools, &server.Metadata,
		&server.CreatedAt, &server.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrServerNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).Str("server_id", id).Msg("Failed to get server")
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	r.logger.Debug().Str("server_id", id).Msg("Server retrieved")
	return &server, nil
}

// Update updates an existing MCP server
func (r *ServerRepository) Update(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error) {
	// First get current server to merge fields
	current, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates (only non-nil fields)
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.URL != nil {
		current.URL = *req.URL
	}
	if req.ProtocolVersion != nil {
		current.ProtocolVersion = *req.ProtocolVersion
	}
	if req.AuthType != nil {
		current.AuthType = *req.AuthType
	}
	if req.AuthConfig != nil {
		current.AuthConfig = req.AuthConfig
	}
	if req.HealthCheckURL != nil {
		current.HealthCheckURL = *req.HealthCheckURL
	}
	if req.HealthCheckInterval != nil {
		current.HealthCheckInterval = *req.HealthCheckInterval
	}
	if req.TimeoutSeconds != nil {
		current.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.MaxConnections != nil {
		current.MaxConnections = *req.MaxConnections
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	if req.Tags != nil {
		current.Tags = *req.Tags
	}
	if req.AllowedTools != nil {
		current.AllowedTools = *req.AllowedTools
	}
	if req.Metadata != nil {
		current.Metadata = req.Metadata
	}

	// Update in database
	query := `
		UPDATE mcp_servers
		SET name = $1, description = $2, url = $3, protocol_version = $4, transport = $5,
		    auth_type = $6, auth_config = $7, health_check_url = $8,
		    health_check_interval = $9, timeout_seconds = $10, max_connections = $11,
		    is_active = $12, tags = $13, allowed_tools = $14, metadata = $15, updated_at = $16
		WHERE id = $17
		RETURNING updated_at
	`

	current.UpdatedAt = time.Now()
	err = r.pool.QueryRow(ctx, query,
		current.Name, current.Description, current.URL, current.ProtocolVersion, current.Transport,
		current.AuthType, current.AuthConfig, current.HealthCheckURL,
		current.HealthCheckInterval, current.TimeoutSeconds, current.MaxConnections,
		current.IsActive, current.Tags, current.AllowedTools, current.Metadata, current.UpdatedAt, id,
	).Scan(&current.UpdatedAt)

	if err != nil {
		r.logger.Error().Err(err).Str("server_id", id).Msg("Failed to update server")
		return nil, fmt.Errorf("failed to update server: %w", err)
	}

	r.logger.Info().
		Str("server_id", id).
		Str("name", current.Name).
		Msg("Server updated successfully")

	return current, nil
}

// Delete deletes an MCP server by ID
func (r *ServerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM mcp_servers WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("server_id", id).Msg("Failed to delete server")
		return fmt.Errorf("failed to delete server: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrServerNotFound
	}

	r.logger.Info().Str("server_id", id).Msg("Server deleted successfully")
	return nil
}

// GetHealthStatus retrieves the latest health status for a server
func (r *ServerRepository) GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error) {
	query := `
		SELECT
			id, server_id, status, response_time_ms, error_message, checked_at
		FROM server_health
		WHERE server_id = $1
		ORDER BY checked_at DESC
		LIMIT 1
	`

	var health domain.ServerHealth
	err := r.pool.QueryRow(ctx, query, serverID).Scan(
		&health.ID, &health.ServerID, &health.Status,
		&health.ResponseTimeMs, &health.ErrorMessage, &health.CheckedAt,
	)

	if err == pgx.ErrNoRows {
		// No health check yet - return default status
		return &domain.ServerHealth{
			ServerID:  serverID,
			Status:    domain.ServerStatusUnknown,
			CheckedAt: time.Now(),
		}, nil
	}
	if err != nil {
		r.logger.Error().Err(err).Str("server_id", serverID).Msg("Failed to get health status")
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	return &health, nil
}

// SaveHealthStatus saves a new health check result
func (r *ServerRepository) SaveHealthStatus(ctx context.Context, health *domain.ServerHealth) error {
	query := `
		INSERT INTO server_health (server_id, status, response_time_ms, error_message, checked_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.pool.QueryRow(ctx, query,
		health.ServerID,
		health.Status,
		health.ResponseTimeMs,
		health.ErrorMessage,
		health.CheckedAt,
	).Scan(&health.ID)

	if err != nil {
		r.logger.Error().Err(err).Str("server_id", health.ServerID).Msg("Failed to save health status")
		return fmt.Errorf("failed to save health status: %w", err)
	}

	r.logger.Debug().
		Str("server_id", health.ServerID).
		Str("status", string(health.Status)).
		Msg("Health status saved")

	return nil
}

// ListForUser retrieves MCP servers filtered by accessible server IDs
// If accessibleServerIDs is nil, returns all servers (admin bypass)
// If accessibleServerIDs is empty slice, returns no servers
// Otherwise, filters servers by the provided IDs
func (r *ServerRepository) ListForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error) {
	query := `
		SELECT
			id, name, description, url, protocol_version, transport,
			auth_type, auth_config, health_check_url, health_check_interval,
			timeout_seconds, max_connections, is_active, tags, allowed_tools, metadata,
			created_at, updated_at
		FROM mcp_servers
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argPos := 1

	// If accessibleServerIDs is not nil, filter by those IDs
	// nil means admin access (all servers)
	// empty slice means no access (no servers)
	if accessibleServerIDs != nil {
		if len(accessibleServerIDs) == 0 {
			// No accessible servers - return empty result
			r.logger.Debug().Msg("No accessible servers for user")
			return []*domain.MCPServer{}, nil
		}
		query += fmt.Sprintf(" AND id = ANY($%d)", argPos)
		args = append(args, accessibleServerIDs)
		argPos++
	}

	// Apply additional filters
	if filter != nil {
		if filter.Name != "" {
			query += fmt.Sprintf(" AND name ILIKE $%d", argPos)
			args = append(args, "%"+filter.Name+"%")
			argPos++
		}
		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argPos)
			args = append(args, *filter.IsActive)
			argPos++
		}
		if len(filter.Tags) > 0 {
			query += fmt.Sprintf(" AND tags && $%d", argPos)
			args = append(args, filter.Tags)
			argPos++
		}
	}

	// Default ordering
	query += " ORDER BY created_at DESC"

	// Apply limit and offset
	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list servers for user")
		return nil, fmt.Errorf("failed to list servers for user: %w", err)
	}
	defer rows.Close()

	var servers []*domain.MCPServer
	for rows.Next() {
		var s domain.MCPServer
		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.URL, &s.ProtocolVersion, &s.Transport,
			&s.AuthType, &s.AuthConfig, &s.HealthCheckURL, &s.HealthCheckInterval,
			&s.TimeoutSeconds, &s.MaxConnections, &s.IsActive, &s.Tags, &s.AllowedTools, &s.Metadata,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan server row")
			continue
		}
		servers = append(servers, &s)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating server rows")
		return nil, fmt.Errorf("error iterating servers: %w", err)
	}

	r.logger.Debug().
		Int("count", len(servers)).
		Bool("filtered", accessibleServerIDs != nil).
		Msg("Servers listed for user")
	return servers, nil
}

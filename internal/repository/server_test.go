package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

func TestServerRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("successfully creates server", func(t *testing.T) {
		req := &domain.ServerCreate{
			Name:                "Test Server",
			Description:         "A test server",
			URL:                 "https://example.com/mcp",
			ProtocolVersion:     "1.0.0",
			Transport:           domain.TransportHTTP,
			AuthType:            domain.ServerAuthNone,
			HealthCheckInterval: 60,
			TimeoutSeconds:      30,
			MaxConnections:      100,
			Tags:                []string{"test"},
		}

		now := time.Now()
		serverID := "server-123"

		mock.ExpectQuery("INSERT INTO mcp_servers").
			WithArgs(
				req.Name, req.Description, req.URL, req.ProtocolVersion, req.Transport,
				req.AuthType, req.AuthConfig, req.HealthCheckURL, req.HealthCheckInterval,
				req.TimeoutSeconds, req.MaxConnections, true, req.Tags, req.AllowedTools, req.Metadata,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(serverID, now, now))

		server, err := repo.Create(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Equal(t, serverID, server.ID)
		assert.Equal(t, req.Name, server.Name)
		assert.Equal(t, req.URL, server.URL)
		assert.True(t, server.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("defaults transport to HTTP when empty", func(t *testing.T) {
		req := &domain.ServerCreate{
			Name: "Test Server",
			URL:  "https://example.com/mcp",
			// Transport is empty, should default to HTTP
		}

		now := time.Now()

		mock.ExpectQuery("INSERT INTO mcp_servers").
			WithArgs(
				req.Name, req.Description, req.URL, req.ProtocolVersion, domain.TransportHTTP,
				req.AuthType, req.AuthConfig, req.HealthCheckURL, req.HealthCheckInterval,
				req.TimeoutSeconds, req.MaxConnections, true, req.Tags, req.AllowedTools, req.Metadata,
			).
			WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow("server-456", now, now))

		server, err := repo.Create(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, domain.TransportHTTP, server.Transport)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		req := &domain.ServerCreate{
			Name: "Test Server",
			URL:  "https://example.com/mcp",
		}

		mock.ExpectQuery("INSERT INTO mcp_servers").
			WithArgs(
				req.Name, req.Description, req.URL, req.ProtocolVersion, domain.TransportHTTP,
				req.AuthType, req.AuthConfig, req.HealthCheckURL, req.HealthCheckInterval,
				req.TimeoutSeconds, req.MaxConnections, true, req.Tags, req.AllowedTools, req.Metadata,
			).
			WillReturnError(errors.New("database error"))

		server, err := repo.Create(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "failed to create server")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets server by ID", func(t *testing.T) {
		serverID := "server-123"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).AddRow(
				serverID, "Test Server", "Description", "https://example.com", "1.0.0", domain.TransportHTTP,
				domain.ServerAuthNone, nil, "", 60,
				30, 100, true, []string{"test"}, nil, nil,
				now, now,
			))

		server, err := repo.Get(context.Background(), serverID)

		require.NoError(t, err)
		require.NotNil(t, server)
		assert.Equal(t, serverID, server.ID)
		assert.Equal(t, "Test Server", server.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrServerNotFound when server does not exist", func(t *testing.T) {
		serverID := "nonexistent"

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			})) // Empty result

		server, err := repo.Get(context.Background(), serverID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrServerNotFound)
		assert.Nil(t, server)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnError(errors.New("connection failed"))

		server, err := repo.Get(context.Background(), serverID)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "failed to get server")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("lists all servers without filter", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 ORDER BY created_at DESC").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Server 1", "", "https://s1.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now).
				AddRow("server-2", "Server 2", "", "https://s2.example.com", "1.0.0", domain.TransportSSE,
					domain.ServerAuthBearer, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.List(context.Background(), nil)

		require.NoError(t, err)
		assert.Len(t, servers, 2)
		assert.Equal(t, "Server 1", servers[0].Name)
		assert.Equal(t, "Server 2", servers[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lists servers with name filter", func(t *testing.T) {
		now := time.Now()
		filter := &domain.ServerFilter{Name: "Test"}

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 AND name ILIKE \\$1 ORDER BY created_at DESC").
			WithArgs("%Test%").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Test Server", "", "https://test.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.List(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, "Test Server", servers[0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lists servers with is_active filter", func(t *testing.T) {
		now := time.Now()
		isActive := true
		filter := &domain.ServerFilter{IsActive: &isActive}

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 AND is_active = \\$1 ORDER BY created_at DESC").
			WithArgs(true).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Active Server", "", "https://active.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.List(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.True(t, servers[0].IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lists servers with pagination", func(t *testing.T) {
		now := time.Now()
		filter := &domain.ServerFilter{Limit: 10, Offset: 5}

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 5).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-6", "Server 6", "", "https://s6.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.List(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no servers found", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 ORDER BY created_at DESC").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}))

		servers, err := repo.List(context.Background(), nil)

		require.NoError(t, err)
		assert.Empty(t, servers)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 ORDER BY created_at DESC").
			WillReturnError(errors.New("query failed"))

		servers, err := repo.List(context.Background(), nil)

		assert.Error(t, err)
		assert.Nil(t, servers)
		assert.Contains(t, err.Error(), "failed to list servers")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("successfully deletes server", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectExec("DELETE FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), serverID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrServerNotFound when server does not exist", func(t *testing.T) {
		serverID := "nonexistent"

		mock.ExpectExec("DELETE FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), serverID)

		assert.ErrorIs(t, err, domain.ErrServerNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectExec("DELETE FROM mcp_servers WHERE id = \\$1").
			WithArgs(serverID).
			WillReturnError(errors.New("delete failed"))

		err := repo.Delete(context.Background(), serverID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete server")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_GetHealthStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("returns existing health status", func(t *testing.T) {
		serverID := "server-123"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM server_health WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "server_id", "status", "response_time_ms", "error_message", "checked_at",
			}).AddRow("health-1", serverID, domain.ServerStatusHealthy, 50, "", now))

		health, err := repo.GetHealthStatus(context.Background(), serverID)

		require.NoError(t, err)
		require.NotNil(t, health)
		assert.Equal(t, domain.ServerStatusHealthy, health.Status)
		assert.Equal(t, 50, health.ResponseTimeMs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns unknown status when no health record exists", func(t *testing.T) {
		serverID := "server-no-health"

		mock.ExpectQuery("SELECT .+ FROM server_health WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "server_id", "status", "response_time_ms", "error_message", "checked_at",
			})) // Empty result

		health, err := repo.GetHealthStatus(context.Background(), serverID)

		require.NoError(t, err)
		require.NotNil(t, health)
		assert.Equal(t, domain.ServerStatusUnknown, health.Status)
		assert.Equal(t, serverID, health.ServerID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectQuery("SELECT .+ FROM server_health WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnError(errors.New("query failed"))

		health, err := repo.GetHealthStatus(context.Background(), serverID)

		assert.Error(t, err)
		assert.Nil(t, health)
		assert.Contains(t, err.Error(), "failed to get health status")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_SaveHealthStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("successfully saves health status", func(t *testing.T) {
		health := &domain.ServerHealth{
			ServerID:       "server-123",
			Status:         domain.ServerStatusHealthy,
			ResponseTimeMs: 100,
			ErrorMessage:   "",
			CheckedAt:      time.Now(),
		}

		mock.ExpectQuery("INSERT INTO server_health").
			WithArgs(health.ServerID, health.Status, health.ResponseTimeMs, health.ErrorMessage, health.CheckedAt).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("health-new"))

		err := repo.SaveHealthStatus(context.Background(), health)

		require.NoError(t, err)
		assert.Equal(t, "health-new", health.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("saves unhealthy status with error message", func(t *testing.T) {
		health := &domain.ServerHealth{
			ServerID:       "server-456",
			Status:         domain.ServerStatusUnhealthy,
			ResponseTimeMs: 5000,
			ErrorMessage:   "Connection timeout",
			CheckedAt:      time.Now(),
		}

		mock.ExpectQuery("INSERT INTO server_health").
			WithArgs(health.ServerID, health.Status, health.ResponseTimeMs, health.ErrorMessage, health.CheckedAt).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("health-err"))

		err := repo.SaveHealthStatus(context.Background(), health)

		require.NoError(t, err)
		assert.Equal(t, "health-err", health.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		health := &domain.ServerHealth{
			ServerID:  "server-123",
			Status:    domain.ServerStatusHealthy,
			CheckedAt: time.Now(),
		}

		mock.ExpectQuery("INSERT INTO server_health").
			WithArgs(health.ServerID, health.Status, health.ResponseTimeMs, health.ErrorMessage, health.CheckedAt).
			WillReturnError(errors.New("insert failed"))

		err := repo.SaveHealthStatus(context.Background(), health)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save health status")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServerRepository_ListForUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewServerRepository(mock, logger.NewNopLogger())

	t.Run("returns all servers when accessibleServerIDs is nil (admin)", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 ORDER BY created_at DESC").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Server 1", "", "https://s1.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now).
				AddRow("server-2", "Server 2", "", "https://s2.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.ListForUser(context.Background(), nil, nil)

		require.NoError(t, err)
		assert.Len(t, servers, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty when accessibleServerIDs is empty slice", func(t *testing.T) {
		servers, err := repo.ListForUser(context.Background(), nil, []string{})

		require.NoError(t, err)
		assert.Empty(t, servers)
		// No database call expected
	})

	t.Run("filters by accessible server IDs", func(t *testing.T) {
		now := time.Now()
		accessibleIDs := []string{"server-1", "server-3"}

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 AND id = ANY\\(\\$1\\) ORDER BY created_at DESC").
			WithArgs(accessibleIDs).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Server 1", "", "https://s1.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now).
				AddRow("server-3", "Server 3", "", "https://s3.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.ListForUser(context.Background(), nil, accessibleIDs)

		require.NoError(t, err)
		assert.Len(t, servers, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("combines accessible IDs with other filters", func(t *testing.T) {
		now := time.Now()
		accessibleIDs := []string{"server-1", "server-2", "server-3"}
		filter := &domain.ServerFilter{Name: "Test", Limit: 10}

		mock.ExpectQuery("SELECT .+ FROM mcp_servers WHERE 1=1 AND id = ANY\\(\\$1\\) AND name ILIKE \\$2 ORDER BY created_at DESC LIMIT \\$3").
			WithArgs(accessibleIDs, "%Test%", 10).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "url", "protocol_version", "transport",
				"auth_type", "auth_config", "health_check_url", "health_check_interval",
				"timeout_seconds", "max_connections", "is_active", "tags", "allowed_tools", "metadata",
				"created_at", "updated_at",
			}).
				AddRow("server-1", "Test Server", "", "https://test.example.com", "1.0.0", domain.TransportHTTP,
					domain.ServerAuthNone, nil, "", 60, 30, 100, true, nil, nil, nil, now, now))

		servers, err := repo.ListForUser(context.Background(), filter, accessibleIDs)

		require.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewServerRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	log := logger.NewNopLogger()
	repo := NewServerRepository(mock, log)

	require.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

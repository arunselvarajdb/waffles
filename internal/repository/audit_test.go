package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/domain"
)

// Note: These tests require a running PostgreSQL database
// Set TEST_DATABASE_URL environment variable to run these tests
// Example: export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/mcp_gateway_test?sslmode=disable"

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	connStr := "postgres://postgres:postgres@localhost:5432/mcp_gateway?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("Could not ping test database: %v", err)
	}

	// Clean up audit logs before each test
	_, err = pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)

	return pool
}

// createTestServer creates a test MCP server and returns its ID for use in audit log tests
func createTestServer(t *testing.T, pool *pgxpool.Pool) string {
	var serverID string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO mcp_servers (name, url, is_active)
		VALUES ($1, $2, true)
		ON CONFLICT (name) DO UPDATE SET url = $2
		RETURNING id
	`, "test-audit-server-"+t.Name(), "http://localhost:8000").Scan(&serverID)
	require.NoError(t, err)
	return serverID
}

// cleanupTestServer removes the test server created for audit tests
func cleanupTestServer(t *testing.T, pool *pgxpool.Pool, serverID string) {
	_, err := pool.Exec(context.Background(), "DELETE FROM mcp_servers WHERE id = $1", serverID)
	require.NoError(t, err)
}

func TestAuditRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewAuditRepository(pool)
	ctx := context.Background()

	// Create a test server for the server ID test case
	testServerID := createTestServer(t, pool)
	defer cleanupTestServer(t, pool, testServerID)

	tests := []struct {
		name    string
		log     *domain.AuditLog
		wantErr bool
	}{
		{
			name: "create minimal audit log",
			log: &domain.AuditLog{
				RequestID: "req-123",
				Method:    "GET",
				Path:      "/api/v1/servers",
				IPAddress: "127.0.0.1",
				UserAgent: "curl/7.68.0",
			},
			wantErr: false,
		},
		{
			name: "create full audit log",
			log: &domain.AuditLog{
				RequestID:      "req-456",
				Method:         "POST",
				Path:           "/api/v1/gateway/server-123/tools/call",
				QueryParams:    json.RawMessage(`{"filter":"calc"}`),
				RequestBody:    json.RawMessage(`{"name":"calculator","args":{"a":1,"b":2}}`),
				ResponseStatus: intPtr(200),
				ResponseBody:   json.RawMessage(`{"result":3}`),
				LatencyMS:      intPtr(45),
				IPAddress:      "192.168.1.1",
				UserAgent:      "Mozilla/5.0",
			},
			wantErr: false,
		},
		{
			name: "create with server ID",
			log: &domain.AuditLog{
				RequestID: "req-789",
				ServerID:  &testServerID, // Use a valid server ID from the database
				Method:    "POST",
				Path:      "/api/v1/gateway/" + testServerID + "/tools/list",
				IPAddress: "127.0.0.1",
				UserAgent: "test-client",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.log)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.log.ID, "ID should be generated")
				assert.False(t, tt.log.CreatedAt.IsZero(), "CreatedAt should be set")
			}
		})
	}
}

func TestAuditRepository_Get(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewAuditRepository(pool)
	ctx := context.Background()

	// Create a test audit log
	testLog := &domain.AuditLog{
		RequestID:      "req-get-test",
		Method:         "GET",
		Path:           "/test/path",
		ResponseStatus: intPtr(200),
		LatencyMS:      intPtr(100),
		IPAddress:      "127.0.0.1",
		UserAgent:      "test-agent",
	}
	err := repo.Create(ctx, testLog)
	require.NoError(t, err)
	require.NotEmpty(t, testLog.ID)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "get existing log",
			id:      testLog.ID,
			wantErr: false,
		},
		{
			name:    "get non-existent log",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := repo.Get(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, log)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, log)
				assert.Equal(t, tt.id, log.ID)
				assert.Equal(t, "req-get-test", log.RequestID)
				assert.Equal(t, "GET", log.Method)
				assert.Equal(t, "/test/path", log.Path)
			}
		})
	}
}

func TestAuditRepository_List(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewAuditRepository(pool)
	ctx := context.Background()

	// Create a test server for the server ID test case
	serverID := createTestServer(t, pool)
	defer cleanupTestServer(t, pool, serverID)

	// Create multiple test logs
	logs := []*domain.AuditLog{
		{
			RequestID: "req-1",
			ServerID:  &serverID,
			Method:    "GET",
			Path:      "/api/v1/servers",
			IPAddress: "127.0.0.1",
			UserAgent: "curl",
		},
		{
			RequestID: "req-2",
			ServerID:  &serverID,
			Method:    "POST",
			Path:      "/api/v1/gateway/" + serverID + "/tools/call",
			IPAddress: "127.0.0.1",
			UserAgent: "curl",
		},
		{
			RequestID: "req-3",
			Method:    "GET",
			Path:      "/api/v1/servers",
			IPAddress: "192.168.1.1",
			UserAgent: "browser",
		},
	}

	for _, log := range logs {
		err := repo.Create(ctx, log)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name      string
		filter    domain.AuditLogFilter
		wantCount int
		checkFunc func(*testing.T, []*domain.AuditLog)
	}{
		{
			name:      "list all logs",
			filter:    domain.AuditLogFilter{Limit: 10},
			wantCount: 3,
		},
		{
			name: "filter by server ID",
			filter: domain.AuditLogFilter{
				ServerID: &serverID,
				Limit:    10,
			},
			wantCount: 2,
			checkFunc: func(t *testing.T, logs []*domain.AuditLog) {
				for _, log := range logs {
					assert.NotNil(t, log.ServerID)
					assert.Equal(t, serverID, *log.ServerID)
				}
			},
		},
		{
			name: "filter by method",
			filter: domain.AuditLogFilter{
				Method: stringPtr("POST"),
				Limit:  10,
			},
			wantCount: 1,
			checkFunc: func(t *testing.T, logs []*domain.AuditLog) {
				assert.Equal(t, "POST", logs[0].Method)
			},
		},
		// Note: PathPrefix filtering would need to be added to domain.AuditLogFilter
		// Skipping this test case for now
		{
			name: "limit results",
			filter: domain.AuditLogFilter{
				Limit: 2,
			},
			wantCount: 2,
		},
		{
			name: "offset results",
			filter: domain.AuditLogFilter{
				Offset: 2,
				Limit:  10,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.List(ctx, tt.filter)

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantCount)

			if tt.checkFunc != nil {
				tt.checkFunc(t, results)
			}
		})
	}
}

func TestAuditRepository_List_EmptyDatabase(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewAuditRepository(pool)
	ctx := context.Background()

	results, err := repo.List(ctx, domain.AuditLogFilter{Limit: 10})

	assert.NoError(t, err)
	assert.Empty(t, results)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

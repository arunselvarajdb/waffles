package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerAuthType_Constants(t *testing.T) {
	assert.Equal(t, ServerAuthType("none"), ServerAuthNone)
	assert.Equal(t, ServerAuthType("basic"), ServerAuthBasic)
	assert.Equal(t, ServerAuthType("bearer"), ServerAuthBearer)
	assert.Equal(t, ServerAuthType("oauth"), ServerAuthOAuth)
}

func TestServerStatus_Constants(t *testing.T) {
	assert.Equal(t, ServerStatus("healthy"), ServerStatusHealthy)
	assert.Equal(t, ServerStatus("degraded"), ServerStatusDegraded)
	assert.Equal(t, ServerStatus("unhealthy"), ServerStatusUnhealthy)
	assert.Equal(t, ServerStatus("unknown"), ServerStatusUnknown)
}

func TestTransportType_Constants(t *testing.T) {
	assert.Equal(t, TransportType("http"), TransportHTTP)
	assert.Equal(t, TransportType("sse"), TransportSSE)
	assert.Equal(t, TransportType("streamable_http"), TransportStreamableHTTP)
}

func TestMCPServer_JSONMarshaling(t *testing.T) {
	server := &MCPServer{
		ID:                  "server-123",
		Name:                "Test Server",
		Description:         "A test MCP server",
		URL:                 "https://example.com/mcp",
		ProtocolVersion:     "2024-11-05",
		Transport:           TransportStreamableHTTP,
		AuthType:            ServerAuthBearer,
		HealthCheckInterval: 30,
		TimeoutSeconds:      60,
		MaxConnections:      10,
		IsActive:            true,
		Tags:                []string{"test", "example"},
		CreatedBy:           "user-456",
		CreatedAt:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(server)
	require.NoError(t, err)

	var parsed MCPServer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, server.ID, parsed.ID)
	assert.Equal(t, server.Name, parsed.Name)
	assert.Equal(t, server.URL, parsed.URL)
	assert.Equal(t, server.Transport, parsed.Transport)
	assert.Equal(t, server.AuthType, parsed.AuthType)
	assert.Equal(t, server.Tags, parsed.Tags)
}

func TestMCPServer_WithCurrentStatus(t *testing.T) {
	health := &ServerHealth{
		ID:             "health-123",
		ServerID:       "server-123",
		Status:         ServerStatusHealthy,
		ResponseTimeMs: 50,
		CheckedAt:      time.Now(),
	}

	server := &MCPServer{
		ID:            "server-123",
		Name:          "Test Server",
		CurrentStatus: health,
	}

	data, err := json.Marshal(server)
	require.NoError(t, err)

	var parsed MCPServer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.CurrentStatus)
	assert.Equal(t, ServerStatusHealthy, parsed.CurrentStatus.Status)
	assert.Equal(t, 50, parsed.CurrentStatus.ResponseTimeMs)
}

func TestServerCreate_JSONMarshaling(t *testing.T) {
	create := &ServerCreate{
		Name:                "New Server",
		Description:         "A new MCP server",
		URL:                 "https://example.com/mcp",
		ProtocolVersion:     "2024-11-05",
		Transport:           TransportHTTP,
		AuthType:            ServerAuthNone,
		HealthCheckInterval: 60,
		TimeoutSeconds:      30,
		MaxConnections:      5,
		Tags:                []string{"production"},
	}

	data, err := json.Marshal(create)
	require.NoError(t, err)

	var parsed ServerCreate
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, create.Name, parsed.Name)
	assert.Equal(t, create.URL, parsed.URL)
	assert.Equal(t, create.Transport, parsed.Transport)
}

func TestServerUpdate_OptionalFields(t *testing.T) {
	name := "Updated Name"
	timeout := 120

	update := &ServerUpdate{
		Name:           &name,
		TimeoutSeconds: &timeout,
	}

	data, err := json.Marshal(update)
	require.NoError(t, err)

	var parsed ServerUpdate
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.Name)
	assert.Equal(t, name, *parsed.Name)

	require.NotNil(t, parsed.TimeoutSeconds)
	assert.Equal(t, timeout, *parsed.TimeoutSeconds)

	// Fields not set should be nil
	assert.Nil(t, parsed.URL)
	assert.Nil(t, parsed.Description)
	assert.Nil(t, parsed.IsActive)
}

func TestServerHealth_JSONMarshaling(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	health := &ServerHealth{
		ID:             "health-123",
		ServerID:       "server-456",
		Status:         ServerStatusDegraded,
		ResponseTimeMs: 1500,
		ErrorMessage:   "High latency detected",
		CheckedAt:      now,
	}

	data, err := json.Marshal(health)
	require.NoError(t, err)

	var parsed ServerHealth
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, health.ID, parsed.ID)
	assert.Equal(t, health.ServerID, parsed.ServerID)
	assert.Equal(t, health.Status, parsed.Status)
	assert.Equal(t, health.ResponseTimeMs, parsed.ResponseTimeMs)
	assert.Equal(t, health.ErrorMessage, parsed.ErrorMessage)
}

func TestServerFilter_Defaults(t *testing.T) {
	filter := &ServerFilter{}

	assert.Equal(t, "", filter.Name)
	assert.Nil(t, filter.IsActive)
	assert.Nil(t, filter.Tags)
	assert.Equal(t, 0, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestServerFilter_WithValues(t *testing.T) {
	isActive := true
	filter := &ServerFilter{
		Name:     "test",
		IsActive: &isActive,
		Tags:     []string{"prod", "api"},
		Limit:    10,
		Offset:   20,
	}

	assert.Equal(t, "test", filter.Name)
	require.NotNil(t, filter.IsActive)
	assert.True(t, *filter.IsActive)
	assert.Equal(t, []string{"prod", "api"}, filter.Tags)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 20, filter.Offset)
}

func TestMCPServer_AuthConfig(t *testing.T) {
	authConfig := json.RawMessage(`{"token": "secret-token"}`)

	server := &MCPServer{
		ID:         "server-123",
		Name:       "Auth Server",
		AuthType:   ServerAuthBearer,
		AuthConfig: authConfig,
	}

	data, err := json.Marshal(server)
	require.NoError(t, err)

	var parsed MCPServer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Compare parsed JSON content, not raw bytes (JSON re-marshaling may change whitespace)
	var expected, actual map[string]string
	require.NoError(t, json.Unmarshal(authConfig, &expected))
	require.NoError(t, json.Unmarshal(parsed.AuthConfig, &actual))
	assert.Equal(t, expected, actual)
}

func TestMCPServer_Metadata(t *testing.T) {
	metadata := json.RawMessage(`{"version": "1.0", "owner": "team-a"}`)

	server := &MCPServer{
		ID:       "server-123",
		Name:     "Metadata Server",
		Metadata: metadata,
	}

	data, err := json.Marshal(server)
	require.NoError(t, err)

	var parsed MCPServer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Compare parsed JSON content, not raw bytes
	var expected, actual map[string]string
	require.NoError(t, json.Unmarshal(metadata, &expected))
	require.NoError(t, json.Unmarshal(parsed.Metadata, &actual))
	assert.Equal(t, expected, actual)
}

func TestMCPServer_AllowedTools(t *testing.T) {
	server := &MCPServer{
		ID:           "server-123",
		Name:         "Tools Server",
		AllowedTools: []string{"read_file", "write_file", "execute"},
	}

	data, err := json.Marshal(server)
	require.NoError(t, err)

	var parsed MCPServer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, []string{"read_file", "write_file", "execute"}, parsed.AllowedTools)
}

func TestServerCreate_AllowedTools(t *testing.T) {
	create := &ServerCreate{
		Name:         "Tools Server",
		URL:          "https://example.com/mcp",
		AllowedTools: []string{"search", "analyze"},
	}

	data, err := json.Marshal(create)
	require.NoError(t, err)

	var parsed ServerCreate
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, []string{"search", "analyze"}, parsed.AllowedTools)
}

func TestServerUpdate_AllowedTools(t *testing.T) {
	tools := []string{"new_tool"}
	update := &ServerUpdate{
		AllowedTools: &tools,
	}

	data, err := json.Marshal(update)
	require.NoError(t, err)

	var parsed ServerUpdate
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.NotNil(t, parsed.AllowedTools)
	assert.Equal(t, tools, *parsed.AllowedTools)
}

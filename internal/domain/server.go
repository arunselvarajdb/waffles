package domain

import (
	"encoding/json"
	"time"
)

// ServerAuthType represents the authentication type for MCP server
type ServerAuthType string

const (
	ServerAuthNone   ServerAuthType = "none"
	ServerAuthBasic  ServerAuthType = "basic"
	ServerAuthBearer ServerAuthType = "bearer"
	ServerAuthOAuth  ServerAuthType = "oauth"
)

// ServerStatus represents the health status of a server
type ServerStatus string

const (
	ServerStatusHealthy   ServerStatus = "healthy"
	ServerStatusDegraded  ServerStatus = "degraded"
	ServerStatusUnhealthy ServerStatus = "unhealthy"
	ServerStatusUnknown   ServerStatus = "unknown"
)

// TransportType represents the MCP transport protocol
type TransportType string

const (
	TransportHTTP           TransportType = "http"            // REST-style HTTP endpoints (legacy)
	TransportSSE            TransportType = "sse"             // Server-Sent Events with JSON-RPC (legacy, deprecated)
	TransportStreamableHTTP TransportType = "streamable_http" // Streamable HTTP (MCP 2025-11-25)
)

// MCPServer represents a registered MCP server
type MCPServer struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	URL                 string          `json:"url"`
	ProtocolVersion     string          `json:"protocol_version"`
	Transport           TransportType   `json:"transport"` // http or sse
	AuthType            ServerAuthType  `json:"auth_type"`
	AuthConfig          json.RawMessage `json:"auth_config,omitempty"` // Encrypted credentials
	HealthCheckURL      string          `json:"health_check_url,omitempty"`
	HealthCheckInterval int             `json:"health_check_interval"` // seconds
	TimeoutSeconds      int             `json:"timeout_seconds"`
	MaxConnections      int             `json:"max_connections"`
	IsActive            bool            `json:"is_active"`
	Tags                []string        `json:"tags,omitempty"`
	AllowedTools        []string        `json:"allowed_tools,omitempty"` // List of tool names users can access (empty = all)
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	CreatedBy           string          `json:"created_by"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`

	// Populated from separate query if needed
	CurrentStatus *ServerHealth `json:"current_status,omitempty"`
}

// ServerCreate represents the data required to create a new MCP server
type ServerCreate struct {
	Name                string          `json:"name" validate:"required,min=3,max=255"`
	Description         string          `json:"description"`
	URL                 string          `json:"url" validate:"required,url"`
	ProtocolVersion     string          `json:"protocol_version,omitempty"`
	Transport           TransportType   `json:"transport,omitempty"` // http (default) or sse
	AuthType            ServerAuthType  `json:"auth_type,omitempty"`
	AuthConfig          json.RawMessage `json:"auth_config,omitempty"`
	HealthCheckURL      string          `json:"health_check_url,omitempty"`
	HealthCheckInterval int             `json:"health_check_interval,omitempty" validate:"omitempty,min=10"`
	TimeoutSeconds      int             `json:"timeout_seconds,omitempty" validate:"omitempty,min=1,max=300"`
	MaxConnections      int             `json:"max_connections,omitempty" validate:"omitempty,min=1"`
	Tags                []string        `json:"tags,omitempty"`
	AllowedTools        []string        `json:"allowed_tools,omitempty"` // List of tool names users can access (empty = all)
	Metadata            json.RawMessage `json:"metadata,omitempty"`
}

// ServerUpdate represents the data that can be updated for an MCP server
type ServerUpdate struct {
	Name                *string         `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Description         *string         `json:"description,omitempty"`
	URL                 *string         `json:"url,omitempty" validate:"omitempty,url"`
	ProtocolVersion     *string         `json:"protocol_version,omitempty"`
	AuthType            *ServerAuthType `json:"auth_type,omitempty"`
	AuthConfig          json.RawMessage `json:"auth_config,omitempty"`
	HealthCheckURL      *string         `json:"health_check_url,omitempty"`
	HealthCheckInterval *int            `json:"health_check_interval,omitempty" validate:"omitempty,min=10"`
	TimeoutSeconds      *int            `json:"timeout_seconds,omitempty" validate:"omitempty,min=1,max=300"`
	MaxConnections      *int            `json:"max_connections,omitempty" validate:"omitempty,min=1"`
	IsActive            *bool           `json:"is_active,omitempty"`
	Tags                *[]string       `json:"tags,omitempty"`
	AllowedTools        *[]string       `json:"allowed_tools,omitempty"` // List of tool names users can access (empty = all)
	Metadata            json.RawMessage `json:"metadata,omitempty"`
}

// ServerHealth represents the health check result for a server
type ServerHealth struct {
	ID             string       `json:"id"`
	ServerID       string       `json:"server_id"`
	Status         ServerStatus `json:"status"`
	ResponseTimeMs int          `json:"response_time_ms,omitempty"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	CheckedAt      time.Time    `json:"checked_at"`
}

// ServerFilter represents query filters for listing servers
type ServerFilter struct {
	Name     string
	IsActive *bool
	Tags     []string
	Limit    int
	Offset   int
}

package domain

import (
	"net"
	"strings"
	"time"
)

// APIKeyScope represents a permission scope for API keys
type APIKeyScope string

const (
	// Viewer+ scopes (viewer, operator, admin)
	ScopeServersRead APIKeyScope = "servers:read"

	// Operator+ scopes (operator, admin)
	ScopeServersWrite   APIKeyScope = "servers:write"
	ScopeGatewayExec    APIKeyScope = "gateway:execute"
	ScopeAuditRead      APIKeyScope = "audit:read"
	ScopeNamespacesRead APIKeyScope = "namespaces:read"

	// Admin-only scopes
	ScopeNamespacesWrite APIKeyScope = "namespaces:write"
	ScopeUsersRead       APIKeyScope = "users:read"
	ScopeUsersWrite      APIKeyScope = "users:write"
	ScopeRolesRead       APIKeyScope = "roles:read"
	ScopeRolesWrite      APIKeyScope = "roles:write"
)

// ValidScopes returns all valid API key scopes
func ValidScopes() []APIKeyScope {
	return []APIKeyScope{
		ScopeServersRead,
		ScopeServersWrite,
		ScopeGatewayExec,
		ScopeAuditRead,
		ScopeNamespacesRead,
		ScopeNamespacesWrite,
		ScopeUsersRead,
		ScopeUsersWrite,
		ScopeRolesRead,
		ScopeRolesWrite,
	}
}

// IsValidScope checks if a scope string is valid
func IsValidScope(scope string) bool {
	for _, s := range ValidScopes() {
		if string(s) == scope {
			return true
		}
	}
	return false
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyHash     string     `json:"-"`      // Never expose key hash
	Prefix      string     `json:"prefix"` // First 8 chars for identification
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Scope restrictions
	Scopes         []string `json:"scopes"`          // Permission scopes
	AllowedServers []string `json:"allowed_servers"` // Server UUIDs (empty = all)
	AllowedTools   []string `json:"allowed_tools"`   // Tool names (empty = all)
	Namespaces     []string `json:"namespaces"`      // Namespace UUIDs (empty = all)
	IPWhitelist    []string `json:"ip_whitelist"`    // CIDR ranges (empty = any)
	ReadOnly       bool     `json:"read_only"`       // Only allow read operations
}

// HasScope checks if the API key has a specific scope
func (k *APIKey) HasScope(scope string) bool {
	// Empty scopes means all scopes (for backward compatibility)
	if len(k.Scopes) == 0 {
		return true
	}
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the API key has any of the specified scopes
func (k *APIKey) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if k.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsServerAllowed checks if the API key can access a specific server
func (k *APIKey) IsServerAllowed(serverID string) bool {
	// Empty means all servers allowed
	if len(k.AllowedServers) == 0 {
		return true
	}
	for _, s := range k.AllowedServers {
		if s == serverID {
			return true
		}
	}
	return false
}

// IsToolAllowed checks if the API key can execute a specific tool
func (k *APIKey) IsToolAllowed(toolName string) bool {
	// Empty means all tools allowed
	if len(k.AllowedTools) == 0 {
		return true
	}
	for _, t := range k.AllowedTools {
		if t == toolName {
			return true
		}
	}
	return false
}

// IsNamespaceAllowed checks if the API key can access a specific namespace
func (k *APIKey) IsNamespaceAllowed(namespaceID string) bool {
	// Empty means all namespaces allowed
	if len(k.Namespaces) == 0 {
		return true
	}
	for _, n := range k.Namespaces {
		if n == namespaceID {
			return true
		}
	}
	return false
}

// IsIPAllowed checks if the client IP is allowed to use this API key
func (k *APIKey) IsIPAllowed(clientIP string) bool {
	// Empty whitelist means any IP is allowed
	if len(k.IPWhitelist) == 0 {
		return true
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidr := range k.IPWhitelist {
		// Handle both single IPs and CIDR ranges
		if !strings.Contains(cidr, "/") {
			if cidr == clientIP {
				return true
			}
			continue
		}

		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// APIKeyCreate represents the data required to create a new API key
type APIKeyCreate struct {
	Name           string     `json:"name" validate:"required,min=3,max=255"`
	Description    string     `json:"description,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Scopes         []string   `json:"scopes,omitempty"`
	AllowedServers []string   `json:"allowed_servers,omitempty"`
	AllowedTools   []string   `json:"allowed_tools,omitempty"`
	Namespaces     []string   `json:"namespaces,omitempty"`
	IPWhitelist    []string   `json:"ip_whitelist,omitempty"`
	ReadOnly       bool       `json:"read_only,omitempty"`
}

// APIKeyResponse includes the plain-text key (only shown once)
type APIKeyResponse struct {
	APIKey
	Key string `json:"key"` // Plain-text key, only returned on creation
}

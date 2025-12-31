package handler

import (
	"context"
	"encoding/json"
	"net/http/httputil"
	"time"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/internal/service/oauth"
	"github.com/waffles/waffles/internal/service/registry"
)

// RegistryServiceInterface defines the interface for registry service operations.
type RegistryServiceInterface interface {
	CreateServer(ctx context.Context, req *domain.ServerCreate) (*domain.MCPServer, error)
	ListServersForUser(ctx context.Context, filter *domain.ServerFilter, accessibleServerIDs []string) ([]*domain.MCPServer, error)
	GetServer(ctx context.Context, id string) (*domain.MCPServer, error)
	UpdateServer(ctx context.Context, id string, req *domain.ServerUpdate) (*domain.MCPServer, error)
	DeleteServer(ctx context.Context, id string) error
	ToggleServer(ctx context.Context, id string, enabled bool) (*domain.MCPServer, error)
	GetHealthStatus(ctx context.Context, serverID string) (*domain.ServerHealth, error)
	CheckHealth(ctx context.Context, serverID string) error
	TestConnection(ctx context.Context, req *registry.TestConnectionRequest) (*registry.TestConnectionResult, error)
	CallTool(ctx context.Context, req *registry.CallToolRequest) (*registry.CallToolResult, error)
}

// ServerAccessServiceInterface defines the interface for server access operations.
type ServerAccessServiceInterface interface {
	GetAccessibleServerIDs(ctx context.Context, roles []string, level domain.AccessLevel) ([]string, error)
	CanAccessServer(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error)
}

// CreateAPIKeyInput contains the parameters for creating a new API key
type CreateAPIKeyInput struct {
	UserID         string
	Name           string
	Description    string
	ExpiresAt      *time.Time
	Scopes         []string
	AllowedServers []string
	AllowedTools   []string
	Namespaces     []string
	IPWhitelist    []string
	ReadOnly       bool
}

// APIKeyRepositoryInterface defines the interface for API key repository operations.
type APIKeyRepositoryInterface interface {
	Create(ctx context.Context, input *CreateAPIKeyInput) (*APIKey, string, error)
	GetByID(ctx context.Context, keyID string) (*APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]*APIKey, error)
	ListAll(ctx context.Context) ([]*APIKey, error)
	Delete(ctx context.Context, keyID, userID string) error
	AdminDelete(ctx context.Context, keyID string) error
	UpdateLastUsed(ctx context.Context, keyID string) error
}

// APIKey represents an API key for use in handler interfaces.
type APIKey struct {
	CreatedAt      time.Time
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	ID             string
	UserID         string
	Name           string
	Description    string
	KeyPrefix      string
	Scopes         []string
	AllowedServers []string
	AllowedTools   []string
	Namespaces     []string
	IPWhitelist    []string
	ReadOnly       bool
}

// UserRepositoryInterface defines the interface for user repository operations.
type UserRepositoryInterface interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
}

// NamespaceRepositoryInterface defines the interface for namespace repository operations.
type NamespaceRepositoryInterface interface {
	Create(ctx context.Context, ns *domain.Namespace) error
	GetByID(ctx context.Context, id string) (*domain.Namespace, error)
	List(ctx context.Context) ([]*domain.Namespace, error)
	Update(ctx context.Context, ns *domain.Namespace) error
	Delete(ctx context.Context, id string) error
	AddServer(ctx context.Context, namespaceID, serverID string) error
	RemoveServer(ctx context.Context, namespaceID, serverID string) error
	ListServers(ctx context.Context, namespaceID string) ([]*domain.MCPServer, error)
	SetRoleAccess(ctx context.Context, namespaceID, roleID string, accessLevel domain.AccessLevel) error
	RemoveRoleAccess(ctx context.Context, namespaceID, roleID string) error
	ListRoleAccess(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error)
}

// GatewayServiceInterface defines the interface for gateway service operations.
type GatewayServiceInterface interface {
	ProxyToServer(ctx context.Context, serverID string) (*httputil.ReverseProxy, *domain.MCPServer, error)
	GetServerInfo(ctx context.Context, serverID string) (*domain.MCPServer, error)
	Initialize(ctx context.Context, serverID string) (*domain.MCPServer, error)
	GetTransportType(ctx context.Context, serverID string) (domain.TransportType, *domain.MCPServer, error)
	CallSSE(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error)
	CallStreamableHTTP(ctx context.Context, serverID string, method string, params interface{}) (json.RawMessage, error)
	InitializeStreamableHTTP(ctx context.Context, serverID string) (*MCPSession, error)
	TerminateStreamableHTTP(ctx context.Context, serverID string) error
}

// MCPSession represents an MCP session (from gateway package).
type MCPSession struct {
	SessionID       string
	ProtocolVersion string
}

// DatabaseHealthChecker defines the interface for database health checks.
type DatabaseHealthChecker interface {
	Health(ctx context.Context) DatabaseHealthStatus
}

// DatabaseHealthStatus represents the health status of the database.
type DatabaseHealthStatus struct {
	Message          string
	TotalConnections int32
	IdleConnections  int32
	MaxConnections   int32
	Healthy          bool
}

// NamespaceRepoInterface defines the interface for namespace repository operations.
type NamespaceRepoInterface interface {
	Create(ctx context.Context, req *domain.NamespaceCreate) (*domain.Namespace, error)
	Get(ctx context.Context, id string) (*domain.Namespace, error)
	List(ctx context.Context) ([]*domain.Namespace, error)
	Update(ctx context.Context, id string, req *domain.NamespaceUpdate) (*domain.Namespace, error)
	Delete(ctx context.Context, id string) error
	AddServerToNamespace(ctx context.Context, serverID, namespaceID string) error
	RemoveServerFromNamespace(ctx context.Context, serverID, namespaceID string) error
	GetNamespaceServers(ctx context.Context, namespaceID string) ([]*domain.NamespaceMember, error)
	SetRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string, level domain.AccessLevel) error
	RemoveRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string) error
	GetNamespaceRoleAccess(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error)
	GetRoleIDByName(ctx context.Context, roleName string) (string, error)
}

// OAuthServiceInterface defines the interface for OAuth service operations.
type OAuthServiceInterface interface {
	IsEnabled() bool
	GenerateState() (string, error)
	GetAuthURL(state string) (string, error)
	ExchangeCode(ctx context.Context, code string) (*oauth.UserInfo, error)
	AutoCreateUsers() bool
	GetDefaultRole() string
}

// OAuthUserRepoInterface defines the interface for OAuth user repository operations.
type OAuthUserRepoInterface interface {
	FindOrCreateOAuthUser(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	AssignRole(ctx context.Context, userID, role string) error
}

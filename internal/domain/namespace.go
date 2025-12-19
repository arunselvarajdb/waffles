package domain

import "time"

// AccessLevel represents the level of access to a namespace
type AccessLevel string

const (
	// AccessLevelView allows viewing servers in the namespace (list operations)
	AccessLevelView AccessLevel = "view"
	// AccessLevelExecute allows using servers via the gateway (includes view)
	AccessLevelExecute AccessLevel = "execute"
)

// IsValid checks if the access level is valid
func (a AccessLevel) IsValid() bool {
	return a == AccessLevelView || a == AccessLevelExecute
}

// Includes checks if this access level includes the other
// execute includes view, but view does not include execute
func (a AccessLevel) Includes(other AccessLevel) bool {
	if a == AccessLevelExecute {
		return true // execute includes both view and execute
	}
	return a == other // view only includes view
}

// Namespace represents a logical grouping of MCP servers
type Namespace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Computed fields (not stored in DB)
	ServerCount int `json:"server_count,omitempty"`
}

// NamespaceCreate represents data to create a namespace
type NamespaceCreate struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description,omitempty"`
}

// NamespaceUpdate represents data to update a namespace
type NamespaceUpdate struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description,omitempty"`
}

// RoleNamespaceAccess represents a role's access to a namespace
type RoleNamespaceAccess struct {
	RoleID        string      `json:"role_id"`
	RoleName      string      `json:"role_name"`
	NamespaceID   string      `json:"namespace_id"`
	NamespaceName string      `json:"namespace_name,omitempty"`
	AccessLevel   AccessLevel `json:"access_level"`
}

// NamespaceMember represents a server's membership in a namespace
type NamespaceMember struct {
	ServerID      string `json:"server_id"`
	ServerName    string `json:"server_name,omitempty"`
	NamespaceID   string `json:"namespace_id"`
	NamespaceName string `json:"namespace_name,omitempty"`
}

// SetRoleAccessRequest represents a request to set role access to a namespace
type SetRoleAccessRequest struct {
	RoleName    string      `json:"role_name" validate:"required"`
	AccessLevel AccessLevel `json:"access_level" validate:"required"`
}

// AddServerToNamespaceRequest represents a request to add a server to a namespace
type AddServerToNamespaceRequest struct {
	ServerID string `json:"server_id" validate:"required,uuid"`
}

// NamespaceFilter represents filter options for listing namespaces
type NamespaceFilter struct {
	Name   string
	Limit  int
	Offset int
}

// Legacy type aliases for backwards compatibility during migration
// These can be removed after all code is updated
type ServerGroup = Namespace
type ServerGroupCreate = NamespaceCreate
type ServerGroupUpdate = NamespaceUpdate
type RoleGroupAccess = RoleNamespaceAccess
type ServerGroupMember = NamespaceMember
type AddServerToGroupRequest = AddServerToNamespaceRequest
type ServerGroupFilter = NamespaceFilter

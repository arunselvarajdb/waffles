package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// NamespaceRepository handles database operations for namespaces
type NamespaceRepository struct {
	db     DBTX
	logger logger.Logger
}

// NewNamespaceRepository creates a new namespace repository
func NewNamespaceRepository(db DBTX, log logger.Logger) *NamespaceRepository {
	return &NamespaceRepository{
		db:     db,
		logger: log,
	}
}

// Create creates a new namespace
func (r *NamespaceRepository) Create(ctx context.Context, req *domain.NamespaceCreate) (*domain.Namespace, error) {
	query := `
		INSERT INTO namespaces (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`

	var ns domain.Namespace
	err := r.db.QueryRow(ctx, query, req.Name, req.Description).Scan(
		&ns.ID,
		&ns.Name,
		&ns.Description,
		&ns.CreatedAt,
		&ns.UpdatedAt,
	)
	if err != nil {
		r.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create namespace")
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	r.logger.Info().Str("namespace_id", ns.ID).Str("name", ns.Name).Msg("Namespace created")
	return &ns, nil
}

// getNamespaceBy is a helper that retrieves a namespace by a given column and value.
func (r *NamespaceRepository) getNamespaceBy(ctx context.Context, column, value, logField string) (*domain.Namespace, error) {
	query := fmt.Sprintf(`
		SELECT n.id, n.name, n.description, n.created_at, n.updated_at,
			   (SELECT COUNT(*) FROM namespace_members WHERE namespace_id = n.id) as server_count
		FROM namespaces n
		WHERE n.%s = $1
	`, column)

	var ns domain.Namespace
	err := r.db.QueryRow(ctx, query, value).Scan(
		&ns.ID,
		&ns.Name,
		&ns.Description,
		&ns.CreatedAt,
		&ns.UpdatedAt,
		&ns.ServerCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		r.logger.Error().Err(err).Str(logField, value).Msg("Failed to get namespace")

		return nil, fmt.Errorf("failed to get namespace by %s: %w", column, err)
	}

	return &ns, nil
}

// Get retrieves a namespace by ID.
func (r *NamespaceRepository) Get(ctx context.Context, id string) (*domain.Namespace, error) {
	return r.getNamespaceBy(ctx, "id", id, "id")
}

// GetByName retrieves a namespace by name.
func (r *NamespaceRepository) GetByName(ctx context.Context, name string) (*domain.Namespace, error) {
	return r.getNamespaceBy(ctx, "name", name, "name")
}

// List retrieves all namespaces
func (r *NamespaceRepository) List(ctx context.Context) ([]*domain.Namespace, error) {
	query := `
		SELECT n.id, n.name, n.description, n.created_at, n.updated_at,
			   (SELECT COUNT(*) FROM namespace_members WHERE namespace_id = n.id) as server_count
		FROM namespaces n
		ORDER BY n.name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list namespaces")
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	defer rows.Close()

	var namespaces []*domain.Namespace
	for rows.Next() {
		var ns domain.Namespace
		if err := rows.Scan(
			&ns.ID,
			&ns.Name,
			&ns.Description,
			&ns.CreatedAt,
			&ns.UpdatedAt,
			&ns.ServerCount,
		); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan namespace row")
			return nil, fmt.Errorf("failed to scan namespace: %w", err)
		}
		namespaces = append(namespaces, &ns)
	}

	return namespaces, nil
}

// Update updates a namespace
func (r *NamespaceRepository) Update(ctx context.Context, id string, req *domain.NamespaceUpdate) (*domain.Namespace, error) {
	// Build dynamic update query
	query := "UPDATE namespaces SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIndex)
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *req.Description)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, name, description, created_at, updated_at", argIndex)
	args = append(args, id)

	var ns domain.Namespace
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&ns.ID,
		&ns.Name,
		&ns.Description,
		&ns.CreatedAt,
		&ns.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		r.logger.Error().Err(err).Str("id", id).Msg("Failed to update namespace")
		return nil, fmt.Errorf("failed to update namespace: %w", err)
	}

	r.logger.Info().Str("namespace_id", ns.ID).Msg("Namespace updated")
	return &ns, nil
}

// Delete deletes a namespace
func (r *NamespaceRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM namespaces WHERE id = $1"

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Msg("Failed to delete namespace")
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	r.logger.Info().Str("namespace_id", id).Msg("Namespace deleted")
	return nil
}

// AddServerToNamespace adds a server to a namespace
func (r *NamespaceRepository) AddServerToNamespace(ctx context.Context, serverID, namespaceID string) error {
	query := `
		INSERT INTO namespace_members (server_id, namespace_id)
		VALUES ($1, $2)
		ON CONFLICT (server_id, namespace_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, serverID, namespaceID)
	if err != nil {
		r.logger.Error().Err(err).
			Str("server_id", serverID).
			Str("namespace_id", namespaceID).
			Msg("Failed to add server to namespace")
		return fmt.Errorf("failed to add server to namespace: %w", err)
	}

	r.logger.Info().
		Str("server_id", serverID).
		Str("namespace_id", namespaceID).
		Msg("Server added to namespace")
	return nil
}

// RemoveServerFromNamespace removes a server from a namespace
func (r *NamespaceRepository) RemoveServerFromNamespace(ctx context.Context, serverID, namespaceID string) error {
	query := "DELETE FROM namespace_members WHERE server_id = $1 AND namespace_id = $2"

	result, err := r.db.Exec(ctx, query, serverID, namespaceID)
	if err != nil {
		r.logger.Error().Err(err).
			Str("server_id", serverID).
			Str("namespace_id", namespaceID).
			Msg("Failed to remove server from namespace")
		return fmt.Errorf("failed to remove server from namespace: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	r.logger.Info().
		Str("server_id", serverID).
		Str("namespace_id", namespaceID).
		Msg("Server removed from namespace")
	return nil
}

// GetServerNamespaces returns all namespace IDs a server belongs to
func (r *NamespaceRepository) GetServerNamespaces(ctx context.Context, serverID string) ([]string, error) {
	query := "SELECT namespace_id FROM namespace_members WHERE server_id = $1"

	rows, err := r.db.Query(ctx, query, serverID)
	if err != nil {
		r.logger.Error().Err(err).Str("server_id", serverID).Msg("Failed to get server namespaces")
		return nil, fmt.Errorf("failed to get server namespaces: %w", err)
	}
	defer rows.Close()

	var namespaceIDs []string
	for rows.Next() {
		var namespaceID string
		if err := rows.Scan(&namespaceID); err != nil {
			return nil, fmt.Errorf("failed to scan namespace id: %w", err)
		}
		namespaceIDs = append(namespaceIDs, namespaceID)
	}

	return namespaceIDs, nil
}

// GetNamespaceServers returns all servers in a namespace
func (r *NamespaceRepository) GetNamespaceServers(ctx context.Context, namespaceID string) ([]*domain.NamespaceMember, error) {
	query := `
		SELECT nm.server_id, s.name, nm.namespace_id, n.name
		FROM namespace_members nm
		JOIN mcp_servers s ON nm.server_id = s.id
		JOIN namespaces n ON nm.namespace_id = n.id
		WHERE nm.namespace_id = $1
		ORDER BY s.name
	`

	rows, err := r.db.Query(ctx, query, namespaceID)
	if err != nil {
		r.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to get namespace servers")
		return nil, fmt.Errorf("failed to get namespace servers: %w", err)
	}
	defer rows.Close()

	var members []*domain.NamespaceMember
	for rows.Next() {
		var member domain.NamespaceMember
		if err := rows.Scan(
			&member.ServerID,
			&member.ServerName,
			&member.NamespaceID,
			&member.NamespaceName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan namespace member: %w", err)
		}
		members = append(members, &member)
	}

	return members, nil
}

// SetRoleNamespaceAccess sets a role's access level to a namespace
func (r *NamespaceRepository) SetRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string, level domain.AccessLevel) error {
	query := `
		INSERT INTO role_namespace_access (role_id, namespace_id, access_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (role_id, namespace_id) DO UPDATE SET access_level = $3
	`

	_, err := r.db.Exec(ctx, query, roleID, namespaceID, string(level))
	if err != nil {
		r.logger.Error().Err(err).
			Str("role_id", roleID).
			Str("namespace_id", namespaceID).
			Str("access_level", string(level)).
			Msg("Failed to set role namespace access")
		return fmt.Errorf("failed to set role namespace access: %w", err)
	}

	r.logger.Info().
		Str("role_id", roleID).
		Str("namespace_id", namespaceID).
		Str("access_level", string(level)).
		Msg("Role namespace access set")
	return nil
}

// RemoveRoleNamespaceAccess removes a role's access to a namespace
func (r *NamespaceRepository) RemoveRoleNamespaceAccess(ctx context.Context, roleID, namespaceID string) error {
	query := "DELETE FROM role_namespace_access WHERE role_id = $1 AND namespace_id = $2"

	result, err := r.db.Exec(ctx, query, roleID, namespaceID)
	if err != nil {
		r.logger.Error().Err(err).
			Str("role_id", roleID).
			Str("namespace_id", namespaceID).
			Msg("Failed to remove role namespace access")
		return fmt.Errorf("failed to remove role namespace access: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	r.logger.Info().
		Str("role_id", roleID).
		Str("namespace_id", namespaceID).
		Msg("Role namespace access removed")
	return nil
}

// GetNamespaceRoleAccess returns all role access entries for a namespace
func (r *NamespaceRepository) GetNamespaceRoleAccess(ctx context.Context, namespaceID string) ([]*domain.RoleNamespaceAccess, error) {
	query := `
		SELECT rna.role_id, ro.name, rna.namespace_id, n.name, rna.access_level
		FROM role_namespace_access rna
		JOIN roles ro ON rna.role_id = ro.id
		JOIN namespaces n ON rna.namespace_id = n.id
		WHERE rna.namespace_id = $1
		ORDER BY ro.name
	`

	rows, err := r.db.Query(ctx, query, namespaceID)
	if err != nil {
		r.logger.Error().Err(err).Str("namespace_id", namespaceID).Msg("Failed to get namespace role access")
		return nil, fmt.Errorf("failed to get namespace role access: %w", err)
	}
	defer rows.Close()

	var accesses []*domain.RoleNamespaceAccess
	for rows.Next() {
		var access domain.RoleNamespaceAccess
		if err := rows.Scan(
			&access.RoleID,
			&access.RoleName,
			&access.NamespaceID,
			&access.NamespaceName,
			&access.AccessLevel,
		); err != nil {
			return nil, fmt.Errorf("failed to scan role namespace access: %w", err)
		}
		accesses = append(accesses, &access)
	}

	return accesses, nil
}

// GetAccessibleServerIDs returns server IDs that the given roles can access at the specified level
// This is the critical query for access control filtering
func (r *NamespaceRepository) GetAccessibleServerIDs(ctx context.Context, roles []string, minAccessLevel domain.AccessLevel) ([]string, error) {
	if len(roles) == 0 {
		return []string{}, nil
	}

	// Build access level condition
	// "execute" grants both execute and view
	// "view" grants only view
	accessCondition := "rna.access_level = $2"
	if minAccessLevel == domain.AccessLevelView {
		// For view access, both "view" and "execute" are acceptable
		accessCondition = "rna.access_level IN ('view', 'execute')"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT s.id
		FROM mcp_servers s
		INNER JOIN namespace_members nm ON s.id = nm.server_id
		INNER JOIN role_namespace_access rna ON nm.namespace_id = rna.namespace_id
		INNER JOIN roles ro ON rna.role_id = ro.id
		WHERE ro.name = ANY($1)
		  AND %s
		  AND s.is_active = true
	`, accessCondition)

	var args []interface{}
	if minAccessLevel == domain.AccessLevelView {
		args = []interface{}{roles}
	} else {
		args = []interface{}{roles, string(minAccessLevel)}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error().Err(err).
			Any("roles", roles).
			Str("access_level", string(minAccessLevel)).
			Msg("Failed to get accessible server IDs")
		return nil, fmt.Errorf("failed to get accessible server IDs: %w", err)
	}
	defer rows.Close()

	var serverIDs []string
	for rows.Next() {
		var serverID string
		if err := rows.Scan(&serverID); err != nil {
			return nil, fmt.Errorf("failed to scan server id: %w", err)
		}
		serverIDs = append(serverIDs, serverID)
	}

	r.logger.Debug().
		Any("roles", roles).
		Str("access_level", string(minAccessLevel)).
		Int("count", len(serverIDs)).
		Msg("Retrieved accessible server IDs")

	return serverIDs, nil
}

// GetRoleIDByName returns the role ID for a given role name
func (r *NamespaceRepository) GetRoleIDByName(ctx context.Context, roleName string) (string, error) {
	query := "SELECT id FROM roles WHERE name = $1"

	var roleID string
	err := r.db.QueryRow(ctx, query, roleName).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("failed to get role by name: %w", err)
	}

	return roleID, nil
}

// Legacy aliases for backwards compatibility
var NewServerGroupRepository = NewNamespaceRepository

type ServerGroupRepository = NamespaceRepository

// Legacy method aliases - these wrap the new methods for backwards compatibility
func (r *NamespaceRepository) AddServerToGroup(ctx context.Context, serverID, groupID string) error {
	return r.AddServerToNamespace(ctx, serverID, groupID)
}

func (r *NamespaceRepository) RemoveServerFromGroup(ctx context.Context, serverID, groupID string) error {
	return r.RemoveServerFromNamespace(ctx, serverID, groupID)
}

func (r *NamespaceRepository) GetServerGroups(ctx context.Context, serverID string) ([]string, error) {
	return r.GetServerNamespaces(ctx, serverID)
}

func (r *NamespaceRepository) GetGroupServers(ctx context.Context, groupID string) ([]*domain.NamespaceMember, error) {
	return r.GetNamespaceServers(ctx, groupID)
}

func (r *NamespaceRepository) SetRoleGroupAccess(ctx context.Context, roleID, groupID string, level domain.AccessLevel) error {
	return r.SetRoleNamespaceAccess(ctx, roleID, groupID, level)
}

func (r *NamespaceRepository) RemoveRoleGroupAccess(ctx context.Context, roleID, groupID string) error {
	return r.RemoveRoleNamespaceAccess(ctx, roleID, groupID)
}

func (r *NamespaceRepository) GetGroupRoleAccess(ctx context.Context, groupID string) ([]*domain.RoleNamespaceAccess, error) {
	return r.GetNamespaceRoleAccess(ctx, groupID)
}

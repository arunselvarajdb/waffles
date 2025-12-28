package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waffles/waffles/pkg/logger"
)

// DBTX defines the database interface needed by the service
type DBTX interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// Role represents a role in the system
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UserCount   int    `json:"user_count"`
	IsBuiltIn   bool   `json:"is_built_in"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// RoleWithPermissions includes a role and its permissions
type RoleWithPermissions struct {
	Role        *Role         `json:"role"`
	Permissions []*Permission `json:"permissions"`
}

// Service handles role business logic
type Service struct {
	db     DBTX
	logger logger.Logger
}

// NewService creates a new role service
func NewService(db *pgxpool.Pool, log logger.Logger) *Service {
	return &Service{
		db:     db,
		logger: log.With().Str("service", "role").Logger(),
	}
}

// Built-in roles that cannot be deleted
var builtInRoles = map[string]bool{
	"admin":    true,
	"operator": true,
	"user":     true,
	"readonly": true,
}

// ErrRoleNotFound is returned when a role is not found
var ErrRoleNotFound = errors.New("role not found")

// ErrBuiltInRole is returned when trying to modify a built-in role
var ErrBuiltInRole = errors.New("cannot delete built-in role")

// ErrRoleNameExists is returned when a role name already exists
var ErrRoleNameExists = errors.New("role name already exists")

// List returns all roles with user counts
func (s *Service) List(ctx context.Context) ([]*Role, error) {
	query := `
		SELECT
			r.id,
			r.name,
			COALESCE(r.description, '') as description,
			r.created_at::text,
			COUNT(ur.user_id) as user_count
		FROM roles r
		LEFT JOIN user_roles ur ON r.id = ur.role_id
		GROUP BY r.id, r.name, r.description, r.created_at
		ORDER BY r.name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list roles")
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UserCount); err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan role")
			continue
		}
		r.IsBuiltIn = builtInRoles[r.Name]
		roles = append(roles, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}

// GetByID returns a role by ID with its permissions
func (s *Service) GetByID(ctx context.Context, id string) (*RoleWithPermissions, error) {
	// Get role
	roleQuery := `
		SELECT
			r.id,
			r.name,
			COALESCE(r.description, '') as description,
			r.created_at::text,
			COUNT(ur.user_id) as user_count
		FROM roles r
		LEFT JOIN user_roles ur ON r.id = ur.role_id
		WHERE r.id = $1
		GROUP BY r.id, r.name, r.description, r.created_at
	`

	var role Role
	err := s.db.QueryRow(ctx, roleQuery, id).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UserCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		s.logger.Error().Err(err).Str("role_id", id).Msg("Failed to get role")
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	role.IsBuiltIn = builtInRoles[role.Name]

	// Get permissions
	permQuery := `
		SELECT p.id, p.name, p.resource, p.action, COALESCE(p.description, '') as description
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := s.db.Query(ctx, permQuery, id)
	if err != nil {
		s.logger.Error().Err(err).Str("role_id", id).Msg("Failed to get role permissions")
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Resource, &p.Action, &p.Description); err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan permission")
			continue
		}
		permissions = append(permissions, &p)
	}

	return &RoleWithPermissions{
		Role:        &role,
		Permissions: permissions,
	}, nil
}

// CreateRequest represents data for creating a role
type CreateRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"` // Permission IDs
}

// Create creates a new role
func (s *Service) Create(ctx context.Context, req CreateRequest) (*RoleWithPermissions, error) {
	// Check if role name exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)", req.Name).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check role existence: %w", err)
	}
	if exists {
		return nil, ErrRoleNameExists
	}

	// Create role
	var roleID string
	err = s.db.QueryRow(ctx,
		"INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id",
		req.Name, req.Description,
	).Scan(&roleID)
	if err != nil {
		s.logger.Error().Err(err).Str("name", req.Name).Msg("Failed to create role")
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// Assign permissions
	for _, permID := range req.Permissions {
		_, err = s.db.Exec(ctx,
			"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			roleID, permID,
		)
		if err != nil {
			s.logger.Warn().Err(err).Str("role_id", roleID).Str("permission_id", permID).Msg("Failed to assign permission")
		}
	}

	s.logger.Info().Str("role_id", roleID).Str("name", req.Name).Msg("Role created")

	return s.GetByID(ctx, roleID)
}

// UpdateRequest represents data for updating a role
type UpdateRequest struct {
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"` // Permission IDs - replaces all existing
}

// Update updates a role's permissions
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*RoleWithPermissions, error) {
	// Verify role exists
	roleWithPerms, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update description if provided
	if req.Description != nil {
		_, err = s.db.Exec(ctx,
			"UPDATE roles SET description = $1 WHERE id = $2",
			*req.Description, id,
		)
		if err != nil {
			s.logger.Error().Err(err).Str("role_id", id).Msg("Failed to update role description")
			return nil, fmt.Errorf("failed to update role: %w", err)
		}
	}

	// Update permissions if provided
	if req.Permissions != nil {
		// Remove all existing permissions
		_, err = s.db.Exec(ctx, "DELETE FROM role_permissions WHERE role_id = $1", id)
		if err != nil {
			s.logger.Error().Err(err).Str("role_id", id).Msg("Failed to remove role permissions")
			return nil, fmt.Errorf("failed to update permissions: %w", err)
		}

		// Add new permissions
		for _, permID := range req.Permissions {
			_, err = s.db.Exec(ctx,
				"INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				id, permID,
			)
			if err != nil {
				s.logger.Warn().Err(err).Str("role_id", id).Str("permission_id", permID).Msg("Failed to assign permission")
			}
		}
	}

	s.logger.Info().Str("role_id", id).Str("name", roleWithPerms.Role.Name).Msg("Role updated")

	return s.GetByID(ctx, id)
}

// Delete deletes a custom role (built-in roles cannot be deleted)
func (s *Service) Delete(ctx context.Context, id string) error {
	// Get role to check if it's built-in
	roleWithPerms, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if roleWithPerms.Role.IsBuiltIn {
		return ErrBuiltInRole
	}

	// Delete role (cascades to user_roles and role_permissions)
	result, err := s.db.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	if err != nil {
		s.logger.Error().Err(err).Str("role_id", id).Msg("Failed to delete role")
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRoleNotFound
	}

	s.logger.Info().Str("role_id", id).Str("name", roleWithPerms.Role.Name).Msg("Role deleted")
	return nil
}

// ListPermissions returns all available permissions
func (s *Service) ListPermissions(ctx context.Context) ([]*Permission, error) {
	query := `
		SELECT id, name, resource, action, COALESCE(description, '') as description
		FROM permissions
		ORDER BY resource, action
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list permissions")
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Resource, &p.Action, &p.Description); err != nil {
			s.logger.Error().Err(err).Msg("Failed to scan permission")
			continue
		}
		permissions = append(permissions, &p)
	}

	return permissions, nil
}

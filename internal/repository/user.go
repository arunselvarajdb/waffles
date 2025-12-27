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

// UserRepository handles user data persistence
type UserRepository struct {
	db     DBTX
	logger logger.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db DBTX, log logger.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: log,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, auth_provider, external_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.AuthProvider,
		user.ExternalID,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		r.logger.Error().Err(err).Str("email", user.Email).Msg("Failed to create user")
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Info().
		Str("user_id", user.ID).
		Str("email", user.Email).
		Msg("User created successfully")

	return nil
}

// getUserBy is a helper that retrieves a user by a given column and value.
func (r *UserRepository) getUserBy(ctx context.Context, column, value, logField string) (*domain.User, error) {
	query := fmt.Sprintf(`
		SELECT id, email, COALESCE(password_hash, ''), COALESCE(name, ''), COALESCE(auth_provider, 'local'), COALESCE(external_id, ''), is_active, created_at, updated_at
		FROM users
		WHERE %s = $1
	`, column)

	var user domain.User
	err := r.db.QueryRow(ctx, query, value).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.AuthProvider,
		&user.ExternalID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).Str(logField, value).Msg("Failed to get user")

		return nil, fmt.Errorf("failed to get user by %s: %w", column, err)
	}

	return &user, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.getUserBy(ctx, "id", id, "user_id")
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getUserBy(ctx, "email", email, "email")
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, name = $2, is_active = $3, updated_at = $4
		WHERE id = $5
		RETURNING updated_at
	`

	user.UpdatedAt = time.Now()
	err := r.db.QueryRow(ctx, query,
		user.Email,
		user.Name,
		user.IsActive,
		user.UpdatedAt,
		user.ID,
	).Scan(&user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to update user")
		return fmt.Errorf("failed to update user: %w", err)
	}

	r.logger.Info().Str("user_id", user.ID).Msg("User updated successfully")
	return nil
}

// UpdatePassword updates a user's password hash
func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.Exec(ctx, query, passwordHash, time.Now(), userID)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to update password")
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	r.logger.Info().Str("user_id", userID).Msg("User password updated")
	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", id).Msg("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	r.logger.Info().Str("user_id", id).Msg("User deleted successfully")
	return nil
}

// List retrieves all users with optional filtering
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM users`
	var total int
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count users")
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT id, email, COALESCE(password_hash, ''), COALESCE(name, ''), COALESCE(auth_provider, 'local'), COALESCE(external_id, ''), is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list users")
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.AuthProvider,
			&user.ExternalID,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan user row")
			continue
		}
		// Clear password hash for security
		user.PasswordHash = ""
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating user rows")
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	r.logger.Debug().Int("count", len(users)).Msg("Users listed")
	return users, total, nil
}

// GetUserRoles retrieves all role names for a user
func (r *UserRepository) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user roles")
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan role")
			continue
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (r *UserRepository) AssignRole(ctx context.Context, userID, roleName string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, roleName)
	if err != nil {
		r.logger.Error().Err(err).
			Str("user_id", userID).
			Str("role", roleName).
			Msg("Failed to assign role")
		return fmt.Errorf("failed to assign role: %w", err)
	}

	r.logger.Info().
		Str("user_id", userID).
		Str("role", roleName).
		Msg("Role assigned successfully")
	return nil
}

// RemoveRole removes a role from a user
func (r *UserRepository) RemoveRole(ctx context.Context, userID, roleName string) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1
		AND role_id = (SELECT id FROM roles WHERE name = $2)
	`

	result, err := r.db.Exec(ctx, query, userID, roleName)
	if err != nil {
		r.logger.Error().Err(err).
			Str("user_id", userID).
			Str("role", roleName).
			Msg("Failed to remove role")
		return fmt.Errorf("failed to remove role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("role assignment not found")
	}

	r.logger.Info().
		Str("user_id", userID).
		Str("role", roleName).
		Msg("Role removed successfully")
	return nil
}

// GetByExternalID retrieves a user by their OAuth provider and external ID
func (r *UserRepository) GetByExternalID(ctx context.Context, provider, externalID string) (*domain.User, error) {
	query := `
		SELECT id, email, COALESCE(password_hash, ''), COALESCE(name, ''), COALESCE(auth_provider, 'local'), COALESCE(external_id, ''), is_active, created_at, updated_at
		FROM users
		WHERE auth_provider = $1 AND external_id = $2
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, provider, externalID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.AuthProvider,
		&user.ExternalID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		r.logger.Error().Err(err).
			Str("provider", provider).
			Str("external_id", externalID).
			Msg("Failed to get user by external ID")
		return nil, fmt.Errorf("failed to get user by external ID: %w", err)
	}

	return &user, nil
}

// FindOrCreateOAuthUser finds a user by OAuth provider/external ID, or creates a new one
func (r *UserRepository) FindOrCreateOAuthUser(ctx context.Context, provider, externalID, email, name string) (*domain.User, bool, error) {
	// First, try to find by external ID
	user, err := r.GetByExternalID(ctx, provider, externalID)
	if err == nil {
		return user, false, nil // Existing user
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, false, err
	}

	// Not found by external ID, check if email exists
	existingUser, err := r.GetByEmail(ctx, email)
	if err == nil {
		// Email exists - update this user to link OAuth provider
		updateQuery := `
			UPDATE users
			SET auth_provider = $1, external_id = $2, name = COALESCE(NULLIF($3, ''), name), updated_at = $4
			WHERE id = $5
			RETURNING updated_at
		`
		existingUser.AuthProvider = domain.AuthProvider(provider)
		existingUser.ExternalID = externalID
		if name != "" {
			existingUser.Name = name
		}
		existingUser.UpdatedAt = time.Now()

		err = r.db.QueryRow(ctx, updateQuery,
			provider, externalID, name, existingUser.UpdatedAt, existingUser.ID,
		).Scan(&existingUser.UpdatedAt)
		if err != nil {
			r.logger.Error().Err(err).Str("user_id", existingUser.ID).Msg("Failed to link OAuth to existing user")
			return nil, false, fmt.Errorf("failed to link OAuth: %w", err)
		}

		r.logger.Info().
			Str("user_id", existingUser.ID).
			Str("provider", provider).
			Msg("Linked OAuth provider to existing user")
		return existingUser, false, nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, false, err
	}

	// Create new user
	newUser := &domain.User{
		Email:        email,
		Name:         name,
		AuthProvider: domain.AuthProvider(provider),
		ExternalID:   externalID,
		IsActive:     true,
	}

	if err := r.Create(ctx, newUser); err != nil {
		return nil, false, err
	}

	r.logger.Info().
		Str("user_id", newUser.ID).
		Str("email", email).
		Str("provider", provider).
		Msg("Created new OAuth user")

	return newUser, true, nil
}

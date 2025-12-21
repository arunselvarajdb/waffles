package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

func TestUserRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully creates user", func(t *testing.T) {
		user := &domain.User{
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			Name:         "Test User",
			AuthProvider: domain.AuthProviderLocal,
			IsActive:     true,
		}

		now := time.Now()

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Email, user.PasswordHash, user.Name, user.AuthProvider, user.ExternalID, user.IsActive).
			WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow("user-123", now, now))

		err := repo.Create(context.Background(), user)

		require.NoError(t, err)
		assert.Equal(t, "user-123", user.ID)
		assert.Equal(t, now, user.CreatedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		user := &domain.User{
			Email:    "fail@example.com",
			IsActive: true,
		}

		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Email, user.PasswordHash, user.Name, user.AuthProvider, user.ExternalID, user.IsActive).
			WillReturnError(errors.New("duplicate email"))

		err := repo.Create(context.Background(), user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets user by ID", func(t *testing.T) {
		userID := "user-123"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}).AddRow(
				userID, "test@example.com", "hash", "Test User", "local", "", true, now, now,
			))

		user, err := repo.GetByID(context.Background(), userID)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test User", user.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when user does not exist", func(t *testing.T) {
		userID := "nonexistent"

		mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			})) // Empty result

		user, err := repo.GetByID(context.Background(), userID)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnError(errors.New("connection failed"))

		user, err := repo.GetByID(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to get user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets user by email", func(t *testing.T) {
		email := "test@example.com"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}).AddRow(
				"user-123", email, "hash", "Test User", "local", "", true, now, now,
			))

		user, err := repo.GetByEmail(context.Background(), email)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when email not found", func(t *testing.T) {
		email := "unknown@example.com"

		mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			})) // Empty result

		user, err := repo.GetByEmail(context.Background(), email)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		email := "test@example.com"

		mock.ExpectQuery("SELECT .+ FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnError(errors.New("connection failed"))

		user, err := repo.GetByEmail(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to get user by email")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully updates user", func(t *testing.T) {
		user := &domain.User{
			ID:       "user-123",
			Email:    "updated@example.com",
			Name:     "Updated User",
			IsActive: true,
		}
		now := time.Now()

		mock.ExpectQuery("UPDATE users").
			WithArgs(user.Email, user.Name, user.IsActive, pgxmock.AnyArg(), user.ID).
			WillReturnRows(pgxmock.NewRows([]string{"updated_at"}).AddRow(now))

		err := repo.Update(context.Background(), user)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when user does not exist", func(t *testing.T) {
		user := &domain.User{
			ID:    "nonexistent",
			Email: "test@example.com",
		}

		mock.ExpectQuery("UPDATE users").
			WithArgs(user.Email, user.Name, user.IsActive, pgxmock.AnyArg(), user.ID).
			WillReturnRows(pgxmock.NewRows([]string{"updated_at"})) // Empty result

		err := repo.Update(context.Background(), user)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		user := &domain.User{
			ID:    "user-123",
			Email: "test@example.com",
		}

		mock.ExpectQuery("UPDATE users").
			WithArgs(user.Email, user.Name, user.IsActive, pgxmock.AnyArg(), user.ID).
			WillReturnError(errors.New("update failed"))

		err := repo.Update(context.Background(), user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully updates password", func(t *testing.T) {
		userID := "user-123"
		newHash := "newpasswordhash"

		mock.ExpectExec("UPDATE users").
			WithArgs(newHash, pgxmock.AnyArg(), userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdatePassword(context.Background(), userID, newHash)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when user does not exist", func(t *testing.T) {
		userID := "nonexistent"
		newHash := "hash"

		mock.ExpectExec("UPDATE users").
			WithArgs(newHash, pgxmock.AnyArg(), userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdatePassword(context.Background(), userID, newHash)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectExec("UPDATE users").
			WithArgs("hash", pgxmock.AnyArg(), userID).
			WillReturnError(errors.New("update failed"))

		err := repo.UpdatePassword(context.Background(), userID, "hash")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update password")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully deletes user", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), userID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when user does not exist", func(t *testing.T) {
		userID := "nonexistent"

		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), userID)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnError(errors.New("delete failed"))

		err := repo.Delete(context.Background(), userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully lists users", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(25))

		mock.ExpectQuery("SELECT .+ FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 0).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}).
				AddRow("user-1", "user1@example.com", "hash1", "User 1", "local", "", true, now, now).
				AddRow("user-2", "user2@example.com", "hash2", "User 2", "local", "", true, now, now))

		users, total, err := repo.List(context.Background(), 10, 0)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, 25, total)
		// Verify password hash is cleared for security
		assert.Empty(t, users[0].PasswordHash)
		assert.Empty(t, users[1].PasswordHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list when no users", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery("SELECT .+ FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 0).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}))

		users, total, err := repo.List(context.Background(), 10, 0)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, 0, total)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on count failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
			WillReturnError(errors.New("count failed"))

		users, total, err := repo.List(context.Background(), 10, 0)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, 0, total)
		assert.Contains(t, err.Error(), "failed to count users")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on query failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(10))

		mock.ExpectQuery("SELECT .+ FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 0).
			WillReturnError(errors.New("query failed"))

		users, total, err := repo.List(context.Background(), 10, 0)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, 0, total)
		assert.Contains(t, err.Error(), "failed to list users")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetUserRoles(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets user roles", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectQuery("SELECT r.name FROM roles").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"name"}).
				AddRow("admin").
				AddRow("user"))

		roles, err := repo.GetUserRoles(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, roles, 2)
		assert.Contains(t, roles, "admin")
		assert.Contains(t, roles, "user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when user has no roles", func(t *testing.T) {
		userID := "user-no-roles"

		mock.ExpectQuery("SELECT r.name FROM roles").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"name"}))

		roles, err := repo.GetUserRoles(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, roles)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"

		mock.ExpectQuery("SELECT r.name FROM roles").
			WithArgs(userID).
			WillReturnError(errors.New("query failed"))

		roles, err := repo.GetUserRoles(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, roles)
		assert.Contains(t, err.Error(), "failed to get user roles")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_AssignRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully assigns role", func(t *testing.T) {
		userID := "user-123"
		roleName := "admin"

		mock.ExpectExec("INSERT INTO user_roles").
			WithArgs(userID, roleName).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.AssignRole(context.Background(), userID, roleName)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"
		roleName := "admin"

		mock.ExpectExec("INSERT INTO user_roles").
			WithArgs(userID, roleName).
			WillReturnError(errors.New("insert failed"))

		err := repo.AssignRole(context.Background(), userID, roleName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to assign role")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_RemoveRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully removes role", func(t *testing.T) {
		userID := "user-123"
		roleName := "admin"

		mock.ExpectExec("DELETE FROM user_roles").
			WithArgs(userID, roleName).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.RemoveRole(context.Background(), userID, roleName)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when role assignment not found", func(t *testing.T) {
		userID := "user-123"
		roleName := "nonexistent"

		mock.ExpectExec("DELETE FROM user_roles").
			WithArgs(userID, roleName).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.RemoveRole(context.Background(), userID, roleName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role assignment not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		userID := "user-123"
		roleName := "admin"

		mock.ExpectExec("DELETE FROM user_roles").
			WithArgs(userID, roleName).
			WillReturnError(errors.New("delete failed"))

		err := repo.RemoveRole(context.Background(), userID, roleName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove role")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByExternalID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets user by external ID", func(t *testing.T) {
		provider := "google"
		externalID := "google-123"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM users WHERE auth_provider = \\$1 AND external_id = \\$2").
			WithArgs(provider, externalID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}).AddRow(
				"user-123", "oauth@example.com", "", "OAuth User", provider, externalID, true, now, now,
			))

		user, err := repo.GetByExternalID(context.Background(), provider, externalID)

		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "oauth@example.com", user.Email)
		assert.Equal(t, domain.AuthProvider(provider), user.AuthProvider)
		assert.Equal(t, externalID, user.ExternalID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrUserNotFound when not found", func(t *testing.T) {
		provider := "google"
		externalID := "unknown"

		mock.ExpectQuery("SELECT .+ FROM users WHERE auth_provider = \\$1 AND external_id = \\$2").
			WithArgs(provider, externalID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "email", "password_hash", "name", "auth_provider", "external_id", "is_active", "created_at", "updated_at",
			}))

		user, err := repo.GetByExternalID(context.Background(), provider, externalID)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		provider := "google"
		externalID := "google-123"

		mock.ExpectQuery("SELECT .+ FROM users WHERE auth_provider = \\$1 AND external_id = \\$2").
			WithArgs(provider, externalID).
			WillReturnError(errors.New("query failed"))

		user, err := repo.GetByExternalID(context.Background(), provider, externalID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to get user by external ID")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewUserRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	log := logger.NewNopLogger()
	repo := NewUserRepository(mock, log)

	require.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

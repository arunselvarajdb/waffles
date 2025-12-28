package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

func TestNamespaceRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully creates namespace", func(t *testing.T) {
		req := &domain.NamespaceCreate{
			Name:        "test-namespace",
			Description: "Test namespace description",
		}
		now := time.Now()

		mock.ExpectQuery("INSERT INTO namespaces").
			WithArgs(req.Name, req.Description).
			WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
				AddRow("ns-123", req.Name, req.Description, now, now))

		ns, err := repo.Create(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, ns)
		assert.Equal(t, "ns-123", ns.ID)
		assert.Equal(t, req.Name, ns.Name)
		assert.Equal(t, req.Description, ns.Description)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		req := &domain.NamespaceCreate{
			Name: "duplicate",
		}

		mock.ExpectQuery("INSERT INTO namespaces").
			WithArgs(req.Name, req.Description).
			WillReturnError(errors.New("duplicate key"))

		ns, err := repo.Create(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, ns)
		assert.Contains(t, err.Error(), "failed to create namespace")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets namespace by ID", func(t *testing.T) {
		nsID := "ns-123"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM namespaces n WHERE n.id = \\$1").
			WithArgs(nsID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}).AddRow(nsID, "test-ns", "desc", now, now, 5))

		ns, err := repo.Get(context.Background(), nsID)

		require.NoError(t, err)
		require.NotNil(t, ns)
		assert.Equal(t, nsID, ns.ID)
		assert.Equal(t, "test-ns", ns.Name)
		assert.Equal(t, 5, ns.ServerCount)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when namespace does not exist", func(t *testing.T) {
		nsID := "nonexistent"

		mock.ExpectQuery("SELECT .+ FROM namespaces n WHERE n.id = \\$1").
			WithArgs(nsID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}))

		ns, err := repo.Get(context.Background(), nsID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, ns)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		nsID := "ns-123"

		mock.ExpectQuery("SELECT .+ FROM namespaces n WHERE n.id = \\$1").
			WithArgs(nsID).
			WillReturnError(errors.New("connection failed"))

		ns, err := repo.Get(context.Background(), nsID)

		assert.Error(t, err)
		assert.Nil(t, ns)
		assert.Contains(t, err.Error(), "failed to get namespace")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetByName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets namespace by name", func(t *testing.T) {
		nsName := "test-ns"
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM namespaces n WHERE n.name = \\$1").
			WithArgs(nsName).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}).AddRow("ns-123", nsName, "desc", now, now, 3))

		ns, err := repo.GetByName(context.Background(), nsName)

		require.NoError(t, err)
		require.NotNil(t, ns)
		assert.Equal(t, nsName, ns.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when name not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM namespaces n WHERE n.name = \\$1").
			WithArgs("unknown").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}))

		ns, err := repo.GetByName(context.Background(), "unknown")

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, ns)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully lists namespaces", func(t *testing.T) {
		now := time.Now()

		mock.ExpectQuery("SELECT .+ FROM namespaces n ORDER BY n.name").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}).
				AddRow("ns-1", "alpha", "Alpha namespace", now, now, 2).
				AddRow("ns-2", "beta", "Beta namespace", now, now, 5))

		namespaces, err := repo.List(context.Background())

		require.NoError(t, err)
		assert.Len(t, namespaces, 2)
		assert.Equal(t, "alpha", namespaces[0].Name)
		assert.Equal(t, "beta", namespaces[1].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list when no namespaces", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM namespaces n ORDER BY n.name").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at", "server_count",
			}))

		namespaces, err := repo.List(context.Background())

		require.NoError(t, err)
		assert.Empty(t, namespaces)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT .+ FROM namespaces n ORDER BY n.name").
			WillReturnError(errors.New("query failed"))

		namespaces, err := repo.List(context.Background())

		assert.Error(t, err)
		assert.Nil(t, namespaces)
		assert.Contains(t, err.Error(), "failed to list namespaces")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully updates namespace name", func(t *testing.T) {
		nsID := "ns-123"
		newName := "updated-name"
		now := time.Now()
		req := &domain.NamespaceUpdate{Name: &newName}

		mock.ExpectQuery("UPDATE namespaces SET").
			WithArgs(pgxmock.AnyArg(), newName, nsID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at",
			}).AddRow(nsID, newName, "desc", now, now))

		ns, err := repo.Update(context.Background(), nsID, req)

		require.NoError(t, err)
		require.NotNil(t, ns)
		assert.Equal(t, newName, ns.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successfully updates namespace description", func(t *testing.T) {
		nsID := "ns-123"
		newDesc := "updated description"
		now := time.Now()
		req := &domain.NamespaceUpdate{Description: &newDesc}

		mock.ExpectQuery("UPDATE namespaces SET").
			WithArgs(pgxmock.AnyArg(), newDesc, nsID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at",
			}).AddRow(nsID, "name", newDesc, now, now))

		ns, err := repo.Update(context.Background(), nsID, req)

		require.NoError(t, err)
		require.NotNil(t, ns)
		assert.Equal(t, newDesc, ns.Description)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when namespace does not exist", func(t *testing.T) {
		nsID := "nonexistent"
		name := "name"
		req := &domain.NamespaceUpdate{Name: &name}

		mock.ExpectQuery("UPDATE namespaces SET").
			WithArgs(pgxmock.AnyArg(), name, nsID).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "name", "description", "created_at", "updated_at",
			}))

		ns, err := repo.Update(context.Background(), nsID, req)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, ns)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully deletes namespace", func(t *testing.T) {
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM namespaces WHERE id = \\$1").
			WithArgs(nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), nsID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when namespace does not exist", func(t *testing.T) {
		nsID := "nonexistent"

		mock.ExpectExec("DELETE FROM namespaces WHERE id = \\$1").
			WithArgs(nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), nsID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM namespaces WHERE id = \\$1").
			WithArgs(nsID).
			WillReturnError(errors.New("delete failed"))

		err := repo.Delete(context.Background(), nsID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete namespace")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_AddServerToNamespace(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully adds server to namespace", func(t *testing.T) {
		serverID := "server-123"
		nsID := "ns-123"

		mock.ExpectExec("INSERT INTO namespace_members").
			WithArgs(serverID, nsID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.AddServerToNamespace(context.Background(), serverID, nsID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		serverID := "server-123"
		nsID := "ns-123"

		mock.ExpectExec("INSERT INTO namespace_members").
			WithArgs(serverID, nsID).
			WillReturnError(errors.New("foreign key violation"))

		err := repo.AddServerToNamespace(context.Background(), serverID, nsID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add server to namespace")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_RemoveServerFromNamespace(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully removes server from namespace", func(t *testing.T) {
		serverID := "server-123"
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM namespace_members WHERE server_id = \\$1 AND namespace_id = \\$2").
			WithArgs(serverID, nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.RemoveServerFromNamespace(context.Background(), serverID, nsID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when membership does not exist", func(t *testing.T) {
		serverID := "server-123"
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM namespace_members WHERE server_id = \\$1 AND namespace_id = \\$2").
			WithArgs(serverID, nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.RemoveServerFromNamespace(context.Background(), serverID, nsID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetServerNamespaces(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets server namespaces", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectQuery("SELECT namespace_id FROM namespace_members WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{"namespace_id"}).
				AddRow("ns-1").
				AddRow("ns-2").
				AddRow("ns-3"))

		namespaceIDs, err := repo.GetServerNamespaces(context.Background(), serverID)

		require.NoError(t, err)
		assert.Len(t, namespaceIDs, 3)
		assert.Contains(t, namespaceIDs, "ns-1")
		assert.Contains(t, namespaceIDs, "ns-2")
		assert.Contains(t, namespaceIDs, "ns-3")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when server has no namespaces", func(t *testing.T) {
		serverID := "server-no-ns"

		mock.ExpectQuery("SELECT namespace_id FROM namespace_members WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnRows(pgxmock.NewRows([]string{"namespace_id"}))

		namespaceIDs, err := repo.GetServerNamespaces(context.Background(), serverID)

		require.NoError(t, err)
		assert.Empty(t, namespaceIDs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		serverID := "server-123"

		mock.ExpectQuery("SELECT namespace_id FROM namespace_members WHERE server_id = \\$1").
			WithArgs(serverID).
			WillReturnError(errors.New("query failed"))

		namespaceIDs, err := repo.GetServerNamespaces(context.Background(), serverID)

		assert.Error(t, err)
		assert.Nil(t, namespaceIDs)
		assert.Contains(t, err.Error(), "failed to get server namespaces")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetNamespaceServers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets namespace servers", func(t *testing.T) {
		nsID := "ns-123"

		mock.ExpectQuery("SELECT nm.server_id, s.name, nm.namespace_id, n.name FROM namespace_members").
			WithArgs(nsID).
			WillReturnRows(pgxmock.NewRows([]string{"server_id", "server_name", "namespace_id", "namespace_name"}).
				AddRow("server-1", "Server 1", nsID, "Test NS").
				AddRow("server-2", "Server 2", nsID, "Test NS"))

		members, err := repo.GetNamespaceServers(context.Background(), nsID)

		require.NoError(t, err)
		assert.Len(t, members, 2)
		assert.Equal(t, "server-1", members[0].ServerID)
		assert.Equal(t, "Server 1", members[0].ServerName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when namespace has no servers", func(t *testing.T) {
		nsID := "ns-empty"

		mock.ExpectQuery("SELECT nm.server_id, s.name, nm.namespace_id, n.name FROM namespace_members").
			WithArgs(nsID).
			WillReturnRows(pgxmock.NewRows([]string{"server_id", "server_name", "namespace_id", "namespace_name"}))

		members, err := repo.GetNamespaceServers(context.Background(), nsID)

		require.NoError(t, err)
		assert.Empty(t, members)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_SetRoleNamespaceAccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully sets role namespace access", func(t *testing.T) {
		roleID := "role-123"
		nsID := "ns-123"
		level := domain.AccessLevelExecute

		mock.ExpectExec("INSERT INTO role_namespace_access").
			WithArgs(roleID, nsID, string(level)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.SetRoleNamespaceAccess(context.Background(), roleID, nsID, level)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		roleID := "role-123"
		nsID := "ns-123"

		mock.ExpectExec("INSERT INTO role_namespace_access").
			WithArgs(roleID, nsID, string(domain.AccessLevelView)).
			WillReturnError(errors.New("insert failed"))

		err := repo.SetRoleNamespaceAccess(context.Background(), roleID, nsID, domain.AccessLevelView)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set role namespace access")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_RemoveRoleNamespaceAccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully removes role namespace access", func(t *testing.T) {
		roleID := "role-123"
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM role_namespace_access WHERE role_id = \\$1 AND namespace_id = \\$2").
			WithArgs(roleID, nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.RemoveRoleNamespaceAccess(context.Background(), roleID, nsID)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when access does not exist", func(t *testing.T) {
		roleID := "role-123"
		nsID := "ns-123"

		mock.ExpectExec("DELETE FROM role_namespace_access WHERE role_id = \\$1 AND namespace_id = \\$2").
			WithArgs(roleID, nsID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.RemoveRoleNamespaceAccess(context.Background(), roleID, nsID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetNamespaceRoleAccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets namespace role access", func(t *testing.T) {
		nsID := "ns-123"

		mock.ExpectQuery("SELECT rna.role_id, ro.name, rna.namespace_id, n.name, rna.access_level FROM role_namespace_access").
			WithArgs(nsID).
			WillReturnRows(pgxmock.NewRows([]string{"role_id", "role_name", "namespace_id", "namespace_name", "access_level"}).
				AddRow("role-1", "admin", nsID, "Test NS", string(domain.AccessLevelExecute)).
				AddRow("role-2", "viewer", nsID, "Test NS", string(domain.AccessLevelView)))

		accesses, err := repo.GetNamespaceRoleAccess(context.Background(), nsID)

		require.NoError(t, err)
		assert.Len(t, accesses, 2)
		assert.Equal(t, "admin", accesses[0].RoleName)
		assert.Equal(t, domain.AccessLevelExecute, accesses[0].AccessLevel)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetAccessibleServerIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("returns empty slice when no roles provided", func(t *testing.T) {
		serverIDs, err := repo.GetAccessibleServerIDs(context.Background(), []string{}, domain.AccessLevelView)

		require.NoError(t, err)
		assert.Empty(t, serverIDs)
	})

	t.Run("successfully gets accessible server IDs for view access", func(t *testing.T) {
		roles := []string{"admin", "user"}

		mock.ExpectQuery("SELECT DISTINCT s.id FROM mcp_servers").
			WithArgs(roles).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).
				AddRow("server-1").
				AddRow("server-2").
				AddRow("server-3"))

		serverIDs, err := repo.GetAccessibleServerIDs(context.Background(), roles, domain.AccessLevelView)

		require.NoError(t, err)
		assert.Len(t, serverIDs, 3)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successfully gets accessible server IDs for execute access", func(t *testing.T) {
		roles := []string{"admin"}

		mock.ExpectQuery("SELECT DISTINCT s.id FROM mcp_servers").
			WithArgs(roles, string(domain.AccessLevelExecute)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).
				AddRow("server-1"))

		serverIDs, err := repo.GetAccessibleServerIDs(context.Background(), roles, domain.AccessLevelExecute)

		require.NoError(t, err)
		assert.Len(t, serverIDs, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		roles := []string{"admin"}

		mock.ExpectQuery("SELECT DISTINCT s.id FROM mcp_servers").
			WithArgs(roles).
			WillReturnError(errors.New("query failed"))

		serverIDs, err := repo.GetAccessibleServerIDs(context.Background(), roles, domain.AccessLevelView)

		assert.Error(t, err)
		assert.Nil(t, serverIDs)
		assert.Contains(t, err.Error(), "failed to get accessible server IDs")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNamespaceRepository_GetRoleIDByName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNamespaceRepository(mock, logger.NewNopLogger())

	t.Run("successfully gets role ID by name", func(t *testing.T) {
		roleName := "admin"

		mock.ExpectQuery("SELECT id FROM roles WHERE name = \\$1").
			WithArgs(roleName).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("role-123"))

		roleID, err := repo.GetRoleIDByName(context.Background(), roleName)

		require.NoError(t, err)
		assert.Equal(t, "role-123", roleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotFound when role does not exist", func(t *testing.T) {
		roleName := "nonexistent"

		mock.ExpectQuery("SELECT id FROM roles WHERE name = \\$1").
			WithArgs(roleName).
			WillReturnRows(pgxmock.NewRows([]string{"id"}))

		roleID, err := repo.GetRoleIDByName(context.Background(), roleName)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Empty(t, roleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on database failure", func(t *testing.T) {
		roleName := "admin"

		mock.ExpectQuery("SELECT id FROM roles WHERE name = \\$1").
			WithArgs(roleName).
			WillReturnError(errors.New("query failed"))

		roleID, err := repo.GetRoleIDByName(context.Background(), roleName)

		assert.Error(t, err)
		assert.Empty(t, roleID)
		assert.Contains(t, err.Error(), "failed to get role by name")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNewNamespaceRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	log := logger.NewNopLogger()
	repo := NewNamespaceRepository(mock, log)

	require.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// Test legacy aliases.
func TestLegacyAliases(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	log := logger.NewNopLogger()

	// Test that legacy constructor works
	repo := NewServerGroupRepository(mock, log)
	require.NotNil(t, repo)

	// Test legacy method aliases call through correctly
	t.Run("AddServerToGroup calls AddServerToNamespace", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO namespace_members").
			WithArgs("server-1", "ns-1").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.AddServerToGroup(context.Background(), "server-1", "ns-1")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveServerFromGroup calls RemoveServerFromNamespace", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM namespace_members WHERE server_id = \\$1 AND namespace_id = \\$2").
			WithArgs("server-1", "ns-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.RemoveServerFromGroup(context.Background(), "server-1", "ns-1")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetServerGroups calls GetServerNamespaces", func(t *testing.T) {
		mock.ExpectQuery("SELECT namespace_id FROM namespace_members WHERE server_id = \\$1").
			WithArgs("server-1").
			WillReturnRows(pgxmock.NewRows([]string{"namespace_id"}).AddRow("ns-1"))

		groups, err := repo.GetServerGroups(context.Background(), "server-1")
		require.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

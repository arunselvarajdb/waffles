package role

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/pkg/logger"
)

// mockDB implements DBTX for testing
type mockDB struct {
	queryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	execFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, sql, args...)
	}
	return nil, nil
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, sql, args...)
	}
	return &mockRow{}
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, sql, args...)
	}
	return pgconn.NewCommandTag(""), nil
}

// mockRow implements pgx.Row for testing
type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	return pgx.ErrNoRows
}

// mockRows implements pgx.Rows for testing
type mockRows struct {
	data    [][]interface{}
	current int
	closed  bool
}

func (m *mockRows) Close()                                       { m.closed = true }
func (m *mockRows) Err() error                                   { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Conn() *pgx.Conn                              { return nil }
func (m *mockRows) Values() ([]interface{}, error)               { return nil, nil }

func (m *mockRows) Next() bool {
	if m.current < len(m.data) {
		m.current++
		return true
	}
	return false
}

func (m *mockRows) Scan(dest ...interface{}) error {
	if m.current == 0 || m.current > len(m.data) {
		return errors.New("no data")
	}
	row := m.data[m.current-1]
	for i, d := range dest {
		if i < len(row) {
			switch v := d.(type) {
			case *string:
				*v = row[i].(string)
			case *int:
				*v = row[i].(int)
			case *bool:
				*v = row[i].(bool)
			}
		}
	}
	return nil
}

func TestBuiltInRoles(t *testing.T) {
	assert.True(t, builtInRoles["admin"])
	assert.True(t, builtInRoles["operator"])
	assert.True(t, builtInRoles["user"])
	assert.True(t, builtInRoles["readonly"])
	assert.False(t, builtInRoles["custom"])
}

func TestErrRoleNotFound(t *testing.T) {
	assert.Error(t, ErrRoleNotFound)
	assert.Equal(t, "role not found", ErrRoleNotFound.Error())
}

func TestErrBuiltInRole(t *testing.T) {
	assert.Error(t, ErrBuiltInRole)
	assert.Equal(t, "cannot delete built-in role", ErrBuiltInRole.Error())
}

func TestErrRoleNameExists(t *testing.T) {
	assert.Error(t, ErrRoleNameExists)
	assert.Equal(t, "role name already exists", ErrRoleNameExists.Error())
}

func TestRole_IsBuiltIn(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		want     bool
	}{
		{"admin is built-in", "admin", true},
		{"operator is built-in", "operator", true},
		{"user is built-in", "user", true},
		{"readonly is built-in", "readonly", true},
		{"custom is not built-in", "custom", false},
		{"empty is not built-in", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, builtInRoles[tt.roleName])
		})
	}
}

func TestService_List_Empty(t *testing.T) {
	db := &mockDB{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
	}
	svc := &Service{db: db, logger: logger.NewNop()}

	roles, err := svc.List(context.Background())

	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestService_List_WithRoles(t *testing.T) {
	db := &mockDB{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &mockRows{
				data: [][]interface{}{
					{"role-1", "admin", "Administrator", "2024-01-01", 5},
					{"role-2", "custom", "Custom Role", "2024-01-02", 2},
				},
			}, nil
		},
	}
	svc := &Service{db: db, logger: logger.NewNop()}

	roles, err := svc.List(context.Background())

	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.Equal(t, "admin", roles[0].Name)
	assert.True(t, roles[0].IsBuiltIn)
	assert.Equal(t, "custom", roles[1].Name)
	assert.False(t, roles[1].IsBuiltIn)
}

func TestService_List_QueryError(t *testing.T) {
	db := &mockDB{
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, errors.New("database error")
		},
	}
	svc := &Service{db: db, logger: logger.NewNop()}

	roles, err := svc.List(context.Background())

	require.Error(t, err)
	assert.Nil(t, roles)
	assert.Contains(t, err.Error(), "failed to list roles")
}

func TestService_GetByID_NotFound(t *testing.T) {
	db := &mockDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					return pgx.ErrNoRows
				},
			}
		},
	}
	svc := &Service{db: db, logger: logger.NewNop()}

	result, err := svc.GetByID(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRoleNotFound))
	assert.Nil(t, result)
}

func TestService_Delete_BuiltInRole(t *testing.T) {
	db := &mockDB{
		queryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					// Simulate returning built-in role
					*dest[0].(*string) = "role-id"
					*dest[1].(*string) = "admin" // Built-in role
					*dest[2].(*string) = "Administrator"
					*dest[3].(*string) = "2024-01-01"
					*dest[4].(*int) = 5
					return nil
				},
			}
		},
		queryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
	}
	svc := &Service{db: db, logger: logger.NewNop()}

	err := svc.Delete(context.Background(), "admin-role-id")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuiltInRole))
}

func TestCreateRequest_Validation(t *testing.T) {
	req := CreateRequest{
		Name:        "test-role",
		Description: "Test description",
		Permissions: []string{"perm-1", "perm-2"},
	}

	assert.Equal(t, "test-role", req.Name)
	assert.Equal(t, "Test description", req.Description)
	assert.Len(t, req.Permissions, 2)
}

func TestUpdateRequest_Validation(t *testing.T) {
	desc := "Updated description"
	req := UpdateRequest{
		Description: &desc,
		Permissions: []string{"perm-1"},
	}

	assert.NotNil(t, req.Description)
	assert.Equal(t, "Updated description", *req.Description)
	assert.Len(t, req.Permissions, 1)
}

func TestPermission_Structure(t *testing.T) {
	perm := Permission{
		ID:          "perm-1",
		Name:        "servers.read",
		Resource:    "servers",
		Action:      "read",
		Description: "Read server details",
	}

	assert.Equal(t, "perm-1", perm.ID)
	assert.Equal(t, "servers.read", perm.Name)
	assert.Equal(t, "servers", perm.Resource)
	assert.Equal(t, "read", perm.Action)
}

func TestRole_Structure(t *testing.T) {
	role := Role{
		ID:          "role-1",
		Name:        "admin",
		Description: "Administrator",
		CreatedAt:   "2024-01-01",
		UserCount:   5,
		IsBuiltIn:   true,
	}

	assert.Equal(t, "role-1", role.ID)
	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, 5, role.UserCount)
	assert.True(t, role.IsBuiltIn)
}

func TestRoleWithPermissions_Structure(t *testing.T) {
	rwp := RoleWithPermissions{
		Role: &Role{
			ID:   "role-1",
			Name: "admin",
		},
		Permissions: []*Permission{
			{ID: "perm-1", Name: "servers.read"},
			{ID: "perm-2", Name: "servers.write"},
		},
	}

	assert.NotNil(t, rwp.Role)
	assert.Len(t, rwp.Permissions, 2)
}

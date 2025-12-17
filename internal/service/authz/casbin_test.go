package authz

import (
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

func TestNewCasbinServiceWithDefaults(t *testing.T) {
	log := logger.NewNopLogger()

	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.NotNil(t, svc.GetEnforcer())
}

func TestCasbinService_Enforce(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	tests := []struct {
		name     string
		sub      string
		obj      string
		act      string
		expected bool
	}{
		// Admin role tests
		{
			name:     "admin can access all endpoints",
			sub:      "admin",
			obj:      "/api/v1/users",
			act:      "GET",
			expected: true,
		},
		{
			name:     "admin can delete users",
			sub:      "admin",
			obj:      "/api/v1/users/123",
			act:      "DELETE",
			expected: true,
		},
		{
			name:     "admin can manage servers (inherited from operator)",
			sub:      "admin",
			obj:      "/api/v1/servers",
			act:      "POST",
			expected: true,
		},

		// Operator role tests
		{
			name:     "operator can list servers",
			sub:      "operator",
			obj:      "/api/v1/servers",
			act:      "GET",
			expected: true,
		},
		{
			name:     "operator can create servers",
			sub:      "operator",
			obj:      "/api/v1/servers",
			act:      "POST",
			expected: true,
		},
		{
			name:     "operator can delete servers",
			sub:      "operator",
			obj:      "/api/v1/servers/123",
			act:      "DELETE",
			expected: true,
		},
		{
			name:     "operator can use gateway",
			sub:      "operator",
			obj:      "/api/v1/gateway/server-1",
			act:      "POST",
			expected: true,
		},
		{
			name:     "operator cannot manage users",
			sub:      "operator",
			obj:      "/api/v1/users",
			act:      "POST",
			expected: false,
		},

		// Viewer role tests
		{
			name:     "viewer can read servers",
			sub:      "viewer",
			obj:      "/api/v1/servers",
			act:      "GET",
			expected: true,
		},
		{
			name:     "viewer can read specific server",
			sub:      "viewer",
			obj:      "/api/v1/servers/123",
			act:      "GET",
			expected: true,
		},
		{
			name:     "viewer cannot create servers",
			sub:      "viewer",
			obj:      "/api/v1/servers",
			act:      "POST",
			expected: false,
		},
		{
			name:     "viewer cannot delete servers",
			sub:      "viewer",
			obj:      "/api/v1/servers/123",
			act:      "DELETE",
			expected: false,
		},
		{
			name:     "viewer can access me endpoint (inherited from user)",
			sub:      "viewer",
			obj:      "/api/v1/me",
			act:      "GET",
			expected: true,
		},

		// User role tests
		{
			name:     "user can access me endpoint",
			sub:      "user",
			obj:      "/api/v1/me",
			act:      "GET",
			expected: true,
		},
		{
			name:     "user can update profile",
			sub:      "user",
			obj:      "/api/v1/me",
			act:      "PUT",
			expected: true,
		},
		{
			name:     "user can list api keys",
			sub:      "user",
			obj:      "/api/v1/api-keys",
			act:      "GET",
			expected: true,
		},
		{
			name:     "user can create api keys",
			sub:      "user",
			obj:      "/api/v1/api-keys",
			act:      "POST",
			expected: true,
		},
		{
			name:     "user can delete own api keys",
			sub:      "user",
			obj:      "/api/v1/api-keys/key-123",
			act:      "DELETE",
			expected: true,
		},
		{
			name:     "user cannot access servers",
			sub:      "user",
			obj:      "/api/v1/servers",
			act:      "GET",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := svc.Enforce(tt.sub, tt.obj, tt.act)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, allowed)
		})
	}
}

func TestCasbinService_RoleHierarchy(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Test role hierarchy: admin > operator > viewer > user
	enforcer := svc.GetEnforcer()

	// Admin inherits from operator
	roles, err := enforcer.GetImplicitRolesForUser("admin")
	require.NoError(t, err)
	assert.Contains(t, roles, "operator")

	// Operator inherits from viewer
	roles, err = enforcer.GetImplicitRolesForUser("operator")
	require.NoError(t, err)
	assert.Contains(t, roles, "viewer")

	// Viewer inherits from user
	roles, err = enforcer.GetImplicitRolesForUser("viewer")
	require.NoError(t, err)
	assert.Contains(t, roles, "user")
}

func TestCasbinService_AddPolicy(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Add a new policy
	added, err := svc.AddPolicy("custom_role", "/api/v1/custom", "GET")
	assert.NoError(t, err)
	assert.True(t, added)

	// Verify the policy works
	allowed, err := svc.Enforce("custom_role", "/api/v1/custom", "GET")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Verify other actions are not allowed
	allowed, err = svc.Enforce("custom_role", "/api/v1/custom", "POST")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCasbinService_RemovePolicy(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Add a policy
	_, err = svc.AddPolicy("temp_role", "/api/v1/temp", "GET")
	require.NoError(t, err)

	// Verify it works
	allowed, err := svc.Enforce("temp_role", "/api/v1/temp", "GET")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Remove the policy
	removed, err := svc.RemovePolicy("temp_role", "/api/v1/temp", "GET")
	assert.NoError(t, err)
	assert.True(t, removed)

	// Verify it no longer works
	allowed, err = svc.Enforce("temp_role", "/api/v1/temp", "GET")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCasbinService_AddRoleForUser(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Add a role for a user
	added, err := svc.AddRoleForUser("user-123", "viewer")
	assert.NoError(t, err)
	assert.True(t, added)

	// Verify the user has the role
	hasRole, err := svc.HasRoleForUser("user-123", "viewer")
	assert.NoError(t, err)
	assert.True(t, hasRole)

	// Verify the user can access viewer endpoints
	allowed, err := svc.Enforce("user-123", "/api/v1/servers", "GET")
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestCasbinService_RemoveRoleForUser(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Add a role for a user
	_, err = svc.AddRoleForUser("user-456", "operator")
	require.NoError(t, err)

	// Verify access
	allowed, err := svc.Enforce("user-456", "/api/v1/servers", "POST")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Remove the role
	removed, err := svc.RemoveRoleForUser("user-456", "operator")
	assert.NoError(t, err)
	assert.True(t, removed)

	// Verify access is revoked
	allowed, err = svc.Enforce("user-456", "/api/v1/servers", "POST")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCasbinService_GetRolesForUser(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	// Add roles for a user
	_, err = svc.AddRoleForUser("user-789", "viewer")
	require.NoError(t, err)
	_, err = svc.AddRoleForUser("user-789", "operator")
	require.NoError(t, err)

	// Get roles
	roles, err := svc.GetRolesForUser("user-789")
	assert.NoError(t, err)
	assert.Contains(t, roles, "viewer")
	assert.Contains(t, roles, "operator")
}

func TestCasbinService_GetEnforcer(t *testing.T) {
	log := logger.NewNopLogger()
	svc, err := NewCasbinServiceWithDefaults(log)
	require.NoError(t, err)

	enforcer := svc.GetEnforcer()
	require.NotNil(t, enforcer)

	// Verify we can use the enforcer directly
	policies, err := enforcer.GetPolicy()
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)
}

func TestKeyMatch2Pattern(t *testing.T) {
	// Test keyMatch2 pattern matching - each test uses a fresh enforcer
	tests := []struct {
		name     string
		policy   string
		request  string
		expected bool
	}{
		{
			name:     "exact match",
			policy:   "/api/v1/servers",
			request:  "/api/v1/servers",
			expected: true,
		},
		{
			name:     "wildcard single segment",
			policy:   "/api/v1/servers/*",
			request:  "/api/v1/servers/123",
			expected: true,
		},
		{
			name:     "wildcard multiple segments",
			policy:   "/api/v1/*",
			request:  "/api/v1/servers/123",
			expected: true,
		},
		{
			name:     "no match different path",
			policy:   "/api/v1/servers",
			request:  "/api/v1/users",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh enforcer for each test to avoid policy leakage
			log := logger.NewNopLogger()
			m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
`)
			require.NoError(t, err)

			enforcer, err := casbin.NewEnforcer(m)
			require.NoError(t, err)
			_ = log // silence unused warning

			// Add the test policy
			_, err = enforcer.AddPolicy("test_role", tt.policy, "GET")
			require.NoError(t, err)

			allowed, err := enforcer.Enforce("test_role", tt.request, "GET")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, allowed)
		})
	}
}

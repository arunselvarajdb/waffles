package authz

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"

	"github.com/waffles/waffles/pkg/logger"
)

// CasbinService wraps the Casbin enforcer with additional functionality
type CasbinService struct {
	enforcer *casbin.Enforcer
	logger   logger.Logger
}

// Config contains configuration for the Casbin service
type Config struct {
	ModelPath  string // Path to casbin_model.conf
	PolicyPath string // Path to casbin_policy.csv
}

// NewCasbinService creates a new Casbin service with file-based policies
func NewCasbinService(cfg Config, log logger.Logger) (*CasbinService, error) {
	// Load the model from file
	enforcer, err := casbin.NewEnforcer(cfg.ModelPath, cfg.PolicyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	log.Info().
		Str("model_path", cfg.ModelPath).
		Str("policy_path", cfg.PolicyPath).
		Msg("Casbin enforcer initialized")

	return &CasbinService{
		enforcer: enforcer,
		logger:   log,
	}, nil
}

// NewCasbinServiceWithDefaults creates a Casbin service with embedded default policies
// This is useful for development and testing
func NewCasbinServiceWithDefaults(log logger.Logger) (*CasbinService, error) {
	// Define the model as a string
	modelText := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	// Add default policies
	policies := [][]string{
		// Admin role - full access
		{"admin", "/api/v1/*", "*"},
		{"admin", "/api/v1/users", "*"},
		{"admin", "/api/v1/users/*", "*"},

		// Operator role - manage servers and gateway
		{"operator", "/api/v1/servers", "*"},
		{"operator", "/api/v1/servers/*", "*"},
		{"operator", "/api/v1/gateway/*", "*"},
		{"operator", "/api/v1/health/*", "GET"},
		{"operator", "/api/v1/audit", "GET"},
		{"operator", "/api/v1/audit/*", "GET"},

		// Viewer role - read-only access
		{"viewer", "/api/v1/servers", "GET"},
		{"viewer", "/api/v1/servers/*", "GET"},
		{"viewer", "/api/v1/health/*", "GET"},

		// User role - basic access
		{"user", "/api/v1/me", "GET"},
		{"user", "/api/v1/me", "PUT"},
		{"user", "/api/v1/api-keys", "GET"},
		{"user", "/api/v1/api-keys", "POST"},
		{"user", "/api/v1/api-keys/*", "DELETE"},
	}

	for _, p := range policies {
		_, err := enforcer.AddPolicy(p)
		if err != nil {
			log.Warn().Err(err).Str("policy", strings.Join(p, ", ")).Msg("Failed to add policy")
		}
	}

	// Add role hierarchy
	roleHierarchy := [][]string{
		{"admin", "operator"},
		{"operator", "viewer"},
		{"viewer", "user"},
	}

	for _, g := range roleHierarchy {
		_, err := enforcer.AddGroupingPolicy(g)
		if err != nil {
			log.Warn().Err(err).Str("grouping", strings.Join(g, ", ")).Msg("Failed to add grouping policy")
		}
	}

	log.Info().Msg("Casbin enforcer initialized with default policies")

	return &CasbinService{
		enforcer: enforcer,
		logger:   log,
	}, nil
}

// GetEnforcer returns the underlying Casbin enforcer
func (s *CasbinService) GetEnforcer() *casbin.Enforcer {
	return s.enforcer
}

// Enforce checks if a subject (role) can perform an action on an object (path)
func (s *CasbinService) Enforce(sub, obj, act string) (bool, error) {
	return s.enforcer.Enforce(sub, obj, act)
}

// AddPolicy adds a new policy rule
func (s *CasbinService) AddPolicy(sub, obj, act string) (bool, error) {
	return s.enforcer.AddPolicy(sub, obj, act)
}

// RemovePolicy removes a policy rule
func (s *CasbinService) RemovePolicy(sub, obj, act string) (bool, error) {
	return s.enforcer.RemovePolicy(sub, obj, act)
}

// AddRoleForUser assigns a role to a user
func (s *CasbinService) AddRoleForUser(user, role string) (bool, error) {
	return s.enforcer.AddGroupingPolicy(user, role)
}

// RemoveRoleForUser removes a role from a user
func (s *CasbinService) RemoveRoleForUser(user, role string) (bool, error) {
	return s.enforcer.RemoveGroupingPolicy(user, role)
}

// GetRolesForUser gets all roles for a user
func (s *CasbinService) GetRolesForUser(user string) ([]string, error) {
	return s.enforcer.GetRolesForUser(user)
}

// HasRoleForUser checks if a user has a specific role
func (s *CasbinService) HasRoleForUser(user, role string) (bool, error) {
	return s.enforcer.HasRoleForUser(user, role)
}

// ReloadPolicy reloads the policy from file
func (s *CasbinService) ReloadPolicy() error {
	return s.enforcer.LoadPolicy()
}

// SavePolicy saves the current policy to file
func (s *CasbinService) SavePolicy() error {
	return s.enforcer.SavePolicy()
}

// LoadPolicyFromFile loads policies from a CSV file
func (s *CasbinService) LoadPolicyFromFile(path string) error {
	adapter := fileadapter.NewAdapter(path)
	s.enforcer.SetAdapter(adapter)
	return s.enforcer.LoadPolicy()
}

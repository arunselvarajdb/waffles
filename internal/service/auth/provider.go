package auth

import (
	"context"
	"errors"
)

// Common errors for authentication providers
var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUserNotFound        = errors.New("user not found")
	ErrAccountLocked       = errors.New("account is locked")
	ErrAccountDisabled     = errors.New("account is disabled")
	ErrPasswordExpired     = errors.New("password has expired")
	ErrProviderUnavailable = errors.New("authentication provider is unavailable")
)

// UserInfo represents authenticated user information from any provider
type UserInfo struct {
	// Unique identifier from the provider (LDAP DN, local user ID, etc.)
	ExternalID string
	// Username used for login
	Username string
	// Email address
	Email string
	// Display name
	DisplayName string
	// Roles mapped from provider groups
	Roles []string
	// Raw groups from the provider (before mapping)
	Groups []string
	// Provider name (ldap, local, oauth)
	Provider string
	// Additional provider-specific attributes
	Attributes map[string]interface{}
}

// Provider defines the interface for authentication providers
// Multiple providers can be enabled simultaneously (OAuth, LDAP, Local)
type Provider interface {
	// Name returns the provider identifier (e.g., "ldap", "local", "oauth")
	Name() string

	// IsEnabled returns whether this provider is configured and active
	IsEnabled() bool

	// Authenticate validates credentials and returns user info
	// Returns ErrInvalidCredentials if authentication fails
	Authenticate(ctx context.Context, username, password string) (*UserInfo, error)

	// GetUser retrieves user information by external ID
	// Used for session refresh and user lookup
	GetUser(ctx context.Context, externalID string) (*UserInfo, error)
}

// ProviderRegistry manages multiple authentication providers
type ProviderRegistry struct {
	providers map[string]Provider
	order     []string // Priority order for authentication attempts
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
		order:     make([]string, 0),
	}
}

// Register adds a provider to the registry
// Providers are tried in registration order during authentication
func (r *ProviderRegistry) Register(p Provider) {
	if p == nil || !p.IsEnabled() {
		return
	}
	name := p.Name()
	r.providers[name] = p
	r.order = append(r.order, name)
}

// Get returns a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names in priority order
func (r *ProviderRegistry) List() []string {
	return r.order
}

// Authenticate tries each provider in order until one succeeds
// Returns the first successful authentication result
func (r *ProviderRegistry) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	if len(r.providers) == 0 {
		return nil, errors.New("no authentication providers configured")
	}

	var lastErr error
	for _, name := range r.order {
		p := r.providers[name]
		user, err := p.Authenticate(ctx, username, password)
		if err == nil {
			return user, nil
		}

		// Track the last error for reporting
		lastErr = err

		// Stop trying other providers for certain errors
		if errors.Is(err, ErrAccountLocked) || errors.Is(err, ErrAccountDisabled) {
			return nil, err
		}
	}

	// Return the last error if all providers failed
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrInvalidCredentials
}

// AuthenticateWithProvider attempts authentication with a specific provider
func (r *ProviderRegistry) AuthenticateWithProvider(ctx context.Context, providerName, username, password string) (*UserInfo, error) {
	p, ok := r.providers[providerName]
	if !ok {
		return nil, errors.New("unknown authentication provider: " + providerName)
	}
	return p.Authenticate(ctx, username, password)
}

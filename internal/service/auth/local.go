package auth

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// UserRepository defines the interface for user database operations
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	ValidatePassword(ctx context.Context, email, password string) (*domain.User, error)
	UpdateFailedAttempts(ctx context.Context, userID string, attempts int, lockedUntil *time.Time) error
	GetFailedAttempts(ctx context.Context, userID string) (int, *time.Time, error)
}

// LocalProvider implements Provider interface for local database authentication
type LocalProvider struct {
	config   config.LocalAuthConfig
	userRepo UserRepository
	logger   logger.Logger

	// In-memory failed attempt tracking (for lockout between DB calls)
	failedAttempts map[string]*loginAttempt
	mu             sync.RWMutex
}

type loginAttempt struct {
	count       int
	lastFailed  time.Time
	lockedUntil *time.Time
}

// NewLocalProvider creates a new local database authentication provider
func NewLocalProvider(cfg config.LocalAuthConfig, userRepo UserRepository, log logger.Logger) *LocalProvider {
	// Set defaults
	if cfg.Lockout.MaxAttempts == 0 {
		cfg.Lockout.MaxAttempts = 5
	}
	if cfg.Lockout.Duration == 0 {
		cfg.Lockout.Duration = 15 * time.Minute
	}
	if cfg.Lockout.ResetAfter == 0 {
		cfg.Lockout.ResetAfter = 24 * time.Hour
	}
	if cfg.PasswordPolicy.MinLength == 0 {
		cfg.PasswordPolicy.MinLength = 12
	}

	return &LocalProvider{
		config:         cfg,
		userRepo:       userRepo,
		logger:         log,
		failedAttempts: make(map[string]*loginAttempt),
	}
}

// Name returns the provider identifier
func (p *LocalProvider) Name() string {
	return "local"
}

// IsEnabled returns whether local authentication is configured and active
func (p *LocalProvider) IsEnabled() bool {
	return p.config.Enabled
}

// Authenticate validates credentials against the local database
func (p *LocalProvider) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	if !p.IsEnabled() {
		return nil, ErrProviderUnavailable
	}

	// Check if account is locked
	if p.isLocked(username) {
		p.logger.Warn().Str("username", username).Msg("Login attempt on locked account")
		return nil, ErrAccountLocked
	}

	// Attempt authentication
	user, err := p.userRepo.ValidatePassword(ctx, username, password)
	if err != nil {
		p.recordFailedAttempt(ctx, username)
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	// Reset failed attempts on successful login
	p.resetFailedAttempts(ctx, username)

	// Get user roles
	roles, err := p.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		p.logger.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
		roles = []string{"viewer"} // Default role
	}

	p.logger.Info().
		Str("username", user.Email).
		Str("user_id", user.ID).
		Str("roles", strings.Join(roles, ",")).
		Msg("Local authentication successful")

	return &UserInfo{
		ExternalID:  user.ID,
		Username:    user.Email,
		Email:       user.Email,
		DisplayName: user.Name,
		Roles:       roles,
		Provider:    "local",
		Attributes: map[string]interface{}{
			"user_id": user.ID,
		},
	}, nil
}

// GetUser retrieves user information by ID
func (p *LocalProvider) GetUser(ctx context.Context, externalID string) (*UserInfo, error) {
	if !p.IsEnabled() {
		return nil, ErrProviderUnavailable
	}

	user, err := p.userRepo.GetByID(ctx, externalID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	roles, err := p.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		roles = []string{"viewer"}
	}

	return &UserInfo{
		ExternalID:  user.ID,
		Username:    user.Email,
		Email:       user.Email,
		DisplayName: user.Name,
		Roles:       roles,
		Provider:    "local",
		Attributes: map[string]interface{}{
			"user_id": user.ID,
		},
	}, nil
}

// isLocked checks if an account is currently locked
func (p *LocalProvider) isLocked(username string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	attempt, exists := p.failedAttempts[username]
	if !exists {
		return false
	}

	if attempt.lockedUntil == nil {
		return false
	}

	if time.Now().After(*attempt.lockedUntil) {
		// Lock has expired
		return false
	}

	return true
}

// recordFailedAttempt tracks a failed login attempt
func (p *LocalProvider) recordFailedAttempt(ctx context.Context, username string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	attempt, exists := p.failedAttempts[username]
	if !exists {
		attempt = &loginAttempt{}
		p.failedAttempts[username] = attempt
	}

	// Reset count if last failure was long ago
	if time.Since(attempt.lastFailed) > p.config.Lockout.ResetAfter {
		attempt.count = 0
	}

	attempt.count++
	attempt.lastFailed = time.Now()

	// Check if we should lock the account
	if attempt.count >= p.config.Lockout.MaxAttempts {
		lockUntil := time.Now().Add(p.config.Lockout.Duration)
		attempt.lockedUntil = &lockUntil

		p.logger.Warn().
			Str("username", username).
			Int("attempts", attempt.count).
			Str("locked_until", lockUntil.Format(time.RFC3339)).
			Msg("Account locked due to failed login attempts")
	}
}

// resetFailedAttempts clears the failed attempt counter
func (p *LocalProvider) resetFailedAttempts(ctx context.Context, username string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.failedAttempts, username)
}

// ValidatePassword checks if a password meets the configured policy
func (p *LocalProvider) ValidatePassword(password string) error {
	policy := p.config.PasswordPolicy

	if len(password) < policy.MinLength {
		return &PasswordPolicyError{
			Message:   "password is too short",
			MinLength: policy.MinLength,
		}
	}

	if policy.RequireUppercase && !containsUppercase(password) {
		return &PasswordPolicyError{Message: "password must contain at least one uppercase letter"}
	}

	if policy.RequireLowercase && !containsLowercase(password) {
		return &PasswordPolicyError{Message: "password must contain at least one lowercase letter"}
	}

	if policy.RequireNumber && !containsNumber(password) {
		return &PasswordPolicyError{Message: "password must contain at least one number"}
	}

	if policy.RequireSpecial && !containsSpecial(password) {
		return &PasswordPolicyError{Message: "password must contain at least one special character"}
	}

	return nil
}

// PasswordPolicyError represents a password policy violation
type PasswordPolicyError struct {
	Message   string
	MinLength int
}

func (e *PasswordPolicyError) Error() string {
	return e.Message
}

// Helper functions for password validation
func containsUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSpecial(s string) bool {
	specials := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	for _, r := range s {
		for _, s := range specials {
			if r == s {
				return true
			}
		}
	}
	return false
}

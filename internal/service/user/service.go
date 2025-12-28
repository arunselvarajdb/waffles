package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// Repository defines the interface for user data persistence
type Repository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	AssignRole(ctx context.Context, userID, roleName string) error
	RemoveRole(ctx context.Context, userID, roleName string) error
}

// Service handles user business logic
type Service struct {
	repo   Repository
	logger logger.Logger
}

// NewService creates a new user service
func NewService(repo Repository, log logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log.With().Str("service", "user").Logger(),
	}
}

// UserWithRoles represents a user with their roles
type UserWithRoles struct {
	*domain.User
	Roles []string `json:"roles"`
}

// ListRequest represents pagination parameters
type ListRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size"`
	Search   string `form:"search" json:"search"`
}

// ListResponse represents a paginated list of users
type ListResponse struct {
	Users      []*UserWithRoles `json:"users"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// CreateRequest represents the data for creating a user
type CreateRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password"` // Optional - if empty, generate temp password
	Role     string `json:"role"`     // Optional - defaults to "user"
}

// CreateResponse includes the created user and temp password if generated
type CreateResponse struct {
	User         *UserWithRoles `json:"user"`
	TempPassword string         `json:"temp_password,omitempty"`
}

// UpdateRequest represents the data for updating a user
type UpdateRequest struct {
	Email    *string `json:"email,omitempty"`
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// RoleAssignment represents role assignment data
type RoleAssignment struct {
	Roles []string `json:"roles" binding:"required"`
}

// List returns a paginated list of users with their roles
func (s *Service) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	users, total, err := s.repo.List(ctx, req.PageSize, offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Fetch roles for each user
	usersWithRoles := make([]*UserWithRoles, len(users))
	for i, user := range users {
		roles, err := s.repo.GetUserRoles(ctx, user.ID)
		if err != nil {
			s.logger.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
			roles = []string{}
		}
		usersWithRoles[i] = &UserWithRoles{
			User:  user,
			Roles: roles,
		}
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return &ListResponse{
		Users:      usersWithRoles,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByID retrieves a user by ID with their roles
func (s *Service) GetByID(ctx context.Context, id string) (*UserWithRoles, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		s.logger.Error().Err(err).Str("user_id", id).Msg("Failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		s.logger.Warn().Err(err).Str("user_id", user.ID).Msg("Failed to get user roles")
		roles = []string{}
	}

	return &UserWithRoles{
		User:  user,
		Roles: roles,
	}, nil
}

// Create creates a new user
func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	// Check if email already exists
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Generate temp password if not provided
	tempPassword := ""
	password := req.Password
	if password == "" {
		tempPassword, err = generateTempPassword(12)
		if err != nil {
			return nil, fmt.Errorf("failed to generate temp password: %w", err)
		}
		password = tempPassword
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		Name:         strings.TrimSpace(req.Name),
		PasswordHash: string(passwordHash),
		AuthProvider: domain.AuthProviderLocal,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		s.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign role (default to "user" if not specified)
	role := req.Role
	if role == "" {
		role = "user"
	}
	if err := s.repo.AssignRole(ctx, user.ID, role); err != nil {
		s.logger.Warn().Err(err).Str("user_id", user.ID).Str("role", role).Msg("Failed to assign role")
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	s.logger.Info().
		Str("user_id", user.ID).
		Str("email", user.Email).
		Str("role", role).
		Msg("User created by admin")

	return &CreateResponse{
		User: &UserWithRoles{
			User:  user,
			Roles: roles,
		},
		TempPassword: tempPassword,
	}, nil
}

// Update updates a user's information
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*UserWithRoles, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Apply updates
	if req.Email != nil {
		user.Email = strings.ToLower(strings.TrimSpace(*req.Email))
	}
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, user); err != nil {
		s.logger.Error().Err(err).Str("user_id", id).Msg("Failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	roles, _ := s.repo.GetUserRoles(ctx, user.ID)

	s.logger.Info().Str("user_id", id).Msg("User updated by admin")

	return &UserWithRoles{
		User:  user,
		Roles: roles,
	}, nil
}

// Deactivate deactivates a user (soft delete)
func (s *Service) Deactivate(ctx context.Context, id string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.IsActive = false
	if err := s.repo.Update(ctx, user); err != nil {
		s.logger.Error().Err(err).Str("user_id", id).Msg("Failed to deactivate user")
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	s.logger.Info().Str("user_id", id).Msg("User deactivated by admin")
	return nil
}

// ResetPassword generates a new temp password for a user
func (s *Service) ResetPassword(ctx context.Context, id string) (string, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrUserNotFound
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new temp password
	tempPassword, err := generateTempPassword(12)
	if err != nil {
		return "", fmt.Errorf("failed to generate temp password: %w", err)
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, user.ID, string(passwordHash)); err != nil {
		s.logger.Error().Err(err).Str("user_id", id).Msg("Failed to reset password")
		return "", fmt.Errorf("failed to reset password: %w", err)
	}

	s.logger.Info().Str("user_id", id).Msg("User password reset by admin")
	return tempPassword, nil
}

// UpdateRoles updates a user's roles
func (s *Service) UpdateRoles(ctx context.Context, userID string, roles []string) (*UserWithRoles, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get current roles
	currentRoles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current roles: %w", err)
	}

	// Create a map for easy lookup
	currentMap := make(map[string]bool)
	for _, r := range currentRoles {
		currentMap[r] = true
	}
	newMap := make(map[string]bool)
	for _, r := range roles {
		newMap[r] = true
	}

	// Remove roles that are no longer needed
	for _, r := range currentRoles {
		if !newMap[r] {
			if err := s.repo.RemoveRole(ctx, userID, r); err != nil {
				s.logger.Warn().Err(err).Str("user_id", userID).Str("role", r).Msg("Failed to remove role")
			}
		}
	}

	// Add new roles
	for _, r := range roles {
		if !currentMap[r] {
			if err := s.repo.AssignRole(ctx, userID, r); err != nil {
				s.logger.Warn().Err(err).Str("user_id", userID).Str("role", r).Msg("Failed to assign role")
			}
		}
	}

	// Fetch updated roles
	updatedRoles, _ := s.repo.GetUserRoles(ctx, userID)

	s.logger.Info().Str("user_id", userID).Any("roles", roles).Msg("User roles updated by admin")

	return &UserWithRoles{
		User:  user,
		Roles: updatedRoles,
	}, nil
}

// generateTempPassword generates a random password
func generateTempPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

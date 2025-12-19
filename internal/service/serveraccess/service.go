package serveraccess

import (
	"context"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/internal/repository"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// Service handles server access control logic
type Service struct {
	namespaceRepo *repository.NamespaceRepository
	logger        logger.Logger
}

// NewService creates a new server access service
func NewService(namespaceRepo *repository.NamespaceRepository, log logger.Logger) *Service {
	return &Service{
		namespaceRepo: namespaceRepo,
		logger:        log,
	}
}

// IsAdmin checks if any of the given roles is "admin"
func (s *Service) IsAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// GetAccessibleServerIDs returns the IDs of servers the given roles can access at the specified level.
// Returns nil for admins (meaning all servers are accessible).
// Returns an empty slice if no servers are accessible.
func (s *Service) GetAccessibleServerIDs(ctx context.Context, roles []string, level domain.AccessLevel) ([]string, error) {
	// Admin bypass - return nil to indicate access to all servers
	if s.IsAdmin(roles) {
		s.logger.Debug().Any("roles", roles).Msg("Admin role detected, granting access to all servers")
		return nil, nil
	}

	// Query accessible servers for these roles
	serverIDs, err := s.namespaceRepo.GetAccessibleServerIDs(ctx, roles, level)
	if err != nil {
		s.logger.Error().Err(err).
			Any("roles", roles).
			Str("access_level", string(level)).
			Msg("Failed to get accessible server IDs")
		return nil, err
	}

	s.logger.Debug().
		Any("roles", roles).
		Str("access_level", string(level)).
		Int("accessible_count", len(serverIDs)).
		Msg("Retrieved accessible servers for roles")

	return serverIDs, nil
}

// CanAccessServer checks if the given roles can access a specific server at the specified level.
// Returns true for admins (they can access all servers).
func (s *Service) CanAccessServer(ctx context.Context, roles []string, serverID string, level domain.AccessLevel) (bool, error) {
	// Admin bypass
	if s.IsAdmin(roles) {
		return true, nil
	}

	// Get accessible server IDs
	accessibleIDs, err := s.GetAccessibleServerIDs(ctx, roles, level)
	if err != nil {
		return false, err
	}

	// Check if the server is in the accessible list
	for _, id := range accessibleIDs {
		if id == serverID {
			s.logger.Debug().
				Str("server_id", serverID).
				Any("roles", roles).
				Str("access_level", string(level)).
				Msg("Access granted to server")
			return true, nil
		}
	}

	s.logger.Debug().
		Str("server_id", serverID).
		Any("roles", roles).
		Str("access_level", string(level)).
		Msg("Access denied to server")
	return false, nil
}

// FilterServerIDs filters a list of server IDs to only those accessible by the given roles.
// Returns the original list if roles include "admin".
func (s *Service) FilterServerIDs(ctx context.Context, roles []string, serverIDs []string, level domain.AccessLevel) ([]string, error) {
	// Admin bypass
	if s.IsAdmin(roles) {
		return serverIDs, nil
	}

	// Get accessible server IDs
	accessibleIDs, err := s.GetAccessibleServerIDs(ctx, roles, level)
	if err != nil {
		return nil, err
	}

	// Create a set for fast lookup
	accessibleSet := make(map[string]bool)
	for _, id := range accessibleIDs {
		accessibleSet[id] = true
	}

	// Filter the input list
	var filtered []string
	for _, id := range serverIDs {
		if accessibleSet[id] {
			filtered = append(filtered, id)
		}
	}

	return filtered, nil
}

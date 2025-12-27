package audit

import (
	"context"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// Repository defines the interface for audit log data access.
type Repository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	Get(ctx context.Context, id string) (*domain.AuditLog, error)
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error)
}

// Service handles audit logging operations
type Service struct {
	repo   Repository
	logger logger.Logger
}

// NewService creates a new audit service
func NewService(repo Repository, log logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log,
	}
}

// Log creates a new audit log entry
func (s *Service) Log(ctx context.Context, log *domain.AuditLog) error {
	s.logger.Info().
		Str("request_id", log.RequestID).
		Str("method", log.Method).
		Str("path", log.Path).
		Msg("Attempting to create audit log")

	if err := s.repo.Create(ctx, log); err != nil {
		s.logger.Error().
			Err(err).
			Str("request_id", log.RequestID).
			Str("path", log.Path).
			Msg("Failed to create audit log")
		return err
	}

	s.logger.Info().
		Str("audit_log_id", log.ID).
		Str("request_id", log.RequestID).
		Str("method", log.Method).
		Str("path", log.Path).
		Msg("Audit log created successfully")

	return nil
}

// Get retrieves a single audit log by ID
func (s *Service) Get(ctx context.Context, id string) (*domain.AuditLog, error) {
	log, err := s.repo.Get(ctx, id)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("audit_log_id", id).
			Msg("Failed to get audit log")
		return nil, err
	}

	return log, nil
}

// List retrieves audit logs with filters
func (s *Service) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	// Set default limit if not specified
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	logs, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error().
			Err(err).
			Msg("Failed to list audit logs")
		return nil, err
	}

	s.logger.Debug().
		Int("count", len(logs)).
		Msg("Audit logs retrieved")

	return logs, nil
}

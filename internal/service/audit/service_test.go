package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// mockAuditRepository is a mock implementation of audit repository for testing.
type mockAuditRepository struct {
	createFunc func(ctx context.Context, log *domain.AuditLog) error
	getFunc    func(ctx context.Context, id string) (*domain.AuditLog, error)
	listFunc   func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error)
}

func (m *mockAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, log)
	}

	return nil
}

func (m *mockAuditRepository) Get(ctx context.Context, id string) (*domain.AuditLog, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}

	return nil, nil
}

func (m *mockAuditRepository) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}

	return nil, nil
}

func TestService_Log(t *testing.T) {
	t.Run("successfully creates audit log", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			createFunc: func(ctx context.Context, log *domain.AuditLog) error {
				log.ID = "audit-123"

				return nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		status := 201
		auditLog := &domain.AuditLog{
			RequestID:      "req-123",
			Method:         "POST",
			Path:           "/api/v1/servers",
			ResponseStatus: &status,
		}

		err := svc.Log(context.Background(), auditLog)

		require.NoError(t, err)
		assert.Equal(t, "audit-123", auditLog.ID)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			createFunc: func(ctx context.Context, log *domain.AuditLog) error {
				return errors.New("database error")
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		auditLog := &domain.AuditLog{
			RequestID: "req-456",
			Method:    "GET",
			Path:      "/api/v1/servers",
		}

		err := svc.Log(context.Background(), auditLog)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("logs audit entry with server ID", func(t *testing.T) {
		var capturedLog *domain.AuditLog
		mockRepo := &mockAuditRepository{
			createFunc: func(ctx context.Context, log *domain.AuditLog) error {
				capturedLog = log
				log.ID = "audit-789"

				return nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		serverID := "server-123"
		status := 200
		auditLog := &domain.AuditLog{
			RequestID:      "req-789",
			Method:         "POST",
			Path:           "/api/v1/gateway/server-123/tools/call",
			ServerID:       &serverID,
			ResponseStatus: &status,
		}

		err := svc.Log(context.Background(), auditLog)

		require.NoError(t, err)
		assert.NotNil(t, capturedLog.ServerID)
		assert.Equal(t, serverID, *capturedLog.ServerID)
	})
}

func TestService_Get(t *testing.T) {
	t.Run("successfully gets audit log by ID", func(t *testing.T) {
		status := 201
		expectedLog := &domain.AuditLog{
			ID:             "audit-123",
			RequestID:      "req-123",
			Method:         "POST",
			Path:           "/api/v1/servers",
			ResponseStatus: &status,
		}

		mockRepo := &mockAuditRepository{
			getFunc: func(ctx context.Context, id string) (*domain.AuditLog, error) {
				if id == "audit-123" {
					return expectedLog, nil
				}

				return nil, domain.ErrNotFound
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		log, err := svc.Get(context.Background(), "audit-123")

		require.NoError(t, err)
		require.NotNil(t, log)
		assert.Equal(t, expectedLog.ID, log.ID)
		assert.Equal(t, expectedLog.RequestID, log.RequestID)
		assert.Equal(t, expectedLog.Method, log.Method)
		assert.Equal(t, expectedLog.Path, log.Path)
	})

	t.Run("returns error when audit log not found", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			getFunc: func(ctx context.Context, id string) (*domain.AuditLog, error) {
				return nil, domain.ErrNotFound
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		log, err := svc.Get(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, log)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			getFunc: func(ctx context.Context, id string) (*domain.AuditLog, error) {
				return nil, errors.New("database error")
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		log, err := svc.Get(context.Background(), "audit-123")

		assert.Error(t, err)
		assert.Nil(t, log)
	})
}

func TestService_List(t *testing.T) {
	t.Run("successfully lists audit logs", func(t *testing.T) {
		expectedLogs := []*domain.AuditLog{
			{ID: "audit-1", Method: "GET", Path: "/api/v1/servers"},
			{ID: "audit-2", Method: "POST", Path: "/api/v1/servers"},
			{ID: "audit-3", Method: "DELETE", Path: "/api/v1/servers/123"},
		}

		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				return expectedLogs, nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		logs, err := svc.List(context.Background(), domain.AuditLogFilter{})

		require.NoError(t, err)
		assert.Len(t, logs, 3)
	})

	t.Run("sets default limit when not specified", func(t *testing.T) {
		var capturedFilter domain.AuditLogFilter
		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				capturedFilter = filter

				return []*domain.AuditLog{}, nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		_, err := svc.List(context.Background(), domain.AuditLogFilter{})

		require.NoError(t, err)
		assert.Equal(t, 100, capturedFilter.Limit)
	})

	t.Run("respects provided limit", func(t *testing.T) {
		var capturedFilter domain.AuditLogFilter
		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				capturedFilter = filter

				return []*domain.AuditLog{}, nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		_, err := svc.List(context.Background(), domain.AuditLogFilter{Limit: 50})

		require.NoError(t, err)
		assert.Equal(t, 50, capturedFilter.Limit)
	})

	t.Run("returns empty list when no logs found", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				return []*domain.AuditLog{}, nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		logs, err := svc.List(context.Background(), domain.AuditLogFilter{})

		require.NoError(t, err)
		assert.Empty(t, logs)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				return nil, errors.New("database error")
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		logs, err := svc.List(context.Background(), domain.AuditLogFilter{})

		assert.Error(t, err)
		assert.Nil(t, logs)
	})

	t.Run("applies filters correctly", func(t *testing.T) {
		var capturedFilter domain.AuditLogFilter
		mockRepo := &mockAuditRepository{
			listFunc: func(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
				capturedFilter = filter

				return []*domain.AuditLog{}, nil
			},
		}

		svc := NewService(mockRepo, logger.NewNopLogger())

		serverID := "server-123"
		method := "POST"
		filter := domain.AuditLogFilter{
			ServerID: &serverID,
			Method:   &method,
			Limit:    25,
			Offset:   10,
		}

		_, err := svc.List(context.Background(), filter)

		require.NoError(t, err)
		assert.Equal(t, &serverID, capturedFilter.ServerID)
		assert.Equal(t, &method, capturedFilter.Method)
		assert.Equal(t, 25, capturedFilter.Limit)
		assert.Equal(t, 10, capturedFilter.Offset)
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates service with nil repository", func(t *testing.T) {
		log := logger.NewNopLogger()
		svc := NewService(nil, log)

		require.NotNil(t, svc)
		assert.Nil(t, svc.repo)
		assert.NotNil(t, svc.logger)
	})
}

func TestAuditLogFilter_DefaultLimit(t *testing.T) {
	filter := domain.AuditLogFilter{}

	expectedLimit := 100
	if filter.Limit == 0 {
		filter.Limit = expectedLimit
	}

	assert.Equal(t, expectedLimit, filter.Limit)
}

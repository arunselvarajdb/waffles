package serveraccess

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/mcp-gateway/internal/domain"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// mockNamespaceRepository is a mock implementation of the namespace repository for testing.
type mockNamespaceRepository struct {
	err                 error
	accessibleServerIDs []string
}

func (m *mockNamespaceRepository) GetAccessibleServerIDs(ctx context.Context, roles []string, level domain.AccessLevel) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.accessibleServerIDs, nil
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "admin role present",
			roles:    []string{"admin"},
			expected: true,
		},
		{
			name:     "admin role among others",
			roles:    []string{"viewer", "admin", "operator"},
			expected: true,
		},
		{
			name:     "no admin role",
			roles:    []string{"viewer", "operator"},
			expected: false,
		},
		{
			name:     "empty roles",
			roles:    []string{},
			expected: false,
		},
		{
			name:     "nil roles",
			roles:    nil,
			expected: false,
		},
		{
			name:     "similar but not admin",
			roles:    []string{"administrator", "admin123", "superadmin"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{}
			result := s.IsAdmin(tt.roles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterServerIDs(t *testing.T) {
	tests := []struct {
		name                string
		roles               []string
		inputServerIDs      []string
		accessibleServerIDs []string
		repoError           error
		expected            []string
		expectError         bool
	}{
		{
			name:                "admin bypasses filter",
			roles:               []string{"admin"},
			inputServerIDs:      []string{"server-1", "server-2", "server-3"},
			accessibleServerIDs: []string{}, // not used for admin
			expected:            []string{"server-1", "server-2", "server-3"},
		},
		{
			name:                "filter to accessible servers",
			roles:               []string{"operator"},
			inputServerIDs:      []string{"server-1", "server-2", "server-3"},
			accessibleServerIDs: []string{"server-1", "server-3"},
			expected:            []string{"server-1", "server-3"},
		},
		{
			name:                "no accessible servers",
			roles:               []string{"viewer"},
			inputServerIDs:      []string{"server-1", "server-2"},
			accessibleServerIDs: []string{},
			expected:            nil, // empty slice becomes nil
		},
		{
			name:                "all servers accessible",
			roles:               []string{"operator"},
			inputServerIDs:      []string{"server-1", "server-2"},
			accessibleServerIDs: []string{"server-1", "server-2", "server-3"},
			expected:            []string{"server-1", "server-2"},
		},
		{
			name:                "repository error",
			roles:               []string{"operator"},
			inputServerIDs:      []string{"server-1"},
			accessibleServerIDs: nil,
			repoError:           errors.New("database error"),
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockNamespaceRepository{
				accessibleServerIDs: tt.accessibleServerIDs,
				err:                 tt.repoError,
			}

			svc := NewService(mockRepo, logger.NewNopLogger())
			ctx := context.Background()

			result, err := svc.FilterServerIDs(ctx, tt.roles, tt.inputServerIDs, domain.AccessLevelView)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanAccessServer(t *testing.T) {
	tests := []struct {
		repoError           error
		name                string
		serverID            string
		roles               []string
		accessibleServerIDs []string
		expected            bool
		expectError         bool
	}{
		{
			name:                "admin always has access",
			roles:               []string{"admin"},
			serverID:            "server-123",
			accessibleServerIDs: []string{},
			expected:            true,
		},
		{
			name:                "admin among other roles",
			roles:               []string{"viewer", "admin"},
			serverID:            "server-123",
			accessibleServerIDs: []string{},
			expected:            true,
		},
		{
			name:                "non-admin with access",
			roles:               []string{"operator"},
			serverID:            "server-1",
			accessibleServerIDs: []string{"server-1", "server-2"},
			expected:            true,
		},
		{
			name:                "non-admin without access",
			roles:               []string{"viewer"},
			serverID:            "server-3",
			accessibleServerIDs: []string{"server-1", "server-2"},
			expected:            false,
		},
		{
			name:        "repository error",
			roles:       []string{"operator"},
			serverID:    "server-1",
			repoError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockNamespaceRepository{
				accessibleServerIDs: tt.accessibleServerIDs,
				err:                 tt.repoError,
			}

			svc := NewService(mockRepo, logger.NewNopLogger())
			ctx := context.Background()

			result, err := svc.CanAccessServer(ctx, tt.roles, tt.serverID, domain.AccessLevelView)

			if tt.expectError {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAccessibleServerIDs(t *testing.T) {
	tests := []struct {
		name                string
		roles               []string
		accessibleServerIDs []string
		repoError           error
		expected            []string
		expectError         bool
	}{
		{
			name:     "admin returns nil for all access",
			roles:    []string{"admin"},
			expected: nil,
		},
		{
			name:                "non-admin returns server IDs",
			roles:               []string{"operator"},
			accessibleServerIDs: []string{"server-1", "server-2"},
			expected:            []string{"server-1", "server-2"},
		},
		{
			name:                "empty accessible list",
			roles:               []string{"viewer"},
			accessibleServerIDs: []string{},
			expected:            []string{},
		},
		{
			name:        "repository error",
			roles:       []string{"operator"},
			repoError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockNamespaceRepository{
				accessibleServerIDs: tt.accessibleServerIDs,
				err:                 tt.repoError,
			}

			svc := NewService(mockRepo, logger.NewNopLogger())
			ctx := context.Background()

			result, err := svc.GetAccessibleServerIDs(ctx, tt.roles, domain.AccessLevelView)

			if tt.expectError {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewService(t *testing.T) {
	mockRepo := &mockNamespaceRepository{}
	log := logger.NewNopLogger()

	svc := NewService(mockRepo, log)

	require.NotNil(t, svc)
	assert.NotNil(t, svc.namespaceRepo)
	assert.NotNil(t, svc.logger)
}

func TestNewService_NilRepo(t *testing.T) {
	log := logger.NewNopLogger()

	svc := NewService(nil, log)

	require.NotNil(t, svc)
	assert.Nil(t, svc.namespaceRepo)
}

func TestAccessLevelConstants(t *testing.T) {
	// Verify access level constants are defined correctly
	assert.Equal(t, domain.AccessLevel("view"), domain.AccessLevelView)
	assert.Equal(t, domain.AccessLevel("execute"), domain.AccessLevelExecute)
}

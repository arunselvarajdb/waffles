package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/internal/domain"
	"github.com/waffles/waffles/pkg/logger"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	users       map[string]*domain.User
	userRoles   map[string][]string
	createErr   error
	getByIDErr  error
	getByEmail  error
	updateErr   error
	deleteErr   error
	listErr     error
	assignErr   error
	removeErr   error
	updatePwErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:     make(map[string]*domain.User),
		userRoles: make(map[string][]string),
	}
}

func (m *mockRepository) Create(ctx context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = "test-user-id"
	m.users[user.ID] = user
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmail != nil {
		return nil, m.getByEmail
	}
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockRepository) Update(ctx context.Context, user *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	if m.updatePwErr != nil {
		return m.updatePwErr
	}
	if user, ok := m.users[userID]; ok {
		user.PasswordHash = passwordHash
		return nil
	}
	return domain.ErrUserNotFound
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.users, id)
	return nil
}

func (m *mockRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	users := make([]*domain.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, len(users), nil
}

func (m *mockRepository) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	roles, ok := m.userRoles[userID]
	if !ok {
		return []string{}, nil
	}
	return roles, nil
}

func (m *mockRepository) AssignRole(ctx context.Context, userID, roleName string) error {
	if m.assignErr != nil {
		return m.assignErr
	}
	m.userRoles[userID] = append(m.userRoles[userID], roleName)
	return nil
}

func (m *mockRepository) RemoveRole(ctx context.Context, userID, roleName string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	roles := m.userRoles[userID]
	for i, r := range roles {
		if r == roleName {
			m.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return nil
}

func TestNewService(t *testing.T) {
	repo := newMockRepository()
	log := logger.NewNop()

	svc := NewService(repo, log)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name        string
		req         CreateRequest
		setupRepo   func(*mockRepository)
		wantErr     bool
		errType     error
		checkResult func(*testing.T, *CreateResponse)
	}{
		{
			name: "successful creation with password",
			req: CreateRequest{
				Email:    "test@example.com",
				Name:     "Test User",
				Password: "password123",
				Role:     "user",
			},
			setupRepo: func(m *mockRepository) {},
			wantErr:   false,
			checkResult: func(t *testing.T, resp *CreateResponse) {
				assert.Equal(t, "test@example.com", resp.User.Email)
				assert.Equal(t, "Test User", resp.User.Name)
				assert.Empty(t, resp.TempPassword) // No temp password when provided
				assert.Contains(t, resp.User.Roles, "user")
			},
		},
		{
			name: "successful creation with temp password",
			req: CreateRequest{
				Email: "test2@example.com",
				Name:  "Test User 2",
				// No password - should generate temp
			},
			setupRepo: func(m *mockRepository) {},
			wantErr:   false,
			checkResult: func(t *testing.T, resp *CreateResponse) {
				assert.NotEmpty(t, resp.TempPassword)
				assert.Len(t, resp.TempPassword, 12)
			},
		},
		{
			name: "user already exists",
			req: CreateRequest{
				Email:    "existing@example.com",
				Name:     "Existing User",
				Password: "password123",
			},
			setupRepo: func(m *mockRepository) {
				m.users["existing-id"] = &domain.User{
					ID:    "existing-id",
					Email: "existing@example.com",
				}
			},
			wantErr: true,
			errType: domain.ErrUserAlreadyExists,
		},
		{
			name: "default role when not specified",
			req: CreateRequest{
				Email:    "noRole@example.com",
				Name:     "No Role User",
				Password: "password123",
			},
			setupRepo: func(m *mockRepository) {},
			wantErr:   false,
			checkResult: func(t *testing.T, resp *CreateResponse) {
				assert.Contains(t, resp.User.Roles, "user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			resp, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType))
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tt.checkResult != nil {
				tt.checkResult(t, resp)
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupRepo func(*mockRepository)
		wantErr   bool
		errType   error
	}{
		{
			name:   "user found",
			userID: "user-1",
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:    "user-1",
					Email: "user@example.com",
					Name:  "Test User",
				}
				m.userRoles["user-1"] = []string{"admin"}
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userID:    "nonexistent",
			setupRepo: func(m *mockRepository) {},
			wantErr:   true,
			errType:   domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			result, err := svc.GetByID(context.Background(), tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType))
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.userID, result.ID)
		})
	}
}

func TestService_Update(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		req       UpdateRequest
		setupRepo func(*mockRepository)
		wantErr   bool
		checkUser func(*testing.T, *UserWithRoles)
	}{
		{
			name:   "update email",
			userID: "user-1",
			req: UpdateRequest{
				Email: strPtr("new@example.com"),
			},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:    "user-1",
					Email: "old@example.com",
					Name:  "Test User",
				}
			},
			wantErr: false,
			checkUser: func(t *testing.T, u *UserWithRoles) {
				assert.Equal(t, "new@example.com", u.Email)
			},
		},
		{
			name:   "update name",
			userID: "user-1",
			req: UpdateRequest{
				Name: strPtr("New Name"),
			},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:    "user-1",
					Email: "test@example.com",
					Name:  "Old Name",
				}
			},
			wantErr: false,
			checkUser: func(t *testing.T, u *UserWithRoles) {
				assert.Equal(t, "New Name", u.Name)
			},
		},
		{
			name:   "deactivate user",
			userID: "user-1",
			req: UpdateRequest{
				IsActive: boolPtr(false),
			},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:       "user-1",
					IsActive: true,
				}
			},
			wantErr: false,
			checkUser: func(t *testing.T, u *UserWithRoles) {
				assert.False(t, u.IsActive)
			},
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			req: UpdateRequest{
				Name: strPtr("New Name"),
			},
			setupRepo: func(m *mockRepository) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			result, err := svc.Update(context.Background(), tt.userID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.checkUser != nil {
				tt.checkUser(t, result)
			}
		})
	}
}

func TestService_Deactivate(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupRepo func(*mockRepository)
		wantErr   bool
	}{
		{
			name:   "successful deactivation",
			userID: "user-1",
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:       "user-1",
					IsActive: true,
				}
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userID:    "nonexistent",
			setupRepo: func(m *mockRepository) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			err := svc.Deactivate(context.Background(), tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			// Verify user is deactivated
			user := repo.users[tt.userID]
			assert.False(t, user.IsActive)
		})
	}
}

func TestService_ResetPassword(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupRepo func(*mockRepository)
		wantErr   bool
	}{
		{
			name:   "successful reset",
			userID: "user-1",
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{
					ID:           "user-1",
					PasswordHash: "old-hash",
				}
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userID:    "nonexistent",
			setupRepo: func(m *mockRepository) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			tempPw, err := svc.ResetPassword(context.Background(), tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, tempPw)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, tempPw)
			assert.Len(t, tempPw, 12)
		})
	}
}

func TestService_UpdateRoles(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		roles     []string
		setupRepo func(*mockRepository)
		wantErr   bool
		wantRoles []string
	}{
		{
			name:   "assign new roles",
			userID: "user-1",
			roles:  []string{"admin", "operator"},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{ID: "user-1"}
				m.userRoles["user-1"] = []string{}
			},
			wantErr:   false,
			wantRoles: []string{"admin", "operator"},
		},
		{
			name:   "replace existing roles",
			userID: "user-1",
			roles:  []string{"readonly"},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{ID: "user-1"}
				m.userRoles["user-1"] = []string{"admin", "operator"}
			},
			wantErr:   false,
			wantRoles: []string{"readonly"},
		},
		{
			name:      "user not found",
			userID:    "nonexistent",
			roles:     []string{"admin"},
			setupRepo: func(m *mockRepository) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			result, err := svc.UpdateRoles(context.Background(), tt.userID, tt.roles)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.ElementsMatch(t, tt.wantRoles, result.Roles)
		})
	}
}

func TestService_List(t *testing.T) {
	tests := []struct {
		name      string
		req       ListRequest
		setupRepo func(*mockRepository)
		wantErr   bool
		wantCount int
	}{
		{
			name: "list all users",
			req:  ListRequest{Page: 1, PageSize: 10},
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{ID: "user-1"}
				m.users["user-2"] = &domain.User{ID: "user-2"}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "empty list",
			req:  ListRequest{Page: 1, PageSize: 10},
			setupRepo: func(m *mockRepository) {
				// No users
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "default pagination",
			req:  ListRequest{}, // No page/size specified
			setupRepo: func(m *mockRepository) {
				m.users["user-1"] = &domain.User{ID: "user-1"}
			},
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			tt.setupRepo(repo)
			svc := NewService(repo, logger.NewNop())

			result, err := svc.List(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Users, tt.wantCount)
			assert.Equal(t, tt.wantCount, result.Total)
		})
	}
}

func TestGenerateTempPassword(t *testing.T) {
	password1, err := generateTempPassword(12)
	require.NoError(t, err)
	assert.Len(t, password1, 12)

	password2, err := generateTempPassword(12)
	require.NoError(t, err)
	assert.Len(t, password2, 12)

	// Passwords should be different (random)
	assert.NotEqual(t, password1, password2)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

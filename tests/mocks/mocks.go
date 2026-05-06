package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	jwtpkg "github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
)

// ---------------------------------------------------------------------------
// MockUserRepository implements repository.UserRepository
// ---------------------------------------------------------------------------

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[model.User], error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PaginatedResult[model.User]), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	args := m.Called(ctx, userID, roleIDs)
	return args.Error(0)
}

func (m *MockUserRepository) GetRoles(ctx context.Context, userID uuid.UUID) ([]model.Role, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Role), args.Error(1)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// MockRoleRepository implements repository.RoleRepository
// ---------------------------------------------------------------------------

type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) GetBySlug(ctx context.Context, slug string) (*model.Role, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) List(ctx context.Context) ([]model.Role, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Role), args.Error(1)
}

// ---------------------------------------------------------------------------
// MockTokenManager implements jwt.TokenManager
// ---------------------------------------------------------------------------

type MockTokenManager struct {
	mock.Mock
}

func (m *MockTokenManager) GenerateAccess(userID uuid.UUID, email string, roles []string) (string, error) {
	args := m.Called(userID, email, roles)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) GenerateRefresh(userID uuid.UUID) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenManager) ValidateAccess(tokenStr string) (*jwtpkg.Claims, error) {
	args := m.Called(tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.Claims), args.Error(1)
}

func (m *MockTokenManager) ValidateRefresh(tokenStr string) (*jwtpkg.Claims, error) {
	args := m.Called(tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtpkg.Claims), args.Error(1)
}

// ---------------------------------------------------------------------------
// MockHasher implements hasher.Hasher
// ---------------------------------------------------------------------------

type MockHasher struct {
	mock.Mock
}

func (m *MockHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockHasher) Compare(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// MockEmailService implements service.EmailService
// ---------------------------------------------------------------------------

type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendPasswordReset(ctx context.Context, to, token string) error {
	args := m.Called(ctx, to, token)
	return args.Error(0)
}

func (m *MockEmailService) SendWelcome(ctx context.Context, to, name string) error {
	args := m.Called(ctx, to, name)
	return args.Error(0)
}

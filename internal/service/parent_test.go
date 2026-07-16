package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/tests/mocks"
)

// mockGuardianStore is a testify mock for the local guardianStore interface.
// It lives in the service package test file because guardianStore is unexported.
type mockGuardianStore struct {
	mock.Mock
}

func (m *mockGuardianStore) GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*model.Guardian, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Guardian), args.Error(1)
}

func (m *mockGuardianStore) GetGuardianByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*model.Guardian, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Guardian), args.Error(1)
}

func (m *mockGuardianStore) ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[model.Guardian], error) {
	args := m.Called(ctx, tenantID, filter, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PaginatedResult[model.Guardian]), args.Error(1)
}

func (m *mockGuardianStore) ListGuardianStudents(ctx context.Context, tenantID, guardianID uuid.UUID) ([]model.GuardianStudent, error) {
	args := m.Called(ctx, tenantID, guardianID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.GuardianStudent), args.Error(1)
}

func (m *mockGuardianStore) ListGuardianStudentsPaginated(ctx context.Context, tenantID, guardianID uuid.UUID, filter model.GuardianStudentFilter, params model.PaginationParams) (*model.PaginatedResult[model.GuardianStudent], error) {
	args := m.Called(ctx, tenantID, guardianID, filter, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PaginatedResult[model.GuardianStudent]), args.Error(1)
}

func (m *mockGuardianStore) ListGuardianStudentsByGuardianIDs(ctx context.Context, tenantID uuid.UUID, guardianIDs []uuid.UUID) (map[uuid.UUID][]model.GuardianStudent, error) {
	args := m.Called(ctx, tenantID, guardianIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID][]model.GuardianStudent), args.Error(1)
}

func (m *mockGuardianStore) SetGuardianUserID(ctx context.Context, tenantID, guardianID uuid.UUID, userID *uuid.UUID) error {
	args := m.Called(ctx, tenantID, guardianID, userID)
	return args.Error(0)
}

func newMockGuardianStore(t *testing.T) *mockGuardianStore {
	t.Helper()
	m := &mockGuardianStore{}
	return m
}

func newParentServiceForTest(t *testing.T) (*parentService, *mockGuardianStore, *mocks.MockUserRepository, *mocks.MockRoleRepository, *mocks.MockAuditRepository) {
	t.Helper()
	guardians := newMockGuardianStore(t)
	userRepo := new(mocks.MockUserRepository)
	roleRepo := new(mocks.MockRoleRepository)
	auditRepo := new(mocks.MockAuditRepository)

	svc := &parentService{
		guardians: guardians,
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		auditRepo: auditRepo,
	}
	return svc, guardians, userRepo, roleRepo, auditRepo
}

func TestParentService_LinkGuardianUser_NotFound(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, guardianID, userID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return((*model.Guardian)(nil), nil)

	resp, err := svc.LinkGuardianUser(context.Background(), actorID, tenantID, guardianID, userID)
	require.Nil(t, resp)
	require.Error(t, err)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.ErrNotFound.Code, appErr.Code)
}

func TestParentService_LinkGuardianUser_AlreadyLinkedSameUser(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, guardianID, userID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID, UserID: &userID}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)

	resp, err := svc.LinkGuardianUser(context.Background(), actorID, tenantID, guardianID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userID, *resp.UserID)
}

func TestParentService_LinkGuardianUser_UserNotFound(t *testing.T) {
	svc, guardians, userRepo, _, _ := newParentServiceForTest(t)
	tenantID, guardianID, userID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return((*model.User)(nil), nil)

	resp, err := svc.LinkGuardianUser(context.Background(), actorID, tenantID, guardianID, userID)
	require.Nil(t, resp)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.ErrNotFound.Code, appErr.Code)
}

func TestParentService_LinkGuardianUser_NotParentRole(t *testing.T) {
	svc, guardians, userRepo, roleRepo, _ := newParentServiceForTest(t)
	tenantID, guardianID, userID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID}
	user := &model.User{ID: userID, Status: "active"} // no roles on the join
	parentRole := &model.Role{ID: uuid.New(), Slug: "parents"}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	roleRepo.On("GetBySlug", mock.Anything, "parents").Return(parentRole, nil)

	resp, err := svc.LinkGuardianUser(context.Background(), actorID, tenantID, guardianID, userID)
	require.Nil(t, resp)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, "PARENT_ROLE_MISSING", appErr.Code)
}

func TestParentService_LinkGuardianUser_Success(t *testing.T) {
	svc, guardians, userRepo, _, auditRepo := newParentServiceForTest(t)
	tenantID, guardianID, userID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID, Name: "Riya"}
	user := &model.User{ID: userID, Status: "active", Roles: []model.Role{{ID: uuid.New(), Slug: "parents"}}}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	guardians.On("SetGuardianUserID", mock.Anything, tenantID, guardianID, &userID).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.LinkGuardianUser(context.Background(), actorID, tenantID, guardianID, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userID, *resp.UserID)
	require.NotNil(t, resp.UserStatus)
	assert.Equal(t, "active", *resp.UserStatus)
}

func TestParentService_UnlinkGuardianUser_Idempotent(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, guardianID, actorID := uuid.New(), uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID} // no user_id

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)

	resp, err := svc.UnlinkGuardianUser(context.Background(), actorID, tenantID, guardianID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.UserID)
}

func TestParentService_UnlinkGuardianUser_Success(t *testing.T) {
	svc, guardians, _, _, auditRepo := newParentServiceForTest(t)
	tenantID, guardianID, actorID := uuid.New(), uuid.New(), uuid.New()
	existingUserID := uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID, UserID: &existingUserID, UserStatus: stringPtr("active")}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)
	guardians.On("SetGuardianUserID", mock.Anything, tenantID, guardianID, (*uuid.UUID)(nil)).Return(nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.UnlinkGuardianUser(context.Background(), actorID, tenantID, guardianID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, resp.UserID)
	assert.Nil(t, resp.UserStatus)
}

func TestParentService_ListGuardianStudents_NotFound(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, guardianID := uuid.New(), uuid.New()

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return((*model.Guardian)(nil), nil)

	resp, err := svc.ListGuardianStudents(context.Background(), tenantID, guardianID)
	require.Nil(t, resp)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.ErrNotFound.Code, appErr.Code)
}

func TestParentService_ListGuardianStudents_Success(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, guardianID := uuid.New(), uuid.New()
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID}
	students := []model.GuardianStudent{
		{StudentID: uuid.New(), AdmissionNumber: "ADM-1", FirstName: "Aarav", LastName: "S", IsPrimary: true, ClassName: "10", SectionName: "A", Status: "active"},
		{StudentID: uuid.New(), AdmissionNumber: "ADM-2", FirstName: "Meera", LastName: "I", IsPrimary: false, ClassName: "9", SectionName: "B", Status: "active"},
	}

	guardians.On("GetGuardian", mock.Anything, tenantID, guardianID).Return(guardian, nil)
	guardians.On("ListGuardianStudents", mock.Anything, tenantID, guardianID).Return(students, nil)

	resp, err := svc.ListGuardianStudents(context.Background(), tenantID, guardianID)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.Equal(t, "ADM-1", resp[0].AdmissionNumber)
	assert.True(t, resp[0].IsPrimary)
	assert.Equal(t, "10", resp[0].ClassName)
}

func TestParentService_ListLinkedChildren_PaginatesAndLimitsFields(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, userID, guardianID := uuid.New(), uuid.New(), uuid.New()
	params := model.PaginationParams{Page: 2, PageSize: 1}
	filter := model.GuardianStudentFilter{Search: "aar"}
	guardian := &model.Guardian{ID: guardianID, TenantID: tenantID, UserID: &userID}
	students := []model.GuardianStudent{{
		GuardianID: guardianID, StudentID: uuid.New(), AdmissionNumber: "ADM-1",
		FirstName: "Aarav", LastName: "Sharma", Relationship: "father", IsPrimary: true,
		ClassName: "Class 5", SectionName: "A", Status: "active",
	}}

	guardians.On("GetGuardianByUserID", mock.Anything, tenantID, userID).Return(guardian, nil)
	guardians.On("ListGuardianStudentsPaginated", mock.Anything, tenantID, guardianID, filter, params).
		Return(model.NewPaginatedResult(students, int64(2), 2, 1), nil)

	result, err := svc.ListLinkedChildren(context.Background(), tenantID, userID, filter, params)
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, students[0].StudentID, result.Data[0].ID)
	assert.Equal(t, "ADM-1", result.Data[0].AdmissionNumber)
	assert.Equal(t, "Class 5", result.Data[0].ClassName)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, int64(2), result.Total)
}

func TestParentService_ListLinkedChildren_UnlinkedParentReturnsEmptyPage(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID, userID := uuid.New(), uuid.New()
	params := model.PaginationParams{Page: 3, PageSize: 5}

	guardians.On("GetGuardianByUserID", mock.Anything, tenantID, userID).Return((*model.Guardian)(nil), nil)

	result, err := svc.ListLinkedChildren(context.Background(), tenantID, userID, model.GuardianStudentFilter{}, params)
	require.NoError(t, err)
	assert.Empty(t, result.Data)
	assert.Equal(t, 3, result.Page)
	assert.Equal(t, 5, result.PageSize)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 0, result.TotalPages)
}

func TestParentService_ListParents_AggregatesGuardianUserAndStudents(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID := uuid.New()
	userID := uuid.New()
	guardianID := uuid.New()
	secondGuardianID := uuid.New()

	guardiansList := []model.Guardian{
		{ID: guardianID, TenantID: tenantID, Name: "Riya", UserID: &userID, UserStatus: stringPtr("active")},
		{ID: secondGuardianID, TenantID: tenantID, Name: "Anil"},
	}
	guardians.On("ListGuardians", mock.Anything, tenantID, mock.Anything, mock.Anything).
		Return(model.NewPaginatedResult(guardiansList, int64(2), 1, 20), nil)

	guardians.On("ListGuardianStudentsByGuardianIDs", mock.Anything, tenantID, []uuid.UUID{guardianID, secondGuardianID}).
		Return(map[uuid.UUID][]model.GuardianStudent{
			guardianID: {
				{GuardianID: guardianID, StudentID: uuid.New(), AdmissionNumber: "ADM-1", FirstName: "Aarav", ClassName: "10", SectionName: "A", Status: "active"},
			},
			secondGuardianID: {},
		}, nil)

	result, err := svc.ListParents(context.Background(), tenantID, model.GuardianFilter{}, model.PaginationParams{})
	require.NoError(t, err)
	require.Len(t, result.Data, 2)

	first := result.Data[0]
	assert.Equal(t, guardianID, first.GuardianID)
	require.NotNil(t, first.UserID)
	assert.Equal(t, userID, *first.UserID)
	require.NotNil(t, first.UserStatus)
	assert.Equal(t, "active", *first.UserStatus)
	require.Len(t, first.LinkedStudents, 1)
	assert.Equal(t, "ADM-1", first.LinkedStudents[0].AdmissionNumber)

	second := result.Data[1]
	assert.Equal(t, secondGuardianID, second.GuardianID)
	assert.Nil(t, second.UserID)
	assert.Nil(t, second.UserStatus)
	assert.Empty(t, second.LinkedStudents)
}

func TestParentService_ListParents_DeletedUserLeavesStatusNil(t *testing.T) {
	svc, guardians, _, _, _ := newParentServiceForTest(t)
	tenantID := uuid.New()
	guardianID := uuid.New()
	userID := uuid.New()

	guardiansList := []model.Guardian{
		{ID: guardianID, TenantID: tenantID, Name: "Riya", UserID: &userID},
	}
	guardians.On("ListGuardians", mock.Anything, tenantID, mock.Anything, mock.Anything).
		Return(model.NewPaginatedResult(guardiansList, int64(1), 1, 20), nil)
	guardians.On("ListGuardianStudentsByGuardianIDs", mock.Anything, tenantID, []uuid.UUID{guardianID}).
		Return(map[uuid.UUID][]model.GuardianStudent{guardianID: {}}, nil)

	result, err := svc.ListParents(context.Background(), tenantID, model.GuardianFilter{}, model.PaginationParams{})
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, guardianID, result.Data[0].GuardianID)
	require.NotNil(t, result.Data[0].UserID)
	assert.Nil(t, result.Data[0].UserStatus)
}

func TestGuardianToResponseIncludesFrontendContractFields(t *testing.T) {
	userID := uuid.New()
	guardian := &model.Guardian{
		ID:                 uuid.New(),
		Name:               "Rajesh Kumar",
		CommunicationOptIn: true,
		OptInWhatsApp:      false,
		Address:            model.Address{Line1: "123, Main Street, City", Country: "India"},
		UserID:             &userID,
		UserStatus:         stringPtr("invited"),
	}

	response := guardianToResponse(guardian)
	assert.Equal(t, "123, Main Street, City", response.Address)
	assert.False(t, response.OptInWhatsApp)
	require.NotNil(t, response.UserID)
	assert.Equal(t, userID, *response.UserID)
	require.NotNil(t, response.UserStatus)
	assert.Equal(t, "invited", *response.UserStatus)
}

package unit

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
	"github.com/itsVijayCoder/edu-wallet-backend/tests/mocks"
)

func TestBootstrapSuperAdminCreatesActiveSuperAdmin(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	roleID := uuid.New()

	userRepo := new(mocks.MockUserRepository)
	roleRepo := new(mocks.MockRoleRepository)
	hasherMock := new(mocks.MockHasher)

	roleRepo.On("GetBySlug", mock.Anything, "super_admin").Return(&model.Role{
		ID:   roleID,
		Name: "Super Admin",
		Slug: "super_admin",
	}, nil)
	hasherMock.On("Hash", "StrongPass123!").Return("hashed-password", nil)
	userRepo.On("GetByEmail", mock.Anything, "admin@eduwallet.in").Return(nil, nil)
	userRepo.On("Create", mock.Anything, mock.MatchedBy(func(user *model.User) bool {
		return user.Email == "admin@eduwallet.in" &&
			user.PasswordHash == "hashed-password" &&
			user.FirstName == "EduWallet" &&
			user.LastName == "Owner" &&
			user.Status == "active"
	})).Run(func(args mock.Arguments) {
		args.Get(1).(*model.User).ID = userID
	}).Return(nil)
	userRepo.On("AssignRoles", mock.Anything, userID, []uuid.UUID{roleID}).Return(nil)
	userRepo.On("GetByID", mock.Anything, userID).Return(&model.User{
		ID:           userID,
		Email:        "admin@eduwallet.in",
		PasswordHash: "hashed-password",
		FirstName:    "EduWallet",
		LastName:     "Owner",
		Status:       "active",
		Roles: []model.Role{{
			ID:   roleID,
			Name: "Super Admin",
			Slug: "super_admin",
		}},
	}, nil)

	resp, err := service.BootstrapSuperAdmin(ctx, userRepo, roleRepo, hasherMock, service.SuperAdminBootstrapInput{
		Email:     " ADMIN@EduWallet.in ",
		Password:  "StrongPass123!",
		FirstName: "EduWallet",
		LastName:  "Owner",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, userID, resp.ID)
	require.Equal(t, "admin@eduwallet.in", resp.Email)
	require.Equal(t, []string{"super_admin"}, resp.Roles)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	hasherMock.AssertExpectations(t)
}

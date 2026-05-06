package unit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	redismock "github.com/go-redis/redismock/v9"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
	"github.com/itsVijayCoder/edu-wallet-backend/tests/mocks"
)

func TestAuthService_Login(t *testing.T) {
	userID := uuid.New()
	hashedPassword := "$2a$12$fakehash"

	activeUser := &model.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "Test",
		LastName:     "User",
		Status:       "active",
		Roles: []model.Role{
			{ID: uuid.New(), Name: "Member", Slug: "member"},
		},
	}

	inactiveUser := &model.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "Test",
		LastName:     "User",
		Status:       "inactive",
		Roles:        []model.Role{},
	}

	tests := []struct {
		name        string
		req         dto.LoginRequest
		setupMocks  func(*mocks.MockUserRepository, *mocks.MockHasher, *mocks.MockTokenManager, redismock.ClientMock)
		wantErr     error
		wantNilResp bool
	}{
		{
			name: "valid credentials",
			req:  dto.LoginRequest{Email: "test@example.com", Password: "password123"},
			setupMocks: func(userRepo *mocks.MockUserRepository, h *mocks.MockHasher, tokenMgr *mocks.MockTokenManager, rmock redismock.ClientMock) {
				userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(activeUser, nil)
				h.On("Compare", hashedPassword, "password123").Return(nil)
				tokenMgr.On("GenerateAccess", userID, "test@example.com", []string{"member"}).Return("access-token", nil)
				tokenMgr.On("GenerateRefresh", userID).Return("refresh-token", nil)
				rmock.ExpectSet("refresh:"+userID.String(), "refresh-token", time.Hour).SetVal("OK")
				userRepo.On("UpdateLastLogin", mock.Anything, userID).Return(nil)
			},
			wantErr:     nil,
			wantNilResp: false,
		},
		{
			name: "invalid password",
			req:  dto.LoginRequest{Email: "test@example.com", Password: "wrongpassword"},
			setupMocks: func(userRepo *mocks.MockUserRepository, h *mocks.MockHasher, tokenMgr *mocks.MockTokenManager, rmock redismock.ClientMock) {
				userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(activeUser, nil)
				h.On("Compare", hashedPassword, "wrongpassword").Return(errors.New("mismatch"))
			},
			wantErr:     apperror.ErrInvalidCredentials,
			wantNilResp: true,
		},
		{
			name: "user not found",
			req:  dto.LoginRequest{Email: "nobody@example.com", Password: "password123"},
			setupMocks: func(userRepo *mocks.MockUserRepository, h *mocks.MockHasher, tokenMgr *mocks.MockTokenManager, rmock redismock.ClientMock) {
				userRepo.On("GetByEmail", mock.Anything, "nobody@example.com").Return(nil, nil)
			},
			wantErr:     apperror.ErrInvalidCredentials,
			wantNilResp: true,
		},
		{
			name: "inactive user",
			req:  dto.LoginRequest{Email: "test@example.com", Password: "password123"},
			setupMocks: func(userRepo *mocks.MockUserRepository, h *mocks.MockHasher, tokenMgr *mocks.MockTokenManager, rmock redismock.ClientMock) {
				userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(inactiveUser, nil)
			},
			wantErr:     apperror.ErrAccountInactive,
			wantNilResp: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := new(mocks.MockUserRepository)
			hasherMock := new(mocks.MockHasher)
			tokenMgr := new(mocks.MockTokenManager)
			emailSvc := new(mocks.MockEmailService)

			rdb, rmock := redismock.NewClientMock()

			tc.setupMocks(userRepo, hasherMock, tokenMgr, rmock)

			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			svc := service.NewAuthService(userRepo, hasherMock, tokenMgr, rdb, time.Hour, emailSvc, log)

			resp, err := svc.Login(context.Background(), tc.req)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tc.wantNilResp {
				assert.Nil(t, resp)
			} else {
				require.NotNil(t, resp)
				assert.Equal(t, "access-token", resp.AccessToken)
				assert.Equal(t, "refresh-token", resp.RefreshToken)
				assert.Equal(t, "test@example.com", resp.User.Email)
			}

			userRepo.AssertExpectations(t)
			hasherMock.AssertExpectations(t)
			tokenMgr.AssertExpectations(t)
		})
	}
}

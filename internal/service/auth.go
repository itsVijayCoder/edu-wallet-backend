package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/hasher"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
)

type authService struct {
	userRepo                  repository.UserRepository
	membershipRepo            repository.TenantMembershipRepository
	tenantRepo                repository.TenantRepository
	hasher                    hasher.Hasher
	tokenMgr                  jwt.TokenManager
	redis                     *redis.Client
	refreshTTL                time.Duration
	emailSvc                  EmailService
	log                       *slog.Logger
	publicRegistrationEnabled bool
}

func NewAuthService(
	userRepo repository.UserRepository,
	h hasher.Hasher,
	tokenMgr jwt.TokenManager,
	rdb *redis.Client,
	refreshTTL time.Duration,
	emailSvc EmailService,
	log *slog.Logger,
	publicRegistrationEnabled bool,
	membershipRepo repository.TenantMembershipRepository,
	tenantRepo repository.TenantRepository,
) AuthService {
	return &authService{
		userRepo:                  userRepo,
		membershipRepo:            membershipRepo,
		tenantRepo:                tenantRepo,
		hasher:                    h,
		tokenMgr:                  tokenMgr,
		redis:                     rdb,
		refreshTTL:                refreshTTL,
		emailSvc:                  emailSvc,
		log:                       log,
		publicRegistrationEnabled: publicRegistrationEnabled,
	}
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, apperror.ErrInvalidCredentials
	}

	if user.Status != "active" {
		return nil, apperror.ErrAccountInactive
	}

	if err := s.hasher.Compare(user.PasswordHash, req.Password); err != nil {
		return nil, apperror.ErrInvalidCredentials
	}

	roleSlugs := rolesToSlugs(user.Roles)

	accessToken, err := s.tokenMgr.GenerateAccess(user.ID, user.Email, roleSlugs)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenMgr.GenerateRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.redis.Set(rctx, refreshKey(user.ID), refreshToken, s.refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return &dto.LoginResponse{
		TokenPair: dto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		User:    userToResponse(user),
		Tenants: s.membershipsForLogin(ctx, user.ID),
	}, nil
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	if !s.publicRegistrationEnabled {
		return nil, apperror.ErrPublicRegistrationDisabled
	}

	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	resp := userToResponse(user)
	return &resp, nil
}

func (s *authService) RefreshToken(ctx context.Context, req dto.RefreshRequest) (*dto.TokenPair, error) {
	claims, err := s.tokenMgr.ValidateRefresh(req.RefreshToken)
	if err != nil {
		return nil, apperror.ErrRefreshTokenInvalid
	}

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	stored, err := s.redis.Get(rctx, refreshKey(claims.UserID)).Result()
	if err != nil || stored != req.RefreshToken {
		return nil, apperror.ErrRefreshTokenInvalid
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil || user.Status != "active" {
		return nil, apperror.ErrRefreshTokenInvalid
	}

	roleSlugs := rolesToSlugs(user.Roles)

	accessToken, err := s.tokenMgr.GenerateAccess(user.ID, user.Email, roleSlugs)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenMgr.GenerateRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	rctx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()
	if err := s.redis.Set(rctx2, refreshKey(user.ID), refreshToken, s.refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) SelectTenant(ctx context.Context, userID uuid.UUID, req dto.SelectTenantRequest) (*dto.TokenPair, error) {
	if s.membershipRepo == nil {
		return nil, apperror.ErrTenantAccessDenied
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil || user.Status != "active" {
		return nil, apperror.ErrTenantAccessDenied
	}

	roles := rolesToSlugs(user.Roles)
	var permissions []string
	tenantID := req.TenantID

	membership, err := s.membershipRepo.GetByUserAndTenant(ctx, userID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant membership: %w", err)
	}

	if membership != nil {
		if membership.Tenant == nil || membership.Role == nil {
			return nil, apperror.ErrTenantAccessDenied
		}
		if membership.Tenant.Status != "active" && membership.Tenant.Status != "trial" {
			return nil, apperror.ErrTenantAccessDenied
		}
		tenantID = membership.TenantID
		roles = appendUnique(roles, membership.Role.Slug)
		permissions = permissionsToCodes(membership.Permissions)
	} else {
		superAdminRole := findRoleBySlug(user.Roles, "super_admin")
		if superAdminRole == nil || s.tenantRepo == nil {
			return nil, apperror.ErrTenantAccessDenied
		}

		tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("get tenant: %w", err)
		}
		if tenant == nil || (tenant.Status != "active" && tenant.Status != "trial") {
			return nil, apperror.ErrTenantAccessDenied
		}

		rolePermissions, err := s.membershipRepo.ListPermissionsByRole(ctx, superAdminRole.ID)
		if err != nil {
			return nil, fmt.Errorf("list super admin permissions: %w", err)
		}
		permissions = permissionsToCodes(rolePermissions)
	}

	accessToken, err := s.tokenMgr.GenerateTenantAccess(user.ID, user.Email, roles, tenantID, permissions)
	if err != nil {
		return nil, fmt.Errorf("generate tenant access token: %w", err)
	}

	refreshToken, err := s.tokenMgr.GenerateRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.redis.Set(rctx, refreshKey(user.ID), refreshToken, s.refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) Logout(ctx context.Context, userID uuid.UUID) error {
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.redis.Del(rctx, refreshKey(userID)).Err()
}

func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil // always nil to prevent email enumeration
	}

	token := uuid.New().String()

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.redis.Set(rctx, resetKey(token), user.ID.String(), time.Hour).Err(); err != nil {
		s.log.Error("failed to store reset token", "error", err)
		return nil
	}

	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer bgCancel()
		_ = s.emailSvc.SendPasswordReset(bgCtx, user.Email, token)
	}()

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	userIDStr, err := s.redis.Get(rctx, resetKey(req.Token)).Result()
	if err != nil {
		return apperror.New("AUTH_RESET_INVALID", "invalid or expired reset token", 400)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperror.New("AUTH_RESET_INVALID", "invalid or expired reset token", 400)
	}

	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return apperror.ErrNotFound
	}

	user.PasswordHash = hash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Invalidate reset token and refresh token
	rctx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
	defer cancel2()
	s.redis.Del(rctx2, resetKey(req.Token))
	s.redis.Del(rctx2, refreshKey(userID))

	return nil
}

// --- Helpers ---

func refreshKey(userID uuid.UUID) string {
	return "refresh:" + userID.String()
}

func resetKey(token string) string {
	return "reset:" + token
}

func rolesToSlugs(roles []model.Role) []string {
	slugs := make([]string, len(roles))
	for i, r := range roles {
		slugs[i] = r.Slug
	}
	return slugs
}

func permissionsToCodes(permissions []model.Permission) []string {
	codes := make([]string, len(permissions))
	for i, permission := range permissions {
		codes[i] = permission.Code
	}
	return codes
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func findRoleBySlug(roles []model.Role, slug string) *model.Role {
	for i := range roles {
		if roles[i].Slug == slug {
			return &roles[i]
		}
	}
	return nil
}

func (s *authService) membershipsForLogin(ctx context.Context, userID uuid.UUID) []dto.TenantMembershipBrief {
	if s.membershipRepo == nil {
		return nil
	}

	memberships, err := s.membershipRepo.ListByUser(ctx, userID)
	if err != nil {
		s.log.Warn("failed to load tenant memberships for login", "user_id", userID, "error", err)
		return nil
	}

	resp := make([]dto.TenantMembershipBrief, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Tenant == nil || membership.Role == nil {
			continue
		}
		resp = append(resp, dto.TenantMembershipBrief{
			TenantID:    membership.TenantID,
			TenantName:  membership.Tenant.Name,
			TenantSlug:  membership.Tenant.Slug,
			Role:        membership.Role.Slug,
			Permissions: permissionsToCodes(membership.Permissions),
		})
	}
	return resp
}

func userToResponse(user *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    user.Status,
		Roles:     rolesToSlugs(user.Roles),
	}
}

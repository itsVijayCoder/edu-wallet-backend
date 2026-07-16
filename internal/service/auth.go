package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
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
	parentAuthRepo            ParentAuthStore
	hasher                    hasher.Hasher
	tokenMgr                  jwt.TokenManager
	redis                     *redis.Client
	refreshTTL                time.Duration
	emailSvc                  EmailService
	otpNotifier               OTPNotifier
	log                       *slog.Logger
	publicRegistrationEnabled bool
}

// ParentAuthStore resolves linked guardian phone numbers to parent accounts.
// It deliberately exposes only the authentication lookup, keeping AuthService
// independent from the broader academic repository.
type ParentAuthStore interface {
	FindParentLoginCandidatesByPhone(ctx context.Context, phone string) ([]model.ParentLoginCandidate, error)
}

const (
	otpTTL           = 5 * time.Minute
	otpRequestWindow = time.Minute
	maxOTPAttempts   = 5
)

type storedOTP struct {
	CodeHash string    `json:"code_hash"`
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
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
	parentAuthRepo ParentAuthStore,
	otpNotifier OTPNotifier,
) AuthService {
	return &authService{
		userRepo:                  userRepo,
		membershipRepo:            membershipRepo,
		tenantRepo:                tenantRepo,
		parentAuthRepo:            parentAuthRepo,
		hasher:                    h,
		tokenMgr:                  tokenMgr,
		redis:                     rdb,
		refreshTTL:                refreshTTL,
		emailSvc:                  emailSvc,
		otpNotifier:               otpNotifier,
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
		TokenPair: s.tokenPair(accessToken, refreshToken),
		User:      userToResponse(user),
		Tenants:   s.membershipsForLogin(ctx, user.ID),
	}, nil
}

func (s *authService) SendOTP(ctx context.Context, req dto.SendOTPRequest) (*dto.SendOTPResponse, error) {
	if s.parentAuthRepo == nil || s.otpNotifier == nil {
		return nil, fmt.Errorf("parent OTP authentication is not configured")
	}

	phone := strings.TrimSpace(req.Phone)
	candidate, err := s.resolveParentLoginCandidate(ctx, phone, req.TenantSlug)
	if err != nil {
		return nil, err
	}

	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	allowed, err := s.redis.SetNX(rctx, otpRequestKey(phone, candidate.TenantID), "1", otpRequestWindow).Result()
	if err != nil {
		return nil, fmt.Errorf("set OTP request rate limit: %w", err)
	}
	if !allowed {
		return nil, apperror.ErrOTPRateLimited
	}

	otp, err := generateOTP(6)
	if err != nil {
		_ = s.redis.Del(rctx, otpRequestKey(phone, candidate.TenantID)).Err()
		return nil, fmt.Errorf("generate OTP: %w", err)
	}
	otpHash, err := s.hasher.Hash(otp)
	if err != nil {
		_ = s.redis.Del(rctx, otpRequestKey(phone, candidate.TenantID)).Err()
		return nil, fmt.Errorf("hash OTP: %w", err)
	}
	value, err := json.Marshal(storedOTP{
		CodeHash: otpHash,
		UserID:   candidate.UserID,
		TenantID: candidate.TenantID,
	})
	if err != nil {
		_ = s.redis.Del(rctx, otpRequestKey(phone, candidate.TenantID)).Err()
		return nil, fmt.Errorf("marshal OTP: %w", err)
	}
	if err := s.redis.Set(rctx, otpKey(phone), value, otpTTL).Err(); err != nil {
		_ = s.redis.Del(rctx, otpRequestKey(phone, candidate.TenantID)).Err()
		return nil, fmt.Errorf("store OTP: %w", err)
	}

	if err := s.otpNotifier.SendOTP(ctx, phone, otp); err != nil {
		if deleteErr := s.redis.Del(rctx, otpKey(phone), otpRequestKey(phone, candidate.TenantID)).Err(); deleteErr != nil {
			s.log.Warn("failed to remove undelivered OTP", "error", deleteErr)
		}
		return nil, fmt.Errorf("deliver OTP: %w", err)
	}

	return &dto.SendOTPResponse{
		Message:          fmt.Sprintf("OTP sent to %s", maskPhone(phone)),
		ExpiresInSeconds: int(otpTTL.Seconds()),
	}, nil
}

func (s *authService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (*dto.LoginResponse, error) {
	phone := strings.TrimSpace(req.Phone)
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	stored, err := s.consumeMatchingOTP(rctx, phone, req.OTP)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("get OTP user: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrPhoneNotFound
	}
	if user.Status != "active" {
		return nil, apperror.ErrAccountInactive
	}
	if !hasRole(user.Roles, parentRoleSlug) {
		return nil, apperror.ErrPhoneNotFound
	}

	candidates, err := s.parentAuthRepo.FindParentLoginCandidatesByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("find parent login candidates: %w", err)
	}
	var selected *model.ParentLoginCandidate
	for i := range candidates {
		if candidates[i].TenantID == stored.TenantID && candidates[i].UserID == user.ID {
			selected = &candidates[i]
			break
		}
	}
	if selected == nil {
		return nil, apperror.ErrPhoneNotFound
	}

	roles := rolesToSlugs(user.Roles)
	accessToken, err := s.tokenMgr.GenerateTenantAccess(user.ID, user.Email, roles, selected.TenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("generate OTP access token: %w", err)
	}
	refreshToken, err := s.tokenMgr.GenerateRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate OTP refresh token: %w", err)
	}
	if err := s.redis.Set(rctx, refreshKey(user.ID), refreshToken, s.refreshTTL).Err(); err != nil {
		return nil, fmt.Errorf("store OTP refresh token: %w", err)
	}
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Warn("failed to update OTP login timestamp", "user_id", user.ID, "error", err)
	}

	return &dto.LoginResponse{
		TokenPair: s.tokenPair(accessToken, refreshToken),
		User:      userToResponse(user),
		Tenants: []dto.TenantMembershipBrief{{
			TenantID:    selected.TenantID,
			TenantName:  selected.TenantName,
			TenantSlug:  selected.TenantSlug,
			Status:      selected.TenantStatus,
			Role:        parentRoleSlug,
			Permissions: []string{},
		}},
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

	returnTokenPair := s.tokenPair(accessToken, refreshToken)
	return &returnTokenPair, nil
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

	returnTokenPair := s.tokenPair(accessToken, refreshToken)
	return &returnTokenPair, nil
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

func otpKey(phone string) string {
	return "auth:otp:" + hashPhone(phone)
}

func otpAttemptKey(phone string) string {
	return "auth:otp-attempts:" + hashPhone(phone)
}

func otpRequestKey(phone string, tenantID uuid.UUID) string {
	return "auth:otp-request:" + hashPhone(phone) + ":" + tenantID.String()
}

func hashPhone(phone string) string {
	sum := sha256.Sum256([]byte(phone))
	return hex.EncodeToString(sum[:])
}

func generateOTP(length int) (string, error) {
	if length < 4 || length > 6 {
		return "", fmt.Errorf("OTP length must be between 4 and 6")
	}
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, n.Int64()), nil
}

func maskPhone(phone string) string {
	if len(phone) <= 6 {
		return phone
	}
	return phone[:len(phone)-6] + "****" + phone[len(phone)-2:]
}

func (s *authService) resolveParentLoginCandidate(ctx context.Context, phone, tenantSlug string) (*model.ParentLoginCandidate, error) {
	candidates, err := s.parentAuthRepo.FindParentLoginCandidatesByPhone(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("find parent login candidates: %w", err)
	}
	if len(candidates) == 0 {
		return nil, apperror.ErrPhoneNotFound
	}

	tenantSlug = strings.TrimSpace(tenantSlug)
	if tenantSlug != "" {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if strings.EqualFold(candidate.TenantSlug, tenantSlug) {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
		if len(candidates) == 0 {
			return nil, apperror.ErrPhoneNotFound
		}
	}
	if len(candidates) > 1 {
		return nil, apperror.ErrTenantSelectionRequired
	}

	candidate := candidates[0]
	user, err := s.userRepo.GetByID(ctx, candidate.UserID)
	if err != nil {
		return nil, fmt.Errorf("get parent OTP user: %w", err)
	}
	if user == nil || user.Status != "active" || !hasRole(user.Roles, parentRoleSlug) {
		return nil, apperror.ErrPhoneNotFound
	}
	return &candidate, nil
}

func (s *authService) consumeMatchingOTP(ctx context.Context, phone, otp string) (*storedOTP, error) {
	key := otpKey(phone)
	var matched *storedOTP
	err := s.redis.Watch(ctx, func(tx *redis.Tx) error {
		raw, err := tx.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return apperror.ErrOTPExpired
		}
		if err != nil {
			return fmt.Errorf("get OTP: %w", err)
		}

		var stored storedOTP
		if err := json.Unmarshal(raw, &stored); err != nil {
			return fmt.Errorf("decode OTP: %w", err)
		}
		if err := s.hasher.Compare(stored.CodeHash, otp); err != nil {
			attempts, err := s.redis.Incr(ctx, otpAttemptKey(phone)).Result()
			if err != nil {
				return fmt.Errorf("record OTP attempt: %w", err)
			}
			if attempts == 1 {
				if err := s.redis.Expire(ctx, otpAttemptKey(phone), otpTTL).Err(); err != nil {
					return fmt.Errorf("set OTP attempt expiry: %w", err)
				}
			}
			if attempts >= maxOTPAttempts {
				if err := s.redis.Del(ctx, key).Err(); err != nil {
					return fmt.Errorf("invalidate OTP after failed attempts: %w", err)
				}
			}
			return apperror.ErrOTPInvalid
		}

		if _, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key, otpAttemptKey(phone))
			return nil
		}); err != nil {
			return fmt.Errorf("consume OTP: %w", err)
		}
		matched = &stored
		return nil
	}, key)
	if err == redis.TxFailedErr {
		return nil, apperror.ErrOTPExpired
	}
	if err != nil {
		return nil, err
	}
	if matched == nil {
		return nil, apperror.ErrOTPExpired
	}
	return matched, nil
}

func (s *authService) tokenPair(accessToken, refreshToken string) dto.TokenPair {
	return dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessTokenExpiresAt(accessToken),
	}
}

func accessTokenExpiresAt(token string) *time.Time {
	claims := jwtlib.MapClaims{}
	if _, _, err := jwtlib.NewParser().ParseUnverified(token, claims); err != nil {
		return nil
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return nil
	}
	value := expiresAt.Time.UTC()
	return &value
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

func hasRole(roles []model.Role, slug string) bool {
	return findRoleBySlug(roles, slug) != nil
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
			Status:      membership.Tenant.Status,
			Role:        membership.Role.Slug,
			Permissions: permissionsToCodes(membership.Permissions),
		})
	}
	return resp
}

func userToResponse(user *model.User) dto.UserResponse {
	roles := rolesToSlugs(user.Roles)
	return dto.UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Name:        strings.TrimSpace(user.FullName()),
		Role:        primaryRole(roles),
		Status:      user.Status,
		Roles:       roles,
		Permissions: []string{},
	}
}

func primaryRole(roles []string) string {
	for _, role := range roles {
		if role == parentRoleSlug {
			return role
		}
	}
	if len(roles) > 0 {
		return roles[0]
	}
	return ""
}

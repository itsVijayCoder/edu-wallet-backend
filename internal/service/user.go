package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/hasher"
)

type userService struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	hasher   hasher.Hasher
	redis    *redis.Client
}

func NewUserService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	h hasher.Hasher,
	rdb *redis.Client,
) UserService {
	return &userService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		hasher:   h,
		redis:    rdb,
	}
}

func (s *userService) Create(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
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

	// Resolve role IDs from slugs and assign
	roleIDs := make([]uuid.UUID, 0, len(req.RoleSlugs))
	for _, slug := range req.RoleSlugs {
		role, err := s.roleRepo.GetBySlug(ctx, slug)
		if err != nil {
			return nil, apperror.New("INVALID_ROLE", fmt.Sprintf("role '%s' not found", slug), 400)
		}
		roleIDs = append(roleIDs, role.ID)
	}

	if err := s.userRepo.AssignRoles(ctx, user.ID, roleIDs); err != nil {
		return nil, fmt.Errorf("assign roles: %w", err)
	}

	// Reload with roles
	user, err = s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("reload user: %w", err)
	}

	resp := userToResponse(user)
	return &resp, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, apperror.ErrNotFound
	}
	resp := userToResponse(user)
	return &resp, nil
}

func (s *userService) List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[dto.UserResponse], error) {
	result, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	responses := make([]dto.UserResponse, len(result.Data))
	for i := range result.Data {
		responses[i] = userToResponse(&result.Data[i])
	}

	return model.NewPaginatedResult(responses, result.Total, result.Page, result.PageSize), nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, apperror.ErrNotFound
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	// Update roles if provided
	if len(req.RoleSlugs) > 0 {
		roleIDs := make([]uuid.UUID, 0, len(req.RoleSlugs))
		for _, slug := range req.RoleSlugs {
			role, err := s.roleRepo.GetBySlug(ctx, slug)
			if err != nil {
				return nil, apperror.New("INVALID_ROLE", fmt.Sprintf("role '%s' not found", slug), 400)
			}
			roleIDs = append(roleIDs, role.ID)
		}
		if err := s.userRepo.AssignRoles(ctx, user.ID, roleIDs); err != nil {
			return nil, fmt.Errorf("assign roles: %w", err)
		}
	}

	user, _ = s.userRepo.GetByID(ctx, id)
	resp := userToResponse(user)
	return &resp, nil
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return apperror.ErrNotFound
	}
	return s.userRepo.SoftDelete(ctx, id)
}

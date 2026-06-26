package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/hasher"
)

type SuperAdminBootstrapInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

func BootstrapSuperAdmin(
	ctx context.Context,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	h hasher.Hasher,
	input SuperAdminBootstrapInput,
) (*dto.UserResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, fmt.Errorf("super admin email is required")
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("super admin password must be at least 8 characters")
	}

	role, err := roleRepo.GetBySlug(ctx, "super_admin")
	if err != nil {
		return nil, fmt.Errorf("lookup super_admin role: %w", err)
	}
	if role == nil {
		return nil, fmt.Errorf("super_admin role is missing; run database migrations first")
	}

	passwordHash, err := h.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash super admin password: %w", err)
	}

	user, err := userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup super admin user: %w", err)
	}

	if user == nil {
		user = &model.User{
			Email:        email,
			PasswordHash: passwordHash,
			FirstName:    defaultBootstrapName(input.FirstName, "EduWallet"),
			LastName:     defaultBootstrapName(input.LastName, "Owner"),
			Status:       "active",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("create super admin user: %w", err)
		}
	} else {
		user.Email = email
		user.PasswordHash = passwordHash
		user.FirstName = defaultBootstrapName(input.FirstName, user.FirstName)
		user.LastName = defaultBootstrapName(input.LastName, user.LastName)
		user.Status = "active"
		if err := userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("update super admin user: %w", err)
		}
	}

	if err := userRepo.AssignRoles(ctx, user.ID, []uuid.UUID{role.ID}); err != nil {
		return nil, fmt.Errorf("assign super_admin role: %w", err)
	}

	user, err = userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("reload super admin user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("reload super admin user: user not found")
	}

	resp := userToResponse(user)
	return &resp, nil
}

func defaultBootstrapName(value, fallback string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return strings.TrimSpace(fallback)
	}
	return clean
}

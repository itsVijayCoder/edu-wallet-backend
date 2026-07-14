package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

const parentRoleSlug = "parents"

// guardianStore is the narrow slice of AcademicRepository that ParentService
// depends on. Defining a small interface here keeps the test surface small and
// explicit (Go idiom: accept interfaces, return structs). The concrete
// repository.AcademicRepository implementation satisfies it without any
// adapter.
type guardianStore interface {
	GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*model.Guardian, error)
	ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[model.Guardian], error)
	ListGuardianStudents(ctx context.Context, tenantID, guardianID uuid.UUID) ([]model.GuardianStudent, error)
	ListGuardianStudentsByGuardianIDs(ctx context.Context, tenantID uuid.UUID, guardianIDs []uuid.UUID) (map[uuid.UUID][]model.GuardianStudent, error)
	SetGuardianUserID(ctx context.Context, tenantID, guardianID uuid.UUID, userID *uuid.UUID) error
}

// parentService implements ParentService. It depends on the academic + user
// repositories plus the audit log so it can record link/unlink events without
// pulling those domains' services together (which would create a circular
// dependency between AcademicService and UserService).
type parentService struct {
	guardians guardianStore
	userRepo  repository.UserRepository
	roleRepo  repository.RoleRepository
	auditRepo repository.AuditRepository
}

// NewParentService wires the parent orchestration service. The academicRepo is
// shared with AcademicService on purpose: it is stateless and DBTX-backed so
// both services can use the same repository instance for both the read and
// write sides of guardian records.
func NewParentService(
	academicRepo repository.AcademicRepository,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	auditRepo repository.AuditRepository,
) ParentService {
	return &parentService{
		guardians: academicRepo,
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		auditRepo: auditRepo,
	}
}

func (s *parentService) LinkGuardianUser(ctx context.Context, actorID, tenantID, guardianID, userID uuid.UUID) (*dto.GuardianResponse, error) {
	guardian, err := s.guardians.GetGuardian(ctx, tenantID, guardianID)
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return nil, apperror.ErrNotFound
	}
	if guardian.UserID != nil && *guardian.UserID == userID {
		resp := guardianToResponse(guardian)
		return &resp, nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, apperror.ErrNotFound
	}
	if err := requireParentRole(ctx, s.roleRepo, user); err != nil {
		return nil, err
	}

	if err := s.guardians.SetGuardianUserID(ctx, tenantID, guardianID, &userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperror.New("GUARDIAN_USER_ALREADY_LINKED", "user is already linked to another guardian in this tenant", 409)
		}
		return nil, fmt.Errorf("link guardian user: %w", err)
	}

	if err := s.audit(ctx, tenantID, actorID, "guardian_user.linked", "guardian", guardianID, "linked guardian user account", map[string]any{"user_id": userID.String()}); err != nil {
		return nil, err
	}

	guardian.UserID = &userID
	resp := guardianToResponse(guardian)
	resp.UserStatus = userStatusPtr(user.Status)
	return &resp, nil
}

func (s *parentService) UnlinkGuardianUser(ctx context.Context, actorID, tenantID, guardianID uuid.UUID) (*dto.GuardianResponse, error) {
	guardian, err := s.guardians.GetGuardian(ctx, tenantID, guardianID)
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return nil, apperror.ErrNotFound
	}
	if guardian.UserID == nil {
		resp := guardianToResponse(guardian)
		return &resp, nil
	}

	previousUserID := *guardian.UserID
	if err := s.guardians.SetGuardianUserID(ctx, tenantID, guardianID, nil); err != nil {
		return nil, fmt.Errorf("unlink guardian user: %w", err)
	}

	if err := s.audit(ctx, tenantID, actorID, "guardian_user.unlinked", "guardian", guardianID, "unlinked guardian user account", map[string]any{"user_id": previousUserID.String()}); err != nil {
		return nil, err
	}

	guardian.UserID = nil
	guardian.UserStatus = nil
	resp := guardianToResponse(guardian)
	return &resp, nil
}

func (s *parentService) ListGuardianStudents(ctx context.Context, tenantID, guardianID uuid.UUID) ([]dto.GuardianStudentResponse, error) {
	guardian, err := s.guardians.GetGuardian(ctx, tenantID, guardianID)
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return nil, apperror.ErrNotFound
	}

	students, err := s.guardians.ListGuardianStudents(ctx, tenantID, guardianID)
	if err != nil {
		return nil, fmt.Errorf("list guardian students: %w", err)
	}

	resp := make([]dto.GuardianStudentResponse, 0, len(students))
	for i := range students {
		resp = append(resp, guardianStudentToResponse(&students[i]))
	}
	return resp, nil
}

func (s *parentService) ListParents(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ParentSummaryResponse], error) {
	result, err := s.guardians.ListGuardians(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list guardians: %w", err)
	}

	guardianIDs := make([]uuid.UUID, 0, len(result.Data))
	for i := range result.Data {
		guardianIDs = append(guardianIDs, result.Data[i].ID)
	}
	studentsByGuardian, err := s.guardians.ListGuardianStudentsByGuardianIDs(ctx, tenantID, guardianIDs)
	if err != nil {
		return nil, fmt.Errorf("list guardian students: %w", err)
	}

	summaries := make([]dto.ParentSummaryResponse, 0, len(result.Data))
	for i := range result.Data {
		guardian := &result.Data[i]
		students := studentsByGuardian[guardian.ID]
		studentResponses := make([]dto.GuardianStudentResponse, 0, len(students))
		for j := range students {
			studentResponses = append(studentResponses, guardianStudentToResponse(&students[j]))
		}

		summaries = append(summaries, dto.ParentSummaryResponse{
			GuardianID:     guardian.ID,
			Name:           guardian.Name,
			Relationship:   guardian.Relationship,
			Phone:          guardian.Phone,
			Email:          guardian.Email,
			UserID:         guardian.UserID,
			UserStatus:     guardian.UserStatus,
			LinkedStudents: studentResponses,
		})
	}

	return model.NewPaginatedResult(summaries, result.Total, result.Page, result.PageSize), nil
}

func (s *parentService) audit(ctx context.Context, tenantID, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, summary string, metadata map[string]any) error {
	if s.auditRepo == nil {
		return nil
	}
	entry := &model.AuditLog{
		TenantID:    &tenantID,
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    &entityID,
		Summary:     summary,
		Metadata:    normalizeMetadata(metadata),
	}
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func requireParentRole(ctx context.Context, roleRepo repository.RoleRepository, user *model.User) error {
	hasRole := false
	for i := range user.Roles {
		if strings.EqualFold(user.Roles[i].Slug, parentRoleSlug) {
			hasRole = true
			break
		}
	}
	if hasRole {
		return nil
	}
	// Some user records loaded through GetByID already include their roles.
	// If for some reason they don't, double-check by asking the repository to
	// load the roles directly. This keeps the link operation safe even when the
	// user is missing roles on the join.
	roles, err := roleRepo.GetBySlug(ctx, parentRoleSlug)
	if err != nil || roles == nil {
		return apperror.New("PARENT_ROLE_MISSING", "linked user does not have the parents role", 400)
	}
	for i := range user.Roles {
		if user.Roles[i].ID == roles.ID {
			return nil
		}
	}
	return apperror.New("PARENT_ROLE_MISSING", "linked user does not have the parents role", 400)
}

func guardianStudentToResponse(item *model.GuardianStudent) dto.GuardianStudentResponse {
	return dto.GuardianStudentResponse{
		StudentID:       item.StudentID,
		AdmissionNumber: item.AdmissionNumber,
		FirstName:       item.FirstName,
		LastName:        item.LastName,
		Relationship:    item.Relationship,
		IsPrimary:       item.IsPrimary,
		ClassName:       item.ClassName,
		SectionName:     item.SectionName,
		Status:          item.Status,
	}
}

func userStatusPtr(status string) *string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

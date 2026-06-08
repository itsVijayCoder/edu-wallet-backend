package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var (
	slugPattern       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	branchCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]*$`)
)

type tenantService struct {
	tenantRepo     repository.TenantRepository
	membershipRepo repository.TenantMembershipRepository
	roleRepo       repository.RoleRepository
	auditRepo      repository.AuditRepository
}

func NewTenantService(
	tenantRepo repository.TenantRepository,
	membershipRepo repository.TenantMembershipRepository,
	roleRepo repository.RoleRepository,
	auditRepo repository.AuditRepository,
) TenantService {
	return &tenantService{
		tenantRepo:     tenantRepo,
		membershipRepo: membershipRepo,
		roleRepo:       roleRepo,
		auditRepo:      auditRepo,
	}
}

func (s *tenantService) Create(ctx context.Context, actorID uuid.UUID, req dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	slug, err := normalizeSlug(req.Slug)
	if err != nil {
		return nil, err
	}

	existing, err := s.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("lookup tenant slug: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	tenant := &model.Tenant{
		Name:         strings.TrimSpace(req.Name),
		Slug:         slug,
		LegalName:    defaultString(strings.TrimSpace(req.LegalName), strings.TrimSpace(req.Name)),
		Domain:       cleanOptionalString(req.Domain),
		ContactEmail: cleanOptionalString(req.ContactEmail),
		ContactPhone: cleanOptionalString(req.ContactPhone),
		Status:       defaultString(req.Status, "active"),
		Address:      addressFromRequest(req.Address),
		Metadata:     normalizeMetadata(req.Metadata),
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	if req.OwnerUserID != nil {
		if err := s.createOwnerMembership(ctx, tenant.ID, *req.OwnerUserID); err != nil {
			return nil, err
		}
	}

	var branches []model.TenantBranch
	if req.Branch != nil {
		branch, err := s.createBranchModel(ctx, tenant.ID, *req.Branch)
		if err != nil {
			return nil, err
		}
		branches = append(branches, *branch)
		if err := s.audit(ctx, &tenant.ID, actorID, "tenant_branch.created", "tenant_branch", branch.ID, "tenant branch created", map[string]any{
			"code": branch.Code,
			"name": branch.Name,
		}); err != nil {
			return nil, err
		}
	}

	if err := s.audit(ctx, &tenant.ID, actorID, "tenant.created", "tenant", tenant.ID, "tenant created", map[string]any{
		"slug": tenant.Slug,
		"name": tenant.Name,
	}); err != nil {
		return nil, err
	}

	resp := tenantToResponse(tenant, branches)
	return &resp, nil
}

func (s *tenantService) List(ctx context.Context, params model.PaginationParams) (*model.PaginatedResult[dto.TenantResponse], error) {
	result, err := s.tenantRepo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}

	responses := make([]dto.TenantResponse, len(result.Data))
	for i := range result.Data {
		responses[i] = tenantToResponse(&result.Data[i], nil)
	}

	return model.NewPaginatedResult(responses, result.Total, result.Page, result.PageSize), nil
}

func (s *tenantService) GetByID(ctx context.Context, id uuid.UUID) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return nil, apperror.ErrNotFound
	}

	branches, err := s.tenantRepo.ListBranches(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("list tenant branches: %w", err)
	}

	resp := tenantToResponse(tenant, branches)
	return &resp, nil
}

func (s *tenantService) Update(ctx context.Context, actorID, id uuid.UUID, req dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return nil, apperror.ErrNotFound
	}

	if req.Name != nil {
		tenant.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		slug, err := normalizeSlug(*req.Slug)
		if err != nil {
			return nil, err
		}
		if slug != tenant.Slug {
			existing, err := s.tenantRepo.GetBySlug(ctx, slug)
			if err != nil {
				return nil, fmt.Errorf("lookup tenant slug: %w", err)
			}
			if existing != nil && existing.ID != tenant.ID {
				return nil, apperror.ErrConflict
			}
		}
		tenant.Slug = slug
	}
	if req.LegalName != nil {
		tenant.LegalName = strings.TrimSpace(*req.LegalName)
	}
	if req.Domain != nil {
		tenant.Domain = cleanOptionalString(req.Domain)
	}
	if req.ContactEmail != nil {
		tenant.ContactEmail = cleanOptionalString(req.ContactEmail)
	}
	if req.ContactPhone != nil {
		tenant.ContactPhone = cleanOptionalString(req.ContactPhone)
	}
	if req.Status != nil {
		tenant.Status = strings.TrimSpace(*req.Status)
	}
	if req.Address != nil {
		tenant.Address = addressFromRequest(*req.Address)
	}
	if req.Metadata != nil {
		tenant.Metadata = normalizeMetadata(req.Metadata)
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	if err := s.audit(ctx, &tenant.ID, actorID, "tenant.updated", "tenant", tenant.ID, "tenant updated", map[string]any{
		"slug": tenant.Slug,
	}); err != nil {
		return nil, err
	}

	branches, err := s.tenantRepo.ListBranches(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("list tenant branches: %w", err)
	}
	resp := tenantToResponse(tenant, branches)
	return &resp, nil
}

func (s *tenantService) CreateBranch(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateBranchRequest) (*dto.BranchResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if tenant == nil {
		return nil, apperror.ErrNotFound
	}

	branch, err := s.createBranchModel(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	if err := s.audit(ctx, &tenantID, actorID, "tenant_branch.created", "tenant_branch", branch.ID, "tenant branch created", map[string]any{
		"code": branch.Code,
		"name": branch.Name,
	}); err != nil {
		return nil, err
	}

	resp := branchToResponse(branch)
	return &resp, nil
}

func (s *tenantService) GetCurrent(ctx context.Context, tenantID uuid.UUID) (*dto.TenantResponse, error) {
	return s.GetByID(ctx, tenantID)
}

func (s *tenantService) UpdateCurrent(ctx context.Context, actorID, tenantID uuid.UUID, req dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	return s.Update(ctx, actorID, tenantID, req)
}

func (s *tenantService) createOwnerMembership(ctx context.Context, tenantID, userID uuid.UUID) error {
	role, err := s.roleRepo.GetBySlug(ctx, "admin")
	if err != nil {
		return fmt.Errorf("lookup admin role: %w", err)
	}
	if role == nil {
		return apperror.New("ROLE_NOT_FOUND", "admin role not found", 500)
	}

	now := time.Now()
	membership := &model.TenantMembership{
		TenantID: tenantID,
		UserID:   userID,
		RoleID:   role.ID,
		Status:   "active",
		JoinedAt: &now,
	}
	if err := s.membershipRepo.CreateMembership(ctx, membership); err != nil {
		return fmt.Errorf("create owner membership: %w", err)
	}
	return nil
}

func (s *tenantService) createBranchModel(ctx context.Context, tenantID uuid.UUID, req dto.CreateBranchRequest) (*model.TenantBranch, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if !branchCodePattern.MatchString(code) {
		return nil, apperror.New("INVALID_BRANCH_CODE", "branch code must contain only letters, numbers, underscores, or dashes", 400)
	}

	branch := &model.TenantBranch{
		TenantID:     tenantID,
		Name:         strings.TrimSpace(req.Name),
		Code:         code,
		ContactEmail: cleanOptionalString(req.ContactEmail),
		ContactPhone: cleanOptionalString(req.ContactPhone),
		Status:       defaultString(req.Status, "active"),
		Address:      addressFromRequest(req.Address),
		Metadata:     normalizeMetadata(req.Metadata),
	}

	if err := s.tenantRepo.CreateBranch(ctx, branch); err != nil {
		return nil, fmt.Errorf("create tenant branch: %w", err)
	}
	return branch, nil
}

func (s *tenantService) audit(
	ctx context.Context,
	tenantID *uuid.UUID,
	actorID uuid.UUID,
	action string,
	entityType string,
	entityID uuid.UUID,
	summary string,
	metadata map[string]any,
) error {
	if s.auditRepo == nil {
		return nil
	}

	entry := &model.AuditLog{
		TenantID:    tenantID,
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    &entityID,
		Summary:     summary,
		Metadata:    metadata,
	}
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func normalizeSlug(value string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(value))
	if !slugPattern.MatchString(slug) {
		return "", apperror.New("INVALID_TENANT_SLUG", "tenant slug must contain lowercase letters, numbers, and single dashes", 400)
	}
	return slug, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	clean := strings.TrimSpace(*value)
	if clean == "" {
		return nil
	}
	return &clean
}

func addressFromRequest(req dto.AddressRequest) model.Address {
	return model.Address{
		Line1:      strings.TrimSpace(req.Line1),
		Line2:      strings.TrimSpace(req.Line2),
		City:       strings.TrimSpace(req.City),
		State:      strings.TrimSpace(req.State),
		PostalCode: strings.TrimSpace(req.PostalCode),
		Country:    defaultString(req.Country, "India"),
	}
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func tenantToResponse(tenant *model.Tenant, branches []model.TenantBranch) dto.TenantResponse {
	resp := dto.TenantResponse{
		ID:           tenant.ID,
		Name:         tenant.Name,
		Slug:         tenant.Slug,
		LegalName:    tenant.LegalName,
		Domain:       tenant.Domain,
		ContactEmail: tenant.ContactEmail,
		ContactPhone: tenant.ContactPhone,
		Status:       tenant.Status,
		Address:      addressToResponse(tenant.Address),
		Metadata:     tenant.Metadata,
		CreatedAt:    tenant.CreatedAt,
		UpdatedAt:    tenant.UpdatedAt,
	}
	for i := range branches {
		resp.Branches = append(resp.Branches, branchToResponse(&branches[i]))
	}
	return resp
}

func branchToResponse(branch *model.TenantBranch) dto.BranchResponse {
	return dto.BranchResponse{
		ID:           branch.ID,
		TenantID:     branch.TenantID,
		Name:         branch.Name,
		Code:         branch.Code,
		ContactEmail: branch.ContactEmail,
		ContactPhone: branch.ContactPhone,
		Status:       branch.Status,
		Address:      addressToResponse(branch.Address),
		Metadata:     branch.Metadata,
		CreatedAt:    branch.CreatedAt,
		UpdatedAt:    branch.UpdatedAt,
	}
}

func addressToResponse(address model.Address) dto.AddressResponse {
	return dto.AddressResponse{
		Line1:      address.Line1,
		Line2:      address.Line2,
		City:       address.City,
		State:      address.State,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

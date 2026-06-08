package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

const (
	dateLayout            = "2006-01-02"
	studentImportType     = "students"
	defaultCurrency       = "INR"
	importMetadataRowsKey = "students"
)

var studentImportHeaders = []string{
	"admission_number",
	"first_name",
	"last_name",
	"academic_year_code",
	"class_code",
	"section_code",
	"roll_number",
	"status",
	"category",
	"phone",
	"email",
	"address_line1",
	"address_line2",
	"city",
	"state",
	"postal_code",
	"country",
	"opening_balance_paise",
	"guardian_name",
	"guardian_relationship",
	"guardian_phone",
	"guardian_whatsapp_phone",
	"guardian_email",
	"guardian_preferred_language",
	"guardian_communication_opt_in",
}

type academicService struct {
	repo        repository.AcademicRepository
	repoFactory repository.AcademicRepositoryFactory
	tx          database.Transactor
	auditRepo   repository.AuditRepository
}

func NewAcademicService(
	repo repository.AcademicRepository,
	repoFactory repository.AcademicRepositoryFactory,
	tx database.Transactor,
	auditRepo repository.AuditRepository,
) AcademicService {
	return &academicService{
		repo:        repo,
		repoFactory: repoFactory,
		tx:          tx,
		auditRepo:   auditRepo,
	}
}

func (s *academicService) CreateAcademicYear(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateAcademicYearRequest) (*dto.AcademicYearResponse, error) {
	startDate, endDate, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	code, err := normalizeCode(req.Code, "INVALID_ACADEMIC_YEAR_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetAcademicYearByCode(ctx, tenantID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup academic year code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	academicYear := &model.AcademicYear{
		TenantID:  tenantID,
		Name:      strings.TrimSpace(req.Name),
		Code:      code,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    defaultString(req.Status, "active"),
		IsActive:  req.IsActive,
		Metadata:  normalizeMetadata(req.Metadata),
	}

	if err := s.repo.CreateAcademicYear(ctx, academicYear); err != nil {
		return nil, mapPersistenceError(err, "create academic year")
	}
	if err := s.audit(ctx, tenantID, actorID, "academic_year.created", "academic_year", academicYear.ID, "academic year created", map[string]any{"code": academicYear.Code}); err != nil {
		return nil, err
	}

	resp := academicYearToResponse(academicYear)
	return &resp, nil
}

func (s *academicService) ListAcademicYears(ctx context.Context, tenantID uuid.UUID, filter model.AcademicYearFilter, params model.PaginationParams) (*model.PaginatedResult[dto.AcademicYearResponse], error) {
	result, err := s.repo.ListAcademicYears(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list academic years: %w", err)
	}
	items := make([]dto.AcademicYearResponse, len(result.Data))
	for i := range result.Data {
		items[i] = academicYearToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) GetAcademicYear(ctx context.Context, tenantID, id uuid.UUID) (*dto.AcademicYearResponse, error) {
	academicYear, err := s.repo.GetAcademicYear(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get academic year: %w", err)
	}
	if academicYear == nil {
		return nil, apperror.ErrNotFound
	}
	resp := academicYearToResponse(academicYear)
	return &resp, nil
}

func (s *academicService) UpdateAcademicYear(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateAcademicYearRequest) (*dto.AcademicYearResponse, error) {
	academicYear, err := s.repo.GetAcademicYear(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get academic year: %w", err)
	}
	if academicYear == nil {
		return nil, apperror.ErrNotFound
	}

	if req.Name != nil {
		academicYear.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_ACADEMIC_YEAR_CODE")
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(code, academicYear.Code) {
			existing, err := s.repo.GetAcademicYearByCode(ctx, tenantID, code)
			if err != nil {
				return nil, fmt.Errorf("lookup academic year code: %w", err)
			}
			if existing != nil && existing.ID != academicYear.ID {
				return nil, apperror.ErrConflict
			}
		}
		academicYear.Code = code
	}
	if req.StartDate != nil {
		parsed, err := parseDate(*req.StartDate)
		if err != nil {
			return nil, err
		}
		academicYear.StartDate = parsed
	}
	if req.EndDate != nil {
		parsed, err := parseDate(*req.EndDate)
		if err != nil {
			return nil, err
		}
		academicYear.EndDate = parsed
	}
	if academicYear.EndDate.Before(academicYear.StartDate) {
		return nil, apperror.New("INVALID_DATE_RANGE", "end_date must be on or after start_date", 400)
	}
	if req.Status != nil {
		academicYear.Status = strings.TrimSpace(*req.Status)
	}
	if req.IsActive != nil {
		academicYear.IsActive = *req.IsActive
	}
	if req.Metadata != nil {
		academicYear.Metadata = normalizeMetadata(req.Metadata)
	}

	if err := s.repo.UpdateAcademicYear(ctx, academicYear); err != nil {
		return nil, mapPersistenceError(err, "update academic year")
	}
	if err := s.audit(ctx, tenantID, actorID, "academic_year.updated", "academic_year", academicYear.ID, "academic year updated", map[string]any{"code": academicYear.Code}); err != nil {
		return nil, err
	}

	resp := academicYearToResponse(academicYear)
	return &resp, nil
}

func (s *academicService) DeleteAcademicYear(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	academicYear, err := s.repo.GetAcademicYear(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get academic year: %w", err)
	}
	if academicYear == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteAcademicYear(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete academic year: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "academic_year.deleted", "academic_year", id, "academic year deleted", map[string]any{"code": academicYear.Code})
}

func (s *academicService) CreateClass(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateClassRequest) (*dto.ClassResponse, error) {
	code, err := normalizeCode(req.Code, "INVALID_CLASS_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetClassByCode(ctx, tenantID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup class code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	class := &model.Class{
		TenantID:  tenantID,
		Name:      strings.TrimSpace(req.Name),
		Code:      code,
		SortOrder: req.SortOrder,
		Status:    defaultString(req.Status, "active"),
		Metadata:  normalizeMetadata(req.Metadata),
	}
	if err := s.repo.CreateClass(ctx, class); err != nil {
		return nil, mapPersistenceError(err, "create class")
	}
	if err := s.audit(ctx, tenantID, actorID, "class.created", "class", class.ID, "class created", map[string]any{"code": class.Code}); err != nil {
		return nil, err
	}
	resp := classToResponse(class)
	return &resp, nil
}

func (s *academicService) ListClasses(ctx context.Context, tenantID uuid.UUID, filter model.ClassFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ClassResponse], error) {
	result, err := s.repo.ListClasses(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	items := make([]dto.ClassResponse, len(result.Data))
	for i := range result.Data {
		items[i] = classToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) GetClass(ctx context.Context, tenantID, id uuid.UUID) (*dto.ClassResponse, error) {
	class, err := s.repo.GetClass(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get class: %w", err)
	}
	if class == nil {
		return nil, apperror.ErrNotFound
	}
	resp := classToResponse(class)
	return &resp, nil
}

func (s *academicService) UpdateClass(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateClassRequest) (*dto.ClassResponse, error) {
	class, err := s.repo.GetClass(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get class: %w", err)
	}
	if class == nil {
		return nil, apperror.ErrNotFound
	}
	if req.Name != nil {
		class.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_CLASS_CODE")
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(code, class.Code) {
			existing, err := s.repo.GetClassByCode(ctx, tenantID, code)
			if err != nil {
				return nil, fmt.Errorf("lookup class code: %w", err)
			}
			if existing != nil && existing.ID != class.ID {
				return nil, apperror.ErrConflict
			}
		}
		class.Code = code
	}
	if req.SortOrder != nil {
		class.SortOrder = *req.SortOrder
	}
	if req.Status != nil {
		class.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		class.Metadata = normalizeMetadata(req.Metadata)
	}

	if err := s.repo.UpdateClass(ctx, class); err != nil {
		return nil, mapPersistenceError(err, "update class")
	}
	if err := s.audit(ctx, tenantID, actorID, "class.updated", "class", class.ID, "class updated", map[string]any{"code": class.Code}); err != nil {
		return nil, err
	}
	resp := classToResponse(class)
	return &resp, nil
}

func (s *academicService) DeleteClass(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	class, err := s.repo.GetClass(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get class: %w", err)
	}
	if class == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteClass(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete class: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "class.deleted", "class", id, "class deleted", map[string]any{"code": class.Code})
}

func (s *academicService) CreateSection(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateSectionRequest) (*dto.SectionResponse, error) {
	code, err := normalizeCode(req.Code, "INVALID_SECTION_CODE")
	if err != nil {
		return nil, err
	}
	if err := validateCapacity(req.Capacity); err != nil {
		return nil, err
	}
	if err := s.validateAcademicStructure(ctx, s.repo, tenantID, req.AcademicYearID, req.ClassID, nil); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetSectionByCode(ctx, tenantID, req.AcademicYearID, req.ClassID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup section code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	section := &model.Section{
		TenantID:       tenantID,
		AcademicYearID: req.AcademicYearID,
		ClassID:        req.ClassID,
		BranchID:       req.BranchID,
		Name:           strings.TrimSpace(req.Name),
		Code:           code,
		Capacity:       req.Capacity,
		Status:         defaultString(req.Status, "active"),
		Metadata:       normalizeMetadata(req.Metadata),
	}
	if err := s.repo.CreateSection(ctx, section); err != nil {
		return nil, mapPersistenceError(err, "create section")
	}
	if err := s.audit(ctx, tenantID, actorID, "section.created", "section", section.ID, "section created", map[string]any{"code": section.Code}); err != nil {
		return nil, err
	}
	return s.GetSection(ctx, tenantID, section.ID)
}

func (s *academicService) ListSections(ctx context.Context, tenantID uuid.UUID, filter model.SectionFilter, params model.PaginationParams) (*model.PaginatedResult[dto.SectionResponse], error) {
	result, err := s.repo.ListSections(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	items := make([]dto.SectionResponse, len(result.Data))
	for i := range result.Data {
		items[i] = sectionToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) GetSection(ctx context.Context, tenantID, id uuid.UUID) (*dto.SectionResponse, error) {
	section, err := s.repo.GetSection(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get section: %w", err)
	}
	if section == nil {
		return nil, apperror.ErrNotFound
	}
	resp := sectionToResponse(section)
	return &resp, nil
}

func (s *academicService) UpdateSection(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateSectionRequest) (*dto.SectionResponse, error) {
	section, err := s.repo.GetSection(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get section: %w", err)
	}
	if section == nil {
		return nil, apperror.ErrNotFound
	}
	if req.AcademicYearID != nil {
		section.AcademicYearID = *req.AcademicYearID
	}
	if req.ClassID != nil {
		section.ClassID = *req.ClassID
	}
	if req.BranchID != nil {
		section.BranchID = req.BranchID
	}
	if req.Name != nil {
		section.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_SECTION_CODE")
		if err != nil {
			return nil, err
		}
		section.Code = code
	}
	if req.Capacity != nil {
		if err := validateCapacity(req.Capacity); err != nil {
			return nil, err
		}
		section.Capacity = req.Capacity
	}
	if req.Status != nil {
		section.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		section.Metadata = normalizeMetadata(req.Metadata)
	}

	if err := s.validateAcademicStructure(ctx, s.repo, tenantID, section.AcademicYearID, section.ClassID, nil); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetSectionByCode(ctx, tenantID, section.AcademicYearID, section.ClassID, section.Code)
	if err != nil {
		return nil, fmt.Errorf("lookup section code: %w", err)
	}
	if existing != nil && existing.ID != section.ID {
		return nil, apperror.ErrConflict
	}

	if err := s.repo.UpdateSection(ctx, section); err != nil {
		return nil, mapPersistenceError(err, "update section")
	}
	if err := s.audit(ctx, tenantID, actorID, "section.updated", "section", section.ID, "section updated", map[string]any{"code": section.Code}); err != nil {
		return nil, err
	}
	return s.GetSection(ctx, tenantID, section.ID)
}

func (s *academicService) DeleteSection(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	section, err := s.repo.GetSection(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get section: %w", err)
	}
	if section == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteSection(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete section: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "section.deleted", "section", id, "section deleted", map[string]any{"code": section.Code})
}

func (s *academicService) CreateGuardian(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateGuardianRequest) (*dto.GuardianResponse, error) {
	optIn := true
	if req.CommunicationOptIn != nil {
		optIn = *req.CommunicationOptIn
	}
	guardian := &model.Guardian{
		TenantID:           tenantID,
		Name:               strings.TrimSpace(req.Name),
		Relationship:       strings.TrimSpace(req.Relationship),
		Phone:              cleanOptionalString(req.Phone),
		WhatsAppPhone:      cleanOptionalString(req.WhatsAppPhone),
		Email:              cleanOptionalString(req.Email),
		PreferredLanguage:  defaultString(req.PreferredLanguage, "en"),
		CommunicationOptIn: optIn,
		Address:            addressFromRequest(req.Address),
		Metadata:           normalizeMetadata(req.Metadata),
	}
	if err := validateOptionalEmail(guardian.Email); err != nil {
		return nil, err
	}
	if err := s.repo.CreateGuardian(ctx, guardian); err != nil {
		return nil, mapPersistenceError(err, "create guardian")
	}
	if err := s.audit(ctx, tenantID, actorID, "guardian.created", "guardian", guardian.ID, "guardian created", map[string]any{"name": guardian.Name}); err != nil {
		return nil, err
	}
	resp := guardianToResponse(guardian)
	return &resp, nil
}

func (s *academicService) ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[dto.GuardianResponse], error) {
	result, err := s.repo.ListGuardians(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list guardians: %w", err)
	}
	items := make([]dto.GuardianResponse, len(result.Data))
	for i := range result.Data {
		items[i] = guardianToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*dto.GuardianResponse, error) {
	guardian, err := s.repo.GetGuardian(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return nil, apperror.ErrNotFound
	}
	resp := guardianToResponse(guardian)
	return &resp, nil
}

func (s *academicService) UpdateGuardian(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateGuardianRequest) (*dto.GuardianResponse, error) {
	guardian, err := s.repo.GetGuardian(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return nil, apperror.ErrNotFound
	}
	if req.Name != nil {
		guardian.Name = strings.TrimSpace(*req.Name)
	}
	if req.Relationship != nil {
		guardian.Relationship = strings.TrimSpace(*req.Relationship)
	}
	if req.Phone != nil {
		guardian.Phone = cleanOptionalString(req.Phone)
	}
	if req.WhatsAppPhone != nil {
		guardian.WhatsAppPhone = cleanOptionalString(req.WhatsAppPhone)
	}
	if req.Email != nil {
		guardian.Email = cleanOptionalString(req.Email)
	}
	if req.PreferredLanguage != nil {
		guardian.PreferredLanguage = defaultString(*req.PreferredLanguage, "en")
	}
	if req.CommunicationOptIn != nil {
		guardian.CommunicationOptIn = *req.CommunicationOptIn
	}
	if req.Address != nil {
		guardian.Address = addressFromRequest(*req.Address)
	}
	if req.Metadata != nil {
		guardian.Metadata = normalizeMetadata(req.Metadata)
	}
	if err := validateOptionalEmail(guardian.Email); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGuardian(ctx, guardian); err != nil {
		return nil, mapPersistenceError(err, "update guardian")
	}
	if err := s.audit(ctx, tenantID, actorID, "guardian.updated", "guardian", guardian.ID, "guardian updated", map[string]any{"name": guardian.Name}); err != nil {
		return nil, err
	}
	resp := guardianToResponse(guardian)
	return &resp, nil
}

func (s *academicService) DeleteGuardian(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	guardian, err := s.repo.GetGuardian(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get guardian: %w", err)
	}
	if guardian == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteGuardian(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete guardian: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "guardian.deleted", "guardian", id, "guardian deleted", map[string]any{"name": guardian.Name})
}

func (s *academicService) CreateStudent(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateStudentRequest) (*dto.StudentResponse, error) {
	admissionNumber := strings.TrimSpace(req.AdmissionNumber)
	student := &model.Student{}

	err := s.runInTx(ctx, func(repo repository.AcademicRepository) error {
		if existing, err := repo.GetStudentByAdmissionNumber(ctx, tenantID, admissionNumber); err != nil {
			return fmt.Errorf("lookup student admission number: %w", err)
		} else if existing != nil {
			return apperror.ErrConflict
		}
		if err := s.validateAcademicStructure(ctx, repo, tenantID, req.AcademicYearID, req.ClassID, &req.SectionID); err != nil {
			return err
		}
		links, err := s.validateGuardianLinks(ctx, repo, tenantID, req.Guardians)
		if err != nil {
			return err
		}

		student = studentFromCreateRequest(tenantID, req)
		if err := validateOptionalEmail(student.Email); err != nil {
			return err
		}
		if err := repo.CreateStudent(ctx, student); err != nil {
			return mapPersistenceError(err, "create student")
		}
		if len(links) > 0 {
			if err := repo.SetStudentGuardians(ctx, tenantID, student.ID, links); err != nil {
				return mapPersistenceError(err, "link student guardians")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "student.created", "student", student.ID, "student created", map[string]any{"admission_number": student.AdmissionNumber}); err != nil {
		return nil, err
	}
	return s.GetStudent(ctx, tenantID, student.ID)
}

func (s *academicService) ListStudents(ctx context.Context, tenantID uuid.UUID, filter model.StudentFilter, params model.PaginationParams) (*model.PaginatedResult[dto.StudentResponse], error) {
	result, err := s.repo.ListStudents(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	items := make([]dto.StudentResponse, len(result.Data))
	for i := range result.Data {
		items[i] = studentToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) GetStudent(ctx context.Context, tenantID, id uuid.UUID) (*dto.StudentResponse, error) {
	student, err := s.repo.GetStudent(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return nil, apperror.ErrNotFound
	}
	guardians, err := s.repo.ListStudentGuardians(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("list student guardians: %w", err)
	}
	student.Guardians = guardians
	resp := studentToResponse(student)
	return &resp, nil
}

func (s *academicService) UpdateStudent(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateStudentRequest) (*dto.StudentResponse, error) {
	err := s.runInTx(ctx, func(repo repository.AcademicRepository) error {
		student, err := repo.GetStudent(ctx, tenantID, id)
		if err != nil {
			return fmt.Errorf("get student: %w", err)
		}
		if student == nil {
			return apperror.ErrNotFound
		}
		if req.AcademicYearID != nil {
			student.AcademicYearID = *req.AcademicYearID
		}
		if req.ClassID != nil {
			student.ClassID = *req.ClassID
		}
		if req.SectionID != nil {
			student.SectionID = *req.SectionID
		}
		if req.BranchID != nil {
			student.BranchID = req.BranchID
		}
		if req.AdmissionNumber != nil {
			nextAdmission := strings.TrimSpace(*req.AdmissionNumber)
			if !strings.EqualFold(nextAdmission, student.AdmissionNumber) {
				existing, err := repo.GetStudentByAdmissionNumber(ctx, tenantID, nextAdmission)
				if err != nil {
					return fmt.Errorf("lookup student admission number: %w", err)
				}
				if existing != nil && existing.ID != student.ID {
					return apperror.ErrConflict
				}
			}
			student.AdmissionNumber = nextAdmission
		}
		if req.FirstName != nil {
			student.FirstName = strings.TrimSpace(*req.FirstName)
		}
		if req.LastName != nil {
			student.LastName = strings.TrimSpace(*req.LastName)
		}
		if req.RollNumber != nil {
			student.RollNumber = cleanOptionalString(req.RollNumber)
		}
		if req.Status != nil {
			student.Status = strings.TrimSpace(*req.Status)
		}
		if req.Category != nil {
			student.Category = strings.TrimSpace(*req.Category)
		}
		if req.Phone != nil {
			student.Phone = cleanOptionalString(req.Phone)
		}
		if req.Email != nil {
			student.Email = cleanOptionalString(req.Email)
		}
		if req.Address != nil {
			student.Address = addressFromRequest(*req.Address)
		}
		if req.OpeningBalancePaise != nil {
			student.OpeningBalancePaise = *req.OpeningBalancePaise
		}
		if req.Metadata != nil {
			student.Metadata = normalizeMetadata(req.Metadata)
		}
		if err := validateOptionalEmail(student.Email); err != nil {
			return err
		}
		if err := s.validateAcademicStructure(ctx, repo, tenantID, student.AcademicYearID, student.ClassID, &student.SectionID); err != nil {
			return err
		}
		if err := repo.UpdateStudent(ctx, student); err != nil {
			return mapPersistenceError(err, "update student")
		}
		if req.Guardians != nil {
			links, err := s.validateGuardianLinks(ctx, repo, tenantID, req.Guardians)
			if err != nil {
				return err
			}
			if err := repo.SetStudentGuardians(ctx, tenantID, student.ID, links); err != nil {
				return mapPersistenceError(err, "replace student guardians")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "student.updated", "student", id, "student updated", nil); err != nil {
		return nil, err
	}
	return s.GetStudent(ctx, tenantID, id)
}

func (s *academicService) DeleteStudent(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	student, err := s.repo.GetStudent(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteStudent(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete student: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "student.deleted", "student", id, "student deleted", map[string]any{"admission_number": student.AdmissionNumber})
}

func (s *academicService) LinkStudentGuardian(ctx context.Context, actorID, tenantID, studentID uuid.UUID, req dto.StudentGuardianRequest) (*dto.StudentResponse, error) {
	if err := s.runInTx(ctx, func(repo repository.AcademicRepository) error {
		student, err := repo.GetStudent(ctx, tenantID, studentID)
		if err != nil {
			return fmt.Errorf("get student: %w", err)
		}
		if student == nil {
			return apperror.ErrNotFound
		}
		guardian, err := repo.GetGuardian(ctx, tenantID, req.GuardianID)
		if err != nil {
			return fmt.Errorf("get guardian: %w", err)
		}
		if guardian == nil {
			return apperror.ErrNotFound
		}
		link := guardianLinkFromRequest(studentID, req)
		link.TenantID = tenantID
		if err := repo.LinkStudentGuardian(ctx, &link); err != nil {
			return mapPersistenceError(err, "link student guardian")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "student_guardian.linked", "student", studentID, "student guardian linked", map[string]any{"guardian_id": req.GuardianID.String()}); err != nil {
		return nil, err
	}
	return s.GetStudent(ctx, tenantID, studentID)
}

func (s *academicService) UnlinkStudentGuardian(ctx context.Context, actorID, tenantID, studentID, guardianID uuid.UUID) (*dto.StudentResponse, error) {
	student, err := s.repo.GetStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return nil, apperror.ErrNotFound
	}
	if err := s.repo.UnlinkStudentGuardian(ctx, tenantID, studentID, guardianID); err != nil {
		return nil, fmt.Errorf("unlink student guardian: %w", err)
	}
	if err := s.audit(ctx, tenantID, actorID, "student_guardian.unlinked", "student", studentID, "student guardian unlinked", map[string]any{"guardian_id": guardianID.String()}); err != nil {
		return nil, err
	}
	return s.GetStudent(ctx, tenantID, studentID)
}

func (s *academicService) StudentImportTemplate() string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write(studentImportHeaders)
	_ = writer.Write([]string{
		"ADM-001",
		"Aarav",
		"Sharma",
		"2026-27",
		"10",
		"A",
		"12",
		"active",
		"general",
		"9876543210",
		"aarav@example.test",
		"12 Main Road",
		"",
		"Bengaluru",
		"Karnataka",
		"560001",
		"India",
		"0",
		"Riya Sharma",
		"mother",
		"9876543211",
		"9876543211",
		"riya@example.test",
		"en",
		"true",
	})
	writer.Flush()
	return buf.String()
}

func (s *academicService) PreviewStudentImport(ctx context.Context, actorID, tenantID uuid.UUID, filename string, csvData []byte) (*dto.StudentImportPreviewResponse, error) {
	rows, importErrors, err := s.validateStudentImportRows(ctx, tenantID, csvData)
	if err != nil {
		return nil, err
	}

	imp := &model.Import{
		TenantID:       tenantID,
		ImportType:     studentImportType,
		Status:         "previewed",
		SourceFilename: strings.TrimSpace(filename),
		TotalRows:      len(rows) + countRowsWithErrors(importErrors),
		ValidRows:      len(rows),
		InvalidRows:    countRowsWithErrors(importErrors),
		CreatedBy:      &actorID,
		Metadata: map[string]any{
			importMetadataRowsKey: rows,
			"headers":             studentImportHeaders,
		},
	}

	err = s.runInTx(ctx, func(repo repository.AcademicRepository) error {
		if err := repo.CreateImport(ctx, imp); err != nil {
			return mapPersistenceError(err, "create import")
		}
		for i := range importErrors {
			importErrors[i].ImportID = imp.ID
			importErrors[i].TenantID = tenantID
		}
		return repo.CreateImportErrors(ctx, importErrors)
	})
	if err != nil {
		return nil, err
	}

	respErrors := make([]dto.ImportErrorResponse, len(importErrors))
	for i := range importErrors {
		respErrors[i] = importErrorToResponse(&importErrors[i])
	}
	return &dto.StudentImportPreviewResponse{
		ImportID:    imp.ID,
		TotalRows:   imp.TotalRows,
		ValidRows:   imp.ValidRows,
		InvalidRows: imp.InvalidRows,
		Errors:      respErrors,
		Template:    studentImportHeaders,
	}, nil
}

func (s *academicService) CommitStudentImport(ctx context.Context, actorID, tenantID uuid.UUID, req dto.StudentImportCommitRequest) (*dto.StudentImportCommitResponse, error) {
	imp, err := s.repo.GetImport(ctx, tenantID, req.ImportID)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}
	if imp == nil {
		return nil, apperror.ErrNotFound
	}
	if imp.ImportType != studentImportType {
		return nil, apperror.New("INVALID_IMPORT_TYPE", "import is not a student import", 400)
	}
	if imp.Status != "previewed" {
		return nil, apperror.New("IMPORT_NOT_COMMITTABLE", "import is not in previewed status", 409)
	}
	if imp.InvalidRows > 0 {
		return nil, apperror.New("IMPORT_HAS_ERRORS", "import has validation errors", 409)
	}

	rows, err := importRowsFromMetadata(imp.Metadata)
	if err != nil {
		return nil, err
	}

	committedRows := 0
	if err := s.runInTx(ctx, func(repo repository.AcademicRepository) error {
		for i := range rows {
			if existing, err := repo.GetStudentByAdmissionNumber(ctx, tenantID, rows[i].AdmissionNumber); err != nil {
				return fmt.Errorf("lookup student admission number: %w", err)
			} else if existing != nil {
				return apperror.New("IMPORT_DUPLICATE_STUDENT", "student admission number already exists", 409)
			}
			if err := s.validateAcademicStructure(ctx, repo, tenantID, rows[i].AcademicYearID, rows[i].ClassID, &rows[i].SectionID); err != nil {
				return err
			}
			student := studentFromImportRow(tenantID, rows[i])
			if err := repo.CreateStudent(ctx, student); err != nil {
				return mapPersistenceError(err, "create imported student")
			}
			if rows[i].Guardian.Name != "" {
				guardian, err := repo.FindGuardianByContact(ctx, tenantID, rows[i].Guardian.Email, rows[i].Guardian.Phone)
				if err != nil {
					return fmt.Errorf("lookup guardian: %w", err)
				}
				if guardian == nil {
					guardian = guardianFromImportRow(tenantID, rows[i].Guardian)
					if err := repo.CreateGuardian(ctx, guardian); err != nil {
						return mapPersistenceError(err, "create imported guardian")
					}
				}
				link := model.StudentGuardian{
					TenantID:     tenantID,
					StudentID:    student.ID,
					GuardianID:   guardian.ID,
					Relationship: rows[i].Guardian.Relationship,
					IsPrimary:    true,
				}
				if err := repo.LinkStudentGuardian(ctx, &link); err != nil {
					return mapPersistenceError(err, "link imported guardian")
				}
			}
			committedRows++
		}

		now := time.Now()
		imp.Status = "committed"
		imp.CommittedRows = committedRows
		imp.CommittedAt = &now
		if err := repo.UpdateImport(ctx, imp); err != nil {
			return mapPersistenceError(err, "commit import")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := s.audit(ctx, tenantID, actorID, "student_import.committed", "import", imp.ID, "student import committed", map[string]any{"committed_rows": committedRows}); err != nil {
		return nil, err
	}
	return &dto.StudentImportCommitResponse{ImportID: imp.ID, CommittedRows: committedRows}, nil
}

func (s *academicService) ListImports(ctx context.Context, tenantID uuid.UUID, filter model.ImportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ImportResponse], error) {
	result, err := s.repo.ListImports(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	items := make([]dto.ImportResponse, len(result.Data))
	for i := range result.Data {
		items[i] = importToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *academicService) validateStudentImportRows(ctx context.Context, tenantID uuid.UUID, csvData []byte) ([]studentImportRow, []model.ImportError, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(csvData, []byte("\xef\xbb\xbf"))))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, apperror.New("EMPTY_IMPORT", "CSV file is empty", 400)
		}
		return nil, nil, apperror.New("INVALID_CSV", "CSV header could not be read", 400)
	}

	headerMap := map[string]int{}
	for i, h := range header {
		headerMap[normalizeHeader(h)] = i
	}
	for _, required := range []string{"admission_number", "first_name", "academic_year_code", "class_code", "section_code"} {
		if _, ok := headerMap[required]; !ok {
			return nil, nil, apperror.New("MISSING_IMPORT_HEADER", "student import CSV is missing required header: "+required, 400)
		}
	}

	validRows := []studentImportRow{}
	importErrors := []model.ImportError{}
	seenAdmissions := map[string]int{}
	rowNumber := 1
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		rowNumber++
		if err != nil {
			importErrors = append(importErrors, model.ImportError{
				RowNumber: rowNumber,
				Field:     "",
				Message:   "row could not be parsed",
				RawData:   map[string]any{},
			})
			continue
		}
		row := rowMap(header, record)
		if isEmptyImportRow(row) {
			continue
		}

		parsed, rowErrors := s.validateStudentImportRow(ctx, tenantID, rowNumber, row, seenAdmissions)
		if len(rowErrors) > 0 {
			importErrors = append(importErrors, rowErrors...)
			continue
		}
		validRows = append(validRows, parsed)
	}
	return validRows, importErrors, nil
}

func (s *academicService) validateStudentImportRow(ctx context.Context, tenantID uuid.UUID, rowNumber int, row map[string]string, seenAdmissions map[string]int) (studentImportRow, []model.ImportError) {
	errorsForRow := []model.ImportError{}
	addError := func(field, message string) {
		errorsForRow = append(errorsForRow, model.ImportError{
			RowNumber: rowNumber,
			Field:     field,
			Message:   message,
			RawData:   stringMapToAny(row),
		})
	}

	admissionNumber := strings.TrimSpace(row["admission_number"])
	firstName := strings.TrimSpace(row["first_name"])
	lastName := strings.TrimSpace(row["last_name"])
	academicYearCode := strings.TrimSpace(row["academic_year_code"])
	classCode := strings.TrimSpace(row["class_code"])
	sectionCode := strings.TrimSpace(row["section_code"])

	if admissionNumber == "" {
		addError("admission_number", "admission_number is required")
	}
	if firstName == "" {
		addError("first_name", "first_name is required")
	}
	if academicYearCode == "" {
		addError("academic_year_code", "academic_year_code is required")
	}
	if classCode == "" {
		addError("class_code", "class_code is required")
	}
	if sectionCode == "" {
		addError("section_code", "section_code is required")
	}
	if previousRow, ok := seenAdmissions[strings.ToLower(admissionNumber)]; admissionNumber != "" && ok {
		addError("admission_number", fmt.Sprintf("duplicate admission_number in CSV; first seen on row %d", previousRow))
	} else if admissionNumber != "" {
		seenAdmissions[strings.ToLower(admissionNumber)] = rowNumber
	}

	if admissionNumber != "" {
		existingStudent, err := s.repo.GetStudentByAdmissionNumber(ctx, tenantID, admissionNumber)
		if err != nil {
			addError("admission_number", "admission_number lookup failed")
		} else if existingStudent != nil {
			addError("admission_number", "admission_number already exists")
		}
	}

	status := strings.ToLower(defaultString(row["status"], "active"))
	if !inSet(status, "active", "inactive", "transferred", "graduated") {
		addError("status", "status must be active, inactive, transferred, or graduated")
	}
	category := strings.ToLower(defaultString(row["category"], "general"))
	if !inSet(category, "general", "scholarship", "staff_child", "sibling", "custom") {
		addError("category", "category must be general, scholarship, staff_child, sibling, or custom")
	}

	openingBalance := int64(0)
	if strings.TrimSpace(row["opening_balance_paise"]) != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(row["opening_balance_paise"]), 10, 64)
		if err != nil {
			addError("opening_balance_paise", "opening_balance_paise must be an integer")
		} else {
			openingBalance = parsed
		}
	}

	email := cleanOptionalStringFromValue(row["email"])
	if err := validateOptionalEmail(email); err != nil {
		addError("email", "email must be valid")
	}
	guardianEmail := cleanOptionalStringFromValue(row["guardian_email"])
	if err := validateOptionalEmail(guardianEmail); err != nil {
		addError("guardian_email", "guardian_email must be valid")
	}

	optIn := true
	if strings.TrimSpace(row["guardian_communication_opt_in"]) != "" {
		parsed, err := parseFlexibleBool(row["guardian_communication_opt_in"])
		if err != nil {
			addError("guardian_communication_opt_in", "guardian_communication_opt_in must be true or false")
		} else {
			optIn = parsed
		}
	}

	var err error
	var academicYear *model.AcademicYear
	if academicYearCode != "" {
		academicYear, err = s.repo.GetAcademicYearByCode(ctx, tenantID, academicYearCode)
		if err != nil {
			addError("academic_year_code", "academic_year_code lookup failed")
		} else if academicYear == nil {
			addError("academic_year_code", "academic_year_code was not found")
		}
	}

	var class *model.Class
	if classCode != "" {
		class, err = s.repo.GetClassByCode(ctx, tenantID, classCode)
		if err != nil {
			addError("class_code", "class_code lookup failed")
		} else if class == nil {
			addError("class_code", "class_code was not found")
		}
	}

	var section *model.Section
	if academicYear != nil && class != nil && sectionCode != "" {
		section, err = s.repo.GetSectionByCode(ctx, tenantID, academicYear.ID, class.ID, sectionCode)
		if err != nil {
			addError("section_code", "section_code lookup failed")
		} else if section == nil {
			addError("section_code", "section_code was not found for the academic year and class")
		}
	}

	guardianName := strings.TrimSpace(row["guardian_name"])
	guardianPhone := cleanOptionalStringFromValue(row["guardian_phone"])
	if guardianName == "" && (guardianPhone != nil || guardianEmail != nil) {
		addError("guardian_name", "guardian_name is required when guardian contact fields are present")
	}

	if len(errorsForRow) > 0 || academicYear == nil || class == nil || section == nil {
		return studentImportRow{}, errorsForRow
	}

	return studentImportRow{
		RowNumber:           rowNumber,
		AdmissionNumber:     admissionNumber,
		FirstName:           firstName,
		LastName:            lastName,
		AcademicYearID:      academicYear.ID,
		ClassID:             class.ID,
		SectionID:           section.ID,
		RollNumber:          cleanOptionalStringFromValue(row["roll_number"]),
		Status:              status,
		Category:            category,
		Phone:               cleanOptionalStringFromValue(row["phone"]),
		Email:               email,
		Address:             addressRequestFromImportRow(row),
		OpeningBalancePaise: openingBalance,
		Guardian: studentImportGuardian{
			Name:               guardianName,
			Relationship:       strings.TrimSpace(row["guardian_relationship"]),
			Phone:              guardianPhone,
			WhatsAppPhone:      cleanOptionalStringFromValue(row["guardian_whatsapp_phone"]),
			Email:              guardianEmail,
			PreferredLanguage:  defaultString(row["guardian_preferred_language"], "en"),
			CommunicationOptIn: optIn,
			Address:            addressRequestFromImportRow(row),
		},
	}, nil
}

func (s *academicService) validateAcademicStructure(ctx context.Context, repo repository.AcademicRepository, tenantID, academicYearID, classID uuid.UUID, sectionID *uuid.UUID) error {
	academicYear, err := repo.GetAcademicYear(ctx, tenantID, academicYearID)
	if err != nil {
		return fmt.Errorf("get academic year: %w", err)
	}
	if academicYear == nil {
		return apperror.New("ACADEMIC_YEAR_NOT_FOUND", "academic year not found for tenant", 404)
	}

	class, err := repo.GetClass(ctx, tenantID, classID)
	if err != nil {
		return fmt.Errorf("get class: %w", err)
	}
	if class == nil {
		return apperror.New("CLASS_NOT_FOUND", "class not found for tenant", 404)
	}

	if sectionID != nil {
		section, err := repo.GetSection(ctx, tenantID, *sectionID)
		if err != nil {
			return fmt.Errorf("get section: %w", err)
		}
		if section == nil {
			return apperror.New("SECTION_NOT_FOUND", "section not found for tenant", 404)
		}
		if section.AcademicYearID != academicYearID || section.ClassID != classID {
			return apperror.New("SECTION_STRUCTURE_MISMATCH", "section does not belong to the provided academic year and class", 400)
		}
	}
	return nil
}

func (s *academicService) validateGuardianLinks(ctx context.Context, repo repository.AcademicRepository, tenantID uuid.UUID, reqs []dto.StudentGuardianRequest) ([]model.StudentGuardian, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	seen := map[uuid.UUID]bool{}
	primaryCount := 0
	links := make([]model.StudentGuardian, 0, len(reqs))
	for _, req := range reqs {
		if seen[req.GuardianID] {
			return nil, apperror.New("DUPLICATE_GUARDIAN_LINK", "guardian can only be linked once to a student", 400)
		}
		seen[req.GuardianID] = true
		if req.IsPrimary {
			primaryCount++
		}
		guardian, err := repo.GetGuardian(ctx, tenantID, req.GuardianID)
		if err != nil {
			return nil, fmt.Errorf("get guardian: %w", err)
		}
		if guardian == nil {
			return nil, apperror.ErrNotFound
		}
		links = append(links, guardianLinkFromRequest(uuid.Nil, req))
	}
	if primaryCount > 1 {
		return nil, apperror.New("MULTIPLE_PRIMARY_GUARDIANS", "only one primary guardian is allowed", 400)
	}
	if primaryCount == 0 && len(links) > 0 {
		links[0].IsPrimary = true
	}
	return links, nil
}

func (s *academicService) runInTx(ctx context.Context, fn func(repository.AcademicRepository) error) error {
	if s.tx == nil || s.repoFactory == nil {
		return fn(s.repo)
	}
	return s.tx.WithinTx(ctx, func(db database.DBTX) error {
		return fn(s.repoFactory(db))
	})
}

func (s *academicService) audit(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, action string, entityType string, entityID uuid.UUID, summary string, metadata map[string]any) error {
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

type studentImportRow struct {
	RowNumber           int                   `json:"row_number"`
	AdmissionNumber     string                `json:"admission_number"`
	FirstName           string                `json:"first_name"`
	LastName            string                `json:"last_name"`
	AcademicYearID      uuid.UUID             `json:"academic_year_id"`
	ClassID             uuid.UUID             `json:"class_id"`
	SectionID           uuid.UUID             `json:"section_id"`
	RollNumber          *string               `json:"roll_number,omitempty"`
	Status              string                `json:"status"`
	Category            string                `json:"category"`
	Phone               *string               `json:"phone,omitempty"`
	Email               *string               `json:"email,omitempty"`
	Address             dto.AddressRequest    `json:"address"`
	OpeningBalancePaise int64                 `json:"opening_balance_paise"`
	Guardian            studentImportGuardian `json:"guardian"`
}

type studentImportGuardian struct {
	Name               string             `json:"name"`
	Relationship       string             `json:"relationship"`
	Phone              *string            `json:"phone,omitempty"`
	WhatsAppPhone      *string            `json:"whatsapp_phone,omitempty"`
	Email              *string            `json:"email,omitempty"`
	PreferredLanguage  string             `json:"preferred_language"`
	CommunicationOptIn bool               `json:"communication_opt_in"`
	Address            dto.AddressRequest `json:"address"`
}

func parseDateRange(start, end string) (time.Time, time.Time, error) {
	startDate, err := parseDate(start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endDate, err := parseDate(end)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, apperror.New("INVALID_DATE_RANGE", "end_date must be on or after start_date", 400)
	}
	return startDate, endDate, nil
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse(dateLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, apperror.New("INVALID_DATE", "date must use YYYY-MM-DD format", 400)
	}
	return parsed, nil
}

func normalizeCode(value, code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if !branchCodePattern.MatchString(normalized) {
		return "", apperror.New(code, "code must contain only letters, numbers, underscores, or dashes", 400)
	}
	return normalized, nil
}

func validateCapacity(capacity *int) error {
	if capacity != nil && *capacity < 0 {
		return apperror.New("INVALID_CAPACITY", "capacity cannot be negative", 400)
	}
	return nil
}

func validateOptionalEmail(value *string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(*value)); err != nil {
		return apperror.New("INVALID_EMAIL", "email must be valid", 400)
	}
	return nil
}

func mapPersistenceError(err error, context string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperror.ErrConflict
		case "23503":
			return apperror.New("INVALID_REFERENCE", "referenced resource was not found", 400)
		case "23514":
			return apperror.ErrValidationFailed
		}
	}
	return fmt.Errorf("%s: %w", context, err)
}

func studentFromCreateRequest(tenantID uuid.UUID, req dto.CreateStudentRequest) *model.Student {
	return &model.Student{
		TenantID:            tenantID,
		AcademicYearID:      req.AcademicYearID,
		ClassID:             req.ClassID,
		SectionID:           req.SectionID,
		BranchID:            req.BranchID,
		AdmissionNumber:     strings.TrimSpace(req.AdmissionNumber),
		FirstName:           strings.TrimSpace(req.FirstName),
		LastName:            strings.TrimSpace(req.LastName),
		RollNumber:          cleanOptionalString(req.RollNumber),
		Status:              defaultString(req.Status, "active"),
		Category:            defaultString(req.Category, "general"),
		Phone:               cleanOptionalString(req.Phone),
		Email:               cleanOptionalString(req.Email),
		Address:             addressFromRequest(req.Address),
		OpeningBalancePaise: req.OpeningBalancePaise,
		Currency:            defaultCurrency,
		Metadata:            normalizeMetadata(req.Metadata),
	}
}

func guardianLinkFromRequest(studentID uuid.UUID, req dto.StudentGuardianRequest) model.StudentGuardian {
	return model.StudentGuardian{
		StudentID:    studentID,
		GuardianID:   req.GuardianID,
		Relationship: strings.TrimSpace(req.Relationship),
		IsPrimary:    req.IsPrimary,
	}
}

func studentFromImportRow(tenantID uuid.UUID, row studentImportRow) *model.Student {
	return &model.Student{
		TenantID:            tenantID,
		AcademicYearID:      row.AcademicYearID,
		ClassID:             row.ClassID,
		SectionID:           row.SectionID,
		AdmissionNumber:     row.AdmissionNumber,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		RollNumber:          row.RollNumber,
		Status:              row.Status,
		Category:            row.Category,
		Phone:               row.Phone,
		Email:               row.Email,
		Address:             addressFromRequest(row.Address),
		OpeningBalancePaise: row.OpeningBalancePaise,
		Currency:            defaultCurrency,
		Metadata:            map[string]any{"import_row_number": row.RowNumber},
	}
}

func guardianFromImportRow(tenantID uuid.UUID, row studentImportGuardian) *model.Guardian {
	return &model.Guardian{
		TenantID:           tenantID,
		Name:               row.Name,
		Relationship:       row.Relationship,
		Phone:              row.Phone,
		WhatsAppPhone:      row.WhatsAppPhone,
		Email:              row.Email,
		PreferredLanguage:  defaultString(row.PreferredLanguage, "en"),
		CommunicationOptIn: row.CommunicationOptIn,
		Address:            addressFromRequest(row.Address),
		Metadata:           map[string]any{"source": "student_import"},
	}
}

func academicYearToResponse(item *model.AcademicYear) dto.AcademicYearResponse {
	return dto.AcademicYearResponse{
		ID:        item.ID,
		TenantID:  item.TenantID,
		Name:      item.Name,
		Code:      item.Code,
		StartDate: item.StartDate.Format(dateLayout),
		EndDate:   item.EndDate.Format(dateLayout),
		Status:    item.Status,
		IsActive:  item.IsActive,
		Metadata:  item.Metadata,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func classToResponse(item *model.Class) dto.ClassResponse {
	return dto.ClassResponse{
		ID:        item.ID,
		TenantID:  item.TenantID,
		Name:      item.Name,
		Code:      item.Code,
		SortOrder: item.SortOrder,
		Status:    item.Status,
		Metadata:  item.Metadata,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func sectionToResponse(item *model.Section) dto.SectionResponse {
	resp := dto.SectionResponse{
		ID:             item.ID,
		TenantID:       item.TenantID,
		AcademicYearID: item.AcademicYearID,
		ClassID:        item.ClassID,
		BranchID:       item.BranchID,
		Name:           item.Name,
		Code:           item.Code,
		Capacity:       item.Capacity,
		Status:         item.Status,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.AcademicYear != nil {
		resp.AcademicYear = &dto.LookupResponse{ID: item.AcademicYear.ID, Name: item.AcademicYear.Name, Code: item.AcademicYear.Code}
	}
	if item.Class != nil {
		resp.Class = &dto.LookupResponse{ID: item.Class.ID, Name: item.Class.Name, Code: item.Class.Code}
	}
	return resp
}

func guardianToResponse(item *model.Guardian) dto.GuardianResponse {
	return dto.GuardianResponse{
		ID:                 item.ID,
		TenantID:           item.TenantID,
		Name:               item.Name,
		Relationship:       item.Relationship,
		Phone:              item.Phone,
		WhatsAppPhone:      item.WhatsAppPhone,
		Email:              item.Email,
		PreferredLanguage:  item.PreferredLanguage,
		CommunicationOptIn: item.CommunicationOptIn,
		Address:            addressToResponse(item.Address),
		Metadata:           item.Metadata,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func studentToResponse(item *model.Student) dto.StudentResponse {
	resp := dto.StudentResponse{
		ID:                  item.ID,
		TenantID:            item.TenantID,
		AcademicYearID:      item.AcademicYearID,
		ClassID:             item.ClassID,
		SectionID:           item.SectionID,
		BranchID:            item.BranchID,
		AdmissionNumber:     item.AdmissionNumber,
		FirstName:           item.FirstName,
		LastName:            item.LastName,
		RollNumber:          item.RollNumber,
		Status:              item.Status,
		Category:            item.Category,
		Phone:               item.Phone,
		Email:               item.Email,
		Address:             addressToResponse(item.Address),
		OpeningBalancePaise: item.OpeningBalancePaise,
		Currency:            item.Currency,
		Metadata:            item.Metadata,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
	if item.AcademicYear != nil {
		resp.AcademicYear = &dto.LookupResponse{ID: item.AcademicYear.ID, Name: item.AcademicYear.Name, Code: item.AcademicYear.Code}
	}
	if item.Class != nil {
		resp.Class = &dto.LookupResponse{ID: item.Class.ID, Name: item.Class.Name, Code: item.Class.Code}
	}
	if item.Section != nil {
		resp.Section = &dto.LookupResponse{ID: item.Section.ID, Name: item.Section.Name, Code: item.Section.Code}
	}
	for i := range item.Guardians {
		resp.Guardians = append(resp.Guardians, studentGuardianToResponse(&item.Guardians[i]))
	}
	return resp
}

func studentGuardianToResponse(item *model.StudentGuardian) dto.StudentGuardianResponse {
	resp := dto.StudentGuardianResponse{
		GuardianID:   item.GuardianID,
		Relationship: item.Relationship,
		IsPrimary:    item.IsPrimary,
	}
	if item.Guardian != nil {
		guardian := guardianToResponse(item.Guardian)
		resp.Guardian = &guardian
	}
	return resp
}

func importToResponse(item *model.Import) dto.ImportResponse {
	return dto.ImportResponse{
		ID:             item.ID,
		TenantID:       item.TenantID,
		ImportType:     item.ImportType,
		Status:         item.Status,
		SourceFilename: item.SourceFilename,
		TotalRows:      item.TotalRows,
		ValidRows:      item.ValidRows,
		InvalidRows:    item.InvalidRows,
		CommittedRows:  item.CommittedRows,
		CreatedBy:      item.CreatedBy,
		CommittedAt:    item.CommittedAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func importErrorToResponse(item *model.ImportError) dto.ImportErrorResponse {
	raw := map[string]string{}
	for k, v := range item.RawData {
		raw[k] = fmt.Sprint(v)
	}
	return dto.ImportErrorResponse{
		RowNumber: item.RowNumber,
		Field:     item.Field,
		Message:   item.Message,
		RawData:   raw,
	}
}

func normalizeHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func rowMap(header, record []string) map[string]string {
	out := map[string]string{}
	for i, h := range header {
		value := ""
		if i < len(record) {
			value = strings.TrimSpace(record[i])
		}
		out[normalizeHeader(h)] = value
	}
	return out
}

func isEmptyImportRow(row map[string]string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func stringMapToAny(row map[string]string) map[string]any {
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = v
	}
	return out
}

func cleanOptionalStringFromValue(value string) *string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return &clean
}

func addressRequestFromImportRow(row map[string]string) dto.AddressRequest {
	return dto.AddressRequest{
		Line1:      strings.TrimSpace(row["address_line1"]),
		Line2:      strings.TrimSpace(row["address_line2"]),
		City:       strings.TrimSpace(row["city"]),
		State:      strings.TrimSpace(row["state"]),
		PostalCode: strings.TrimSpace(row["postal_code"]),
		Country:    defaultString(row["country"], "India"),
	}
}

func parseFlexibleBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "yes", "y", "1":
		return true, nil
	case "false", "f", "no", "n", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool")
	}
}

func inSet(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func countRowsWithErrors(importErrors []model.ImportError) int {
	seen := map[int]bool{}
	for _, item := range importErrors {
		seen[item.RowNumber] = true
	}
	return len(seen)
}

func importRowsFromMetadata(metadata map[string]any) ([]studentImportRow, error) {
	raw, ok := metadata[importMetadataRowsKey]
	if !ok {
		return nil, apperror.New("IMPORT_PAYLOAD_MISSING", "import payload is missing", 409)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, apperror.New("IMPORT_PAYLOAD_INVALID", "import payload is invalid", 409)
	}
	var rows []studentImportRow
	if err := json.Unmarshal(encoded, &rows); err != nil {
		return nil, apperror.New("IMPORT_PAYLOAD_INVALID", "import payload is invalid", 409)
	}
	return rows, nil
}

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

type billingService struct {
	repo         repository.BillingRepository
	repoFactory  repository.BillingRepositoryFactory
	academicRepo repository.AcademicRepository
	tx           database.Transactor
	auditRepo    repository.AuditRepository
}

func NewBillingService(
	repo repository.BillingRepository,
	repoFactory repository.BillingRepositoryFactory,
	academicRepo repository.AcademicRepository,
	tx database.Transactor,
	auditRepo repository.AuditRepository,
) BillingService {
	return &billingService{
		repo:         repo,
		repoFactory:  repoFactory,
		academicRepo: academicRepo,
		tx:           tx,
		auditRepo:    auditRepo,
	}
}

func (s *billingService) CreateFeeHead(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeHeadRequest) (*dto.FeeHeadResponse, error) {
	code, err := normalizeCode(req.Code, "INVALID_FEE_HEAD_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetFeeHeadByCode(ctx, tenantID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup fee head code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	feeHead := &model.FeeHead{
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Code:        code,
		Description: strings.TrimSpace(req.Description),
		Category:    defaultString(req.Category, "tuition"),
		Status:      defaultString(req.Status, "active"),
		Taxable:     req.Taxable,
		TaxRateBps:  req.TaxRateBps,
		Metadata:    normalizeMetadata(req.Metadata),
	}
	normalizeFeeHeadTax(feeHead)

	if err := s.repo.CreateFeeHead(ctx, feeHead); err != nil {
		return nil, mapPersistenceError(err, "create fee head")
	}
	if err := s.audit(ctx, tenantID, actorID, "fee_head.created", "fee_head", feeHead.ID, "fee head created", map[string]any{"code": feeHead.Code}); err != nil {
		return nil, err
	}
	resp := feeHeadToResponse(feeHead)
	return &resp, nil
}

func (s *billingService) ListFeeHeads(ctx context.Context, tenantID uuid.UUID, filter model.FeeHeadFilter, params model.PaginationParams) (*model.PaginatedResult[dto.FeeHeadResponse], error) {
	result, err := s.repo.ListFeeHeads(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list fee heads: %w", err)
	}
	items := make([]dto.FeeHeadResponse, len(result.Data))
	for i := range result.Data {
		items[i] = feeHeadToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *billingService) GetFeeHead(ctx context.Context, tenantID, id uuid.UUID) (*dto.FeeHeadResponse, error) {
	feeHead, err := s.repo.GetFeeHead(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get fee head: %w", err)
	}
	if feeHead == nil {
		return nil, apperror.ErrNotFound
	}
	resp := feeHeadToResponse(feeHead)
	return &resp, nil
}

func (s *billingService) UpdateFeeHead(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateFeeHeadRequest) (*dto.FeeHeadResponse, error) {
	feeHead, err := s.repo.GetFeeHead(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get fee head: %w", err)
	}
	if feeHead == nil {
		return nil, apperror.ErrNotFound
	}
	if req.Name != nil {
		feeHead.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_FEE_HEAD_CODE")
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(code, feeHead.Code) {
			existing, err := s.repo.GetFeeHeadByCode(ctx, tenantID, code)
			if err != nil {
				return nil, fmt.Errorf("lookup fee head code: %w", err)
			}
			if existing != nil && existing.ID != feeHead.ID {
				return nil, apperror.ErrConflict
			}
		}
		feeHead.Code = code
	}
	if req.Description != nil {
		feeHead.Description = strings.TrimSpace(*req.Description)
	}
	if req.Category != nil {
		feeHead.Category = strings.TrimSpace(*req.Category)
	}
	if req.Status != nil {
		feeHead.Status = strings.TrimSpace(*req.Status)
	}
	if req.Taxable != nil {
		feeHead.Taxable = *req.Taxable
	}
	if req.TaxRateBps != nil {
		feeHead.TaxRateBps = *req.TaxRateBps
	}
	if req.Metadata != nil {
		feeHead.Metadata = normalizeMetadata(req.Metadata)
	}
	normalizeFeeHeadTax(feeHead)

	if err := s.repo.UpdateFeeHead(ctx, feeHead); err != nil {
		return nil, mapPersistenceError(err, "update fee head")
	}
	if err := s.audit(ctx, tenantID, actorID, "fee_head.updated", "fee_head", feeHead.ID, "fee head updated", map[string]any{"code": feeHead.Code}); err != nil {
		return nil, err
	}
	resp := feeHeadToResponse(feeHead)
	return &resp, nil
}

func (s *billingService) DeleteFeeHead(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	feeHead, err := s.repo.GetFeeHead(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get fee head: %w", err)
	}
	if feeHead == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteFeeHead(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete fee head: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "fee_head.deleted", "fee_head", id, "fee head deleted", map[string]any{"code": feeHead.Code})
}

func (s *billingService) CreateFeeStructure(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeStructureRequest) (*dto.FeeStructureResponse, error) {
	if _, err := s.requireAcademicYear(ctx, tenantID, req.AcademicYearID); err != nil {
		return nil, err
	}
	code, err := normalizeCode(req.Code, "INVALID_FEE_STRUCTURE_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetFeeStructureByCode(ctx, tenantID, req.AcademicYearID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup fee structure code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}

	feeStructure := &model.FeeStructure{
		TenantID:                  tenantID,
		AcademicYearID:            req.AcademicYearID,
		Name:                      strings.TrimSpace(req.Name),
		Code:                      code,
		Description:               strings.TrimSpace(req.Description),
		BillingCycle:              defaultString(req.BillingCycle, "one_time"),
		Status:                    defaultString(req.Status, "active"),
		Currency:                  defaultCurrency,
		AllowPartialPayment:       req.AllowPartialPayment,
		MinimumPartialAmountPaise: req.MinimumPartialAmountPaise,
		DueDay:                    req.DueDay,
		Metadata:                  normalizeMetadata(req.Metadata),
	}
	normalizePartialPaymentRule(feeStructure)

	items, err := s.feeStructureItemsFromRequests(ctx, tenantID, uuid.Nil, req.Items)
	if err != nil {
		return nil, err
	}
	if err := validateFeeStructureItems(items, feeStructure); err != nil {
		return nil, err
	}

	err = s.runInTx(ctx, func(repo repository.BillingRepository) error {
		if err := repo.CreateFeeStructure(ctx, feeStructure); err != nil {
			return mapPersistenceError(err, "create fee structure")
		}
		if err := repo.ReplaceFeeStructureItems(ctx, tenantID, feeStructure.ID, items); err != nil {
			return mapPersistenceError(err, "create fee structure items")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "fee_structure.created", "fee_structure", feeStructure.ID, "fee structure created", map[string]any{"code": feeStructure.Code}); err != nil {
		return nil, err
	}
	return s.GetFeeStructure(ctx, tenantID, feeStructure.ID)
}

func (s *billingService) ListFeeStructures(ctx context.Context, tenantID uuid.UUID, filter model.FeeStructureFilter, params model.PaginationParams) (*model.PaginatedResult[dto.FeeStructureResponse], error) {
	result, err := s.repo.ListFeeStructures(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list fee structures: %w", err)
	}
	items := make([]dto.FeeStructureResponse, len(result.Data))
	for i := range result.Data {
		items[i] = feeStructureToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *billingService) GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*dto.FeeStructureResponse, error) {
	feeStructure, err := s.repo.GetFeeStructure(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get fee structure: %w", err)
	}
	if feeStructure == nil {
		return nil, apperror.ErrNotFound
	}
	resp := feeStructureToResponse(feeStructure)
	return &resp, nil
}

func (s *billingService) UpdateFeeStructure(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateFeeStructureRequest) (*dto.FeeStructureResponse, error) {
	feeStructure, err := s.repo.GetFeeStructure(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get fee structure: %w", err)
	}
	if feeStructure == nil {
		return nil, apperror.ErrNotFound
	}

	if req.AcademicYearID != nil {
		if _, err := s.requireAcademicYear(ctx, tenantID, *req.AcademicYearID); err != nil {
			return nil, err
		}
		feeStructure.AcademicYearID = *req.AcademicYearID
	}
	if req.Name != nil {
		feeStructure.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_FEE_STRUCTURE_CODE")
		if err != nil {
			return nil, err
		}
		feeStructure.Code = code
	}
	existing, err := s.repo.GetFeeStructureByCode(ctx, tenantID, feeStructure.AcademicYearID, feeStructure.Code)
	if err != nil {
		return nil, fmt.Errorf("lookup fee structure code: %w", err)
	}
	if existing != nil && existing.ID != feeStructure.ID {
		return nil, apperror.ErrConflict
	}
	if req.Description != nil {
		feeStructure.Description = strings.TrimSpace(*req.Description)
	}
	if req.BillingCycle != nil {
		feeStructure.BillingCycle = strings.TrimSpace(*req.BillingCycle)
	}
	if req.Status != nil {
		feeStructure.Status = strings.TrimSpace(*req.Status)
	}
	if req.AllowPartialPayment != nil {
		feeStructure.AllowPartialPayment = *req.AllowPartialPayment
	}
	if req.MinimumPartialAmountPaise != nil {
		feeStructure.MinimumPartialAmountPaise = *req.MinimumPartialAmountPaise
	}
	if req.DueDay != nil {
		feeStructure.DueDay = req.DueDay
	}
	if req.Metadata != nil {
		feeStructure.Metadata = normalizeMetadata(req.Metadata)
	}
	normalizePartialPaymentRule(feeStructure)

	var items []model.FeeStructureItem
	if req.Items != nil {
		if len(*req.Items) == 0 {
			return nil, apperror.New("FEE_STRUCTURE_ITEMS_REQUIRED", "fee structure must include at least one item", 400)
		}
		items, err = s.feeStructureItemsFromRequests(ctx, tenantID, feeStructure.ID, *req.Items)
		if err != nil {
			return nil, err
		}
		if err := validateFeeStructureItems(items, feeStructure); err != nil {
			return nil, err
		}
	}

	err = s.runInTx(ctx, func(repo repository.BillingRepository) error {
		if err := repo.UpdateFeeStructure(ctx, feeStructure); err != nil {
			return mapPersistenceError(err, "update fee structure")
		}
		if req.Items != nil {
			if err := repo.ReplaceFeeStructureItems(ctx, tenantID, feeStructure.ID, items); err != nil {
				return mapPersistenceError(err, "replace fee structure items")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "fee_structure.updated", "fee_structure", feeStructure.ID, "fee structure updated", map[string]any{"code": feeStructure.Code}); err != nil {
		return nil, err
	}
	return s.GetFeeStructure(ctx, tenantID, feeStructure.ID)
}

func (s *billingService) DeleteFeeStructure(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	feeStructure, err := s.repo.GetFeeStructure(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get fee structure: %w", err)
	}
	if feeStructure == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteFeeStructure(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete fee structure: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "fee_structure.deleted", "fee_structure", id, "fee structure deleted", map[string]any{"code": feeStructure.Code})
}

func (s *billingService) CreateFeeAssignment(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateFeeAssignmentRequest) (*dto.FeeAssignmentResponse, error) {
	feeStructure, err := s.repo.GetFeeStructure(ctx, tenantID, req.FeeStructureID)
	if err != nil {
		return nil, fmt.Errorf("get fee structure: %w", err)
	}
	if feeStructure == nil {
		return nil, apperror.ErrNotFound
	}
	if feeStructure.Status != "active" {
		return nil, apperror.New("FEE_STRUCTURE_NOT_ACTIVE", "fee structure must be active before assignment", 409)
	}

	effectiveFrom := todayDate()
	if strings.TrimSpace(req.EffectiveFrom) != "" {
		effectiveFrom, err = parseDate(req.EffectiveFrom)
		if err != nil {
			return nil, err
		}
	}
	var effectiveUntil *time.Time
	if req.EffectiveUntil != nil {
		effectiveUntil, err = parseDatePointer(req.EffectiveUntil)
		if err != nil {
			return nil, err
		}
		if effectiveUntil != nil && effectiveUntil.Before(effectiveFrom) {
			return nil, apperror.New("INVALID_DATE_RANGE", "effective_until must be on or after effective_from", 400)
		}
	}

	assignment := &model.StudentFeeAssignment{
		TenantID:       tenantID,
		FeeStructureID: feeStructure.ID,
		AcademicYearID: feeStructure.AcademicYearID,
		AssignmentType: strings.TrimSpace(req.AssignmentType),
		Status:         defaultString(req.Status, "active"),
		EffectiveFrom:  effectiveFrom,
		EffectiveUntil: effectiveUntil,
		CreatedBy:      &actorID,
		Metadata:       normalizeMetadata(req.Metadata),
	}
	if req.AcademicYearID != nil && *req.AcademicYearID != feeStructure.AcademicYearID {
		return nil, apperror.New("ACADEMIC_YEAR_MISMATCH", "assignment academic_year_id must match the fee structure", 400)
	}
	if err := s.populateAssignmentTarget(ctx, assignment, req); err != nil {
		return nil, err
	}

	if err := s.repo.CreateFeeAssignment(ctx, assignment); err != nil {
		return nil, mapPersistenceError(err, "create fee assignment")
	}
	if err := s.audit(ctx, tenantID, actorID, "fee_assignment.created", "fee_assignment", assignment.ID, "fee assignment created", map[string]any{
		"fee_structure_id": feeStructure.ID.String(),
		"assignment_type":  assignment.AssignmentType,
	}); err != nil {
		return nil, err
	}
	return s.getFeeAssignmentResponse(ctx, tenantID, assignment.ID)
}

func (s *billingService) GenerateInvoices(ctx context.Context, actorID, tenantID uuid.UUID, req dto.GenerateInvoicesRequest) (*dto.GenerateInvoicesResponse, error) {
	generated := []model.Invoice{}
	skipped := 0
	var assignmentID uuid.UUID

	err := s.runInTx(ctx, func(repo repository.BillingRepository) error {
		assignment, err := repo.GetFeeAssignment(ctx, tenantID, req.AssignmentID)
		if err != nil {
			return fmt.Errorf("get fee assignment: %w", err)
		}
		if assignment == nil {
			return apperror.ErrNotFound
		}
		assignmentID = assignment.ID
		if assignment.Status != "active" {
			return apperror.New("FEE_ASSIGNMENT_NOT_ACTIVE", "fee assignment must be active before invoice generation", 409)
		}

		feeStructure, err := repo.GetFeeStructure(ctx, tenantID, assignment.FeeStructureID)
		if err != nil {
			return fmt.Errorf("get fee structure: %w", err)
		}
		if feeStructure == nil {
			return apperror.ErrNotFound
		}
		if len(feeStructure.Items) == 0 {
			return apperror.New("FEE_STRUCTURE_ITEMS_REQUIRED", "fee structure has no active fee items", 409)
		}

		issueDate, dueDate, periodStart, periodEnd, err := resolveInvoiceDates(req, feeStructure)
		if err != nil {
			return err
		}
		if err := validateRequestedStudentIDs(req.StudentIDs); err != nil {
			return err
		}
		students, err := repo.ListStudentsForAssignment(ctx, assignment, req.StudentIDs)
		if err != nil {
			return fmt.Errorf("list assignment students: %w", err)
		}
		if len(students) == 0 {
			return apperror.New("NO_STUDENTS_FOR_ASSIGNMENT", "no active students matched the fee assignment", 404)
		}
		if len(req.StudentIDs) > 0 && len(students) != len(req.StudentIDs) {
			return apperror.New("STUDENT_OUTSIDE_ASSIGNMENT", "one or more students do not match the fee assignment target", 400)
		}

		prefix := invoiceNumberPrefix(feeStructure)
		for i := range students {
			student := students[i]
			generationKey := invoiceGenerationKey(assignment.ID, student.ID, dueDate, periodStart, periodEnd)
			existing, err := repo.GetInvoiceByGenerationKey(ctx, tenantID, generationKey)
			if err != nil {
				return fmt.Errorf("lookup invoice generation key: %w", err)
			}
			if existing != nil {
				skipped++
				continue
			}

			concessions, err := repo.ListActiveConcessions(ctx, tenantID, student.ID, assignment.AcademicYearID, dueDate)
			if err != nil {
				return fmt.Errorf("list active concessions: %w", err)
			}
			sequence, err := repo.NextInvoiceSequence(ctx, tenantID, assignment.AcademicYearID, prefix)
			if err != nil {
				return fmt.Errorf("next invoice sequence: %w", err)
			}
			invoice := invoiceFromStructure(tenantID, actorID, student, assignment, feeStructure, sequence, generationKey, issueDate, dueDate, periodStart, periodEnd, concessions, req.Metadata)
			if err := repo.CreateInvoice(ctx, &invoice); err != nil {
				return mapPersistenceError(err, "create invoice")
			}
			for itemIndex := range invoice.Items {
				invoice.Items[itemIndex].InvoiceID = invoice.ID
			}
			if err := repo.CreateInvoiceItems(ctx, invoice.Items); err != nil {
				return mapPersistenceError(err, "create invoice items")
			}
			generated = append(generated, invoice)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(generated) > 0 {
		if err := s.audit(ctx, tenantID, actorID, "invoices.generated", "fee_assignment", assignmentID, "invoices generated", map[string]any{
			"generated_count": len(generated),
			"skipped_count":   skipped,
		}); err != nil {
			return nil, err
		}
	}

	resp := &dto.GenerateInvoicesResponse{
		AssignmentID:   assignmentID,
		GeneratedCount: len(generated),
		SkippedCount:   skipped,
		Invoices:       make([]dto.InvoiceResponse, len(generated)),
	}
	for i := range generated {
		resp.Invoices[i] = invoiceToResponse(&generated[i])
	}
	return resp, nil
}

func (s *billingService) ListInvoices(ctx context.Context, tenantID uuid.UUID, filter model.InvoiceFilter, params model.PaginationParams) (*model.PaginatedResult[dto.InvoiceResponse], error) {
	result, err := s.repo.ListInvoices(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	items := make([]dto.InvoiceResponse, len(result.Data))
	for i := range result.Data {
		items[i] = invoiceToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *billingService) GetInvoice(ctx context.Context, tenantID, id uuid.UUID) (*dto.InvoiceResponse, error) {
	invoice, err := s.repo.GetInvoice(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	if invoice == nil {
		return nil, apperror.ErrNotFound
	}
	resp := invoiceToResponse(invoice)
	return &resp, nil
}

func (s *billingService) GetStudentLedger(ctx context.Context, tenantID, studentID uuid.UUID) (*dto.StudentLedgerResponse, error) {
	student, err := s.academicRepo.GetStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return nil, apperror.ErrNotFound
	}
	invoices, err := s.repo.ListStudentInvoices(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student invoices: %w", err)
	}

	ledger := dto.StudentLedgerResponse{
		Student:             studentBriefToResponse(student),
		Currency:            defaultCurrency,
		OpeningBalancePaise: student.OpeningBalancePaise,
		Entries:             []dto.StudentLedgerEntryResponse{},
	}
	balance := student.OpeningBalancePaise
	if student.OpeningBalancePaise != 0 {
		ledger.Entries = append(ledger.Entries, dto.StudentLedgerEntryResponse{
			EntryType:         "opening_balance",
			EntryID:           student.ID,
			ReferenceNumber:   "OPENING_BALANCE",
			Description:       "Opening balance",
			EntryDate:         student.CreatedAt.Format(dateLayout),
			DebitAmountPaise:  positiveOnly(student.OpeningBalancePaise),
			CreditAmountPaise: positiveOnly(-student.OpeningBalancePaise),
			BalanceAfterPaise: balance,
			Status:            "posted",
		})
	}
	for i := range invoices {
		invoice := invoices[i]
		ledger.TotalBilledPaise += invoice.TotalAmountPaise
		ledger.TotalPaidPaise += invoice.PaidAmountPaise
		balance += invoice.BalanceAmountPaise
		dueDate := invoice.DueDate.Format(dateLayout)
		ledger.Entries = append(ledger.Entries, dto.StudentLedgerEntryResponse{
			EntryType:         "invoice",
			EntryID:           invoice.ID,
			ReferenceNumber:   invoice.InvoiceNumber,
			Description:       "Invoice " + invoice.InvoiceNumber,
			EntryDate:         invoice.IssueDate.Format(dateLayout),
			DueDate:           &dueDate,
			DebitAmountPaise:  invoice.TotalAmountPaise,
			CreditAmountPaise: invoice.PaidAmountPaise,
			BalanceAfterPaise: balance,
			Status:            invoice.Status,
		})
	}
	ledger.BalancePaise = balance
	return &ledger, nil
}

func (s *billingService) GetParentChildDues(ctx context.Context, tenantID, studentID uuid.UUID) (*dto.ParentDuesResponse, error) {
	student, err := s.academicRepo.GetStudent(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if student == nil {
		return nil, apperror.ErrNotFound
	}
	invoices, err := s.repo.ListStudentInvoices(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student invoices: %w", err)
	}

	today := todayDate()
	resp := &dto.ParentDuesResponse{
		Student:  studentBriefToResponse(student),
		Currency: defaultCurrency,
		Invoices: []dto.InvoiceResponse{},
	}
	for i := range invoices {
		invoice := invoices[i]
		if invoice.BalanceAmountPaise <= 0 {
			continue
		}
		resp.TotalDuePaise += invoice.BalanceAmountPaise
		if invoice.DueDate.Before(today) {
			resp.OverduePaise += invoice.BalanceAmountPaise
		}
		if invoice.AllowPartialPayment {
			resp.AllowPartial = true
			if invoice.MinimumPartialAmountPaise > 0 && (resp.MinimumPayablePaise == 0 || invoice.MinimumPartialAmountPaise < resp.MinimumPayablePaise) {
				resp.MinimumPayablePaise = invoice.MinimumPartialAmountPaise
			}
		}
		resp.Invoices = append(resp.Invoices, invoiceToResponse(&invoice))
	}
	if resp.MinimumPayablePaise > resp.TotalDuePaise {
		resp.MinimumPayablePaise = resp.TotalDuePaise
	}
	return resp, nil
}

func (s *billingService) getFeeAssignmentResponse(ctx context.Context, tenantID, id uuid.UUID) (*dto.FeeAssignmentResponse, error) {
	assignment, err := s.repo.GetFeeAssignment(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get fee assignment: %w", err)
	}
	if assignment == nil {
		return nil, apperror.ErrNotFound
	}
	resp := feeAssignmentToResponse(assignment)
	return &resp, nil
}

func (s *billingService) populateAssignmentTarget(ctx context.Context, assignment *model.StudentFeeAssignment, req dto.CreateFeeAssignmentRequest) error {
	switch assignment.AssignmentType {
	case "class":
		if req.ClassID == nil {
			return apperror.New("CLASS_REQUIRED", "class_id is required for class fee assignment", 400)
		}
		class, err := s.academicRepo.GetClass(ctx, assignment.TenantID, *req.ClassID)
		if err != nil {
			return fmt.Errorf("get class: %w", err)
		}
		if class == nil {
			return apperror.New("CLASS_NOT_FOUND", "class not found for tenant", 404)
		}
		assignment.ClassID = req.ClassID
	case "section":
		if req.SectionID == nil {
			return apperror.New("SECTION_REQUIRED", "section_id is required for section fee assignment", 400)
		}
		section, err := s.academicRepo.GetSection(ctx, assignment.TenantID, *req.SectionID)
		if err != nil {
			return fmt.Errorf("get section: %w", err)
		}
		if section == nil {
			return apperror.New("SECTION_NOT_FOUND", "section not found for tenant", 404)
		}
		if section.AcademicYearID != assignment.AcademicYearID {
			return apperror.New("SECTION_STRUCTURE_MISMATCH", "section does not belong to the fee structure academic year", 400)
		}
		assignment.ClassID = &section.ClassID
		assignment.SectionID = &section.ID
	case "student":
		if req.StudentID == nil {
			return apperror.New("STUDENT_REQUIRED", "student_id is required for student fee assignment", 400)
		}
		student, err := s.academicRepo.GetStudent(ctx, assignment.TenantID, *req.StudentID)
		if err != nil {
			return fmt.Errorf("get student: %w", err)
		}
		if student == nil {
			return apperror.New("STUDENT_NOT_FOUND", "student not found for tenant", 404)
		}
		if student.AcademicYearID != assignment.AcademicYearID {
			return apperror.New("STUDENT_STRUCTURE_MISMATCH", "student does not belong to the fee structure academic year", 400)
		}
		assignment.ClassID = &student.ClassID
		assignment.SectionID = &student.SectionID
		assignment.StudentID = &student.ID
	default:
		return apperror.ErrValidationFailed
	}
	return nil
}

func (s *billingService) feeStructureItemsFromRequests(ctx context.Context, tenantID, feeStructureID uuid.UUID, reqs []dto.CreateFeeStructureItemRequest) ([]model.FeeStructureItem, error) {
	if len(reqs) == 0 {
		return nil, apperror.New("FEE_STRUCTURE_ITEMS_REQUIRED", "fee structure must include at least one item", 400)
	}
	items := make([]model.FeeStructureItem, 0, len(reqs))
	for i, req := range reqs {
		feeHead, err := s.repo.GetFeeHead(ctx, tenantID, req.FeeHeadID)
		if err != nil {
			return nil, fmt.Errorf("get fee head: %w", err)
		}
		if feeHead == nil {
			return nil, apperror.New("FEE_HEAD_NOT_FOUND", "fee head not found for tenant", 404)
		}
		if feeHead.Status != "active" {
			return nil, apperror.New("FEE_HEAD_NOT_ACTIVE", "fee head must be active before it can be used in a structure", 409)
		}
		taxRate := 0
		if feeHead.Taxable {
			taxRate = feeHead.TaxRateBps
		}
		if req.TaxRateBps != nil {
			taxRate = *req.TaxRateBps
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = feeHead.Name
		}
		sortOrder := req.SortOrder
		if sortOrder == 0 {
			sortOrder = i + 1
		}
		items = append(items, model.FeeStructureItem{
			TenantID:       tenantID,
			FeeStructureID: feeStructureID,
			FeeHeadID:      feeHead.ID,
			Name:           name,
			Description:    strings.TrimSpace(req.Description),
			AmountPaise:    req.AmountPaise,
			TaxRateBps:     taxRate,
			SortOrder:      sortOrder,
			Optional:       req.Optional,
			Metadata:       normalizeMetadata(req.Metadata),
			FeeHead:        feeHead,
		})
	}
	return items, nil
}

func (s *billingService) requireAcademicYear(ctx context.Context, tenantID, academicYearID uuid.UUID) (*model.AcademicYear, error) {
	academicYear, err := s.academicRepo.GetAcademicYear(ctx, tenantID, academicYearID)
	if err != nil {
		return nil, fmt.Errorf("get academic year: %w", err)
	}
	if academicYear == nil {
		return nil, apperror.New("ACADEMIC_YEAR_NOT_FOUND", "academic year not found for tenant", 404)
	}
	return academicYear, nil
}

func (s *billingService) runInTx(ctx context.Context, fn func(repository.BillingRepository) error) error {
	if s.tx == nil || s.repoFactory == nil {
		return fn(s.repo)
	}
	return s.tx.WithinTx(ctx, func(db database.DBTX) error {
		return fn(s.repoFactory(db))
	})
}

func (s *billingService) audit(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, action string, entityType string, entityID uuid.UUID, summary string, metadata map[string]any) error {
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

func normalizeFeeHeadTax(feeHead *model.FeeHead) {
	if !feeHead.Taxable {
		feeHead.TaxRateBps = 0
	}
}

func normalizePartialPaymentRule(feeStructure *model.FeeStructure) {
	if !feeStructure.AllowPartialPayment {
		feeStructure.MinimumPartialAmountPaise = 0
	}
}

func validateFeeStructureItems(items []model.FeeStructureItem, feeStructure *model.FeeStructure) error {
	total := int64(0)
	for _, item := range items {
		if item.AmountPaise < 0 {
			return apperror.New("INVALID_FEE_AMOUNT", "fee item amount cannot be negative", 400)
		}
		total += item.AmountPaise
	}
	if total <= 0 {
		return apperror.New("INVALID_FEE_STRUCTURE_TOTAL", "fee structure total must be greater than zero", 400)
	}
	if feeStructure.AllowPartialPayment && feeStructure.MinimumPartialAmountPaise > total {
		return apperror.New("INVALID_PARTIAL_PAYMENT_MINIMUM", "minimum partial amount cannot exceed fee structure total", 400)
	}
	return nil
}

func validateRequestedStudentIDs(ids []uuid.UUID) error {
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if id == uuid.Nil {
			return apperror.New("INVALID_STUDENT_ID", "student_ids cannot contain an empty UUID", 400)
		}
		if seen[id] {
			return apperror.New("DUPLICATE_STUDENT_ID", "student_ids cannot contain duplicates", 400)
		}
		seen[id] = true
	}
	return nil
}

func resolveInvoiceDates(req dto.GenerateInvoicesRequest, feeStructure *model.FeeStructure) (time.Time, time.Time, *time.Time, *time.Time, error) {
	issueDate := todayDate()
	var err error
	if strings.TrimSpace(req.IssueDate) != "" {
		issueDate, err = parseDate(req.IssueDate)
		if err != nil {
			return time.Time{}, time.Time{}, nil, nil, err
		}
	}

	periodStart, err := parseDatePointer(req.BillingPeriodStart)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, err
	}
	periodEnd, err := parseDatePointer(req.BillingPeriodEnd)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, err
	}
	if (periodStart == nil) != (periodEnd == nil) {
		return time.Time{}, time.Time{}, nil, nil, apperror.New("INVALID_BILLING_PERIOD", "billing_period_start and billing_period_end must be provided together", 400)
	}
	if periodStart != nil && periodEnd.Before(*periodStart) {
		return time.Time{}, time.Time{}, nil, nil, apperror.New("INVALID_BILLING_PERIOD", "billing_period_end must be on or after billing_period_start", 400)
	}
	if inSet(feeStructure.BillingCycle, "monthly", "quarterly", "term") && periodStart == nil {
		return time.Time{}, time.Time{}, nil, nil, apperror.New("BILLING_PERIOD_REQUIRED", "billing period is required for recurring fee structures", 400)
	}

	var dueDate time.Time
	if strings.TrimSpace(req.DueDate) != "" {
		dueDate, err = parseDate(req.DueDate)
		if err != nil {
			return time.Time{}, time.Time{}, nil, nil, err
		}
	} else if feeStructure.DueDay != nil {
		base := issueDate
		if periodStart != nil {
			base = *periodStart
		}
		dueDate = dateWithDueDay(base, *feeStructure.DueDay)
	} else if periodEnd != nil {
		dueDate = *periodEnd
	} else {
		dueDate = issueDate
	}
	if dueDate.Before(issueDate) {
		return time.Time{}, time.Time{}, nil, nil, apperror.New("INVALID_DUE_DATE", "due_date must be on or after issue_date", 400)
	}
	return issueDate, dueDate, periodStart, periodEnd, nil
}

func parseDatePointer(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := parseDate(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func todayDate() time.Time {
	return dateOnly(time.Now().UTC())
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func dateWithDueDay(base time.Time, dueDay int) time.Time {
	year, month, _ := base.Date()
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if dueDay > lastDay {
		dueDay = lastDay
	}
	return time.Date(year, month, dueDay, 0, 0, 0, 0, time.UTC)
}

func invoiceFromStructure(
	tenantID uuid.UUID,
	actorID uuid.UUID,
	student model.Student,
	assignment *model.StudentFeeAssignment,
	feeStructure *model.FeeStructure,
	sequence int64,
	generationKey string,
	issueDate time.Time,
	dueDate time.Time,
	periodStart *time.Time,
	periodEnd *time.Time,
	concessions []model.Concession,
	requestMetadata map[string]any,
) model.Invoice {
	issuedAt := time.Now().UTC()
	feeStructureID := feeStructure.ID
	assignmentID := assignment.ID
	invoice := model.Invoice{
		TenantID:                  tenantID,
		InvoiceNumber:             fmt.Sprintf("%s-%06d", invoiceNumberPrefix(feeStructure), sequence),
		StudentID:                 student.ID,
		AcademicYearID:            assignment.AcademicYearID,
		ClassID:                   student.ClassID,
		SectionID:                 student.SectionID,
		FeeStructureID:            &feeStructureID,
		AssignmentID:              &assignmentID,
		IssueDate:                 issueDate,
		DueDate:                   dueDate,
		BillingPeriodStart:        periodStart,
		BillingPeriodEnd:          periodEnd,
		GenerationKey:             generationKey,
		Currency:                  feeStructure.Currency,
		AllowPartialPayment:       feeStructure.AllowPartialPayment,
		MinimumPartialAmountPaise: feeStructure.MinimumPartialAmountPaise,
		GeneratedBy:               &actorID,
		Metadata:                  invoiceMetadata(feeStructure, requestMetadata),
		IssuedAt:                  &issuedAt,
		Student:                   &student,
		AcademicYear:              feeStructure.AcademicYear,
		FeeStructure:              feeStructure,
	}

	discounts := calculateItemDiscounts(feeStructure.Items, concessions)
	for i := range feeStructure.Items {
		source := feeStructure.Items[i]
		discount := discounts[source.ID]
		taxableAmount := source.AmountPaise - discount
		taxAmount := taxableAmount * int64(source.TaxRateBps) / 10000
		total := taxableAmount + taxAmount
		sourceID := source.ID
		invoice.Items = append(invoice.Items, model.InvoiceItem{
			TenantID:            tenantID,
			FeeHeadID:           source.FeeHeadID,
			FeeStructureItemID:  &sourceID,
			Description:         defaultString(source.Description, source.Name),
			AmountPaise:         source.AmountPaise,
			DiscountAmountPaise: discount,
			FineAmountPaise:     0,
			TaxAmountPaise:      taxAmount,
			TotalAmountPaise:    total,
			SortOrder:           source.SortOrder,
			Metadata:            map[string]any{"fee_structure_item_id": source.ID.String()},
			FeeHead:             source.FeeHead,
		})
		invoice.SubtotalAmountPaise += source.AmountPaise
		invoice.DiscountAmountPaise += discount
		invoice.TaxAmountPaise += taxAmount
	}
	invoice.TotalAmountPaise = invoice.SubtotalAmountPaise - invoice.DiscountAmountPaise + invoice.FineAmountPaise + invoice.TaxAmountPaise
	invoice.PaidAmountPaise = 0
	invoice.BalanceAmountPaise = invoice.TotalAmountPaise
	invoice.Status = recalculateInvoiceStatus(invoice, todayDate())
	return invoice
}

func calculateItemDiscounts(items []model.FeeStructureItem, concessions []model.Concession) map[uuid.UUID]int64 {
	discounts := map[uuid.UUID]int64{}
	globalFixed := int64(0)
	for _, concession := range concessions {
		if concession.FeeHeadID == nil && concession.ConcessionType == "fixed_amount" {
			globalFixed += concession.AmountPaise
			continue
		}
		for _, item := range items {
			if concession.FeeHeadID != nil && *concession.FeeHeadID != item.FeeHeadID {
				continue
			}
			discount := int64(0)
			if concession.ConcessionType == "percentage" {
				discount = item.AmountPaise * int64(concession.PercentageBps) / 10000
			} else {
				discount = concession.AmountPaise
			}
			discounts[item.ID] += discount
			if discounts[item.ID] > item.AmountPaise {
				discounts[item.ID] = item.AmountPaise
			}
		}
	}
	for _, item := range items {
		if globalFixed <= 0 {
			break
		}
		remainingItemAmount := item.AmountPaise - discounts[item.ID]
		if remainingItemAmount <= 0 {
			continue
		}
		applied := remainingItemAmount
		if globalFixed < applied {
			applied = globalFixed
		}
		discounts[item.ID] += applied
		globalFixed -= applied
	}
	return discounts
}

func recalculateInvoiceStatus(invoice model.Invoice, asOf time.Time) string {
	if invoice.Status == "cancelled" || invoice.Status == "void" {
		return invoice.Status
	}
	if invoice.BalanceAmountPaise <= 0 {
		return "paid"
	}
	if invoice.DueDate.Before(asOf) {
		return "overdue"
	}
	if invoice.PaidAmountPaise > 0 {
		return "partially_paid"
	}
	return "issued"
}

func invoiceGenerationKey(assignmentID, studentID uuid.UUID, dueDate time.Time, periodStart, periodEnd *time.Time) string {
	periodKey := dueDate.Format(dateLayout)
	if periodStart != nil && periodEnd != nil {
		periodKey = periodStart.Format(dateLayout) + ":" + periodEnd.Format(dateLayout)
	}
	return fmt.Sprintf("assignment:%s:student:%s:period:%s:due:%s", assignmentID, studentID, periodKey, dueDate.Format(dateLayout))
}

func invoiceNumberPrefix(feeStructure *model.FeeStructure) string {
	if feeStructure.AcademicYear != nil && strings.TrimSpace(feeStructure.AcademicYear.Code) != "" {
		return "INV-" + feeStructure.AcademicYear.Code
	}
	return "INV"
}

func invoiceMetadata(feeStructure *model.FeeStructure, requestMetadata map[string]any) map[string]any {
	metadata := normalizeMetadata(requestMetadata)
	metadata["fee_structure_code"] = feeStructure.Code
	metadata["billing_cycle"] = feeStructure.BillingCycle
	return metadata
}

func positiveOnly(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func feeHeadToResponse(item *model.FeeHead) dto.FeeHeadResponse {
	return dto.FeeHeadResponse{
		ID:          item.ID,
		TenantID:    item.TenantID,
		Name:        item.Name,
		Code:        item.Code,
		Description: item.Description,
		Category:    item.Category,
		Status:      item.Status,
		Taxable:     item.Taxable,
		TaxRateBps:  item.TaxRateBps,
		Metadata:    item.Metadata,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func feeStructureToResponse(item *model.FeeStructure) dto.FeeStructureResponse {
	resp := dto.FeeStructureResponse{
		ID:                        item.ID,
		TenantID:                  item.TenantID,
		AcademicYearID:            item.AcademicYearID,
		Name:                      item.Name,
		Code:                      item.Code,
		Description:               item.Description,
		BillingCycle:              item.BillingCycle,
		Status:                    item.Status,
		Currency:                  item.Currency,
		AllowPartialPayment:       item.AllowPartialPayment,
		MinimumPartialAmountPaise: item.MinimumPartialAmountPaise,
		DueDay:                    item.DueDay,
		Metadata:                  item.Metadata,
		CreatedAt:                 item.CreatedAt,
		UpdatedAt:                 item.UpdatedAt,
	}
	if item.AcademicYear != nil {
		resp.AcademicYear = &dto.LookupResponse{ID: item.AcademicYear.ID, Name: item.AcademicYear.Name, Code: item.AcademicYear.Code}
	}
	for i := range item.Items {
		resp.Items = append(resp.Items, feeStructureItemToResponse(&item.Items[i]))
	}
	return resp
}

func feeStructureItemToResponse(item *model.FeeStructureItem) dto.FeeStructureItemResponse {
	resp := dto.FeeStructureItemResponse{
		ID:             item.ID,
		FeeStructureID: item.FeeStructureID,
		FeeHeadID:      item.FeeHeadID,
		Name:           item.Name,
		Description:    item.Description,
		AmountPaise:    item.AmountPaise,
		TaxRateBps:     item.TaxRateBps,
		SortOrder:      item.SortOrder,
		Optional:       item.Optional,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.FeeHead != nil {
		feeHead := feeHeadToResponse(item.FeeHead)
		resp.FeeHead = &feeHead
	}
	return resp
}

func feeAssignmentToResponse(item *model.StudentFeeAssignment) dto.FeeAssignmentResponse {
	resp := dto.FeeAssignmentResponse{
		ID:             item.ID,
		TenantID:       item.TenantID,
		FeeStructureID: item.FeeStructureID,
		AcademicYearID: item.AcademicYearID,
		AssignmentType: item.AssignmentType,
		ClassID:        item.ClassID,
		SectionID:      item.SectionID,
		StudentID:      item.StudentID,
		Status:         item.Status,
		EffectiveFrom:  item.EffectiveFrom.Format(dateLayout),
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.EffectiveUntil != nil {
		formatted := item.EffectiveUntil.Format(dateLayout)
		resp.EffectiveUntil = &formatted
	}
	if item.FeeStructure != nil {
		feeStructure := feeStructureToResponse(item.FeeStructure)
		resp.FeeStructure = &feeStructure
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
	if item.Student != nil {
		student := studentBriefToResponse(item.Student)
		resp.Student = &student
	}
	return resp
}

func invoiceToResponse(item *model.Invoice) dto.InvoiceResponse {
	resp := dto.InvoiceResponse{
		ID:                        item.ID,
		TenantID:                  item.TenantID,
		InvoiceNumber:             item.InvoiceNumber,
		StudentID:                 item.StudentID,
		AcademicYearID:            item.AcademicYearID,
		ClassID:                   item.ClassID,
		SectionID:                 item.SectionID,
		FeeStructureID:            item.FeeStructureID,
		AssignmentID:              item.AssignmentID,
		IssueDate:                 item.IssueDate.Format(dateLayout),
		DueDate:                   item.DueDate.Format(dateLayout),
		Status:                    item.Status,
		Currency:                  item.Currency,
		AllowPartialPayment:       item.AllowPartialPayment,
		MinimumPartialAmountPaise: item.MinimumPartialAmountPaise,
		SubtotalAmountPaise:       item.SubtotalAmountPaise,
		DiscountAmountPaise:       item.DiscountAmountPaise,
		FineAmountPaise:           item.FineAmountPaise,
		TaxAmountPaise:            item.TaxAmountPaise,
		TotalAmountPaise:          item.TotalAmountPaise,
		PaidAmountPaise:           item.PaidAmountPaise,
		BalanceAmountPaise:        item.BalanceAmountPaise,
		Metadata:                  item.Metadata,
		CreatedAt:                 item.CreatedAt,
		UpdatedAt:                 item.UpdatedAt,
	}
	if item.BillingPeriodStart != nil {
		formatted := item.BillingPeriodStart.Format(dateLayout)
		resp.BillingPeriodStart = &formatted
	}
	if item.BillingPeriodEnd != nil {
		formatted := item.BillingPeriodEnd.Format(dateLayout)
		resp.BillingPeriodEnd = &formatted
	}
	if item.Student != nil {
		student := studentBriefToResponse(item.Student)
		resp.Student = &student
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
	if item.FeeStructure != nil {
		resp.FeeStructure = &dto.LookupResponse{ID: item.FeeStructure.ID, Name: item.FeeStructure.Name, Code: item.FeeStructure.Code}
	}
	for i := range item.Items {
		resp.Items = append(resp.Items, invoiceItemToResponse(&item.Items[i]))
	}
	return resp
}

func invoiceItemToResponse(item *model.InvoiceItem) dto.InvoiceItemResponse {
	resp := dto.InvoiceItemResponse{
		ID:                  item.ID,
		InvoiceID:           item.InvoiceID,
		FeeHeadID:           item.FeeHeadID,
		FeeStructureItemID:  item.FeeStructureItemID,
		Description:         item.Description,
		AmountPaise:         item.AmountPaise,
		DiscountAmountPaise: item.DiscountAmountPaise,
		FineAmountPaise:     item.FineAmountPaise,
		TaxAmountPaise:      item.TaxAmountPaise,
		TotalAmountPaise:    item.TotalAmountPaise,
		SortOrder:           item.SortOrder,
		Metadata:            item.Metadata,
		CreatedAt:           item.CreatedAt,
	}
	if item.FeeHead != nil {
		feeHead := feeHeadToResponse(item.FeeHead)
		resp.FeeHead = &feeHead
	}
	return resp
}

func studentBriefToResponse(item *model.Student) dto.StudentBriefResponse {
	return dto.StudentBriefResponse{
		ID:              item.ID,
		AdmissionNumber: item.AdmissionNumber,
		FirstName:       item.FirstName,
		LastName:        item.LastName,
		Status:          item.Status,
	}
}

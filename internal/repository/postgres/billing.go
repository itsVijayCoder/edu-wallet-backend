package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var _ repository.BillingRepository = (*billingRepository)(nil)

type billingRepository struct {
	db database.DBTX
}

func NewBillingRepository(db database.DBTX) repository.BillingRepository {
	return &billingRepository{db: db}
}

var (
	allowedFeeHeadSortColumns       = map[string]bool{"created_at": true, "name": true, "code": true, "category": true, "status": true}
	allowedFeeStructureSortColumns  = map[string]bool{"created_at": true, "name": true, "code": true, "billing_cycle": true, "status": true}
	allowedFeeAssignmentSortColumns = map[string]bool{"created_at": true, "assignment_type": true, "status": true, "effective_from": true}
	allowedInvoiceSortColumns       = map[string]bool{"created_at": true, "invoice_number": true, "due_date": true, "issue_date": true, "status": true, "total_amount_paise": true, "balance_amount_paise": true}
)

func (r *billingRepository) CreateFeeHead(ctx context.Context, feeHead *model.FeeHead) error {
	const query = `INSERT INTO fee_heads (
			tenant_id, name, code, description, category, status, taxable, tax_rate_bps, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		feeHead.TenantID,
		feeHead.Name,
		feeHead.Code,
		feeHead.Description,
		feeHead.Category,
		feeHead.Status,
		feeHead.Taxable,
		feeHead.TaxRateBps,
		mustJSON(feeHead.Metadata),
	).Scan(&feeHead.ID, &feeHead.CreatedAt, &feeHead.UpdatedAt)
}

func (r *billingRepository) GetFeeHead(ctx context.Context, tenantID, id uuid.UUID) (*model.FeeHead, error) {
	const query = feeHeadSelect + ` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`
	return r.scanFeeHead(ctx, query, tenantID, id)
}

func (r *billingRepository) GetFeeHeadByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.FeeHead, error) {
	const query = feeHeadSelect + ` WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL`
	return r.scanFeeHead(ctx, query, tenantID, code)
}

func (r *billingRepository) ListFeeHeads(ctx context.Context, tenantID uuid.UUID, filter model.FeeHeadFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeHead], error) {
	params.Normalize()
	where, args := feeHeadWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fee_heads `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedFeeHeadSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, feeHeadSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FeeHead, 0, params.PageSize)
	for rows.Next() {
		item, err := scanFeeHeadScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *billingRepository) UpdateFeeHead(ctx context.Context, feeHead *model.FeeHead) error {
	const query = `UPDATE fee_heads SET
			name = $1,
			code = $2,
			description = $3,
			category = $4,
			status = $5,
			taxable = $6,
			tax_rate_bps = $7,
			metadata = $8,
			updated_at = NOW()
		WHERE tenant_id = $9 AND id = $10 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		feeHead.Name,
		feeHead.Code,
		feeHead.Description,
		feeHead.Category,
		feeHead.Status,
		feeHead.Taxable,
		feeHead.TaxRateBps,
		mustJSON(feeHead.Metadata),
		feeHead.TenantID,
		feeHead.ID,
	).Scan(&feeHead.UpdatedAt)
}

func (r *billingRepository) SoftDeleteFeeHead(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE fee_heads SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *billingRepository) CreateFeeStructure(ctx context.Context, feeStructure *model.FeeStructure) error {
	const query = `INSERT INTO fee_structures (
			tenant_id, academic_year_id, name, code, description, billing_cycle, status,
			currency, allow_partial_payment, minimum_partial_amount_paise, due_day, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		feeStructure.TenantID,
		feeStructure.AcademicYearID,
		feeStructure.Name,
		feeStructure.Code,
		feeStructure.Description,
		feeStructure.BillingCycle,
		feeStructure.Status,
		feeStructure.Currency,
		feeStructure.AllowPartialPayment,
		feeStructure.MinimumPartialAmountPaise,
		feeStructure.DueDay,
		mustJSON(feeStructure.Metadata),
	).Scan(&feeStructure.ID, &feeStructure.CreatedAt, &feeStructure.UpdatedAt)
}

func (r *billingRepository) GetFeeStructure(ctx context.Context, tenantID, id uuid.UUID) (*model.FeeStructure, error) {
	const query = feeStructureSelect + ` WHERE fs.tenant_id = $1 AND fs.id = $2 AND fs.deleted_at IS NULL`
	item, err := r.scanFeeStructure(ctx, query, tenantID, id)
	if err != nil || item == nil {
		return item, err
	}
	items, err := r.ListFeeStructureItems(ctx, tenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Items = items
	return item, nil
}

func (r *billingRepository) GetFeeStructureByCode(ctx context.Context, tenantID, academicYearID uuid.UUID, code string) (*model.FeeStructure, error) {
	const query = feeStructureSelect + ` WHERE fs.tenant_id = $1 AND fs.academic_year_id = $2 AND lower(fs.code) = lower($3) AND fs.deleted_at IS NULL`
	return r.scanFeeStructure(ctx, query, tenantID, academicYearID, code)
}

func (r *billingRepository) ListFeeStructures(ctx context.Context, tenantID uuid.UUID, filter model.FeeStructureFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeStructure], error) {
	params.Normalize()
	where, args := feeStructureWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fee_structures fs `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedFeeStructureSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY fs.%s %s LIMIT %s OFFSET %s`, feeStructureSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FeeStructure, 0, params.PageSize)
	for rows.Next() {
		item, err := scanFeeStructureScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *billingRepository) UpdateFeeStructure(ctx context.Context, feeStructure *model.FeeStructure) error {
	const query = `UPDATE fee_structures SET
			academic_year_id = $1,
			name = $2,
			code = $3,
			description = $4,
			billing_cycle = $5,
			status = $6,
			currency = $7,
			allow_partial_payment = $8,
			minimum_partial_amount_paise = $9,
			due_day = $10,
			metadata = $11,
			updated_at = NOW()
		WHERE tenant_id = $12 AND id = $13 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		feeStructure.AcademicYearID,
		feeStructure.Name,
		feeStructure.Code,
		feeStructure.Description,
		feeStructure.BillingCycle,
		feeStructure.Status,
		feeStructure.Currency,
		feeStructure.AllowPartialPayment,
		feeStructure.MinimumPartialAmountPaise,
		feeStructure.DueDay,
		mustJSON(feeStructure.Metadata),
		feeStructure.TenantID,
		feeStructure.ID,
	).Scan(&feeStructure.UpdatedAt)
}

func (r *billingRepository) SoftDeleteFeeStructure(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE fee_structures SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *billingRepository) ReplaceFeeStructureItems(ctx context.Context, tenantID, feeStructureID uuid.UUID, items []model.FeeStructureItem) error {
	if _, err := r.db.Exec(ctx, `UPDATE fee_structure_items SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND fee_structure_id = $2 AND deleted_at IS NULL`, tenantID, feeStructureID); err != nil {
		return err
	}
	for i := range items {
		items[i].TenantID = tenantID
		items[i].FeeStructureID = feeStructureID
		if err := r.createFeeStructureItem(ctx, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *billingRepository) ListFeeStructureItems(ctx context.Context, tenantID, feeStructureID uuid.UUID) ([]model.FeeStructureItem, error) {
	const query = feeStructureItemSelect + ` WHERE fsi.tenant_id = $1 AND fsi.fee_structure_id = $2 AND fsi.deleted_at IS NULL ORDER BY fsi.sort_order ASC, fsi.created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, feeStructureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.FeeStructureItem{}
	for rows.Next() {
		item, err := scanFeeStructureItemScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *billingRepository) CreateFeeAssignment(ctx context.Context, assignment *model.StudentFeeAssignment) error {
	const query = `INSERT INTO student_fee_assignments (
			tenant_id, fee_structure_id, academic_year_id, assignment_type, class_id, section_id,
			student_id, status, effective_from, effective_until, created_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		assignment.TenantID,
		assignment.FeeStructureID,
		assignment.AcademicYearID,
		assignment.AssignmentType,
		assignment.ClassID,
		assignment.SectionID,
		assignment.StudentID,
		assignment.Status,
		assignment.EffectiveFrom,
		assignment.EffectiveUntil,
		assignment.CreatedBy,
		mustJSON(assignment.Metadata),
	).Scan(&assignment.ID, &assignment.CreatedAt, &assignment.UpdatedAt)
}

func (r *billingRepository) GetFeeAssignment(ctx context.Context, tenantID, id uuid.UUID) (*model.StudentFeeAssignment, error) {
	const query = feeAssignmentSelect + ` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`
	return r.scanFeeAssignment(ctx, query, tenantID, id)
}

func (r *billingRepository) ListFeeAssignments(ctx context.Context, tenantID uuid.UUID, filter model.FeeAssignmentFilter, params model.PaginationParams) (*model.PaginatedResult[model.StudentFeeAssignment], error) {
	params.Normalize()
	where, args := feeAssignmentWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM student_fee_assignments `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedFeeAssignmentSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, feeAssignmentSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.StudentFeeAssignment, 0, params.PageSize)
	for rows.Next() {
		item, err := scanFeeAssignmentScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *billingRepository) ListStudentsForAssignment(ctx context.Context, assignment *model.StudentFeeAssignment, onlyStudentIDs []uuid.UUID) ([]model.Student, error) {
	clauses := []string{
		"s.tenant_id = $1",
		"s.academic_year_id = $2",
		"s.status = 'active'",
		"s.deleted_at IS NULL",
	}
	args := []any{assignment.TenantID, assignment.AcademicYearID}

	switch assignment.AssignmentType {
	case "class":
		args = append(args, *assignment.ClassID)
		clauses = append(clauses, fmt.Sprintf("s.class_id = $%d", len(args)))
	case "section":
		args = append(args, *assignment.ClassID)
		clauses = append(clauses, fmt.Sprintf("s.class_id = $%d", len(args)))
		args = append(args, *assignment.SectionID)
		clauses = append(clauses, fmt.Sprintf("s.section_id = $%d", len(args)))
	case "student":
		args = append(args, *assignment.StudentID)
		clauses = append(clauses, fmt.Sprintf("s.id = $%d", len(args)))
	}

	if len(onlyStudentIDs) > 0 {
		placeholders := make([]string, 0, len(onlyStudentIDs))
		for _, id := range onlyStudentIDs {
			args = append(args, id)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		clauses = append(clauses, "s.id IN ("+strings.Join(placeholders, ", ")+")")
	}

	query := studentSelect + ` WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY s.admission_number ASC`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []model.Student{}
	for rows.Next() {
		student, err := scanStudentScanner(rows)
		if err != nil {
			return nil, err
		}
		students = append(students, *student)
	}
	return students, rows.Err()
}

func (r *billingRepository) ListActiveConcessions(ctx context.Context, tenantID, studentID, academicYearID uuid.UUID, asOf time.Time) ([]model.Concession, error) {
	const query = concessionSelect + `
		WHERE tenant_id = $1
		  AND student_id = $2
		  AND academic_year_id = $3
		  AND status = 'active'
		  AND deleted_at IS NULL
		  AND starts_on <= $4
		  AND (ends_on IS NULL OR ends_on >= $4)
		ORDER BY fee_head_id NULLS LAST, created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, studentID, academicYearID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Concession{}
	for rows.Next() {
		item, err := scanConcessionScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *billingRepository) NextInvoiceSequence(ctx context.Context, tenantID, academicYearID uuid.UUID, prefix string) (int64, error) {
	const query = `INSERT INTO invoice_counters (tenant_id, academic_year_id, prefix, next_number)
		VALUES ($1, $2, $3, 2)
		ON CONFLICT (tenant_id, academic_year_id)
		DO UPDATE SET next_number = invoice_counters.next_number + 1,
			prefix = EXCLUDED.prefix,
			updated_at = NOW()
		RETURNING next_number - 1`

	var sequence int64
	if err := r.db.QueryRow(ctx, query, tenantID, academicYearID, prefix).Scan(&sequence); err != nil {
		return 0, err
	}
	return sequence, nil
}

func (r *billingRepository) CreateInvoice(ctx context.Context, invoice *model.Invoice) error {
	const query = `INSERT INTO invoices (
			tenant_id, invoice_number, student_id, academic_year_id, class_id, section_id,
			fee_structure_id, assignment_id, issue_date, due_date, billing_period_start,
			billing_period_end, generation_key, status, currency, allow_partial_payment,
			minimum_partial_amount_paise, subtotal_amount_paise, discount_amount_paise, fine_amount_paise, tax_amount_paise,
			total_amount_paise, paid_amount_paise, balance_amount_paise, generated_by, metadata, issued_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26, $27
		)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		invoice.TenantID,
		invoice.InvoiceNumber,
		invoice.StudentID,
		invoice.AcademicYearID,
		invoice.ClassID,
		invoice.SectionID,
		invoice.FeeStructureID,
		invoice.AssignmentID,
		invoice.IssueDate,
		invoice.DueDate,
		invoice.BillingPeriodStart,
		invoice.BillingPeriodEnd,
		invoice.GenerationKey,
		invoice.Status,
		invoice.Currency,
		invoice.AllowPartialPayment,
		invoice.MinimumPartialAmountPaise,
		invoice.SubtotalAmountPaise,
		invoice.DiscountAmountPaise,
		invoice.FineAmountPaise,
		invoice.TaxAmountPaise,
		invoice.TotalAmountPaise,
		invoice.PaidAmountPaise,
		invoice.BalanceAmountPaise,
		invoice.GeneratedBy,
		mustJSON(invoice.Metadata),
		invoice.IssuedAt,
	).Scan(&invoice.ID, &invoice.CreatedAt, &invoice.UpdatedAt)
}

func (r *billingRepository) CreateInvoiceItems(ctx context.Context, items []model.InvoiceItem) error {
	const query = `INSERT INTO invoice_items (
			tenant_id, invoice_id, fee_head_id, fee_structure_item_id, description,
			amount_paise, discount_amount_paise, fine_amount_paise, tax_amount_paise,
			total_amount_paise, sort_order, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`

	for i := range items {
		if err := r.db.QueryRow(ctx, query,
			items[i].TenantID,
			items[i].InvoiceID,
			items[i].FeeHeadID,
			items[i].FeeStructureItemID,
			items[i].Description,
			items[i].AmountPaise,
			items[i].DiscountAmountPaise,
			items[i].FineAmountPaise,
			items[i].TaxAmountPaise,
			items[i].TotalAmountPaise,
			items[i].SortOrder,
			mustJSON(items[i].Metadata),
		).Scan(&items[i].ID, &items[i].CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *billingRepository) GetInvoice(ctx context.Context, tenantID, id uuid.UUID) (*model.Invoice, error) {
	const query = invoiceSelect + ` WHERE i.tenant_id = $1 AND i.id = $2`
	item, err := r.scanInvoice(ctx, query, tenantID, id)
	if err != nil || item == nil {
		return item, err
	}
	items, err := r.ListInvoiceItems(ctx, tenantID, item.ID)
	if err != nil {
		return nil, err
	}
	item.Items = items
	return item, nil
}

func (r *billingRepository) GetInvoiceByGenerationKey(ctx context.Context, tenantID uuid.UUID, generationKey string) (*model.Invoice, error) {
	const query = invoiceSelect + ` WHERE i.tenant_id = $1 AND i.generation_key = $2 AND i.status NOT IN ('cancelled', 'void')`
	return r.scanInvoice(ctx, query, tenantID, generationKey)
}

func (r *billingRepository) ListInvoices(ctx context.Context, tenantID uuid.UUID, filter model.InvoiceFilter, params model.PaginationParams) (*model.PaginatedResult[model.Invoice], error) {
	params.Normalize()
	where, args := invoiceWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM invoices i JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedInvoiceSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY i.%s %s LIMIT %s OFFSET %s`, invoiceSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Invoice, 0, params.PageSize)
	for rows.Next() {
		item, err := scanInvoiceScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *billingRepository) ListInvoiceItems(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]model.InvoiceItem, error) {
	const query = invoiceItemSelect + ` WHERE ii.tenant_id = $1 AND ii.invoice_id = $2 ORDER BY ii.sort_order ASC, ii.created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.InvoiceItem{}
	for rows.Next() {
		item, err := scanInvoiceItemScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *billingRepository) ListStudentInvoices(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.Invoice, error) {
	const query = invoiceSelect + ` WHERE i.tenant_id = $1 AND i.student_id = $2 AND i.status NOT IN ('cancelled', 'void') ORDER BY i.issue_date ASC, i.created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Invoice{}
	for rows.Next() {
		item, err := scanInvoiceScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *billingRepository) createFeeStructureItem(ctx context.Context, item *model.FeeStructureItem) error {
	const query = `INSERT INTO fee_structure_items (
			tenant_id, fee_structure_id, fee_head_id, name, description, amount_paise,
			tax_rate_bps, sort_order, optional, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		item.TenantID,
		item.FeeStructureID,
		item.FeeHeadID,
		item.Name,
		item.Description,
		item.AmountPaise,
		item.TaxRateBps,
		item.SortOrder,
		item.Optional,
		mustJSON(item.Metadata),
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

const feeHeadSelect = `SELECT id, tenant_id, name, code, description, category, status, taxable, tax_rate_bps, metadata, created_at, updated_at, deleted_at FROM fee_heads`

const feeStructureSelect = `SELECT
		fs.id, fs.tenant_id, fs.academic_year_id, fs.name, fs.code, fs.description,
		fs.billing_cycle, fs.status, fs.currency, fs.allow_partial_payment,
		fs.minimum_partial_amount_paise, fs.due_day, fs.metadata, fs.created_at, fs.updated_at, fs.deleted_at,
		ay.id, ay.name, ay.code
	FROM fee_structures fs
	JOIN academic_years ay ON ay.tenant_id = fs.tenant_id AND ay.id = fs.academic_year_id`

const feeStructureItemSelect = `SELECT
		fsi.id, fsi.tenant_id, fsi.fee_structure_id, fsi.fee_head_id, fsi.name,
		fsi.description, fsi.amount_paise, fsi.tax_rate_bps, fsi.sort_order, fsi.optional,
		fsi.metadata, fsi.created_at, fsi.updated_at, fsi.deleted_at,
		fh.id, fh.tenant_id, fh.name, fh.code, fh.description, fh.category, fh.status,
		fh.taxable, fh.tax_rate_bps, fh.metadata, fh.created_at, fh.updated_at, fh.deleted_at
	FROM fee_structure_items fsi
	JOIN fee_heads fh ON fh.tenant_id = fsi.tenant_id AND fh.id = fsi.fee_head_id`

const feeAssignmentSelect = `SELECT
		id, tenant_id, fee_structure_id, academic_year_id, assignment_type, class_id,
		section_id, student_id, status, effective_from, effective_until, created_by,
		metadata, created_at, updated_at, deleted_at
	FROM student_fee_assignments`

const concessionSelect = `SELECT
		id, tenant_id, academic_year_id, student_id, fee_head_id, name, code,
		concession_type, amount_paise, percentage_bps, reason, status, starts_on,
		ends_on, metadata, created_at, updated_at, deleted_at
	FROM concessions`

const invoiceSelect = `SELECT
		i.id, i.tenant_id, i.invoice_number, i.student_id, i.academic_year_id, i.class_id,
		i.section_id, i.fee_structure_id, i.assignment_id, i.issue_date, i.due_date,
		i.billing_period_start, i.billing_period_end, i.generation_key, i.status, i.currency,
		i.allow_partial_payment, i.minimum_partial_amount_paise, i.subtotal_amount_paise, i.discount_amount_paise,
		i.fine_amount_paise, i.tax_amount_paise, i.total_amount_paise, i.paid_amount_paise,
		i.balance_amount_paise, i.generated_by, i.metadata, i.issued_at, i.cancelled_at,
		i.voided_at, i.created_at, i.updated_at,
		s.id, s.admission_number, s.first_name, s.last_name, s.status,
		ay.id, ay.name, ay.code,
		c.id, c.name, c.code,
		sec.id, sec.name, sec.code
	FROM invoices i
	JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id
	JOIN academic_years ay ON ay.tenant_id = i.tenant_id AND ay.id = i.academic_year_id
	JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
	JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id`

const invoiceItemSelect = `SELECT
		ii.id, ii.tenant_id, ii.invoice_id, ii.fee_head_id, ii.fee_structure_item_id,
		ii.description, ii.amount_paise, ii.discount_amount_paise, ii.fine_amount_paise,
		ii.tax_amount_paise, ii.total_amount_paise, ii.sort_order, ii.metadata, ii.created_at,
		fh.id, fh.tenant_id, fh.name, fh.code, fh.description, fh.category, fh.status,
		fh.taxable, fh.tax_rate_bps, fh.metadata, fh.created_at, fh.updated_at, fh.deleted_at
	FROM invoice_items ii
	JOIN fee_heads fh ON fh.tenant_id = ii.tenant_id AND fh.id = ii.fee_head_id`

func (r *billingRepository) scanFeeHead(ctx context.Context, query string, args ...any) (*model.FeeHead, error) {
	item, err := scanFeeHeadScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *billingRepository) scanFeeStructure(ctx context.Context, query string, args ...any) (*model.FeeStructure, error) {
	item, err := scanFeeStructureScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *billingRepository) scanFeeAssignment(ctx context.Context, query string, args ...any) (*model.StudentFeeAssignment, error) {
	item, err := scanFeeAssignmentScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *billingRepository) scanInvoice(ctx context.Context, query string, args ...any) (*model.Invoice, error) {
	item, err := scanInvoiceScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func scanFeeHeadScanner(row rowScanner) (*model.FeeHead, error) {
	var item model.FeeHead
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Code,
		&item.Description,
		&item.Category,
		&item.Status,
		&item.Taxable,
		&item.TaxRateBps,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func scanFeeStructureScanner(row rowScanner) (*model.FeeStructure, error) {
	var item model.FeeStructure
	var metadata []byte
	academicYear := model.AcademicYear{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AcademicYearID,
		&item.Name,
		&item.Code,
		&item.Description,
		&item.BillingCycle,
		&item.Status,
		&item.Currency,
		&item.AllowPartialPayment,
		&item.MinimumPartialAmountPaise,
		&item.DueDay,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
		&academicYear.ID,
		&academicYear.Name,
		&academicYear.Code,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.AcademicYear = &academicYear
	return &item, nil
}

func scanFeeStructureItemScanner(row rowScanner) (*model.FeeStructureItem, error) {
	var item model.FeeStructureItem
	var metadata []byte
	feeHead, err := scanFeeHeadTail(row, &item.ID, &item.TenantID, &item.FeeStructureID, &item.FeeHeadID, &item.Name, &item.Description, &item.AmountPaise, &item.TaxRateBps, &item.SortOrder, &item.Optional, &metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	if err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.FeeHead = feeHead
	return &item, nil
}

func scanFeeAssignmentScanner(row rowScanner) (*model.StudentFeeAssignment, error) {
	var item model.StudentFeeAssignment
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.FeeStructureID,
		&item.AcademicYearID,
		&item.AssignmentType,
		&item.ClassID,
		&item.SectionID,
		&item.StudentID,
		&item.Status,
		&item.EffectiveFrom,
		&item.EffectiveUntil,
		&item.CreatedBy,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func scanConcessionScanner(row rowScanner) (*model.Concession, error) {
	var item model.Concession
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AcademicYearID,
		&item.StudentID,
		&item.FeeHeadID,
		&item.Name,
		&item.Code,
		&item.ConcessionType,
		&item.AmountPaise,
		&item.PercentageBps,
		&item.Reason,
		&item.Status,
		&item.StartsOn,
		&item.EndsOn,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func scanInvoiceScanner(row rowScanner) (*model.Invoice, error) {
	var item model.Invoice
	var metadata []byte
	student := model.Student{}
	academicYear := model.AcademicYear{}
	class := model.Class{}
	section := model.Section{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.InvoiceNumber,
		&item.StudentID,
		&item.AcademicYearID,
		&item.ClassID,
		&item.SectionID,
		&item.FeeStructureID,
		&item.AssignmentID,
		&item.IssueDate,
		&item.DueDate,
		&item.BillingPeriodStart,
		&item.BillingPeriodEnd,
		&item.GenerationKey,
		&item.Status,
		&item.Currency,
		&item.AllowPartialPayment,
		&item.MinimumPartialAmountPaise,
		&item.SubtotalAmountPaise,
		&item.DiscountAmountPaise,
		&item.FineAmountPaise,
		&item.TaxAmountPaise,
		&item.TotalAmountPaise,
		&item.PaidAmountPaise,
		&item.BalanceAmountPaise,
		&item.GeneratedBy,
		&metadata,
		&item.IssuedAt,
		&item.CancelledAt,
		&item.VoidedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&student.Status,
		&academicYear.ID,
		&academicYear.Name,
		&academicYear.Code,
		&class.ID,
		&class.Name,
		&class.Code,
		&section.ID,
		&section.Name,
		&section.Code,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.Student = &student
	item.AcademicYear = &academicYear
	item.Class = &class
	item.Section = &section
	return &item, nil
}

func scanInvoiceItemScanner(row rowScanner) (*model.InvoiceItem, error) {
	var item model.InvoiceItem
	var metadata []byte
	feeHead, err := scanFeeHeadTail(row, &item.ID, &item.TenantID, &item.InvoiceID, &item.FeeHeadID, &item.FeeStructureItemID, &item.Description, &item.AmountPaise, &item.DiscountAmountPaise, &item.FineAmountPaise, &item.TaxAmountPaise, &item.TotalAmountPaise, &item.SortOrder, &metadata, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.FeeHead = feeHead
	return &item, nil
}

func scanFeeHeadTail(row rowScanner, prefixDest ...any) (*model.FeeHead, error) {
	feeHead := model.FeeHead{}
	var feeHeadMetadata []byte
	dest := append(prefixDest,
		&feeHead.ID,
		&feeHead.TenantID,
		&feeHead.Name,
		&feeHead.Code,
		&feeHead.Description,
		&feeHead.Category,
		&feeHead.Status,
		&feeHead.Taxable,
		&feeHead.TaxRateBps,
		&feeHeadMetadata,
		&feeHead.CreatedAt,
		&feeHead.UpdatedAt,
		&feeHead.DeletedAt,
	)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	feeHead.Metadata = parseJSON(feeHeadMetadata)
	return &feeHead, nil
}

func feeHeadWhere(tenantID uuid.UUID, filter model.FeeHeadFilter) (string, []any) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, strings.TrimSpace(filter.Category))
		clauses = append(clauses, fmt.Sprintf("category = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(name) LIKE $%d OR lower(code) LIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func feeStructureWhere(tenantID uuid.UUID, filter model.FeeStructureFilter) (string, []any) {
	clauses := []string{"fs.tenant_id = $1", "fs.deleted_at IS NULL"}
	args := []any{tenantID}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("fs.academic_year_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("fs.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.BillingCycle) != "" {
		args = append(args, strings.TrimSpace(filter.BillingCycle))
		clauses = append(clauses, fmt.Sprintf("fs.billing_cycle = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(fs.name) LIKE $%d OR lower(fs.code) LIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func feeAssignmentWhere(tenantID uuid.UUID, filter model.FeeAssignmentFilter) (string, []any) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	if filter.FeeStructureID != nil {
		args = append(args, *filter.FeeStructureID)
		clauses = append(clauses, fmt.Sprintf("fee_structure_id = $%d", len(args)))
	}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("academic_year_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.AssignmentType) != "" {
		args = append(args, strings.TrimSpace(filter.AssignmentType))
		clauses = append(clauses, fmt.Sprintf("assignment_type = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func invoiceWhere(tenantID uuid.UUID, filter model.InvoiceFilter) (string, []any) {
	clauses := []string{"i.tenant_id = $1"}
	args := []any{tenantID}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("i.student_id = $%d", len(args)))
	}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("i.academic_year_id = $%d", len(args)))
	}
	if filter.ClassID != nil {
		args = append(args, *filter.ClassID)
		clauses = append(clauses, fmt.Sprintf("i.class_id = $%d", len(args)))
	}
	if filter.SectionID != nil {
		args = append(args, *filter.SectionID)
		clauses = append(clauses, fmt.Sprintf("i.section_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("i.status = $%d", len(args)))
	}
	if filter.DueFrom != nil {
		args = append(args, *filter.DueFrom)
		clauses = append(clauses, fmt.Sprintf("i.due_date >= $%d", len(args)))
	}
	if filter.DueTo != nil {
		args = append(args, *filter.DueTo)
		clauses = append(clauses, fmt.Sprintf("i.due_date <= $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(i.invoice_number) LIKE $%d OR lower(s.admission_number) LIKE $%d OR lower(s.first_name) LIKE $%d OR lower(s.last_name) LIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

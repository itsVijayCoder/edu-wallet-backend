package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

var _ repository.AcademicRepository = (*academicRepository)(nil)

type academicRepository struct {
	db database.DBTX
}

func NewAcademicRepository(db database.DBTX) repository.AcademicRepository {
	return &academicRepository{db: db}
}

var (
	allowedAcademicYearSortColumns = map[string]bool{"created_at": true, "name": true, "code": true, "start_date": true, "end_date": true, "status": true}
	allowedClassSortColumns        = map[string]bool{"created_at": true, "name": true, "code": true, "sort_order": true, "status": true}
	allowedSectionSortColumns      = map[string]bool{"created_at": true, "name": true, "code": true, "status": true}
	allowedStudentSortColumns      = map[string]bool{"created_at": true, "admission_number": true, "first_name": true, "last_name": true, "status": true, "roll_number": true}
	allowedGuardianSortColumns     = map[string]bool{"created_at": true, "name": true, "relationship": true}
	allowedImportSortColumns       = map[string]bool{"created_at": true, "status": true, "import_type": true}
)

func (r *academicRepository) CreateAcademicYear(ctx context.Context, academicYear *model.AcademicYear) error {
	const query = `INSERT INTO academic_years (
			tenant_id, name, code, start_date, end_date, status, is_active, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		academicYear.TenantID,
		academicYear.Name,
		academicYear.Code,
		academicYear.StartDate,
		academicYear.EndDate,
		academicYear.Status,
		academicYear.IsActive,
		mustJSON(academicYear.Metadata),
	).Scan(&academicYear.ID, &academicYear.CreatedAt, &academicYear.UpdatedAt)
}

func (r *academicRepository) GetAcademicYear(ctx context.Context, tenantID, id uuid.UUID) (*model.AcademicYear, error) {
	const query = academicYearSelect + ` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`
	return r.scanAcademicYear(ctx, query, tenantID, id)
}

func (r *academicRepository) GetAcademicYearByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.AcademicYear, error) {
	const query = academicYearSelect + ` WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL`
	return r.scanAcademicYear(ctx, query, tenantID, code)
}

func (r *academicRepository) ListAcademicYears(ctx context.Context, tenantID uuid.UUID, filter model.AcademicYearFilter, params model.PaginationParams) (*model.PaginatedResult[model.AcademicYear], error) {
	params.Normalize()
	where, args := academicYearWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM academic_years `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedAcademicYearSortColumns, "start_date", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, academicYearSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AcademicYear, 0, params.PageSize)
	for rows.Next() {
		item, err := scanAcademicYearScanner(rows)
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

func (r *academicRepository) UpdateAcademicYear(ctx context.Context, academicYear *model.AcademicYear) error {
	const query = `UPDATE academic_years SET
			name = $1,
			code = $2,
			start_date = $3,
			end_date = $4,
			status = $5,
			is_active = $6,
			metadata = $7,
			updated_at = NOW()
		WHERE tenant_id = $8 AND id = $9 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		academicYear.Name,
		academicYear.Code,
		academicYear.StartDate,
		academicYear.EndDate,
		academicYear.Status,
		academicYear.IsActive,
		mustJSON(academicYear.Metadata),
		academicYear.TenantID,
		academicYear.ID,
	).Scan(&academicYear.UpdatedAt)
}

func (r *academicRepository) SoftDeleteAcademicYear(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE academic_years SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *academicRepository) CreateClass(ctx context.Context, class *model.Class) error {
	const query = `INSERT INTO classes (tenant_id, name, code, sort_order, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		class.TenantID,
		class.Name,
		class.Code,
		class.SortOrder,
		class.Status,
		mustJSON(class.Metadata),
	).Scan(&class.ID, &class.CreatedAt, &class.UpdatedAt)
}

func (r *academicRepository) GetClass(ctx context.Context, tenantID, id uuid.UUID) (*model.Class, error) {
	const query = classSelect + ` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`
	return r.scanClass(ctx, query, tenantID, id)
}

func (r *academicRepository) GetClassByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.Class, error) {
	const query = classSelect + ` WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL`
	return r.scanClass(ctx, query, tenantID, code)
}

func (r *academicRepository) ListClasses(ctx context.Context, tenantID uuid.UUID, filter model.ClassFilter, params model.PaginationParams) (*model.PaginatedResult[model.Class], error) {
	params.Normalize()
	where, args := classWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM classes `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedClassSortColumns, "sort_order", "ASC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s, name ASC LIMIT %s OFFSET %s`, classSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Class, 0, params.PageSize)
	for rows.Next() {
		item, err := scanClassScanner(rows)
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

func (r *academicRepository) UpdateClass(ctx context.Context, class *model.Class) error {
	const query = `UPDATE classes SET
			name = $1,
			code = $2,
			sort_order = $3,
			status = $4,
			metadata = $5,
			updated_at = NOW()
		WHERE tenant_id = $6 AND id = $7 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		class.Name,
		class.Code,
		class.SortOrder,
		class.Status,
		mustJSON(class.Metadata),
		class.TenantID,
		class.ID,
	).Scan(&class.UpdatedAt)
}

func (r *academicRepository) SoftDeleteClass(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE classes SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *academicRepository) CreateSection(ctx context.Context, section *model.Section) error {
	const query = `INSERT INTO sections (
			tenant_id, academic_year_id, class_id, branch_id, name, code, capacity, status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		section.TenantID,
		section.AcademicYearID,
		section.ClassID,
		section.BranchID,
		section.Name,
		section.Code,
		section.Capacity,
		section.Status,
		mustJSON(section.Metadata),
	).Scan(&section.ID, &section.CreatedAt, &section.UpdatedAt)
}

func (r *academicRepository) GetSection(ctx context.Context, tenantID, id uuid.UUID) (*model.Section, error) {
	const query = sectionSelect + ` WHERE s.tenant_id = $1 AND s.id = $2 AND s.deleted_at IS NULL`
	return r.scanSection(ctx, query, tenantID, id)
}

func (r *academicRepository) GetSectionByCode(ctx context.Context, tenantID, academicYearID, classID uuid.UUID, code string) (*model.Section, error) {
	const query = sectionSelect + ` WHERE s.tenant_id = $1 AND s.academic_year_id = $2 AND s.class_id = $3 AND lower(s.code) = lower($4) AND s.deleted_at IS NULL`
	return r.scanSection(ctx, query, tenantID, academicYearID, classID, code)
}

func (r *academicRepository) ListSections(ctx context.Context, tenantID uuid.UUID, filter model.SectionFilter, params model.PaginationParams) (*model.PaginatedResult[model.Section], error) {
	params.Normalize()
	where, args := sectionWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM sections s `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedSectionSortColumns, "name", "ASC")
	query := fmt.Sprintf(`%s %s ORDER BY s.%s %s LIMIT %s OFFSET %s`, sectionSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Section, 0, params.PageSize)
	for rows.Next() {
		item, err := scanSectionScanner(rows)
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

func (r *academicRepository) UpdateSection(ctx context.Context, section *model.Section) error {
	const query = `UPDATE sections SET
			academic_year_id = $1,
			class_id = $2,
			branch_id = $3,
			name = $4,
			code = $5,
			capacity = $6,
			status = $7,
			metadata = $8,
			updated_at = NOW()
		WHERE tenant_id = $9 AND id = $10 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		section.AcademicYearID,
		section.ClassID,
		section.BranchID,
		section.Name,
		section.Code,
		section.Capacity,
		section.Status,
		mustJSON(section.Metadata),
		section.TenantID,
		section.ID,
	).Scan(&section.UpdatedAt)
}

func (r *academicRepository) SoftDeleteSection(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE sections SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *academicRepository) CreateStudent(ctx context.Context, student *model.Student) error {
	const query = `INSERT INTO students (
			tenant_id, academic_year_id, class_id, section_id, branch_id, admission_number,
			first_name, last_name, roll_number, status, category, phone, email,
			address_line1, address_line2, city, state, postal_code, country,
			opening_balance_paise, currency, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		student.TenantID,
		student.AcademicYearID,
		student.ClassID,
		student.SectionID,
		student.BranchID,
		student.AdmissionNumber,
		student.FirstName,
		student.LastName,
		student.RollNumber,
		student.Status,
		student.Category,
		student.Phone,
		student.Email,
		student.Address.Line1,
		student.Address.Line2,
		student.Address.City,
		student.Address.State,
		student.Address.PostalCode,
		student.Address.Country,
		student.OpeningBalancePaise,
		student.Currency,
		mustJSON(student.Metadata),
	).Scan(&student.ID, &student.CreatedAt, &student.UpdatedAt)
}

func (r *academicRepository) GetStudent(ctx context.Context, tenantID, id uuid.UUID) (*model.Student, error) {
	const query = studentSelect + ` WHERE s.tenant_id = $1 AND s.id = $2 AND s.deleted_at IS NULL`
	return r.scanStudent(ctx, query, tenantID, id)
}

func (r *academicRepository) GetStudentByAdmissionNumber(ctx context.Context, tenantID uuid.UUID, admissionNumber string) (*model.Student, error) {
	const query = studentSelect + ` WHERE s.tenant_id = $1 AND lower(s.admission_number) = lower($2) AND s.deleted_at IS NULL`
	return r.scanStudent(ctx, query, tenantID, admissionNumber)
}

func (r *academicRepository) ListStudents(ctx context.Context, tenantID uuid.UUID, filter model.StudentFilter, params model.PaginationParams) (*model.PaginatedResult[model.Student], error) {
	params.Normalize()
	where, args := studentWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM students s `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedStudentSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY s.%s %s LIMIT %s OFFSET %s`, studentSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Student, 0, params.PageSize)
	for rows.Next() {
		item, err := scanStudentScanner(rows)
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

func (r *academicRepository) UpdateStudent(ctx context.Context, student *model.Student) error {
	const query = `UPDATE students SET
			academic_year_id = $1,
			class_id = $2,
			section_id = $3,
			branch_id = $4,
			admission_number = $5,
			first_name = $6,
			last_name = $7,
			roll_number = $8,
			status = $9,
			category = $10,
			phone = $11,
			email = $12,
			address_line1 = $13,
			address_line2 = $14,
			city = $15,
			state = $16,
			postal_code = $17,
			country = $18,
			opening_balance_paise = $19,
			currency = $20,
			metadata = $21,
			updated_at = NOW()
		WHERE tenant_id = $22 AND id = $23 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		student.AcademicYearID,
		student.ClassID,
		student.SectionID,
		student.BranchID,
		student.AdmissionNumber,
		student.FirstName,
		student.LastName,
		student.RollNumber,
		student.Status,
		student.Category,
		student.Phone,
		student.Email,
		student.Address.Line1,
		student.Address.Line2,
		student.Address.City,
		student.Address.State,
		student.Address.PostalCode,
		student.Address.Country,
		student.OpeningBalancePaise,
		student.Currency,
		mustJSON(student.Metadata),
		student.TenantID,
		student.ID,
	).Scan(&student.UpdatedAt)
}

func (r *academicRepository) SoftDeleteStudent(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE students SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *academicRepository) CreateGuardian(ctx context.Context, guardian *model.Guardian) error {
	const query = `INSERT INTO guardians (
			tenant_id, name, relationship, phone, whatsapp_phone, email, preferred_language,
			communication_opt_in, opt_in_whatsapp, address_line1, address_line2, city, state, postal_code, country, user_id, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		guardian.TenantID,
		guardian.Name,
		guardian.Relationship,
		guardian.Phone,
		guardian.WhatsAppPhone,
		guardian.Email,
		guardian.PreferredLanguage,
		guardian.CommunicationOptIn,
		guardian.OptInWhatsApp,
		guardian.Address.Line1,
		guardian.Address.Line2,
		guardian.Address.City,
		guardian.Address.State,
		guardian.Address.PostalCode,
		guardian.Address.Country,
		guardian.UserID,
		mustJSON(guardian.Metadata),
	).Scan(&guardian.ID, &guardian.CreatedAt, &guardian.UpdatedAt)
}

func (r *academicRepository) GetGuardian(ctx context.Context, tenantID, id uuid.UUID) (*model.Guardian, error) {
	const query = guardianSelect + ` WHERE g.tenant_id = $1 AND g.id = $2 AND g.deleted_at IS NULL`
	return r.scanGuardian(ctx, query, tenantID, id)
}

func (r *academicRepository) GetGuardianByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*model.Guardian, error) {
	const query = guardianSelect + ` WHERE g.tenant_id = $1 AND g.user_id = $2 AND g.deleted_at IS NULL`
	return r.scanGuardian(ctx, query, tenantID, userID)
}

func (r *academicRepository) FindGuardianByContact(ctx context.Context, tenantID uuid.UUID, email, phone *string) (*model.Guardian, error) {
	clauses := []string{"g.tenant_id = $1", "g.deleted_at IS NULL"}
	args := []any{tenantID}
	contactClauses := []string{}
	if email != nil && strings.TrimSpace(*email) != "" {
		args = append(args, strings.TrimSpace(*email))
		contactClauses = append(contactClauses, fmt.Sprintf("lower(g.email) = lower($%d)", len(args)))
	}
	if phone != nil && strings.TrimSpace(*phone) != "" {
		args = append(args, strings.TrimSpace(*phone))
		contactClauses = append(contactClauses, fmt.Sprintf("g.phone = $%d", len(args)))
	}
	if len(contactClauses) == 0 {
		return nil, nil
	}
	clauses = append(clauses, "("+strings.Join(contactClauses, " OR ")+")")
	query := guardianSelect + ` WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY g.created_at ASC LIMIT 1`
	return r.scanGuardian(ctx, query, args...)
}

func (r *academicRepository) ListGuardians(ctx context.Context, tenantID uuid.UUID, filter model.GuardianFilter, params model.PaginationParams) (*model.PaginatedResult[model.Guardian], error) {
	params.Normalize()
	where, args := guardianWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM guardians g `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedGuardianSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY g.%s %s LIMIT %s OFFSET %s`, guardianSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Guardian, 0, params.PageSize)
	for rows.Next() {
		item, err := scanGuardianScanner(rows)
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

func (r *academicRepository) UpdateGuardian(ctx context.Context, guardian *model.Guardian) error {
	const query = `UPDATE guardians SET
			name = $1,
			relationship = $2,
			phone = $3,
			whatsapp_phone = $4,
			email = $5,
			preferred_language = $6,
			communication_opt_in = $7,
			opt_in_whatsapp = $8,
			address_line1 = $9,
			address_line2 = $10,
			city = $11,
			state = $12,
			postal_code = $13,
			country = $14,
			user_id = $15,
			metadata = $16,
			updated_at = NOW()
		WHERE tenant_id = $17 AND id = $18 AND deleted_at IS NULL
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		guardian.Name,
		guardian.Relationship,
		guardian.Phone,
		guardian.WhatsAppPhone,
		guardian.Email,
		guardian.PreferredLanguage,
		guardian.CommunicationOptIn,
		guardian.OptInWhatsApp,
		guardian.Address.Line1,
		guardian.Address.Line2,
		guardian.Address.City,
		guardian.Address.State,
		guardian.Address.PostalCode,
		guardian.Address.Country,
		guardian.UserID,
		mustJSON(guardian.Metadata),
		guardian.TenantID,
		guardian.ID,
	).Scan(&guardian.UpdatedAt)
}

func (r *academicRepository) SoftDeleteGuardian(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE guardians SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *academicRepository) SetGuardianUserID(ctx context.Context, tenantID, guardianID uuid.UUID, userID *uuid.UUID) error {
	const query = `UPDATE guardians SET user_id = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3 AND deleted_at IS NULL
		RETURNING id`
	var returned uuid.UUID
	if err := r.db.QueryRow(ctx, query, userID, tenantID, guardianID).Scan(&returned); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

func (r *academicRepository) SetStudentGuardians(ctx context.Context, tenantID, studentID uuid.UUID, links []model.StudentGuardian) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM student_guardians WHERE tenant_id = $1 AND student_id = $2`, tenantID, studentID); err != nil {
		return err
	}
	for i := range links {
		links[i].TenantID = tenantID
		links[i].StudentID = studentID
		if err := r.LinkStudentGuardian(ctx, &links[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *academicRepository) LinkStudentGuardian(ctx context.Context, link *model.StudentGuardian) error {
	if link.IsPrimary {
		if _, err := r.db.Exec(ctx, `UPDATE student_guardians SET is_primary = FALSE WHERE tenant_id = $1 AND student_id = $2`, link.TenantID, link.StudentID); err != nil {
			return err
		}
	}

	const query = `INSERT INTO student_guardians (tenant_id, student_id, guardian_id, relationship, is_primary)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (student_id, guardian_id)
		DO UPDATE SET relationship = EXCLUDED.relationship, is_primary = EXCLUDED.is_primary
		RETURNING created_at`

	return r.db.QueryRow(ctx, query,
		link.TenantID,
		link.StudentID,
		link.GuardianID,
		link.Relationship,
		link.IsPrimary,
	).Scan(&link.CreatedAt)
}

func (r *academicRepository) UnlinkStudentGuardian(ctx context.Context, tenantID, studentID, guardianID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM student_guardians WHERE tenant_id = $1 AND student_id = $2 AND guardian_id = $3`, tenantID, studentID, guardianID)
	return err
}

func (r *academicRepository) ListStudentGuardians(ctx context.Context, tenantID, studentID uuid.UUID) ([]model.StudentGuardian, error) {
	const query = `SELECT
			sg.tenant_id, sg.student_id, sg.guardian_id, sg.relationship, sg.is_primary, sg.created_at,
			g.id, g.tenant_id, g.name, g.relationship, g.phone, g.whatsapp_phone, g.email,
			g.preferred_language, g.communication_opt_in,
			g.address_line1, g.address_line2, g.city, g.state, g.postal_code, g.country,
			g.user_id, g.metadata, g.created_at, g.updated_at, g.deleted_at
		FROM student_guardians sg
		JOIN guardians g ON g.tenant_id = sg.tenant_id AND g.id = sg.guardian_id
		WHERE sg.tenant_id = $1 AND sg.student_id = $2 AND g.deleted_at IS NULL
		ORDER BY sg.is_primary DESC, g.name ASC`

	rows, err := r.db.Query(ctx, query, tenantID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []model.StudentGuardian{}
	for rows.Next() {
		link, err := scanStudentGuardianRow(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *link)
	}
	return links, rows.Err()
}

// ListGuardianStudents is the reverse projection: given a guardian ID, return
// the students linked to it with the student, class, and section details
// required to render the admin "Parents" view without a second round trip.
func (r *academicRepository) ListGuardianStudents(ctx context.Context, tenantID, guardianID uuid.UUID) ([]model.GuardianStudent, error) {
	itemsByGuardian, err := r.ListGuardianStudentsByGuardianIDs(ctx, tenantID, []uuid.UUID{guardianID})
	if err != nil {
		return nil, err
	}
	return itemsByGuardian[guardianID], nil
}

// ListGuardianStudentsPaginated is the parent portal projection. The query
// keeps the guardian predicate in SQL, so search and pagination never operate
// on students belonging to another guardian.
func (r *academicRepository) ListGuardianStudentsPaginated(ctx context.Context, tenantID, guardianID uuid.UUID, filter model.GuardianStudentFilter, params model.PaginationParams) (*model.PaginatedResult[model.GuardianStudent], error) {
	params.Normalize()
	clauses := []string{"sg.tenant_id = $1", "sg.guardian_id = $2", "s.deleted_at IS NULL"}
	args := []any{tenantID, guardianID}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(s.first_name) LIKE $%d OR lower(s.last_name) LIKE $%d OR lower(s.admission_number) LIKE $%d)", len(args), len(args), len(args)))
	}
	where := "WHERE " + strings.Join(clauses, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM student_guardians sg JOIN students s ON s.tenant_id = sg.tenant_id AND s.id = sg.student_id ` + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	const selectQuery = `SELECT
			sg.guardian_id, sg.student_id, s.admission_number, s.first_name, s.last_name,
			sg.relationship, sg.is_primary,
			coalesce(c.name, '') AS class_name,
			coalesce(sec.name, '') AS section_name,
			s.status
		FROM student_guardians sg
		JOIN students s ON s.tenant_id = sg.tenant_id AND s.id = sg.student_id
		LEFT JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id
		LEFT JOIN sections sec ON sec.tenant_id = s.tenant_id AND sec.id = s.section_id`
	query := selectQuery + ` ` + where + ` ORDER BY s.first_name ASC, s.last_name ASC, s.admission_number ASC LIMIT ` + limitParam + ` OFFSET ` + offsetParam
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.GuardianStudent, 0, params.PageSize)
	for rows.Next() {
		item, err := scanGuardianStudentRow(rows)
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

// ListGuardianStudentsByGuardianIDs resolves the linked students for an
// entire guardian page in one query, avoiding one query per parent row.
func (r *academicRepository) ListGuardianStudentsByGuardianIDs(ctx context.Context, tenantID uuid.UUID, guardianIDs []uuid.UUID) (map[uuid.UUID][]model.GuardianStudent, error) {
	itemsByGuardian := make(map[uuid.UUID][]model.GuardianStudent, len(guardianIDs))
	for i := range guardianIDs {
		itemsByGuardian[guardianIDs[i]] = []model.GuardianStudent{}
	}
	if len(guardianIDs) == 0 {
		return itemsByGuardian, nil
	}

	const query = `SELECT
			sg.guardian_id, sg.student_id, s.admission_number, s.first_name, s.last_name,
			sg.relationship, sg.is_primary,
			coalesce(c.name, '') AS class_name,
			coalesce(sec.name, '') AS section_name,
			s.status
		FROM student_guardians sg
		JOIN students s ON s.tenant_id = sg.tenant_id AND s.id = sg.student_id AND s.deleted_at IS NULL
		LEFT JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id
		LEFT JOIN sections sec ON sec.tenant_id = s.tenant_id AND sec.id = s.section_id
		WHERE sg.tenant_id = $1 AND sg.guardian_id = ANY($2::uuid[])
		ORDER BY sg.guardian_id, sg.is_primary DESC, s.first_name ASC, s.last_name ASC`

	rows, err := r.db.Query(ctx, query, tenantID, guardianIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanGuardianStudentRow(rows)
		if err != nil {
			return nil, err
		}
		itemsByGuardian[item.GuardianID] = append(itemsByGuardian[item.GuardianID], *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemsByGuardian, nil
}

func (r *academicRepository) CreateImport(ctx context.Context, imp *model.Import) error {
	const query = `INSERT INTO imports (
			tenant_id, import_type, status, source_filename, total_rows, valid_rows,
			invalid_rows, committed_rows, created_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		imp.TenantID,
		imp.ImportType,
		imp.Status,
		imp.SourceFilename,
		imp.TotalRows,
		imp.ValidRows,
		imp.InvalidRows,
		imp.CommittedRows,
		imp.CreatedBy,
		mustJSON(imp.Metadata),
	).Scan(&imp.ID, &imp.CreatedAt, &imp.UpdatedAt)
}

func (r *academicRepository) GetImport(ctx context.Context, tenantID, id uuid.UUID) (*model.Import, error) {
	const query = importSelect + ` WHERE tenant_id = $1 AND id = $2`
	return r.scanImport(ctx, query, tenantID, id)
}

func (r *academicRepository) ListImports(ctx context.Context, tenantID uuid.UUID, filter model.ImportFilter, params model.PaginationParams) (*model.PaginatedResult[model.Import], error) {
	params.Normalize()
	where, args := importWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM imports `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedImportSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, importSelect, where, sortCol, sortDir, limitParam, offsetParam)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Import, 0, params.PageSize)
	for rows.Next() {
		item, err := scanImportScanner(rows)
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

func (r *academicRepository) UpdateImport(ctx context.Context, imp *model.Import) error {
	const query = `UPDATE imports SET
			status = $1,
			total_rows = $2,
			valid_rows = $3,
			invalid_rows = $4,
			committed_rows = $5,
			metadata = $6,
			committed_at = $7,
			updated_at = NOW()
		WHERE tenant_id = $8 AND id = $9
		RETURNING updated_at`

	return r.db.QueryRow(ctx, query,
		imp.Status,
		imp.TotalRows,
		imp.ValidRows,
		imp.InvalidRows,
		imp.CommittedRows,
		mustJSON(imp.Metadata),
		imp.CommittedAt,
		imp.TenantID,
		imp.ID,
	).Scan(&imp.UpdatedAt)
}

func (r *academicRepository) CreateImportErrors(ctx context.Context, importErrors []model.ImportError) error {
	const query = `INSERT INTO import_errors (import_id, tenant_id, row_number, field, message, raw_data)
		VALUES ($1, $2, $3, $4, $5, $6)`

	for i := range importErrors {
		if _, err := r.db.Exec(ctx, query,
			importErrors[i].ImportID,
			importErrors[i].TenantID,
			importErrors[i].RowNumber,
			importErrors[i].Field,
			importErrors[i].Message,
			mustJSON(importErrors[i].RawData),
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *academicRepository) ListImportErrors(ctx context.Context, tenantID, importID uuid.UUID) ([]model.ImportError, error) {
	const query = `SELECT id, import_id, tenant_id, row_number, field, message, raw_data, created_at
		FROM import_errors
		WHERE tenant_id = $1 AND import_id = $2
		ORDER BY row_number ASC, field ASC`

	rows, err := r.db.Query(ctx, query, tenantID, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ImportError{}
	for rows.Next() {
		var item model.ImportError
		var rawData []byte
		if err := rows.Scan(&item.ID, &item.ImportID, &item.TenantID, &item.RowNumber, &item.Field, &item.Message, &rawData, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.RawData = parseJSON(rawData)
		items = append(items, item)
	}
	return items, rows.Err()
}

const academicYearSelect = `SELECT id, tenant_id, name, code, start_date, end_date, status, is_active, metadata, created_at, updated_at, deleted_at FROM academic_years`

const classSelect = `SELECT id, tenant_id, name, code, sort_order, status, metadata, created_at, updated_at, deleted_at FROM classes`

const sectionSelect = `SELECT
		s.id, s.tenant_id, s.academic_year_id, s.class_id, s.branch_id, s.name, s.code, s.capacity,
		s.status, s.metadata, s.created_at, s.updated_at, s.deleted_at,
		ay.id, ay.name, ay.code,
		c.id, c.name, c.code
	FROM sections s
	JOIN academic_years ay ON ay.tenant_id = s.tenant_id AND ay.id = s.academic_year_id
	JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id`

const studentSelect = `SELECT
		s.id, s.tenant_id, s.academic_year_id, s.class_id, s.section_id, s.branch_id,
		s.admission_number, s.first_name, s.last_name, s.roll_number, s.status, s.category,
		s.phone, s.email, s.address_line1, s.address_line2, s.city, s.state, s.postal_code, s.country,
		s.opening_balance_paise, s.currency, s.metadata, s.created_at, s.updated_at, s.deleted_at,
		ay.id, ay.name, ay.code,
		c.id, c.name, c.code,
		sec.id, sec.name, sec.code
	FROM students s
	JOIN academic_years ay ON ay.tenant_id = s.tenant_id AND ay.id = s.academic_year_id
	JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id
	JOIN sections sec ON sec.tenant_id = s.tenant_id AND sec.id = s.section_id`

const guardianSelect = `SELECT
		g.id, g.tenant_id, g.name, g.relationship, g.phone, g.whatsapp_phone, g.email,
		g.preferred_language, g.communication_opt_in, g.opt_in_whatsapp,
		g.address_line1, g.address_line2, g.city, g.state, g.postal_code, g.country,
		g.user_id, u.status, g.metadata, g.created_at, g.updated_at, g.deleted_at
	FROM guardians g
	LEFT JOIN users u ON u.id = g.user_id AND u.deleted_at IS NULL`

const importSelect = `SELECT
		id, tenant_id, import_type, status, source_filename, total_rows, valid_rows,
		invalid_rows, committed_rows, created_by, metadata, committed_at, created_at, updated_at
	FROM imports`

func (r *academicRepository) scanAcademicYear(ctx context.Context, query string, args ...any) (*model.AcademicYear, error) {
	item, err := scanAcademicYearScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *academicRepository) scanClass(ctx context.Context, query string, args ...any) (*model.Class, error) {
	item, err := scanClassScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *academicRepository) scanSection(ctx context.Context, query string, args ...any) (*model.Section, error) {
	item, err := scanSectionScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *academicRepository) scanStudent(ctx context.Context, query string, args ...any) (*model.Student, error) {
	item, err := scanStudentScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *academicRepository) scanGuardian(ctx context.Context, query string, args ...any) (*model.Guardian, error) {
	item, err := scanGuardianScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *academicRepository) scanImport(ctx context.Context, query string, args ...any) (*model.Import, error) {
	item, err := scanImportScanner(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func scanAcademicYearScanner(row rowScanner) (*model.AcademicYear, error) {
	var item model.AcademicYear
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Code,
		&item.StartDate,
		&item.EndDate,
		&item.Status,
		&item.IsActive,
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

func scanClassScanner(row rowScanner) (*model.Class, error) {
	var item model.Class
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Code,
		&item.SortOrder,
		&item.Status,
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

func scanSectionScanner(row rowScanner) (*model.Section, error) {
	var item model.Section
	var metadata []byte
	academicYear := model.AcademicYear{}
	class := model.Class{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AcademicYearID,
		&item.ClassID,
		&item.BranchID,
		&item.Name,
		&item.Code,
		&item.Capacity,
		&item.Status,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
		&academicYear.ID,
		&academicYear.Name,
		&academicYear.Code,
		&class.ID,
		&class.Name,
		&class.Code,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	item.AcademicYear = &academicYear
	item.Class = &class
	return &item, nil
}

func scanStudentScanner(row rowScanner) (*model.Student, error) {
	var item model.Student
	var metadata []byte
	academicYear := model.AcademicYear{}
	class := model.Class{}
	section := model.Section{}
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AcademicYearID,
		&item.ClassID,
		&item.SectionID,
		&item.BranchID,
		&item.AdmissionNumber,
		&item.FirstName,
		&item.LastName,
		&item.RollNumber,
		&item.Status,
		&item.Category,
		&item.Phone,
		&item.Email,
		&item.Address.Line1,
		&item.Address.Line2,
		&item.Address.City,
		&item.Address.State,
		&item.Address.PostalCode,
		&item.Address.Country,
		&item.OpeningBalancePaise,
		&item.Currency,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
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
	item.AcademicYear = &academicYear
	item.Class = &class
	item.Section = &section
	return &item, nil
}

func scanGuardianScanner(row rowScanner) (*model.Guardian, error) {
	var item model.Guardian
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Relationship,
		&item.Phone,
		&item.WhatsAppPhone,
		&item.Email,
		&item.PreferredLanguage,
		&item.CommunicationOptIn,
		&item.OptInWhatsApp,
		&item.Address.Line1,
		&item.Address.Line2,
		&item.Address.City,
		&item.Address.State,
		&item.Address.PostalCode,
		&item.Address.Country,
		&item.UserID,
		&item.UserStatus,
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

func scanStudentGuardianRow(rows pgx.Rows) (*model.StudentGuardian, error) {
	var link model.StudentGuardian
	guardian := model.Guardian{}
	var metadata []byte
	if err := rows.Scan(
		&link.TenantID,
		&link.StudentID,
		&link.GuardianID,
		&link.Relationship,
		&link.IsPrimary,
		&link.CreatedAt,
		&guardian.ID,
		&guardian.TenantID,
		&guardian.Name,
		&guardian.Relationship,
		&guardian.Phone,
		&guardian.WhatsAppPhone,
		&guardian.Email,
		&guardian.PreferredLanguage,
		&guardian.CommunicationOptIn,
		&guardian.Address.Line1,
		&guardian.Address.Line2,
		&guardian.Address.City,
		&guardian.Address.State,
		&guardian.Address.PostalCode,
		&guardian.Address.Country,
		&guardian.UserID,
		&metadata,
		&guardian.CreatedAt,
		&guardian.UpdatedAt,
		&guardian.DeletedAt,
	); err != nil {
		return nil, err
	}
	guardian.Metadata = parseJSON(metadata)
	link.Guardian = &guardian
	return &link, nil
}

func scanGuardianStudentRow(rows pgx.Rows) (*model.GuardianStudent, error) {
	var item model.GuardianStudent
	if err := rows.Scan(
		&item.GuardianID,
		&item.StudentID,
		&item.AdmissionNumber,
		&item.FirstName,
		&item.LastName,
		&item.Relationship,
		&item.IsPrimary,
		&item.ClassName,
		&item.SectionName,
		&item.Status,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanImportScanner(row rowScanner) (*model.Import, error) {
	var item model.Import
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ImportType,
		&item.Status,
		&item.SourceFilename,
		&item.TotalRows,
		&item.ValidRows,
		&item.InvalidRows,
		&item.CommittedRows,
		&item.CreatedBy,
		&metadata,
		&item.CommittedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func academicYearWhere(tenantID uuid.UUID, filter model.AcademicYearFilter) (string, []any) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(name) LIKE $%d OR lower(code) LIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func classWhere(tenantID uuid.UUID, filter model.ClassFilter) (string, []any) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(name) LIKE $%d OR lower(code) LIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func sectionWhere(tenantID uuid.UUID, filter model.SectionFilter) (string, []any) {
	clauses := []string{"s.tenant_id = $1", "s.deleted_at IS NULL"}
	args := []any{tenantID}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("s.academic_year_id = $%d", len(args)))
	}
	if filter.ClassID != nil {
		args = append(args, *filter.ClassID)
		clauses = append(clauses, fmt.Sprintf("s.class_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(s.name) LIKE $%d OR lower(s.code) LIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func studentWhere(tenantID uuid.UUID, filter model.StudentFilter) (string, []any) {
	clauses := []string{"s.tenant_id = $1", "s.deleted_at IS NULL"}
	args := []any{tenantID}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("s.academic_year_id = $%d", len(args)))
	}
	if filter.ClassID != nil {
		args = append(args, *filter.ClassID)
		clauses = append(clauses, fmt.Sprintf("s.class_id = $%d", len(args)))
	}
	if filter.SectionID != nil {
		args = append(args, *filter.SectionID)
		clauses = append(clauses, fmt.Sprintf("s.section_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(s.first_name) LIKE $%d OR lower(s.last_name) LIKE $%d OR lower(s.admission_number) LIKE $%d)", len(args), len(args), len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func guardianWhere(tenantID uuid.UUID, filter model.GuardianFilter) (string, []any) {
	clauses := []string{"g.tenant_id = $1", "g.deleted_at IS NULL"}
	args := []any{tenantID}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		clauses = append(clauses, fmt.Sprintf("(lower(g.name) LIKE $%d OR lower(coalesce(g.email, '')) LIKE $%d OR coalesce(g.phone, '') LIKE $%d)", len(args), len(args), len(args)))
	}
	if filter.OnlyLinked {
		clauses = append(clauses, "g.user_id IS NOT NULL")
	}
	if filter.OnlyUnlinked {
		clauses = append(clauses, "g.user_id IS NULL")
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func importWhere(tenantID uuid.UUID, filter model.ImportFilter) (string, []any) {
	clauses := []string{"tenant_id = $1"}
	args := []any{tenantID}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.ImportType) != "" {
		args = append(args, strings.TrimSpace(filter.ImportType))
		clauses = append(clauses, fmt.Sprintf("import_type = $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func sanitizeSort(params model.PaginationParams, allowed map[string]bool, fallbackCol, fallbackDir string) (string, string) {
	sortCol := fallbackCol
	if allowed[params.SortBy] {
		sortCol = params.SortBy
	}

	sortDir := fallbackDir
	if strings.EqualFold(params.SortDir, "asc") {
		sortDir = "ASC"
	}
	if strings.EqualFold(params.SortDir, "desc") {
		sortDir = "DESC"
	}
	return sortCol, sortDir
}

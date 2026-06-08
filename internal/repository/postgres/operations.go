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

var _ repository.OperationsRepository = (*operationsRepository)(nil)

const reminderJobType = "reminder.send"

type operationsRepository struct {
	db database.DBTX
}

func NewOperationsRepository(db database.DBTX) repository.OperationsRepository {
	return &operationsRepository{db: db}
}

var (
	allowedReminderTemplateSortColumns = map[string]bool{"created_at": true, "updated_at": true, "name": true, "code": true, "channel": true, "status": true}
	allowedReminderRuleSortColumns     = map[string]bool{"created_at": true, "updated_at": true, "name": true, "code": true, "channel": true, "trigger_type": true, "status": true}
	allowedReminderLogSortColumns      = map[string]bool{"created_at": true, "scheduled_for": true, "attempted_at": true, "sent_at": true, "status": true, "channel": true}
	allowedExportJobSortColumns        = map[string]bool{"created_at": true, "updated_at": true, "completed_at": true, "status": true, "export_type": true}
)

func (r *operationsRepository) CreateReminderTemplate(ctx context.Context, template *model.ReminderTemplate) error {
	const query = `INSERT INTO reminder_templates (
			tenant_id, name, code, channel, subject, body, tone, status, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		template.TenantID,
		template.Name,
		template.Code,
		template.Channel,
		template.Subject,
		template.Body,
		template.Tone,
		template.Status,
		mustJSON(template.Metadata),
	).Scan(&template.ID, &template.CreatedAt, &template.UpdatedAt)
}

func (r *operationsRepository) GetReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderTemplate, error) {
	return r.scanReminderTemplate(ctx, reminderTemplateSelect+` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
}

func (r *operationsRepository) GetReminderTemplateByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.ReminderTemplate, error) {
	return r.scanReminderTemplate(ctx, reminderTemplateSelect+` WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL`, tenantID, code)
}

func (r *operationsRepository) ListReminderTemplates(ctx context.Context, tenantID uuid.UUID, filter model.ReminderTemplateFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderTemplate], error) {
	params.Normalize()
	where, args := reminderTemplateWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reminder_templates `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedReminderTemplateSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, reminderTemplateSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ReminderTemplate, 0, params.PageSize)
	for rows.Next() {
		item, err := scanReminderTemplateScanner(rows)
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

func (r *operationsRepository) UpdateReminderTemplate(ctx context.Context, template *model.ReminderTemplate) error {
	const query = `UPDATE reminder_templates SET
			name = $1,
			code = $2,
			channel = $3,
			subject = $4,
			body = $5,
			tone = $6,
			status = $7,
			metadata = $8,
			updated_at = NOW()
		WHERE tenant_id = $9 AND id = $10 AND deleted_at IS NULL
		RETURNING updated_at`
	return r.db.QueryRow(ctx, query,
		template.Name,
		template.Code,
		template.Channel,
		template.Subject,
		template.Body,
		template.Tone,
		template.Status,
		mustJSON(template.Metadata),
		template.TenantID,
		template.ID,
	).Scan(&template.UpdatedAt)
}

func (r *operationsRepository) SoftDeleteReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE reminder_templates SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *operationsRepository) CreateReminderRule(ctx context.Context, rule *model.ReminderRule) error {
	const query = `INSERT INTO reminder_rules (
			tenant_id, template_id, name, code, channel, trigger_type, offset_days,
			target_statuses, status, max_attempts, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		rule.TenantID,
		rule.TemplateID,
		rule.Name,
		rule.Code,
		rule.Channel,
		rule.TriggerType,
		rule.OffsetDays,
		rule.TargetStatuses,
		rule.Status,
		rule.MaxAttempts,
		mustJSON(rule.Metadata),
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *operationsRepository) GetReminderRule(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderRule, error) {
	rule, err := r.scanReminderRule(ctx, reminderRuleSelect+` WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil || rule == nil || rule.TemplateID == nil {
		return rule, err
	}
	template, err := r.GetReminderTemplate(ctx, tenantID, *rule.TemplateID)
	if err != nil {
		return nil, err
	}
	rule.Template = template
	return rule, nil
}

func (r *operationsRepository) GetReminderRuleByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.ReminderRule, error) {
	return r.scanReminderRule(ctx, reminderRuleSelect+` WHERE tenant_id = $1 AND lower(code) = lower($2) AND deleted_at IS NULL`, tenantID, code)
}

func (r *operationsRepository) ListReminderRules(ctx context.Context, tenantID uuid.UUID, filter model.ReminderRuleFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderRule], error) {
	params.Normalize()
	where, args := reminderRuleWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reminder_rules `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedReminderRuleSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, reminderRuleSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ReminderRule, 0, params.PageSize)
	for rows.Next() {
		item, err := scanReminderRuleScanner(rows)
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

func (r *operationsRepository) UpdateReminderRule(ctx context.Context, rule *model.ReminderRule) error {
	const query = `UPDATE reminder_rules SET
			template_id = $1,
			name = $2,
			code = $3,
			channel = $4,
			trigger_type = $5,
			offset_days = $6,
			target_statuses = $7,
			status = $8,
			max_attempts = $9,
			metadata = $10,
			updated_at = NOW()
		WHERE tenant_id = $11 AND id = $12 AND deleted_at IS NULL
		RETURNING updated_at`
	return r.db.QueryRow(ctx, query,
		rule.TemplateID,
		rule.Name,
		rule.Code,
		rule.Channel,
		rule.TriggerType,
		rule.OffsetDays,
		rule.TargetStatuses,
		rule.Status,
		rule.MaxAttempts,
		mustJSON(rule.Metadata),
		rule.TenantID,
		rule.ID,
	).Scan(&rule.UpdatedAt)
}

func (r *operationsRepository) SoftDeleteReminderRule(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE reminder_rules SET deleted_at = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	return err
}

func (r *operationsRepository) ListReminderCandidates(ctx context.Context, tenantID uuid.UUID, filter model.ReminderCandidateFilter) ([]model.ReminderCandidate, error) {
	clauses := []string{
		"i.tenant_id = $1",
		"i.balance_amount_paise > 0",
		"i.status NOT IN ('paid', 'cancelled', 'void')",
		"s.deleted_at IS NULL",
	}
	args := []any{tenantID}
	if len(filter.InvoiceIDs) > 0 {
		args = append(args, filter.InvoiceIDs)
		clauses = append(clauses, fmt.Sprintf("i.id = ANY($%d)", len(args)))
	}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("i.student_id = $%d", len(args)))
	}
	if filter.ClassID != nil {
		args = append(args, *filter.ClassID)
		clauses = append(clauses, fmt.Sprintf("i.class_id = $%d", len(args)))
	}
	if filter.SectionID != nil {
		args = append(args, *filter.SectionID)
		clauses = append(clauses, fmt.Sprintf("i.section_id = $%d", len(args)))
	}
	if filter.AcademicYearID != nil {
		args = append(args, *filter.AcademicYearID)
		clauses = append(clauses, fmt.Sprintf("i.academic_year_id = $%d", len(args)))
	}
	if filter.DueOnOrBefore != nil {
		args = append(args, *filter.DueOnOrBefore)
		clauses = append(clauses, fmt.Sprintf("i.due_date <= $%d", len(args)))
	}
	if len(filter.Statuses) > 0 {
		args = append(args, filter.Statuses)
		clauses = append(clauses, fmt.Sprintf("i.status = ANY($%d)", len(args)))
	}

	query := `SELECT
			i.id,
			i.invoice_number,
			s.id,
			s.admission_number,
			s.first_name,
			s.last_name,
			s.email,
			s.phone,
			g.guardian_id,
			COALESCE(g.name, ''),
			g.email,
			g.phone,
			g.whatsapp_phone,
			c.name,
			sec.name,
			i.due_date,
			i.balance_amount_paise,
			i.currency
		FROM invoices i
		JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id
		JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
		JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id
		LEFT JOIN LATERAL (
			SELECT sg.guardian_id, gu.name, gu.email, gu.phone, gu.whatsapp_phone
			FROM student_guardians sg
			JOIN guardians gu ON gu.tenant_id = sg.tenant_id AND gu.id = sg.guardian_id AND gu.deleted_at IS NULL
			WHERE sg.tenant_id = i.tenant_id AND sg.student_id = i.student_id
			ORDER BY sg.is_primary DESC, sg.created_at ASC
			LIMIT 1
		) g ON TRUE
		WHERE ` + strings.Join(clauses, " AND ") + `
		ORDER BY i.due_date ASC, s.admission_number ASC
		LIMIT 1000`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ReminderCandidate{}
	for rows.Next() {
		var item model.ReminderCandidate
		if err := rows.Scan(
			&item.InvoiceID,
			&item.InvoiceNumber,
			&item.StudentID,
			&item.AdmissionNumber,
			&item.StudentFirstName,
			&item.StudentLastName,
			&item.StudentEmail,
			&item.StudentPhone,
			&item.GuardianID,
			&item.GuardianName,
			&item.GuardianEmail,
			&item.GuardianPhone,
			&item.GuardianWhatsApp,
			&item.ClassName,
			&item.SectionName,
			&item.DueDate,
			&item.BalanceAmountPaise,
			&item.Currency,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *operationsRepository) CreateJob(ctx context.Context, job *model.Job) error {
	const query = `INSERT INTO jobs (
			tenant_id, job_type, status, priority, run_at, attempts, max_attempts,
			locked_by, idempotency_key, payload, last_error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO UPDATE
		SET updated_at = jobs.updated_at
		RETURNING id, status, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		job.TenantID,
		job.JobType,
		job.Status,
		job.Priority,
		job.RunAt,
		job.Attempts,
		job.MaxAttempts,
		job.LockedBy,
		job.IdempotencyKey,
		mustJSON(job.Payload),
		job.LastError,
	).Scan(&job.ID, &job.Status, &job.CreatedAt, &job.UpdatedAt)
}

func (r *operationsRepository) GetJob(ctx context.Context, tenantID, id uuid.UUID) (*model.Job, error) {
	return r.scanJob(ctx, jobSelect+` WHERE tenant_id = $1 AND id = $2`, tenantID, id)
}

func (r *operationsRepository) ListTenantsWithDueJobs(ctx context.Context, jobType string, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT tenant_id
		FROM jobs
		WHERE job_type = $1
		  AND status IN ('queued', 'failed')
		  AND run_at <= NOW()
		  AND attempts < max_attempts
		ORDER BY tenant_id
		LIMIT $2`, jobType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenantIDs := []uuid.UUID{}
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	return tenantIDs, rows.Err()
}

func (r *operationsRepository) ClaimDueJobs(ctx context.Context, tenantID uuid.UUID, jobType, workerID string, limit int) ([]model.Job, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `WITH candidate AS (
			SELECT id
			FROM jobs
			WHERE tenant_id = $1
			  AND job_type = $2
			  AND status IN ('queued', 'failed')
			  AND run_at <= NOW()
			  AND attempts < max_attempts
			ORDER BY priority DESC, run_at ASC, created_at ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		),
		claimed AS (
			UPDATE jobs j
			SET status = 'running',
				locked_at = NOW(),
				locked_by = $4,
				attempts = j.attempts + 1,
				updated_at = NOW()
			FROM candidate c
			WHERE j.id = c.id
			RETURNING j.id, j.tenant_id, j.job_type, j.status, j.priority, j.run_at,
				j.attempts, j.max_attempts, j.locked_at, j.locked_by, j.idempotency_key,
				j.payload, j.last_error, j.created_at, j.updated_at
		)
		SELECT id, tenant_id, job_type, status, priority, run_at, attempts, max_attempts,
			locked_at, locked_by, idempotency_key, payload, last_error, created_at, updated_at
		FROM claimed
		ORDER BY priority DESC, run_at ASC, created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, jobType, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Job{}
	for rows.Next() {
		item, err := scanJobScanner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *operationsRepository) MarkJobSucceeded(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE jobs SET status = 'succeeded', locked_at = NULL, locked_by = '', last_error = '', updated_at = NOW() WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}

func (r *operationsRepository) MarkJobFailed(ctx context.Context, tenantID, id uuid.UUID, lastError string, retryAt *time.Time) error {
	const query = `UPDATE jobs
		SET status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'failed' END,
			run_at = COALESCE($3, run_at),
			locked_at = NULL,
			locked_by = '',
			last_error = $4,
			updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2`
	_, err := r.db.Exec(ctx, query, tenantID, id, retryAt, lastError)
	return err
}

func (r *operationsRepository) CreateReminderLog(ctx context.Context, log *model.ReminderLog) error {
	const query = `INSERT INTO reminder_logs (
			id, tenant_id, rule_id, template_id, job_id, invoice_id, student_id, guardian_id,
			channel, recipient, subject, message, status, provider, provider_message_id,
			provider_response, error_message, scheduled_for, attempted_at, sent_at,
			attempt_count, created_by, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) RETURNING created_at, updated_at`
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query,
		log.ID,
		log.TenantID,
		log.RuleID,
		log.TemplateID,
		log.JobID,
		log.InvoiceID,
		log.StudentID,
		log.GuardianID,
		log.Channel,
		log.Recipient,
		log.Subject,
		log.Message,
		log.Status,
		log.Provider,
		log.ProviderMessageID,
		mustJSON(log.ProviderResponse),
		log.ErrorMessage,
		log.ScheduledFor,
		log.AttemptedAt,
		log.SentAt,
		log.AttemptCount,
		log.CreatedBy,
		mustJSON(log.Metadata),
	).Scan(&log.CreatedAt, &log.UpdatedAt)
}

func (r *operationsRepository) GetReminderLog(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderLog, error) {
	return r.scanReminderLog(ctx, reminderLogSelect+` WHERE rl.tenant_id = $1 AND rl.id = $2`, tenantID, id)
}

func (r *operationsRepository) ListReminderLogs(ctx context.Context, tenantID uuid.UUID, filter model.ReminderLogFilter, params model.PaginationParams) (*model.PaginatedResult[model.ReminderLog], error) {
	params.Normalize()
	where, args := reminderLogWhere(tenantID, filter)

	var total int64
	countQuery := `SELECT COUNT(*)
		FROM reminder_logs rl
		JOIN students s ON s.tenant_id = rl.tenant_id AND s.id = rl.student_id
		LEFT JOIN guardians g ON g.tenant_id = rl.tenant_id AND g.id = rl.guardian_id
		LEFT JOIN invoices i ON i.tenant_id = rl.tenant_id AND i.id = rl.invoice_id ` + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedReminderLogSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY rl.%s %s LIMIT %s OFFSET %s`, reminderLogSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ReminderLog, 0, params.PageSize)
	for rows.Next() {
		item, err := scanReminderLogScanner(rows)
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

func (r *operationsRepository) UpdateReminderLogDelivery(ctx context.Context, log *model.ReminderLog) error {
	const query = `UPDATE reminder_logs SET
			status = $1,
			provider = $2,
			provider_message_id = $3,
			provider_response = $4,
			error_message = $5,
			attempted_at = $6,
			sent_at = $7,
			attempt_count = $8,
			updated_at = NOW()
		WHERE tenant_id = $9 AND id = $10
		RETURNING updated_at`
	return r.db.QueryRow(ctx, query,
		log.Status,
		log.Provider,
		log.ProviderMessageID,
		mustJSON(log.ProviderResponse),
		log.ErrorMessage,
		log.AttemptedAt,
		log.SentAt,
		log.AttemptCount,
		log.TenantID,
		log.ID,
	).Scan(&log.UpdatedAt)
}

func (r *operationsRepository) CreateNotificationLog(ctx context.Context, log *model.NotificationLog) error {
	const query = `INSERT INTO notification_logs (
			tenant_id, reminder_log_id, channel, recipient, provider, status,
			provider_message_id, provider_response, error_message, attempted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	return r.db.QueryRow(ctx, query,
		log.TenantID,
		log.ReminderLogID,
		log.Channel,
		log.Recipient,
		log.Provider,
		log.Status,
		log.ProviderMessageID,
		mustJSON(log.ProviderResponse),
		log.ErrorMessage,
		log.AttemptedAt,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *operationsRepository) GetDashboardSummary(ctx context.Context, tenantID uuid.UUID, asOf time.Time, recentLimit int) (*model.DashboardSummary, error) {
	if recentLimit <= 0 {
		recentLimit = 10
	}
	startOfDay := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())
	startOfMonth := time.Date(asOf.Year(), asOf.Month(), 1, 0, 0, 0, 0, asOf.Location())
	nextDay := startOfDay.AddDate(0, 0, 1)
	nextMonth := startOfMonth.AddDate(0, 1, 0)

	summary := &model.DashboardSummary{Currency: "INR"}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_students,
			COUNT(*) FILTER (WHERE status = 'active') AS active_students
		FROM students
		WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&summary.TotalStudents, &summary.ActiveStudents); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount_paise) FILTER (WHERE COALESCE(paid_at, created_at) >= $2 AND COALESCE(paid_at, created_at) < $3), 0),
			COALESCE(SUM(amount_paise) FILTER (WHERE COALESCE(paid_at, created_at) >= $4 AND COALESCE(paid_at, created_at) < $5), 0)
		FROM payments
		WHERE tenant_id = $1 AND status IN ('success', 'manually_verified', 'settled')`,
		tenantID, startOfDay, nextDay, startOfMonth, nextMonth,
	).Scan(&summary.TodayCollectionPaise, &summary.MonthCollectionPaise); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(balance_amount_paise), 0),
			COALESCE(SUM(balance_amount_paise) FILTER (WHERE due_date < $2), 0),
			COUNT(DISTINCT student_id) FILTER (WHERE due_date < $2),
			COUNT(*)
		FROM invoices
		WHERE tenant_id = $1
		  AND balance_amount_paise > 0
		  AND status NOT IN ('paid', 'cancelled', 'void')`,
		tenantID, asOf,
	).Scan(&summary.TotalDuePaise, &summary.OverduePaise, &summary.DefaulterCount, &summary.UnpaidInvoiceCount); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT payment_method, COUNT(*), COALESCE(SUM(amount_paise), 0)
		FROM payments
		WHERE tenant_id = $1
		  AND status IN ('success', 'manually_verified', 'settled')
		  AND COALESCE(paid_at, created_at) >= $2
		  AND COALESCE(paid_at, created_at) < $3
		GROUP BY payment_method
		ORDER BY payment_method`, tenantID, startOfMonth, nextMonth)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.PaymentMethodSummary
		if err := rows.Scan(&item.PaymentMethod, &item.PaymentCount, &item.AmountPaise); err != nil {
			rows.Close()
			return nil, err
		}
		summary.PaymentMethodBreakdown = append(summary.PaymentMethodBreakdown, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	eventRows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, payment_id, attempt_id, receipt_id, student_id, event_type,
			status, amount_paise, message, metadata, occurred_at, created_at
		FROM payment_events
		WHERE tenant_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2`, tenantID, recentLimit)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()
	for eventRows.Next() {
		event, err := scanPaymentEventBare(eventRows)
		if err != nil {
			return nil, err
		}
		summary.RecentPaymentEvents = append(summary.RecentPaymentEvents, *event)
	}
	return summary, eventRows.Err()
}

func (r *operationsRepository) ListCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.CollectionReportRow], error) {
	params.Normalize()
	where, args := collectionReportWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*)
		FROM payments p
		JOIN students s ON s.tenant_id = p.tenant_id AND s.id = p.student_id
		JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id
		JOIN sections sec ON sec.tenant_id = s.tenant_id AND sec.id = s.section_id
		LEFT JOIN receipts r ON r.tenant_id = p.tenant_id AND r.payment_id = p.id `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	query := `SELECT
			p.id,
			r.id,
			COALESCE(r.receipt_number, ''),
			s.id,
			s.admission_number,
			trim(s.first_name || ' ' || s.last_name),
			c.name,
			sec.name,
			p.payment_method,
			p.provider,
			p.amount_paise,
			p.amount_applied_paise,
			COALESCE(p.paid_at, p.created_at),
			r.issued_at
		FROM payments p
		JOIN students s ON s.tenant_id = p.tenant_id AND s.id = p.student_id
		JOIN classes c ON c.tenant_id = s.tenant_id AND c.id = s.class_id
		JOIN sections sec ON sec.tenant_id = s.tenant_id AND sec.id = s.section_id
		LEFT JOIN receipts r ON r.tenant_id = p.tenant_id AND r.payment_id = p.id ` + where + `
		ORDER BY COALESCE(p.paid_at, p.created_at) DESC, p.created_at DESC
		LIMIT ` + limitParam + ` OFFSET ` + offsetParam

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CollectionReportRow, 0, params.PageSize)
	for rows.Next() {
		var item model.CollectionReportRow
		if err := rows.Scan(
			&item.PaymentID,
			&item.ReceiptID,
			&item.ReceiptNumber,
			&item.StudentID,
			&item.AdmissionNumber,
			&item.StudentName,
			&item.ClassName,
			&item.SectionName,
			&item.PaymentMethod,
			&item.Provider,
			&item.AmountPaise,
			&item.AmountAppliedPaise,
			&item.PaidAt,
			&item.ReceiptIssuedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *operationsRepository) ListDefaulterReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.DefaulterReportRow], error) {
	params.Normalize()
	where, args := defaulterReportWhere(tenantID, filter)
	groupQuery := defaulterReportSelect + where + defaulterReportGroup

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+groupQuery+`) counted`, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	query := groupQuery + ` ORDER BY oldest_due_date ASC, student_name ASC LIMIT ` + limitParam + ` OFFSET ` + offsetParam
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DefaulterReportRow, 0, params.PageSize)
	for rows.Next() {
		var item model.DefaulterReportRow
		if err := rows.Scan(
			&item.StudentID,
			&item.AdmissionNumber,
			&item.StudentName,
			&item.ClassID,
			&item.ClassName,
			&item.SectionID,
			&item.SectionName,
			&item.GuardianName,
			&item.GuardianPhone,
			&item.GuardianEmail,
			&item.InvoiceCount,
			&item.TotalDuePaise,
			&item.OverduePaise,
			&item.OldestDueDate,
			&item.LastReminderAt,
			&item.LastReminderStatus,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *operationsRepository) ListDueReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.DueReportRow], error) {
	params.Normalize()
	where, args := dueReportWhere(tenantID, filter)
	groupQuery := dueReportSelect + where + ` GROUP BY c.id, c.name, sec.id, sec.name`

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+groupQuery+`) counted`, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	query := groupQuery + ` ORDER BY c.name ASC, sec.name ASC LIMIT ` + limitParam + ` OFFSET ` + offsetParam
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DueReportRow, 0, params.PageSize)
	for rows.Next() {
		var item model.DueReportRow
		if err := rows.Scan(
			&item.ClassID,
			&item.ClassName,
			&item.SectionID,
			&item.SectionName,
			&item.StudentCount,
			&item.InvoiceCount,
			&item.TotalBilledPaise,
			&item.TotalPaidPaise,
			&item.TotalDuePaise,
			&item.OverduePaise,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *operationsRepository) ListFeeHeadCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.FeeHeadCollectionRow], error) {
	params.Normalize()
	where, args := feeHeadCollectionWhere(tenantID, filter)
	groupQuery := feeHeadCollectionSelect + where + ` GROUP BY fh.id, fh.name, fh.code, fh.category`

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+groupQuery+`) counted`, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	query := groupQuery + ` ORDER BY fh.name ASC LIMIT ` + limitParam + ` OFFSET ` + offsetParam
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.FeeHeadCollectionRow, 0, params.PageSize)
	for rows.Next() {
		var item model.FeeHeadCollectionRow
		if err := rows.Scan(&item.FeeHeadID, &item.FeeHeadName, &item.FeeHeadCode, &item.Category, &item.CollectedPaise, &item.InvoiceCount, &item.PaymentCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *operationsRepository) ListPaymentMethodReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[model.PaymentMethodReportRow], error) {
	params.Normalize()
	where, args := paymentMethodReportWhere(tenantID, filter)
	groupQuery := `SELECT p.payment_method, p.provider, COUNT(*), COALESCE(SUM(p.amount_paise), 0)
		FROM payments p ` + where + ` GROUP BY p.payment_method, p.provider`

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM (`+groupQuery+`) counted`, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())
	query := groupQuery + ` ORDER BY p.payment_method ASC, p.provider ASC LIMIT ` + limitParam + ` OFFSET ` + offsetParam
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PaymentMethodReportRow, 0, params.PageSize)
	for rows.Next() {
		var item model.PaymentMethodReportRow
		if err := rows.Scan(&item.PaymentMethod, &item.Provider, &item.PaymentCount, &item.AmountPaise); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return model.NewPaginatedResult(items, total, params.Page, params.PageSize), nil
}

func (r *operationsRepository) CreateExportJob(ctx context.Context, job *model.ExportJob) error {
	const query = `INSERT INTO export_jobs (
			tenant_id, export_type, status, format, params, file_name, content_type,
			content, row_count, requested_by, completed_at, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		job.TenantID,
		job.ExportType,
		job.Status,
		job.Format,
		mustJSON(job.Params),
		job.FileName,
		job.ContentType,
		job.Content,
		job.RowCount,
		job.RequestedBy,
		job.CompletedAt,
		job.ErrorMessage,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *operationsRepository) GetExportJob(ctx context.Context, tenantID, id uuid.UUID) (*model.ExportJob, error) {
	return r.scanExportJob(ctx, exportJobSelect+` WHERE tenant_id = $1 AND id = $2`, tenantID, id)
}

func (r *operationsRepository) ListExportJobs(ctx context.Context, tenantID uuid.UUID, filter model.ExportJobFilter, params model.PaginationParams) (*model.PaginatedResult[model.ExportJob], error) {
	params.Normalize()
	where, args := exportJobWhere(tenantID, filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM export_jobs `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	queryArgs := append([]any{}, args...)
	limitParam := fmt.Sprintf("$%d", len(queryArgs)+1)
	offsetParam := fmt.Sprintf("$%d", len(queryArgs)+2)
	queryArgs = append(queryArgs, params.PageSize, params.Offset())

	sortCol, sortDir := sanitizeSort(params, allowedExportJobSortColumns, "created_at", "DESC")
	query := fmt.Sprintf(`%s %s ORDER BY %s %s LIMIT %s OFFSET %s`, exportJobSelect, where, sortCol, sortDir, limitParam, offsetParam)
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ExportJob, 0, params.PageSize)
	for rows.Next() {
		item, err := scanExportJobScanner(rows)
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

func (r *operationsRepository) MarkExportJobProcessing(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE export_jobs SET status = 'processing', updated_at = NOW() WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}

func (r *operationsRepository) CompleteExportJob(ctx context.Context, job *model.ExportJob) error {
	const query = `UPDATE export_jobs SET
			status = 'succeeded',
			file_name = $1,
			content_type = $2,
			content = $3,
			row_count = $4,
			completed_at = NOW(),
			error_message = '',
			updated_at = NOW()
		WHERE tenant_id = $5 AND id = $6
		RETURNING completed_at, updated_at`
	return r.db.QueryRow(ctx, query,
		job.FileName,
		job.ContentType,
		job.Content,
		job.RowCount,
		job.TenantID,
		job.ID,
	).Scan(&job.CompletedAt, &job.UpdatedAt)
}

func (r *operationsRepository) FailExportJob(ctx context.Context, tenantID, id uuid.UUID, message string) error {
	_, err := r.db.Exec(ctx, `UPDATE export_jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`, message, tenantID, id)
	return err
}

const reminderTemplateSelect = `SELECT id, tenant_id, name, code, channel, subject, body, tone, status, metadata, created_at, updated_at, deleted_at FROM reminder_templates`

func (r *operationsRepository) scanReminderTemplate(ctx context.Context, query string, args ...any) (*model.ReminderTemplate, error) {
	item, err := scanReminderTemplateScanner(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanReminderTemplateScanner(row rowScanner) (*model.ReminderTemplate, error) {
	var item model.ReminderTemplate
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Code,
		&item.Channel,
		&item.Subject,
		&item.Body,
		&item.Tone,
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

const reminderRuleSelect = `SELECT id, tenant_id, template_id, name, code, channel, trigger_type, offset_days, target_statuses, status, max_attempts, metadata, created_at, updated_at, deleted_at FROM reminder_rules`

func (r *operationsRepository) scanReminderRule(ctx context.Context, query string, args ...any) (*model.ReminderRule, error) {
	item, err := scanReminderRuleScanner(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanReminderRuleScanner(row rowScanner) (*model.ReminderRule, error) {
	var item model.ReminderRule
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.TemplateID,
		&item.Name,
		&item.Code,
		&item.Channel,
		&item.TriggerType,
		&item.OffsetDays,
		&item.TargetStatuses,
		&item.Status,
		&item.MaxAttempts,
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

const jobSelect = `SELECT jobs.id, jobs.tenant_id, jobs.job_type, jobs.status, jobs.priority, jobs.run_at, jobs.attempts, jobs.max_attempts, jobs.locked_at, jobs.locked_by, jobs.idempotency_key, jobs.payload, jobs.last_error, jobs.created_at, jobs.updated_at FROM jobs`

func (r *operationsRepository) scanJob(ctx context.Context, query string, args ...any) (*model.Job, error) {
	item, err := scanJobScanner(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanJobScanner(row rowScanner) (*model.Job, error) {
	var item model.Job
	var payload []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.JobType,
		&item.Status,
		&item.Priority,
		&item.RunAt,
		&item.Attempts,
		&item.MaxAttempts,
		&item.LockedAt,
		&item.LockedBy,
		&item.IdempotencyKey,
		&payload,
		&item.LastError,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Payload = parseJSON(payload)
	return &item, nil
}

const reminderLogSelect = `SELECT
		rl.id,
		rl.tenant_id,
		rl.rule_id,
		rl.template_id,
		rl.job_id,
		rl.invoice_id,
		rl.student_id,
		rl.guardian_id,
		rl.channel,
		rl.recipient,
		rl.subject,
		rl.message,
		rl.status,
		rl.provider,
		rl.provider_message_id,
		rl.provider_response,
		rl.error_message,
		rl.scheduled_for,
		rl.attempted_at,
		rl.sent_at,
		rl.attempt_count,
		rl.created_by,
		rl.metadata,
		rl.created_at,
		rl.updated_at,
		s.id,
		s.admission_number,
		s.first_name,
		s.last_name,
		s.status,
		COALESCE(g.name, ''),
		COALESCE(i.invoice_number, '')
	FROM reminder_logs rl
	JOIN students s ON s.tenant_id = rl.tenant_id AND s.id = rl.student_id
	LEFT JOIN guardians g ON g.tenant_id = rl.tenant_id AND g.id = rl.guardian_id
	LEFT JOIN invoices i ON i.tenant_id = rl.tenant_id AND i.id = rl.invoice_id`

func (r *operationsRepository) scanReminderLog(ctx context.Context, query string, args ...any) (*model.ReminderLog, error) {
	item, err := scanReminderLogScanner(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanReminderLogScanner(row rowScanner) (*model.ReminderLog, error) {
	var item model.ReminderLog
	var providerResponse []byte
	var metadata []byte
	var student model.Student
	var studentStatus string
	var guardianName string
	var invoiceNumber string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.RuleID,
		&item.TemplateID,
		&item.JobID,
		&item.InvoiceID,
		&item.StudentID,
		&item.GuardianID,
		&item.Channel,
		&item.Recipient,
		&item.Subject,
		&item.Message,
		&item.Status,
		&item.Provider,
		&item.ProviderMessageID,
		&providerResponse,
		&item.ErrorMessage,
		&item.ScheduledFor,
		&item.AttemptedAt,
		&item.SentAt,
		&item.AttemptCount,
		&item.CreatedBy,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
		&student.ID,
		&student.AdmissionNumber,
		&student.FirstName,
		&student.LastName,
		&studentStatus,
		&guardianName,
		&invoiceNumber,
	); err != nil {
		return nil, err
	}
	student.Status = studentStatus
	item.Student = &student
	if guardianName != "" {
		item.Guardian = &model.Guardian{Name: guardianName}
	}
	if invoiceNumber != "" {
		item.Invoice = &model.Invoice{InvoiceNumber: invoiceNumber}
	}
	item.ProviderResponse = parseJSON(providerResponse)
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

const exportJobSelect = `SELECT id, tenant_id, export_type, status, format, params, file_name, content_type, content, row_count, requested_by, completed_at, error_message, created_at, updated_at FROM export_jobs`

func (r *operationsRepository) scanExportJob(ctx context.Context, query string, args ...any) (*model.ExportJob, error) {
	item, err := scanExportJobScanner(r.db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func scanExportJobScanner(row rowScanner) (*model.ExportJob, error) {
	var item model.ExportJob
	var params []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ExportType,
		&item.Status,
		&item.Format,
		&params,
		&item.FileName,
		&item.ContentType,
		&item.Content,
		&item.RowCount,
		&item.RequestedBy,
		&item.CompletedAt,
		&item.ErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Params = parseJSON(params)
	return &item, nil
}

func scanPaymentEventBare(row rowScanner) (*model.PaymentEvent, error) {
	var item model.PaymentEvent
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.PaymentID,
		&item.AttemptID,
		&item.ReceiptID,
		&item.StudentID,
		&item.EventType,
		&item.Status,
		&item.AmountPaise,
		&item.Message,
		&metadata,
		&item.OccurredAt,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = parseJSON(metadata)
	return &item, nil
}

func reminderTemplateWhere(tenantID uuid.UUID, filter model.ReminderTemplateFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	if strings.TrimSpace(filter.Channel) != "" {
		args = append(args, strings.TrimSpace(filter.Channel))
		clauses = append(clauses, fmt.Sprintf("channel = $%d", len(args)))
	}
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

func reminderRuleWhere(tenantID uuid.UUID, filter model.ReminderRuleFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	if strings.TrimSpace(filter.Channel) != "" {
		args = append(args, strings.TrimSpace(filter.Channel))
		clauses = append(clauses, fmt.Sprintf("channel = $%d", len(args)))
	}
	if strings.TrimSpace(filter.TriggerType) != "" {
		args = append(args, strings.TrimSpace(filter.TriggerType))
		clauses = append(clauses, fmt.Sprintf("trigger_type = $%d", len(args)))
	}
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

func reminderLogWhere(tenantID uuid.UUID, filter model.ReminderLogFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"rl.tenant_id = $1"}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("rl.student_id = $%d", len(args)))
	}
	if filter.InvoiceID != nil {
		args = append(args, *filter.InvoiceID)
		clauses = append(clauses, fmt.Sprintf("rl.invoice_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Channel) != "" {
		args = append(args, strings.TrimSpace(filter.Channel))
		clauses = append(clauses, fmt.Sprintf("rl.channel = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("rl.status = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("rl.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("rl.created_at < $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func collectionReportWhere(tenantID uuid.UUID, filter model.ReportFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"p.tenant_id = $1", "p.status IN ('success', 'manually_verified', 'settled')"}
	addReportPaymentFilters(&args, &clauses, filter)
	if filter.ClassID != nil {
		args = append(args, *filter.ClassID)
		clauses = append(clauses, fmt.Sprintf("s.class_id = $%d", len(args)))
	}
	if filter.SectionID != nil {
		args = append(args, *filter.SectionID)
		clauses = append(clauses, fmt.Sprintf("s.section_id = $%d", len(args)))
	}
	if filter.StudentID != nil {
		args = append(args, *filter.StudentID)
		clauses = append(clauses, fmt.Sprintf("p.student_id = $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func addReportPaymentFilters(args *[]any, clauses *[]string, filter model.ReportFilter) {
	if filter.From != nil {
		*args = append(*args, *filter.From)
		*clauses = append(*clauses, fmt.Sprintf("COALESCE(p.paid_at, p.created_at) >= $%d", len(*args)))
	}
	if filter.To != nil {
		*args = append(*args, *filter.To)
		*clauses = append(*clauses, fmt.Sprintf("COALESCE(p.paid_at, p.created_at) < $%d", len(*args)))
	}
	if strings.TrimSpace(filter.PaymentMethod) != "" {
		*args = append(*args, strings.TrimSpace(filter.PaymentMethod))
		*clauses = append(*clauses, fmt.Sprintf("p.payment_method = $%d", len(*args)))
	}
	if strings.TrimSpace(filter.Provider) != "" {
		*args = append(*args, strings.TrimSpace(filter.Provider))
		*clauses = append(*clauses, fmt.Sprintf("p.provider = $%d", len(*args)))
	}
}

const defaulterReportSelect = `SELECT
		s.id,
		s.admission_number,
		trim(s.first_name || ' ' || s.last_name) AS student_name,
		c.id,
		c.name,
		sec.id,
		sec.name,
		COALESCE(g.name, ''),
		COALESCE(g.phone, ''),
		COALESCE(g.email, ''),
		COUNT(DISTINCT i.id),
		COALESCE(SUM(i.balance_amount_paise), 0),
		COALESCE(SUM(i.balance_amount_paise) FILTER (WHERE i.due_date < $2), 0),
		MIN(i.due_date) AS oldest_due_date,
		lr.created_at,
		COALESCE(lr.status, '')
	FROM invoices i
	JOIN students s ON s.tenant_id = i.tenant_id AND s.id = i.student_id AND s.deleted_at IS NULL
	JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
	JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id
	LEFT JOIN LATERAL (
		SELECT gu.name, gu.phone, gu.email
		FROM student_guardians sg
		JOIN guardians gu ON gu.tenant_id = sg.tenant_id AND gu.id = sg.guardian_id AND gu.deleted_at IS NULL
		WHERE sg.tenant_id = i.tenant_id AND sg.student_id = i.student_id
		ORDER BY sg.is_primary DESC, sg.created_at ASC
		LIMIT 1
	) g ON TRUE
	LEFT JOIN LATERAL (
		SELECT created_at, status
		FROM reminder_logs
		WHERE tenant_id = i.tenant_id AND student_id = i.student_id
		ORDER BY created_at DESC
		LIMIT 1
	) lr ON TRUE `

const defaulterReportGroup = ` GROUP BY s.id, s.admission_number, s.first_name, s.last_name, c.id, c.name, sec.id, sec.name, g.name, g.phone, g.email, lr.created_at, lr.status`

func defaulterReportWhere(tenantID uuid.UUID, filter model.ReportFilter) (string, []any) {
	asOf := time.Now()
	if filter.AsOf != nil {
		asOf = *filter.AsOf
	}
	args := []any{tenantID, asOf}
	clauses := []string{
		"i.tenant_id = $1",
		"i.balance_amount_paise > 0",
		"i.due_date <= $2",
		"i.status NOT IN ('paid', 'cancelled', 'void')",
	}
	addInvoiceDimensionFilters(&args, &clauses, filter, "i")
	return "WHERE " + strings.Join(clauses, " AND "), args
}

const dueReportSelect = `SELECT
		c.id,
		c.name,
		sec.id,
		sec.name,
		COUNT(DISTINCT i.student_id),
		COUNT(i.id),
		COALESCE(SUM(i.total_amount_paise), 0),
		COALESCE(SUM(i.paid_amount_paise), 0),
		COALESCE(SUM(i.balance_amount_paise), 0),
		COALESCE(SUM(i.balance_amount_paise) FILTER (WHERE i.due_date < $2), 0)
	FROM invoices i
	JOIN classes c ON c.tenant_id = i.tenant_id AND c.id = i.class_id
	JOIN sections sec ON sec.tenant_id = i.tenant_id AND sec.id = i.section_id `

func dueReportWhere(tenantID uuid.UUID, filter model.ReportFilter) (string, []any) {
	asOf := time.Now()
	if filter.AsOf != nil {
		asOf = *filter.AsOf
	}
	args := []any{tenantID, asOf}
	clauses := []string{"i.tenant_id = $1", "i.status NOT IN ('cancelled', 'void')"}
	addInvoiceDimensionFilters(&args, &clauses, filter, "i")
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func addInvoiceDimensionFilters(args *[]any, clauses *[]string, filter model.ReportFilter, alias string) {
	if filter.AcademicYearID != nil {
		*args = append(*args, *filter.AcademicYearID)
		*clauses = append(*clauses, fmt.Sprintf("%s.academic_year_id = $%d", alias, len(*args)))
	}
	if filter.ClassID != nil {
		*args = append(*args, *filter.ClassID)
		*clauses = append(*clauses, fmt.Sprintf("%s.class_id = $%d", alias, len(*args)))
	}
	if filter.SectionID != nil {
		*args = append(*args, *filter.SectionID)
		*clauses = append(*clauses, fmt.Sprintf("%s.section_id = $%d", alias, len(*args)))
	}
	if filter.StudentID != nil {
		*args = append(*args, *filter.StudentID)
		*clauses = append(*clauses, fmt.Sprintf("%s.student_id = $%d", alias, len(*args)))
	}
}

const feeHeadCollectionSelect = `SELECT
		fh.id,
		fh.name,
		fh.code,
		fh.category,
		COALESCE(SUM(ROUND((pa.amount_paise::numeric * ii.total_amount_paise::numeric) / NULLIF(i.total_amount_paise, 0))), 0)::bigint,
		COUNT(DISTINCT i.id),
		COUNT(DISTINCT p.id)
	FROM payment_allocations pa
	JOIN payments p ON p.tenant_id = pa.tenant_id AND p.id = pa.payment_id
	JOIN invoices i ON i.tenant_id = pa.tenant_id AND i.id = pa.invoice_id
	JOIN invoice_items ii ON ii.tenant_id = i.tenant_id AND ii.invoice_id = i.id
	JOIN fee_heads fh ON fh.tenant_id = ii.tenant_id AND fh.id = ii.fee_head_id `

func feeHeadCollectionWhere(tenantID uuid.UUID, filter model.ReportFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"p.tenant_id = $1", "p.status IN ('success', 'manually_verified', 'settled')"}
	addReportPaymentFilters(&args, &clauses, filter)
	addInvoiceDimensionFilters(&args, &clauses, filter, "i")
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func paymentMethodReportWhere(tenantID uuid.UUID, filter model.ReportFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"p.tenant_id = $1", "p.status IN ('success', 'manually_verified', 'settled')"}
	addReportPaymentFilters(&args, &clauses, filter)
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func exportJobWhere(tenantID uuid.UUID, filter model.ExportJobFilter) (string, []any) {
	args := []any{tenantID}
	clauses := []string{"tenant_id = $1"}
	if strings.TrimSpace(filter.ExportType) != "" {
		args = append(args, strings.TrimSpace(filter.ExportType))
		clauses = append(clauses, fmt.Sprintf("export_type = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("created_at < $%d", len(args)))
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

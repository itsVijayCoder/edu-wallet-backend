package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository"
)

const (
	reminderJobTypeSend  = "reminder.send"
	exportContentTypeCSV = "text/csv; charset=utf-8"
)

type operationsService struct {
	repo        repository.OperationsRepository
	repoFactory repository.OperationsRepositoryFactory
	tx          database.Transactor
	auditRepo   repository.AuditRepository
	notifier    NotificationProvider
	workerID    string
}

func NewOperationsService(
	repo repository.OperationsRepository,
	repoFactory repository.OperationsRepositoryFactory,
	tx database.Transactor,
	auditRepo repository.AuditRepository,
	notifier NotificationProvider,
) OperationsService {
	if notifier == nil {
		notifier = NewNotificationProvider(nil)
	}
	return &operationsService{
		repo:        repo,
		repoFactory: repoFactory,
		tx:          tx,
		auditRepo:   auditRepo,
		notifier:    notifier,
		workerID:    "api-inline",
	}
}

func (s *operationsService) CreateReminderTemplate(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateReminderTemplateRequest) (*dto.ReminderTemplateResponse, error) {
	code, err := normalizeCode(req.Code, "INVALID_REMINDER_TEMPLATE_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetReminderTemplateByCode(ctx, tenantID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup reminder template code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}
	template := &model.ReminderTemplate{
		TenantID: tenantID,
		Name:     strings.TrimSpace(req.Name),
		Code:     code,
		Channel:  defaultString(req.Channel, "email"),
		Subject:  strings.TrimSpace(req.Subject),
		Body:     strings.TrimSpace(req.Body),
		Tone:     defaultString(req.Tone, "polite"),
		Status:   defaultString(req.Status, "active"),
		Metadata: normalizeMetadata(req.Metadata),
	}
	if err := s.repo.CreateReminderTemplate(ctx, template); err != nil {
		return nil, mapPersistenceError(err, "create reminder template")
	}
	if err := s.audit(ctx, tenantID, actorID, "reminder_template.created", "reminder_template", template.ID, "reminder template created", map[string]any{"code": template.Code}); err != nil {
		return nil, err
	}
	resp := reminderTemplateToResponse(template)
	return &resp, nil
}

func (s *operationsService) ListReminderTemplates(ctx context.Context, tenantID uuid.UUID, filter model.ReminderTemplateFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ReminderTemplateResponse], error) {
	result, err := s.repo.ListReminderTemplates(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list reminder templates: %w", err)
	}
	items := make([]dto.ReminderTemplateResponse, len(result.Data))
	for i := range result.Data {
		items[i] = reminderTemplateToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) GetReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) (*dto.ReminderTemplateResponse, error) {
	template, err := s.repo.GetReminderTemplate(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get reminder template: %w", err)
	}
	if template == nil {
		return nil, apperror.ErrNotFound
	}
	resp := reminderTemplateToResponse(template)
	return &resp, nil
}

func (s *operationsService) UpdateReminderTemplate(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateReminderTemplateRequest) (*dto.ReminderTemplateResponse, error) {
	template, err := s.repo.GetReminderTemplate(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get reminder template: %w", err)
	}
	if template == nil {
		return nil, apperror.ErrNotFound
	}
	if req.Name != nil {
		template.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_REMINDER_TEMPLATE_CODE")
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(code, template.Code) {
			existing, err := s.repo.GetReminderTemplateByCode(ctx, tenantID, code)
			if err != nil {
				return nil, fmt.Errorf("lookup reminder template code: %w", err)
			}
			if existing != nil && existing.ID != template.ID {
				return nil, apperror.ErrConflict
			}
		}
		template.Code = code
	}
	if req.Channel != nil {
		template.Channel = strings.TrimSpace(*req.Channel)
	}
	if req.Subject != nil {
		template.Subject = strings.TrimSpace(*req.Subject)
	}
	if req.Body != nil {
		template.Body = strings.TrimSpace(*req.Body)
	}
	if req.Tone != nil {
		template.Tone = strings.TrimSpace(*req.Tone)
	}
	if req.Status != nil {
		template.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		template.Metadata = normalizeMetadata(req.Metadata)
	}
	if err := s.repo.UpdateReminderTemplate(ctx, template); err != nil {
		return nil, mapPersistenceError(err, "update reminder template")
	}
	if err := s.audit(ctx, tenantID, actorID, "reminder_template.updated", "reminder_template", template.ID, "reminder template updated", map[string]any{"code": template.Code}); err != nil {
		return nil, err
	}
	resp := reminderTemplateToResponse(template)
	return &resp, nil
}

func (s *operationsService) DeleteReminderTemplate(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	template, err := s.repo.GetReminderTemplate(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get reminder template: %w", err)
	}
	if template == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteReminderTemplate(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete reminder template: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "reminder_template.deleted", "reminder_template", id, "reminder template deleted", map[string]any{"code": template.Code})
}

func (s *operationsService) CreateReminderRule(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateReminderRuleRequest) (*dto.ReminderRuleResponse, error) {
	code, err := normalizeCode(req.Code, "INVALID_REMINDER_RULE_CODE")
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetReminderRuleByCode(ctx, tenantID, code)
	if err != nil {
		return nil, fmt.Errorf("lookup reminder rule code: %w", err)
	}
	if existing != nil {
		return nil, apperror.ErrConflict
	}
	if req.TemplateID != nil {
		if _, err := s.requireReminderTemplate(ctx, tenantID, *req.TemplateID); err != nil {
			return nil, err
		}
	}
	rule := &model.ReminderRule{
		TenantID:       tenantID,
		TemplateID:     req.TemplateID,
		Name:           strings.TrimSpace(req.Name),
		Code:           code,
		Channel:        defaultString(req.Channel, "email"),
		TriggerType:    defaultString(req.TriggerType, "manual"),
		OffsetDays:     req.OffsetDays,
		TargetStatuses: normalizeReminderStatuses(req.TargetStatuses),
		Status:         defaultString(req.Status, "active"),
		MaxAttempts:    defaultInt(req.MaxAttempts, 3),
		Metadata:       normalizeMetadata(req.Metadata),
	}
	if err := s.repo.CreateReminderRule(ctx, rule); err != nil {
		return nil, mapPersistenceError(err, "create reminder rule")
	}
	refreshed, err := s.repo.GetReminderRule(ctx, tenantID, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("get reminder rule: %w", err)
	}
	if refreshed != nil {
		rule = refreshed
	}
	if err := s.audit(ctx, tenantID, actorID, "reminder_rule.created", "reminder_rule", rule.ID, "reminder rule created", map[string]any{"code": rule.Code}); err != nil {
		return nil, err
	}
	resp := reminderRuleToResponse(rule)
	return &resp, nil
}

func (s *operationsService) ListReminderRules(ctx context.Context, tenantID uuid.UUID, filter model.ReminderRuleFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ReminderRuleResponse], error) {
	result, err := s.repo.ListReminderRules(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list reminder rules: %w", err)
	}
	items := make([]dto.ReminderRuleResponse, len(result.Data))
	for i := range result.Data {
		items[i] = reminderRuleToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) GetReminderRule(ctx context.Context, tenantID, id uuid.UUID) (*dto.ReminderRuleResponse, error) {
	rule, err := s.repo.GetReminderRule(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get reminder rule: %w", err)
	}
	if rule == nil {
		return nil, apperror.ErrNotFound
	}
	resp := reminderRuleToResponse(rule)
	return &resp, nil
}

func (s *operationsService) UpdateReminderRule(ctx context.Context, actorID, tenantID, id uuid.UUID, req dto.UpdateReminderRuleRequest) (*dto.ReminderRuleResponse, error) {
	rule, err := s.repo.GetReminderRule(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get reminder rule: %w", err)
	}
	if rule == nil {
		return nil, apperror.ErrNotFound
	}
	if req.ClearTemplate {
		rule.TemplateID = nil
		rule.Template = nil
	}
	if req.TemplateID != nil {
		if _, err := s.requireReminderTemplate(ctx, tenantID, *req.TemplateID); err != nil {
			return nil, err
		}
		rule.TemplateID = req.TemplateID
	}
	if req.Name != nil {
		rule.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code, err := normalizeCode(*req.Code, "INVALID_REMINDER_RULE_CODE")
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(code, rule.Code) {
			existing, err := s.repo.GetReminderRuleByCode(ctx, tenantID, code)
			if err != nil {
				return nil, fmt.Errorf("lookup reminder rule code: %w", err)
			}
			if existing != nil && existing.ID != rule.ID {
				return nil, apperror.ErrConflict
			}
		}
		rule.Code = code
	}
	if req.Channel != nil {
		rule.Channel = strings.TrimSpace(*req.Channel)
	}
	if req.TriggerType != nil {
		rule.TriggerType = strings.TrimSpace(*req.TriggerType)
	}
	if req.OffsetDays != nil {
		rule.OffsetDays = *req.OffsetDays
	}
	if req.TargetStatuses != nil {
		rule.TargetStatuses = normalizeReminderStatuses(*req.TargetStatuses)
	}
	if req.Status != nil {
		rule.Status = strings.TrimSpace(*req.Status)
	}
	if req.MaxAttempts != nil {
		rule.MaxAttempts = *req.MaxAttempts
	}
	if req.Metadata != nil {
		rule.Metadata = normalizeMetadata(req.Metadata)
	}
	if err := s.repo.UpdateReminderRule(ctx, rule); err != nil {
		return nil, mapPersistenceError(err, "update reminder rule")
	}
	refreshed, err := s.repo.GetReminderRule(ctx, tenantID, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("get reminder rule: %w", err)
	}
	if refreshed != nil {
		rule = refreshed
	}
	if err := s.audit(ctx, tenantID, actorID, "reminder_rule.updated", "reminder_rule", rule.ID, "reminder rule updated", map[string]any{"code": rule.Code}); err != nil {
		return nil, err
	}
	resp := reminderRuleToResponse(rule)
	return &resp, nil
}

func (s *operationsService) DeleteReminderRule(ctx context.Context, actorID, tenantID, id uuid.UUID) error {
	rule, err := s.repo.GetReminderRule(ctx, tenantID, id)
	if err != nil {
		return fmt.Errorf("get reminder rule: %w", err)
	}
	if rule == nil {
		return apperror.ErrNotFound
	}
	if err := s.repo.SoftDeleteReminderRule(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete reminder rule: %w", err)
	}
	return s.audit(ctx, tenantID, actorID, "reminder_rule.deleted", "reminder_rule", id, "reminder rule deleted", map[string]any{"code": rule.Code})
}

func (s *operationsService) SendReminders(ctx context.Context, actorID, tenantID uuid.UUID, req dto.SendReminderRequest) (*dto.SendReminderResponse, error) {
	rule, template, channel, subject, body, err := s.resolveReminderContent(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}
	filter, err := reminderCandidateFilterFromRequest(req, rule)
	if err != nil {
		return nil, err
	}
	candidates, err := s.repo.ListReminderCandidates(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("list reminder candidates: %w", err)
	}
	if len(candidates) == 0 {
		return &dto.SendReminderResponse{}, nil
	}

	logs := make([]model.ReminderLog, 0, len(candidates))
	err = s.runInTx(ctx, func(repo repository.OperationsRepository) error {
		for _, candidate := range candidates {
			recipient := reminderRecipient(channel, candidate)
			renderedSubject := renderReminderText(subject, candidate)
			renderedBody := renderReminderText(body, candidate)
			log := model.ReminderLog{
				ID:           uuid.New(),
				TenantID:     tenantID,
				InvoiceID:    &candidate.InvoiceID,
				StudentID:    candidate.StudentID,
				GuardianID:   candidate.GuardianID,
				Channel:      channel,
				Recipient:    recipient,
				Subject:      renderedSubject,
				Message:      renderedBody,
				Status:       "queued",
				ScheduledFor: time.Now().UTC(),
				CreatedBy:    &actorID,
				Metadata:     normalizeMetadata(req.Metadata),
			}
			if rule != nil {
				log.RuleID = &rule.ID
			}
			if template != nil {
				log.TemplateID = &template.ID
			}
			if recipient == "" {
				log.Status = "skipped"
				log.ErrorMessage = "no recipient available for selected channel"
				if err := repo.CreateReminderLog(ctx, &log); err != nil {
					return mapPersistenceError(err, "create skipped reminder log")
				}
				logs = append(logs, log)
				continue
			}
			job := model.Job{
				TenantID:       tenantID,
				JobType:        reminderJobTypeSend,
				Status:         "queued",
				Priority:       0,
				RunAt:          time.Now().UTC(),
				MaxAttempts:    3,
				IdempotencyKey: stringPtr(fmt.Sprintf("reminder:%s:%s:%s", tenantID, candidate.InvoiceID, log.ID)),
				Payload: map[string]any{
					"reminder_log_id": log.ID.String(),
					"channel":         channel,
					"recipient":       recipient,
					"subject":         renderedSubject,
					"message":         renderedBody,
				},
			}
			if rule != nil {
				job.MaxAttempts = rule.MaxAttempts
			}
			if err := repo.CreateJob(ctx, &job); err != nil {
				return mapPersistenceError(err, "create reminder job")
			}
			log.JobID = &job.ID
			if err := repo.CreateReminderLog(ctx, &log); err != nil {
				return mapPersistenceError(err, "create reminder log")
			}
			logs = append(logs, log)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, tenantID, actorID, "reminders.queued", "reminder_log", uuid.Nil, "reminders queued", map[string]any{"count": len(logs), "channel": channel}); err != nil {
		return nil, err
	}

	resp := reminderSendResponseFromLogs(logs)
	if req.ProcessNow == nil || *req.ProcessNow {
		processed, err := s.ProcessDueReminderJobs(ctx, tenantID, len(logs))
		if err != nil {
			return nil, err
		}
		resp.SentCount += processed.SentCount
		resp.FailedCount += processed.FailedCount
		resp.SkippedCount += processed.SkippedCount
		if len(processed.ReminderLogs) > 0 {
			resp.ReminderLogs = processed.ReminderLogs
		}
	}
	return resp, nil
}

func (s *operationsService) ListReminderLogs(ctx context.Context, tenantID uuid.UUID, filter model.ReminderLogFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ReminderLogResponse], error) {
	result, err := s.repo.ListReminderLogs(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list reminder logs: %w", err)
	}
	items := make([]dto.ReminderLogResponse, len(result.Data))
	for i := range result.Data {
		items[i] = reminderLogToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) ProcessDueReminderJobs(ctx context.Context, tenantID uuid.UUID, limit int) (*dto.SendReminderResponse, error) {
	jobs, err := s.repo.ClaimDueJobs(ctx, tenantID, reminderJobTypeSend, s.workerID, limit)
	if err != nil {
		return nil, fmt.Errorf("claim reminder jobs: %w", err)
	}
	resp := &dto.SendReminderResponse{}
	for _, job := range jobs {
		log, err := s.processReminderJob(ctx, tenantID, job)
		if err != nil {
			resp.FailedCount++
			continue
		}
		switch log.Status {
		case "sent":
			resp.SentCount++
		case "skipped":
			resp.SkippedCount++
		default:
			resp.FailedCount++
		}
		resp.ReminderLogs = append(resp.ReminderLogs, reminderLogToResponse(log))
	}
	return resp, nil
}

func (s *operationsService) ProcessDueReminderJobsForAllTenants(ctx context.Context, limit int) (*dto.SendReminderResponse, error) {
	tenantIDs, err := s.repo.ListTenantsWithDueJobs(ctx, reminderJobTypeSend, 100)
	if err != nil {
		return nil, fmt.Errorf("list tenants with reminder jobs: %w", err)
	}
	total := &dto.SendReminderResponse{}
	for _, tenantID := range tenantIDs {
		resp, err := s.ProcessDueReminderJobs(ctx, tenantID, limit)
		if err != nil {
			total.FailedCount++
			continue
		}
		total.QueuedCount += resp.QueuedCount
		total.SentCount += resp.SentCount
		total.FailedCount += resp.FailedCount
		total.SkippedCount += resp.SkippedCount
		total.ReminderLogs = append(total.ReminderLogs, resp.ReminderLogs...)
	}
	return total, nil
}

func (s *operationsService) GetDashboard(ctx context.Context, tenantID uuid.UUID, asOf time.Time) (*dto.DashboardResponse, error) {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	summary, err := s.repo.GetDashboardSummary(ctx, tenantID, asOf, 10)
	if err != nil {
		return nil, fmt.Errorf("get dashboard summary: %w", err)
	}
	resp := dashboardToResponse(summary)
	return &resp, nil
}

func (s *operationsService) ListCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.CollectionReportRowResponse], error) {
	result, err := s.repo.ListCollectionReport(ctx, tenantID, normalizeReportFilter(filter), params)
	if err != nil {
		return nil, fmt.Errorf("list collection report: %w", err)
	}
	items := make([]dto.CollectionReportRowResponse, len(result.Data))
	for i := range result.Data {
		items[i] = collectionReportRowToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) ListDefaulterReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.DefaulterReportRowResponse], error) {
	result, err := s.repo.ListDefaulterReport(ctx, tenantID, normalizeReportFilter(filter), params)
	if err != nil {
		return nil, fmt.Errorf("list defaulter report: %w", err)
	}
	items := make([]dto.DefaulterReportRowResponse, len(result.Data))
	for i := range result.Data {
		items[i] = defaulterReportRowToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) ListDueReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.DueReportRowResponse], error) {
	result, err := s.repo.ListDueReport(ctx, tenantID, normalizeReportFilter(filter), params)
	if err != nil {
		return nil, fmt.Errorf("list due report: %w", err)
	}
	items := make([]dto.DueReportRowResponse, len(result.Data))
	for i := range result.Data {
		items[i] = dueReportRowToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) ListFeeHeadCollectionReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.FeeHeadCollectionRowResponse], error) {
	result, err := s.repo.ListFeeHeadCollectionReport(ctx, tenantID, normalizeReportFilter(filter), params)
	if err != nil {
		return nil, fmt.Errorf("list fee-head collection report: %w", err)
	}
	items := make([]dto.FeeHeadCollectionRowResponse, len(result.Data))
	for i := range result.Data {
		items[i] = feeHeadCollectionRowToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) ListPaymentMethodReport(ctx context.Context, tenantID uuid.UUID, filter model.ReportFilter, params model.PaginationParams) (*model.PaginatedResult[dto.PaymentMethodReportRowResponse], error) {
	result, err := s.repo.ListPaymentMethodReport(ctx, tenantID, normalizeReportFilter(filter), params)
	if err != nil {
		return nil, fmt.Errorf("list payment-method report: %w", err)
	}
	items := make([]dto.PaymentMethodReportRowResponse, len(result.Data))
	for i := range result.Data {
		items[i] = paymentMethodReportRowToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) CreateExport(ctx context.Context, actorID, tenantID uuid.UUID, req dto.CreateExportRequest) (*dto.ExportJobResponse, error) {
	if defaultString(req.Format, "csv") != "csv" {
		return nil, apperror.New("UNSUPPORTED_EXPORT_FORMAT", "only CSV exports are supported", http.StatusBadRequest)
	}
	filter, err := reportFilterFromExportRequest(req)
	if err != nil {
		return nil, err
	}
	params := exportParams(req)
	job := &model.ExportJob{
		TenantID:    tenantID,
		ExportType:  strings.TrimSpace(req.ExportType),
		Status:      "queued",
		Format:      "csv",
		Params:      params,
		ContentType: exportContentTypeCSV,
		RequestedBy: &actorID,
	}
	if err := s.repo.CreateExportJob(ctx, job); err != nil {
		return nil, mapPersistenceError(err, "create export job")
	}
	if err := s.repo.MarkExportJobProcessing(ctx, tenantID, job.ID); err != nil {
		return nil, fmt.Errorf("mark export processing: %w", err)
	}
	csvBytes, rowCount, filename, err := s.buildCSVExport(ctx, tenantID, job.ExportType, filter)
	if err != nil {
		_ = s.repo.FailExportJob(ctx, tenantID, job.ID, err.Error())
		return nil, err
	}
	job.FileName = filename
	job.ContentType = exportContentTypeCSV
	job.Content = csvBytes
	job.RowCount = rowCount
	if err := s.repo.CompleteExportJob(ctx, job); err != nil {
		return nil, fmt.Errorf("complete export job: %w", err)
	}
	refreshed, err := s.repo.GetExportJob(ctx, tenantID, job.ID)
	if err != nil {
		return nil, fmt.Errorf("get export job: %w", err)
	}
	if refreshed != nil {
		job = refreshed
	}
	if err := s.audit(ctx, tenantID, actorID, "report.exported", "export_job", job.ID, "report exported", map[string]any{"export_type": job.ExportType, "row_count": job.RowCount}); err != nil {
		return nil, err
	}
	resp := exportJobToResponse(job)
	return &resp, nil
}

func (s *operationsService) ListExports(ctx context.Context, tenantID uuid.UUID, filter model.ExportJobFilter, params model.PaginationParams) (*model.PaginatedResult[dto.ExportJobResponse], error) {
	result, err := s.repo.ListExportJobs(ctx, tenantID, filter, params)
	if err != nil {
		return nil, fmt.Errorf("list export jobs: %w", err)
	}
	items := make([]dto.ExportJobResponse, len(result.Data))
	for i := range result.Data {
		items[i] = exportJobToResponse(&result.Data[i])
	}
	return model.NewPaginatedResult(items, result.Total, result.Page, result.PageSize), nil
}

func (s *operationsService) GetExport(ctx context.Context, tenantID, id uuid.UUID) (*dto.ExportJobResponse, error) {
	job, err := s.repo.GetExportJob(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get export job: %w", err)
	}
	if job == nil {
		return nil, apperror.ErrNotFound
	}
	resp := exportJobToResponse(job)
	return &resp, nil
}

func (s *operationsService) DownloadExport(ctx context.Context, tenantID, id uuid.UUID) (*dto.ExportDownloadResponse, error) {
	job, err := s.repo.GetExportJob(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get export job: %w", err)
	}
	if job == nil {
		return nil, apperror.ErrNotFound
	}
	if job.Status != "succeeded" || len(job.Content) == 0 {
		return nil, apperror.New("EXPORT_NOT_READY", "export is not ready for download", http.StatusConflict)
	}
	return &dto.ExportDownloadResponse{
		Filename:    job.FileName,
		ContentType: job.ContentType,
		Bytes:       job.Content,
	}, nil
}

func (s *operationsService) processReminderJob(ctx context.Context, tenantID uuid.UUID, job model.Job) (*model.ReminderLog, error) {
	logID, err := uuid.Parse(fmt.Sprint(job.Payload["reminder_log_id"]))
	if err != nil {
		retryAt := time.Now().UTC().Add(5 * time.Minute)
		_ = s.repo.MarkJobFailed(ctx, tenantID, job.ID, "invalid reminder log id in job payload", &retryAt)
		return nil, err
	}
	log, err := s.repo.GetReminderLog(ctx, tenantID, logID)
	if err != nil {
		return nil, fmt.Errorf("get reminder log: %w", err)
	}
	if log == nil {
		retryAt := time.Now().UTC().Add(5 * time.Minute)
		_ = s.repo.MarkJobFailed(ctx, tenantID, job.ID, "reminder log not found", &retryAt)
		return nil, apperror.ErrNotFound
	}
	now := time.Now().UTC()
	log.AttemptedAt = &now
	log.AttemptCount = job.Attempts
	result, sendErr := s.notifier.Send(ctx, NotificationMessage{
		Channel:   log.Channel,
		Recipient: log.Recipient,
		Subject:   log.Subject,
		Body:      strings.ReplaceAll(log.Message, "\n", "<br>"),
		Metadata:  log.Metadata,
	})
	notification := &model.NotificationLog{
		TenantID:      tenantID,
		ReminderLogID: &log.ID,
		Channel:       log.Channel,
		Recipient:     log.Recipient,
		AttemptedAt:   now,
	}
	if sendErr != nil {
		log.Status = "failed"
		log.ErrorMessage = sendErr.Error()
		notification.Provider = "unknown"
		notification.Status = "failed"
		notification.ErrorMessage = sendErr.Error()
		retryAt := now.Add(time.Duration(job.Attempts) * 5 * time.Minute)
		if err := s.repo.CreateNotificationLog(ctx, notification); err != nil {
			return nil, fmt.Errorf("create notification log: %w", err)
		}
		if err := s.repo.UpdateReminderLogDelivery(ctx, log); err != nil {
			return nil, fmt.Errorf("update reminder log: %w", err)
		}
		_ = s.repo.MarkJobFailed(ctx, tenantID, job.ID, sendErr.Error(), &retryAt)
		return log, sendErr
	}
	log.Provider = result.Provider
	log.ProviderMessageID = result.MessageID
	log.ProviderResponse = normalizeMetadata(result.Response)
	log.ErrorMessage = ""
	log.Status = result.Status
	if log.Status == "" {
		log.Status = "sent"
	}
	if log.Status == "sent" {
		log.SentAt = &now
	}
	if log.Status != "sent" && log.Status != "skipped" {
		log.Status = "failed"
	}
	notification.Provider = result.Provider
	notification.Status = log.Status
	notification.ProviderMessageID = result.MessageID
	notification.ProviderResponse = normalizeMetadata(result.Response)
	if err := s.repo.CreateNotificationLog(ctx, notification); err != nil {
		return nil, fmt.Errorf("create notification log: %w", err)
	}
	if err := s.repo.UpdateReminderLogDelivery(ctx, log); err != nil {
		return nil, fmt.Errorf("update reminder log: %w", err)
	}
	if log.Status == "failed" {
		retryAt := now.Add(time.Duration(job.Attempts) * 5 * time.Minute)
		_ = s.repo.MarkJobFailed(ctx, tenantID, job.ID, "notification provider returned failed", &retryAt)
		return log, nil
	}
	if err := s.repo.MarkJobSucceeded(ctx, tenantID, job.ID); err != nil {
		return nil, fmt.Errorf("mark reminder job succeeded: %w", err)
	}
	return log, nil
}

func (s *operationsService) resolveReminderContent(ctx context.Context, tenantID uuid.UUID, req dto.SendReminderRequest) (*model.ReminderRule, *model.ReminderTemplate, string, string, string, error) {
	var rule *model.ReminderRule
	var template *model.ReminderTemplate
	var err error
	if req.RuleID != nil {
		rule, err = s.repo.GetReminderRule(ctx, tenantID, *req.RuleID)
		if err != nil {
			return nil, nil, "", "", "", fmt.Errorf("get reminder rule: %w", err)
		}
		if rule == nil {
			return nil, nil, "", "", "", apperror.ErrNotFound
		}
		template = rule.Template
	}
	if req.TemplateID != nil {
		template, err = s.requireReminderTemplate(ctx, tenantID, *req.TemplateID)
		if err != nil {
			return nil, nil, "", "", "", err
		}
	}
	channel := strings.TrimSpace(req.Channel)
	if channel == "" && rule != nil {
		channel = rule.Channel
	}
	if channel == "" && template != nil {
		channel = template.Channel
	}
	channel = defaultString(channel, "email")
	subject := strings.TrimSpace(req.Subject)
	body := strings.TrimSpace(req.Message)
	if subject == "" && template != nil {
		subject = template.Subject
	}
	if body == "" && template != nil {
		body = template.Body
	}
	if subject == "" {
		subject = "Fee payment reminder for {{student_name}}"
	}
	if body == "" {
		body = "Dear guardian, {{student_name}} has an outstanding fee balance of {{amount_due}} for invoice {{invoice_number}} due on {{due_date}}."
	}
	return rule, template, channel, subject, body, nil
}

func (s *operationsService) requireReminderTemplate(ctx context.Context, tenantID, id uuid.UUID) (*model.ReminderTemplate, error) {
	template, err := s.repo.GetReminderTemplate(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get reminder template: %w", err)
	}
	if template == nil {
		return nil, apperror.ErrNotFound
	}
	return template, nil
}

func (s *operationsService) buildCSVExport(ctx context.Context, tenantID uuid.UUID, exportType string, filter model.ReportFilter) ([]byte, int, string, error) {
	params := model.PaginationParams{Page: 1, PageSize: 10000}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rowCount := 0
	write := func(record []string) error {
		if err := w.Write(record); err != nil {
			return err
		}
		rowCount++
		return nil
	}

	switch exportType {
	case "collections", "offline_payments", "receipt_register":
		if exportType == "offline_payments" && filter.Provider == "" {
			filter.Provider = "offline"
		}
		result, err := s.repo.ListCollectionReport(ctx, tenantID, normalizeReportFilter(filter), params)
		if err != nil {
			return nil, 0, "", fmt.Errorf("build collection export: %w", err)
		}
		if err := write([]string{"payment_id", "receipt_number", "student", "admission_number", "class", "section", "payment_method", "provider", "amount_paise", "paid_at"}); err != nil {
			return nil, 0, "", err
		}
		for _, row := range result.Data {
			if err := write([]string{
				row.PaymentID.String(),
				row.ReceiptNumber,
				row.StudentName,
				row.AdmissionNumber,
				row.ClassName,
				row.SectionName,
				row.PaymentMethod,
				row.Provider,
				strconv.FormatInt(row.AmountPaise, 10),
				row.PaidAt.Format(time.RFC3339),
			}); err != nil {
				return nil, 0, "", err
			}
		}
	case "defaulters":
		result, err := s.repo.ListDefaulterReport(ctx, tenantID, normalizeReportFilter(filter), params)
		if err != nil {
			return nil, 0, "", fmt.Errorf("build defaulter export: %w", err)
		}
		if err := write([]string{"student", "admission_number", "class", "section", "guardian_name", "guardian_phone", "guardian_email", "invoice_count", "total_due_paise", "overdue_paise", "oldest_due_date", "last_reminder_status"}); err != nil {
			return nil, 0, "", err
		}
		for _, row := range result.Data {
			if err := write([]string{
				row.StudentName,
				row.AdmissionNumber,
				row.ClassName,
				row.SectionName,
				row.GuardianName,
				row.GuardianPhone,
				row.GuardianEmail,
				strconv.FormatInt(row.InvoiceCount, 10),
				strconv.FormatInt(row.TotalDuePaise, 10),
				strconv.FormatInt(row.OverduePaise, 10),
				row.OldestDueDate.Format(dateLayout),
				row.LastReminderStatus,
			}); err != nil {
				return nil, 0, "", err
			}
		}
	case "dues":
		result, err := s.repo.ListDueReport(ctx, tenantID, normalizeReportFilter(filter), params)
		if err != nil {
			return nil, 0, "", fmt.Errorf("build dues export: %w", err)
		}
		if err := write([]string{"class", "section", "student_count", "invoice_count", "total_billed_paise", "total_paid_paise", "total_due_paise", "overdue_paise"}); err != nil {
			return nil, 0, "", err
		}
		for _, row := range result.Data {
			if err := write([]string{row.ClassName, row.SectionName, strconv.FormatInt(row.StudentCount, 10), strconv.FormatInt(row.InvoiceCount, 10), strconv.FormatInt(row.TotalBilledPaise, 10), strconv.FormatInt(row.TotalPaidPaise, 10), strconv.FormatInt(row.TotalDuePaise, 10), strconv.FormatInt(row.OverduePaise, 10)}); err != nil {
				return nil, 0, "", err
			}
		}
	case "fee_heads":
		result, err := s.repo.ListFeeHeadCollectionReport(ctx, tenantID, normalizeReportFilter(filter), params)
		if err != nil {
			return nil, 0, "", fmt.Errorf("build fee-head export: %w", err)
		}
		if err := write([]string{"fee_head", "code", "category", "collected_paise", "invoice_count", "payment_count"}); err != nil {
			return nil, 0, "", err
		}
		for _, row := range result.Data {
			if err := write([]string{row.FeeHeadName, row.FeeHeadCode, row.Category, strconv.FormatInt(row.CollectedPaise, 10), strconv.FormatInt(row.InvoiceCount, 10), strconv.FormatInt(row.PaymentCount, 10)}); err != nil {
				return nil, 0, "", err
			}
		}
	case "payment_methods":
		result, err := s.repo.ListPaymentMethodReport(ctx, tenantID, normalizeReportFilter(filter), params)
		if err != nil {
			return nil, 0, "", fmt.Errorf("build payment-method export: %w", err)
		}
		if err := write([]string{"payment_method", "provider", "payment_count", "amount_paise"}); err != nil {
			return nil, 0, "", err
		}
		for _, row := range result.Data {
			if err := write([]string{row.PaymentMethod, row.Provider, strconv.FormatInt(row.PaymentCount, 10), strconv.FormatInt(row.AmountPaise, 10)}); err != nil {
				return nil, 0, "", err
			}
		}
	default:
		return nil, 0, "", apperror.New("UNSUPPORTED_EXPORT_TYPE", "unsupported export type", http.StatusBadRequest)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, "", err
	}
	filename := fmt.Sprintf("%s-%s.csv", exportType, time.Now().UTC().Format("20060102-150405"))
	return buf.Bytes(), max(rowCount-1, 0), filename, nil
}

func (s *operationsService) runInTx(ctx context.Context, fn func(repository.OperationsRepository) error) error {
	if s.tx == nil || s.repoFactory == nil {
		return fn(s.repo)
	}
	return s.tx.WithinTx(ctx, func(db database.DBTX) error {
		return fn(s.repoFactory(db))
	})
}

func (s *operationsService) audit(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, action string, entityType string, entityID uuid.UUID, summary string, metadata map[string]any) error {
	if s.auditRepo == nil {
		return nil
	}
	entry := &model.AuditLog{
		TenantID:    &tenantID,
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  entityType,
		Summary:     summary,
		Metadata:    normalizeMetadata(metadata),
	}
	if entityID != uuid.Nil {
		entry.EntityID = &entityID
	}
	if err := s.auditRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func reminderCandidateFilterFromRequest(req dto.SendReminderRequest, rule *model.ReminderRule) (model.ReminderCandidateFilter, error) {
	var dueOnOrBefore *time.Time
	if strings.TrimSpace(req.DueOnOrBefore) != "" {
		parsed, err := parseDate(req.DueOnOrBefore)
		if err != nil {
			return model.ReminderCandidateFilter{}, err
		}
		dueOnOrBefore = &parsed
	} else if len(req.InvoiceIDs) == 0 {
		today := dateOnly(time.Now().UTC())
		dueOnOrBefore = &today
	}
	statuses := []string{}
	if rule != nil {
		statuses = normalizeReminderStatuses(rule.TargetStatuses)
	}
	return model.ReminderCandidateFilter{
		InvoiceIDs:     req.InvoiceIDs,
		StudentID:      req.StudentID,
		ClassID:        req.ClassID,
		SectionID:      req.SectionID,
		AcademicYearID: req.AcademicYearID,
		DueOnOrBefore:  dueOnOrBefore,
		Statuses:       statuses,
	}, nil
}

func reportFilterFromExportRequest(req dto.CreateExportRequest) (model.ReportFilter, error) {
	filter := model.ReportFilter{
		StudentID:     req.StudentID,
		ClassID:       req.ClassID,
		SectionID:     req.SectionID,
		PaymentMethod: strings.TrimSpace(req.PaymentMethod),
		Provider:      strings.TrimSpace(req.Provider),
	}
	if strings.TrimSpace(req.From) != "" {
		parsed, err := parseDate(req.From)
		if err != nil {
			return filter, err
		}
		filter.From = &parsed
	}
	if strings.TrimSpace(req.To) != "" {
		parsed, err := parseDate(req.To)
		if err != nil {
			return filter, err
		}
		parsed = parsed.AddDate(0, 0, 1)
		filter.To = &parsed
	}
	if strings.TrimSpace(req.AsOf) != "" {
		parsed, err := parseDate(req.AsOf)
		if err != nil {
			return filter, err
		}
		filter.AsOf = &parsed
	}
	return filter, nil
}

func exportParams(req dto.CreateExportRequest) map[string]any {
	params := normalizeMetadata(req.Metadata)
	params["export_type"] = req.ExportType
	params["format"] = defaultString(req.Format, "csv")
	if req.From != "" {
		params["from"] = req.From
	}
	if req.To != "" {
		params["to"] = req.To
	}
	if req.AsOf != "" {
		params["as_of"] = req.AsOf
	}
	if req.StudentID != nil {
		params["student_id"] = req.StudentID.String()
	}
	if req.ClassID != nil {
		params["class_id"] = req.ClassID.String()
	}
	if req.SectionID != nil {
		params["section_id"] = req.SectionID.String()
	}
	if req.PaymentMethod != "" {
		params["payment_method"] = req.PaymentMethod
	}
	if req.Provider != "" {
		params["provider"] = req.Provider
	}
	return params
}

func normalizeReportFilter(filter model.ReportFilter) model.ReportFilter {
	if filter.AsOf == nil {
		asOf := dateOnly(time.Now().UTC())
		filter.AsOf = &asOf
	}
	return filter
}

func normalizeReminderStatuses(values []string) []string {
	if len(values) == 0 {
		return []string{"issued", "partially_paid", "overdue"}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	if len(out) == 0 {
		return []string{"issued", "partially_paid", "overdue"}
	}
	return out
}

func reminderRecipient(channel string, candidate model.ReminderCandidate) string {
	switch channel {
	case "email":
		return firstNonEmptyStringPtr(candidate.GuardianEmail, candidate.StudentEmail)
	case "sms":
		return firstNonEmptyStringPtr(candidate.GuardianPhone, candidate.StudentPhone)
	case "whatsapp":
		return firstNonEmptyStringPtr(candidate.GuardianWhatsApp, candidate.GuardianPhone, candidate.StudentPhone)
	case "in_app":
		return candidate.StudentID.String()
	default:
		return ""
	}
}

func firstNonEmptyStringPtr(values ...*string) string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return strings.TrimSpace(*value)
		}
	}
	return ""
}

func renderReminderText(template string, candidate model.ReminderCandidate) string {
	replacements := map[string]string{
		"{{student_name}}":     strings.TrimSpace(candidate.StudentFirstName + " " + candidate.StudentLastName),
		"{{admission_number}}": candidate.AdmissionNumber,
		"{{invoice_number}}":   candidate.InvoiceNumber,
		"{{amount_due}}":       formatReminderAmount(candidate.BalanceAmountPaise, candidate.Currency),
		"{{due_date}}":         candidate.DueDate.Format(dateLayout),
		"{{class_name}}":       candidate.ClassName,
		"{{section_name}}":     candidate.SectionName,
		"{{guardian_name}}":    candidate.GuardianName,
	}
	rendered := template
	for placeholder, value := range replacements {
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}
	return rendered
}

func formatReminderAmount(amount int64, currency string) string {
	return fmt.Sprintf("%s %.2f", defaultString(currency, defaultCurrency), float64(amount)/100)
}

func reminderSendResponseFromLogs(logs []model.ReminderLog) *dto.SendReminderResponse {
	resp := &dto.SendReminderResponse{ReminderLogs: make([]dto.ReminderLogResponse, len(logs))}
	for i := range logs {
		switch logs[i].Status {
		case "queued":
			resp.QueuedCount++
		case "sent":
			resp.SentCount++
		case "skipped":
			resp.SkippedCount++
		default:
			resp.FailedCount++
		}
		resp.ReminderLogs[i] = reminderLogToResponse(&logs[i])
	}
	return resp
}

func reminderTemplateToResponse(item *model.ReminderTemplate) dto.ReminderTemplateResponse {
	if item == nil {
		return dto.ReminderTemplateResponse{}
	}
	return dto.ReminderTemplateResponse{
		ID:        item.ID,
		TenantID:  item.TenantID,
		Name:      item.Name,
		Code:      item.Code,
		Channel:   item.Channel,
		Subject:   item.Subject,
		Body:      item.Body,
		Tone:      item.Tone,
		Status:    item.Status,
		Metadata:  item.Metadata,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func reminderRuleToResponse(item *model.ReminderRule) dto.ReminderRuleResponse {
	resp := dto.ReminderRuleResponse{
		ID:             item.ID,
		TenantID:       item.TenantID,
		TemplateID:     item.TemplateID,
		Name:           item.Name,
		Code:           item.Code,
		Channel:        item.Channel,
		TriggerType:    item.TriggerType,
		OffsetDays:     item.OffsetDays,
		TargetStatuses: item.TargetStatuses,
		Status:         item.Status,
		MaxAttempts:    item.MaxAttempts,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.Template != nil {
		template := reminderTemplateToResponse(item.Template)
		resp.Template = &template
	}
	return resp
}

func reminderLogToResponse(item *model.ReminderLog) dto.ReminderLogResponse {
	resp := dto.ReminderLogResponse{
		ID:                item.ID,
		TenantID:          item.TenantID,
		RuleID:            item.RuleID,
		TemplateID:        item.TemplateID,
		JobID:             item.JobID,
		InvoiceID:         item.InvoiceID,
		StudentID:         item.StudentID,
		GuardianID:        item.GuardianID,
		Channel:           item.Channel,
		Recipient:         item.Recipient,
		Subject:           item.Subject,
		Message:           item.Message,
		Status:            item.Status,
		Provider:          item.Provider,
		ProviderMessageID: item.ProviderMessageID,
		ErrorMessage:      item.ErrorMessage,
		ScheduledFor:      item.ScheduledFor,
		AttemptedAt:       item.AttemptedAt,
		SentAt:            item.SentAt,
		AttemptCount:      item.AttemptCount,
		Metadata:          item.Metadata,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.Student != nil {
		resp.Student = &dto.StudentBriefResponse{
			ID:              item.Student.ID,
			AdmissionNumber: item.Student.AdmissionNumber,
			FirstName:       item.Student.FirstName,
			LastName:        item.Student.LastName,
			Status:          item.Student.Status,
		}
	}
	if item.Guardian != nil {
		resp.GuardianName = item.Guardian.Name
	}
	if item.Invoice != nil {
		resp.InvoiceNumber = item.Invoice.InvoiceNumber
	}
	return resp
}

func dashboardToResponse(item *model.DashboardSummary) dto.DashboardResponse {
	resp := dto.DashboardResponse{
		Currency:               item.Currency,
		TotalStudents:          item.TotalStudents,
		ActiveStudents:         item.ActiveStudents,
		TodayCollectionPaise:   item.TodayCollectionPaise,
		MonthCollectionPaise:   item.MonthCollectionPaise,
		TotalDuePaise:          item.TotalDuePaise,
		OverduePaise:           item.OverduePaise,
		DefaulterCount:         item.DefaulterCount,
		UnpaidInvoiceCount:     item.UnpaidInvoiceCount,
		PaymentMethodBreakdown: make([]dto.PaymentMethodSummaryResponse, len(item.PaymentMethodBreakdown)),
		RecentPaymentEvents:    make([]dto.PaymentEventResponse, len(item.RecentPaymentEvents)),
	}
	for i := range item.PaymentMethodBreakdown {
		resp.PaymentMethodBreakdown[i] = dto.PaymentMethodSummaryResponse{
			PaymentMethod: item.PaymentMethodBreakdown[i].PaymentMethod,
			PaymentCount:  item.PaymentMethodBreakdown[i].PaymentCount,
			AmountPaise:   item.PaymentMethodBreakdown[i].AmountPaise,
		}
	}
	for i := range item.RecentPaymentEvents {
		resp.RecentPaymentEvents[i] = paymentEventToResponse(&item.RecentPaymentEvents[i])
	}
	return resp
}

func collectionReportRowToResponse(item *model.CollectionReportRow) dto.CollectionReportRowResponse {
	return dto.CollectionReportRowResponse(*item)
}

func defaulterReportRowToResponse(item *model.DefaulterReportRow) dto.DefaulterReportRowResponse {
	return dto.DefaulterReportRowResponse{
		StudentID:          item.StudentID,
		AdmissionNumber:    item.AdmissionNumber,
		StudentName:        item.StudentName,
		ClassID:            item.ClassID,
		ClassName:          item.ClassName,
		SectionID:          item.SectionID,
		SectionName:        item.SectionName,
		GuardianName:       item.GuardianName,
		GuardianPhone:      item.GuardianPhone,
		GuardianEmail:      item.GuardianEmail,
		InvoiceCount:       item.InvoiceCount,
		TotalDuePaise:      item.TotalDuePaise,
		OverduePaise:       item.OverduePaise,
		OldestDueDate:      item.OldestDueDate.Format(dateLayout),
		LastReminderAt:     item.LastReminderAt,
		LastReminderStatus: item.LastReminderStatus,
	}
}

func dueReportRowToResponse(item *model.DueReportRow) dto.DueReportRowResponse {
	return dto.DueReportRowResponse(*item)
}

func feeHeadCollectionRowToResponse(item *model.FeeHeadCollectionRow) dto.FeeHeadCollectionRowResponse {
	return dto.FeeHeadCollectionRowResponse(*item)
}

func paymentMethodReportRowToResponse(item *model.PaymentMethodReportRow) dto.PaymentMethodReportRowResponse {
	return dto.PaymentMethodReportRowResponse(*item)
}

func exportJobToResponse(item *model.ExportJob) dto.ExportJobResponse {
	return dto.ExportJobResponse{
		ID:           item.ID,
		TenantID:     item.TenantID,
		ExportType:   item.ExportType,
		Status:       item.Status,
		Format:       item.Format,
		Params:       item.Params,
		FileName:     item.FileName,
		ContentType:  item.ContentType,
		RowCount:     item.RowCount,
		RequestedBy:  item.RequestedBy,
		CompletedAt:  item.CompletedAt,
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func stringPtr(value string) *string {
	return &value
}

package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

type OperationsHandler struct {
	opsSvc service.OperationsService
}

func NewOperationsHandler(opsSvc service.OperationsService) *OperationsHandler {
	return &OperationsHandler{opsSvc: opsSvc}
}

func (h *OperationsHandler) CreateReminderTemplate(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateReminderTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.CreateReminderTemplate(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *OperationsHandler) ListReminderTemplates(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.ReminderTemplateFilter{
		Channel: c.Query("channel"),
		Status:  c.Query("status"),
		Search:  c.Query("search"),
	}
	result, err := h.opsSvc.ListReminderTemplates(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) GetReminderTemplate(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.opsSvc.GetReminderTemplate(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) UpdateReminderTemplate(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateReminderTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.UpdateReminderTemplate(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) DeleteReminderTemplate(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.opsSvc.DeleteReminderTemplate(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "reminder template deleted")
}

func (h *OperationsHandler) CreateReminderRule(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateReminderRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.CreateReminderRule(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *OperationsHandler) ListReminderRules(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.ReminderRuleFilter{
		Channel:     c.Query("channel"),
		TriggerType: c.Query("trigger_type"),
		Status:      c.Query("status"),
		Search:      c.Query("search"),
	}
	result, err := h.opsSvc.ListReminderRules(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) GetReminderRule(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.opsSvc.GetReminderRule(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) UpdateReminderRule(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateReminderRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.UpdateReminderRule(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) DeleteReminderRule(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.opsSvc.DeleteReminderRule(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "reminder rule deleted")
}

func (h *OperationsHandler) SendReminders(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.SendReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.SendReminders(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *OperationsHandler) ListReminderLogs(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return
	}
	invoiceID, ok := queryUUID(c, "invoice_id")
	if !ok {
		return
	}
	from, to, ok := queryDateRange(c)
	if !ok {
		return
	}
	filter := model.ReminderLogFilter{
		StudentID: studentID,
		InvoiceID: invoiceID,
		Channel:   c.Query("channel"),
		Status:    c.Query("status"),
		From:      from,
		To:        to,
	}
	result, err := h.opsSvc.ListReminderLogs(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) GetDashboard(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	asOf, ok := queryDate(c, "as_of")
	if !ok {
		return
	}
	var date time.Time
	if asOf != nil {
		date = *asOf
	}
	resp, err := h.opsSvc.GetDashboard(c.Request.Context(), tenantID, date)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) CollectionsReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	result, err := h.opsSvc.ListCollectionReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) DefaultersReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	result, err := h.opsSvc.ListDefaulterReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) DuesReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	result, err := h.opsSvc.ListDueReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) FeeHeadsReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	result, err := h.opsSvc.ListFeeHeadCollectionReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) PaymentMethodsReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	result, err := h.opsSvc.ListPaymentMethodReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) OfflinePaymentsReport(c *gin.Context) {
	tenantID, filter, ok := h.reportFilter(c)
	if !ok {
		return
	}
	if strings.TrimSpace(filter.Provider) == "" {
		filter.Provider = "offline"
	}
	result, err := h.opsSvc.ListCollectionReport(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) CreateExport(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.opsSvc.CreateExport(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *OperationsHandler) ListExports(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	from, to, ok := queryDateRange(c)
	if !ok {
		return
	}
	filter := model.ExportJobFilter{
		ExportType: c.Query("export_type"),
		Status:     c.Query("status"),
		From:       from,
		To:         to,
	}
	result, err := h.opsSvc.ListExports(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *OperationsHandler) GetExport(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.opsSvc.GetExport(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *OperationsHandler) DownloadExport(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.opsSvc.DownloadExport(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resp.Filename))
	c.Data(http.StatusOK, resp.ContentType, resp.Bytes)
}

func (h *OperationsHandler) reportFilter(c *gin.Context) (uuid.UUID, model.ReportFilter, bool) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return uuid.Nil, model.ReportFilter{}, false
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	classID, ok := queryUUID(c, "class_id")
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	sectionID, ok := queryUUID(c, "section_id")
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	from, to, ok := queryDateRange(c)
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	asOf, ok := queryDate(c, "as_of")
	if !ok {
		return uuid.Nil, model.ReportFilter{}, false
	}
	return tenantID, model.ReportFilter{
		From:           from,
		To:             to,
		AsOf:           asOf,
		StudentID:      studentID,
		ClassID:        classID,
		SectionID:      sectionID,
		AcademicYearID: academicYearID,
		PaymentMethod:  c.Query("payment_method"),
		Provider:       c.Query("provider"),
	}, true
}

func queryDateRange(c *gin.Context) (*time.Time, *time.Time, bool) {
	from, ok := queryDate(c, "from")
	if !ok {
		return nil, nil, false
	}
	to, ok := queryDate(c, "to")
	if !ok {
		return nil, nil, false
	}
	if to != nil {
		endExclusive := to.AddDate(0, 0, 1)
		to = &endExclusive
	}
	return from, to, true
}

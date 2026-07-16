package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

const queryDateLayout = "2006-01-02"

type BillingHandler struct {
	billingSvc service.BillingService
}

func NewBillingHandler(billingSvc service.BillingService) *BillingHandler {
	return &BillingHandler{billingSvc: billingSvc}
}

func (h *BillingHandler) CreateFeeHead(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateFeeHeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.CreateFeeHead(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *BillingHandler) ListFeeHeads(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.FeeHeadFilter{
		Status:   c.Query("status"),
		Category: c.Query("category"),
		Search:   c.Query("search"),
	}
	result, err := h.billingSvc.ListFeeHeads(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *BillingHandler) GetFeeHead(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.billingSvc.GetFeeHead(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) UpdateFeeHead(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateFeeHeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.UpdateFeeHead(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) DeleteFeeHead(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.billingSvc.DeleteFeeHead(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "fee head deleted")
}

func (h *BillingHandler) CreateFeeStructure(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateFeeStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.CreateFeeStructure(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *BillingHandler) ListFeeStructures(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return
	}
	filter := model.FeeStructureFilter{
		AcademicYearID: academicYearID,
		Status:         c.Query("status"),
		BillingCycle:   c.Query("billing_cycle"),
		Search:         c.Query("search"),
	}
	result, err := h.billingSvc.ListFeeStructures(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *BillingHandler) GetFeeStructure(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.billingSvc.GetFeeStructure(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) UpdateFeeStructure(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateFeeStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.UpdateFeeStructure(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) DeleteFeeStructure(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.billingSvc.DeleteFeeStructure(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "fee structure deleted")
}

func (h *BillingHandler) CreateFeeAssignment(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateFeeAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.CreateFeeAssignment(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *BillingHandler) ListFeeAssignments(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	feeStructureID, ok := queryUUID(c, "fee_structure_id")
	if !ok {
		return
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return
	}
	filter := model.FeeAssignmentFilter{
		FeeStructureID: feeStructureID,
		AcademicYearID: academicYearID,
		AssignmentType: c.Query("assignment_type"),
		Status:         c.Query("status"),
		Search:         c.Query("search"),
	}
	result, err := h.billingSvc.ListFeeAssignments(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *BillingHandler) GetFeeAssignment(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.billingSvc.GetFeeAssignment(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) UpdateFeeAssignment(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateFeeAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.UpdateFeeAssignment(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) DeleteFeeAssignment(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.billingSvc.DeleteFeeAssignment(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "fee assignment deleted")
}

func (h *BillingHandler) GenerateInvoices(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.GenerateInvoicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.billingSvc.GenerateInvoices(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *BillingHandler) ListInvoices(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return
	}
	classID, ok := queryUUID(c, "class_id")
	if !ok {
		return
	}
	sectionID, ok := queryUUID(c, "section_id")
	if !ok {
		return
	}
	dueFrom, ok := queryDate(c, "due_from")
	if !ok {
		return
	}
	dueTo, ok := queryDate(c, "due_to")
	if !ok {
		return
	}
	if dueFrom != nil && dueTo != nil && dueFrom.After(*dueTo) {
		RespondValidationError(c, []string{"due_from must be on or before due_to"})
		return
	}
	filter := model.InvoiceFilter{
		StudentID:      studentID,
		AcademicYearID: academicYearID,
		ClassID:        classID,
		SectionID:      sectionID,
		Status:         c.Query("status"),
		DueFrom:        dueFrom,
		DueTo:          dueTo,
		Search:         c.Query("search"),
	}
	result, err := h.billingSvc.ListInvoices(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *BillingHandler) GetInvoice(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.billingSvc.GetInvoice(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) GetStudentLedger(c *gin.Context) {
	tenantID, studentID, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.billingSvc.GetStudentLedger(c.Request.Context(), tenantID, studentID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *BillingHandler) GetParentChildDues(c *gin.Context) {
	actorID, tenantID, childID, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	dueFrom, ok := queryDate(c, "due_from")
	if !ok {
		return
	}
	dueTo, ok := queryDate(c, "due_to")
	if !ok {
		return
	}
	if dueFrom != nil && dueTo != nil && dueFrom.After(*dueTo) {
		RespondValidationError(c, []string{"due_from must be on or before due_to"})
		return
	}
	status, ok := parentInvoiceStatus(c.Query("status"))
	if !ok {
		RespondValidationError(c, []string{"status must be one of paid, pending, partial, overdue, failed"})
		return
	}
	filter := model.InvoiceFilter{
		Status:  status,
		DueFrom: dueFrom,
		DueTo:   dueTo,
		Search:  c.Query("search"),
	}
	resp, err := h.billingSvc.GetParentChildDues(c.Request.Context(), tenantID, actorID, childID, filter)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func parentInvoiceStatus(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", true
	case "paid":
		return "paid", true
	case "pending":
		return "issued", true
	case "partial":
		return "partially_paid", true
	case "overdue":
		return "overdue", true
	case "failed":
		// Invoices do not have a failed lifecycle state. The accepted frontend
		// category is represented by an empty result rather than being silently
		// treated as an unrelated invoice state.
		return "failed", true
	default:
		return "", false
	}
}

func queryDate(c *gin.Context, name string) (*time.Time, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse(queryDateLayout, value)
	if err != nil {
		HandleError(c, apperror.New("INVALID_QUERY_DATE", name+" must use YYYY-MM-DD format", 400))
		return nil, false
	}
	return &parsed, true
}

package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

const maxWebhookBodyBytes = 1 << 20

type PaymentHandler struct {
	paymentSvc service.PaymentService
}

func NewPaymentHandler(paymentSvc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

func (h *PaymentHandler) CreatePaymentOrder(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreatePaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.paymentSvc.CreatePaymentOrder(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.paymentSvc.VerifyPayment(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	payload, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes))
	if err != nil {
		RespondValidationError(c, []string{"webhook payload is too large or unreadable"})
		return
	}
	resp, err := h.paymentSvc.ProcessRazorpayWebhook(
		c.Request.Context(),
		payload,
		c.GetHeader("X-Razorpay-Signature"),
		c.GetHeader("X-Razorpay-Event-Id"),
	)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *PaymentHandler) CreateOfflinePayment(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateOfflinePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.paymentSvc.CreateOfflinePayment(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *PaymentHandler) ListPayments(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return
	}
	from, ok := queryDate(c, "from")
	if !ok {
		return
	}
	to, ok := queryDate(c, "to")
	if !ok {
		return
	}
	filter := model.PaymentFilter{
		StudentID:     studentID,
		Status:        c.Query("status"),
		PaymentMethod: c.Query("payment_method"),
		Provider:      c.Query("provider"),
		From:          from,
		To:            to,
		Search:        c.Query("search"),
	}
	result, err := h.paymentSvc.ListPayments(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.paymentSvc.GetPayment(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *PaymentHandler) ListReceipts(c *gin.Context) {
	tenantID, filter, ok := receiptFilterFromQuery(c)
	if !ok {
		return
	}
	result, err := h.paymentSvc.ListReceipts(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *PaymentHandler) GetReceipt(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.paymentSvc.GetReceipt(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *PaymentHandler) DownloadReceipt(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	file, err := h.paymentSvc.DownloadReceipt(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+file.Filename+`"`)
	c.Data(http.StatusOK, file.ContentType, file.Bytes)
}

func (h *PaymentHandler) ListParentReceipts(c *gin.Context) {
	tenantID, filter, ok := receiptFilterFromQuery(c)
	if !ok {
		return
	}
	result, err := h.paymentSvc.ListParentReceipts(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *PaymentHandler) DownloadParentReceipt(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	file, err := h.paymentSvc.DownloadParentReceipt(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+file.Filename+`"`)
	c.Data(http.StatusOK, file.ContentType, file.Bytes)
}

func (h *PaymentHandler) ListPaymentEvents(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return
	}
	from, ok := queryDate(c, "from")
	if !ok {
		return
	}
	to, ok := queryDate(c, "to")
	if !ok {
		return
	}
	filter := model.PaymentEventFilter{
		StudentID: studentID,
		EventType: c.Query("event_type"),
		Status:    c.Query("status"),
		From:      from,
		To:        to,
	}
	result, err := h.paymentSvc.ListPaymentEvents(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func receiptFilterFromQuery(c *gin.Context) (uuid.UUID, model.ReceiptFilter, bool) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return uuid.Nil, model.ReceiptFilter{}, false
	}
	studentID, ok := queryUUID(c, "student_id")
	if !ok {
		return uuid.Nil, model.ReceiptFilter{}, false
	}
	from, ok := queryDate(c, "from")
	if !ok {
		return uuid.Nil, model.ReceiptFilter{}, false
	}
	to, ok := queryDate(c, "to")
	if !ok {
		return uuid.Nil, model.ReceiptFilter{}, false
	}
	return tenantID, model.ReceiptFilter{
		StudentID: studentID,
		Status:    c.Query("status"),
		From:      from,
		To:        to,
		Search:    c.Query("search"),
	}, true
}

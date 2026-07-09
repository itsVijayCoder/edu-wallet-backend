package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

// ParentHandler exposes the unified "Parents" admin surface: linking guardian
// contacts to parent user accounts, listing a guardian's linked students, and
// returning the aggregated parents summary used by the admin UI.
type ParentHandler struct {
	parentSvc service.ParentService
}

func NewParentHandler(parentSvc service.ParentService) *ParentHandler {
	return &ParentHandler{parentSvc: parentSvc}
}

func (h *ParentHandler) LinkGuardianUser(c *gin.Context) {
	actorID, tenantID, guardianID, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.LinkGuardianUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.parentSvc.LinkGuardianUser(c.Request.Context(), actorID, tenantID, guardianID, req.UserID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *ParentHandler) UnlinkGuardianUser(c *gin.Context) {
	actorID, tenantID, guardianID, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.parentSvc.UnlinkGuardianUser(c.Request.Context(), actorID, tenantID, guardianID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *ParentHandler) ListGuardianStudents(c *gin.Context) {
	tenantID, guardianID, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.parentSvc.ListGuardianStudents(c.Request.Context(), tenantID, guardianID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *ParentHandler) ListParents(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.GuardianFilter{
		Search:       c.Query("search"),
		OnlyLinked:   c.Query("linked") == "true",
		OnlyUnlinked: c.Query("linked") == "false",
	}
	result, err := h.parentSvc.ListParents(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

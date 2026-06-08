package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

type TenantHandler struct {
	tenantSvc service.TenantService
}

func NewTenantHandler(tenantSvc service.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: tenantSvc}
}

func (h *TenantHandler) PlatformCreate(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.tenantSvc.Create(c.Request.Context(), actorID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *TenantHandler) PlatformList(c *gin.Context) {
	params := dto.ExtractPagination(c)
	result, err := h.tenantSvc.List(c.Request.Context(), params)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *TenantHandler) PlatformGet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	resp, err := h.tenantSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *TenantHandler) PlatformUpdate(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.tenantSvc.Update(c.Request.Context(), actorID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *TenantHandler) PlatformCreateBranch(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	var req dto.CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.tenantSvc.CreateBranch(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *TenantHandler) AdminGet(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	resp, err := h.tenantSvc.GetCurrent(c.Request.Context(), tenantID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *TenantHandler) AdminUpdate(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.tenantSvc.UpdateCurrent(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

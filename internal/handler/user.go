package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

type AdminUserHandler struct {
	userSvc service.UserService
}

func NewAdminUserHandler(userSvc service.UserService) *AdminUserHandler {
	return &AdminUserHandler{userSvc: userSvc}
}

func (h *AdminUserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.userSvc.Create(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondCreated(c, resp)
}

func (h *AdminUserHandler) List(c *gin.Context) {
	params := dto.ExtractPagination(c)

	result, err := h.userSvc.List(c.Request.Context(), params)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AdminUserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	resp, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondOK(c, resp)
}

func (h *AdminUserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.userSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondOK(c, resp)
}

func (h *AdminUserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for id"})
		return
	}

	if err := h.userSvc.Delete(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}

	RespondMessage(c, "user deleted successfully")
}

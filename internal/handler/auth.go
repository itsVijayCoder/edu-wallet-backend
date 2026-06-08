package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

type AuthHandler struct {
	authSvc service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondOK(c, resp)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.authSvc.Register(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondCreated(c, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.authSvc.RefreshToken(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondOK(c, resp)
}

func (h *AuthHandler) SelectTenant(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return
	}

	var req dto.SelectTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	resp, err := h.authSvc.SelectTenant(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	RespondOK(c, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		HandleError(c, apperror.ErrInvalidToken)
		return
	}

	if err := h.authSvc.Logout(c.Request.Context(), userID); err != nil {
		HandleError(c, err)
		return
	}

	RespondMessage(c, "logged out successfully")
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	// Always respond 200 regardless of outcome to prevent email enumeration.
	_ = h.authSvc.ForgotPassword(c.Request.Context(), req)

	RespondMessage(c, "if that email is registered, a reset link has been sent")
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}

	if err := h.authSvc.ResetPassword(c.Request.Context(), req); err != nil {
		HandleError(c, err)
		return
	}

	RespondMessage(c, "password has been reset successfully")
}

// extractValidationErrors converts binding/validation errors into a string slice.
func extractValidationErrors(err error) []string {
	var details []string
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			details = append(details, fe.Field()+" failed on '"+fe.Tag()+"' validation")
		}
	} else {
		details = append(details, err.Error())
	}
	return details
}

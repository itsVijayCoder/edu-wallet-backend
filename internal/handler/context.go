package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
)

func currentUserID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, apperror.ErrInvalidToken
	}
	userID, ok := raw.(string)
	if !ok {
		return uuid.Nil, apperror.ErrInvalidToken
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, apperror.ErrInvalidToken
	}
	return id, nil
}

func currentTenantID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, apperror.ErrTenantRequired
	}
	tenantID, ok := raw.(string)
	if !ok {
		return uuid.Nil, apperror.ErrTenantRequired
	}
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return uuid.Nil, apperror.ErrTenantRequired
	}
	return id, nil
}

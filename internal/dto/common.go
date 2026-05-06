package dto

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
)

// ExtractPagination reads pagination params from the query string with defaults.
func ExtractPagination(c *gin.Context) model.PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	params := model.PaginationParams{
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "created_at"),
		SortDir:  c.DefaultQuery("sort_dir", "desc"),
	}
	params.Normalize()
	return params
}

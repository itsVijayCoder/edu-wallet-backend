package model

import "math"

// PaginationParams holds pagination parameters extracted from query strings.
type PaginationParams struct {
	Page     int    `form:"page"      binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by"`
	SortDir  string `form:"sort_dir"  binding:"omitempty,oneof=asc desc"`
}

func (p *PaginationParams) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	if p.SortDir == "" {
		p.SortDir = "desc"
	}
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginatedResult wraps a page of results with metadata.
type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

func NewPaginatedResult[T any](data []T, total int64, page, pageSize int) *PaginatedResult[T] {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &PaginatedResult[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

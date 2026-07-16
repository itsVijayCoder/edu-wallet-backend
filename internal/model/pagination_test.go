package model

import "testing"

func TestPaginationParamsNormalize(t *testing.T) {
	t.Parallel()

	params := PaginationParams{Page: -1, PageSize: 101}
	params.Normalize()
	if params.Page != 1 || params.PageSize != 100 || params.SortBy != "created_at" || params.SortDir != "desc" {
		t.Fatalf("unexpected normalized pagination: %+v", params)
	}
}

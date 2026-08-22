package webui

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePage reads the 1-indexed ?page= query param, defaulting to (and
// floor-clamping at) 1.
func ParsePage(c *gin.Context) int {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// BuildPagination computes prev/next pagination links from the current
// page, total row count, and page size, calling linkFor(p) to build each
// link's URL. Returns nil when everything fits on one page, so callers can
// assign it straight to TableData.Pagination without an extra nil check.
func BuildPagination(page, totalCount, pageSize int, linkFor func(page int) string) *PaginationData {
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if totalPages <= 1 {
		return nil
	}

	data := &PaginationData{Page: page, TotalPages: totalPages}
	if page > 1 {
		data.PrevURL = linkFor(page - 1)
	}
	if page < totalPages {
		data.NextURL = linkFor(page + 1)
	}
	return data
}

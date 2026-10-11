package pagination

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	appErrors "educore/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage     = 1
	DefaultLimit    = 20
	MaxLimit        = 100
	DefaultOrder    = "asc"
	OrderAscending  = "asc"
	OrderDescending = "desc"
)

// Params holds parsed query parameters for pagination and sorting.
type Params struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort,omitempty"`
	Order string `json:"order,omitempty"`
}

// Meta holds pagination metadata returned in API responses.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// Parse extracts and validates pagination and sorting parameters from gin.Context.
func Parse(c *gin.Context) (Params, error) {
	p := Params{
		Page:  DefaultPage,
		Limit: DefaultLimit,
		Order: DefaultOrder,
	}

	pageStr := strings.TrimSpace(c.Query("page"))
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return p, appErrors.ErrBadRequest("Parameter 'page' harus berupa bilangan bulat positif (minimal 1)")
		}
		p.Page = page
	}

	limitStr := strings.TrimSpace(c.Query("limit"))
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return p, appErrors.ErrBadRequest("Parameter 'limit' harus berupa bilangan bulat positif (minimal 1)")
		}
		if limit > MaxLimit {
			return p, appErrors.ErrBadRequest(fmt.Sprintf("Parameter 'limit' tidak boleh melebihi batas maksimum %d", MaxLimit))
		}
		p.Limit = limit
	}

	sort := strings.TrimSpace(c.Query("sort"))
	if sort != "" {
		p.Sort = sort
	}

	order := strings.ToLower(strings.TrimSpace(c.Query("order")))
	if order != "" {
		if order != OrderAscending && order != OrderDescending {
			return p, appErrors.ErrBadRequest("Parameter 'order' hanya menerima nilai 'asc' atau 'desc'")
		}
		p.Order = order
	}

	return p, nil
}

// Offset computes the safe SQL offset, preventing integer overflow.
func (p Params) Offset() int {
	if p.Page <= 1 {
		return 0
	}

	// Guard against potential integer overflow on extreme page numbers:
	// If (p.Page - 1) > math.MaxInt / p.Limit, cap it to MaxInt.
	if p.Page-1 > math.MaxInt/p.Limit {
		return math.MaxInt
	}

	return (p.Page - 1) * p.Limit
}

// ValidateSort checks if the requested sort column is within an explicit allowlist.
// This prevents SQL injection or exposure of unintended internal column names.
func (p Params) ValidateSort(allowedFields []string) error {
	if p.Sort == "" {
		return nil
	}

	for _, field := range allowedFields {
		if strings.EqualFold(p.Sort, field) {
			return nil
		}
	}

	return appErrors.ErrBadRequest(fmt.Sprintf("Kolom sorting '%s' tidak diizinkan. Pilihan yang tersedia: [%s]",
		p.Sort, strings.Join(allowedFields, ", ")))
}

// NewMeta constructs pagination metadata safely.
func NewMeta(page, limit, totalItems int) Meta {
	totalPages := 0
	if limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	return Meta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

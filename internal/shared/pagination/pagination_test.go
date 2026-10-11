package pagination_test

import (
	"math"
	"net/http/httptest"
	"testing"

	"educore/internal/shared/pagination"

	"github.com/gin-gonic/gin"
)

func createGinContext(queryString string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test?"+queryString, nil)
	c.Request = req
	return c
}

func TestParse_Defaults(t *testing.T) {
	c := createGinContext("")
	p, err := pagination.Parse(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.Page != pagination.DefaultPage {
		t.Errorf("expected default page %d, got %d", pagination.DefaultPage, p.Page)
	}
	if p.Limit != pagination.DefaultLimit {
		t.Errorf("expected default limit %d, got %d", pagination.DefaultLimit, p.Limit)
	}
	if p.Order != pagination.DefaultOrder {
		t.Errorf("expected default order %s, got %s", pagination.DefaultOrder, p.Order)
	}
	if p.Offset() != 0 {
		t.Errorf("expected offset 0 for page 1, got %d", p.Offset())
	}
}

func TestParse_ValidCustom(t *testing.T) {
	c := createGinContext("page=3&limit=50&sort=name&order=desc")
	p, err := pagination.Parse(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.Page != 3 {
		t.Errorf("expected page 3, got %d", p.Page)
	}
	if p.Limit != 50 {
		t.Errorf("expected limit 50, got %d", p.Limit)
	}
	if p.Sort != "name" {
		t.Errorf("expected sort 'name', got %s", p.Sort)
	}
	if p.Order != "desc" {
		t.Errorf("expected order 'desc', got %s", p.Order)
	}
	if p.Offset() != 100 {
		t.Errorf("expected offset 100 ((3-1)*50), got %d", p.Offset())
	}
}

func TestParse_InvalidPage(t *testing.T) {
	cases := []string{"page=0", "page=-5", "page=abc"}
	for _, q := range cases {
		c := createGinContext(q)
		_, err := pagination.Parse(c)
		if err == nil {
			t.Errorf("expected error for query %q, got nil", q)
		}
	}
}

func TestParse_InvalidLimit(t *testing.T) {
	cases := []string{"limit=0", "limit=-1", "limit=101", "limit=xyz"}
	for _, q := range cases {
		c := createGinContext(q)
		_, err := pagination.Parse(c)
		if err == nil {
			t.Errorf("expected error for query %q, got nil", q)
		}
	}
}

func TestParse_InvalidOrder(t *testing.T) {
	c := createGinContext("order=random")
	_, err := pagination.Parse(c)
	if err == nil {
		t.Errorf("expected error for invalid order, got nil")
	}
}

func TestOffset_OverflowProtection(t *testing.T) {
	p := pagination.Params{
		Page:  math.MaxInt,
		Limit: 100,
	}

	offset := p.Offset()
	if offset != math.MaxInt {
		t.Errorf("expected capped offset math.MaxInt on overflow, got %d", offset)
	}
}

func TestValidateSort(t *testing.T) {
	allowed := []string{"created_at", "name", "nim"}

	validParams := pagination.Params{Sort: "name"}
	if err := validParams.ValidateSort(allowed); err != nil {
		t.Errorf("expected allowed sort to pass, got error: %v", err)
	}

	emptyParams := pagination.Params{Sort: ""}
	if err := emptyParams.ValidateSort(allowed); err != nil {
		t.Errorf("expected empty sort to pass, got error: %v", err)
	}

	invalidParams := pagination.Params{Sort: "password_hash"}
	if err := invalidParams.ValidateSort(allowed); err == nil {
		t.Errorf("expected disallowed sort to return error, got nil")
	}
}

func TestNewMeta(t *testing.T) {
	meta := pagination.NewMeta(2, 20, 45)
	if meta.Page != 2 {
		t.Errorf("expected page 2, got %d", meta.Page)
	}
	if meta.Limit != 20 {
		t.Errorf("expected limit 20, got %d", meta.Limit)
	}
	if meta.TotalItems != 45 {
		t.Errorf("expected total items 45, got %d", meta.TotalItems)
	}
	if meta.TotalPages != 3 {
		t.Errorf("expected total pages 3, got %d", meta.TotalPages)
	}
}

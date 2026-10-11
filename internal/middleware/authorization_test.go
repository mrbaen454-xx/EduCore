package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"educore/internal/modules/auth"
	"educore/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupRoleTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/admin-only", RequireRoles("admin"), func(c *gin.Context) {
		response.OK(c, "Admin access granted", nil, nil)
	})

	router.GET("/staff-or-dosen", RequireRoles("akademik", "dosen"), func(c *gin.Context) {
		response.OK(c, "Staff or Dosen access granted", nil, nil)
	})

	return router
}

func TestRequireRoles_Unauthenticated(t *testing.T) {
	router := setupRoleTestEngine()

	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireRoles_RoleMatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyPrincipal, &auth.Principal{
			UserID: uuid.New(),
			Email:  "admin@example.com",
			Roles:  []string{"admin"},
		})
		c.Next()
	})

	router.GET("/admin-only", RequireRoles("admin"), func(c *gin.Context) {
		response.OK(c, "Admin access granted", nil, nil)
	})

	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireRoles_ForbiddenRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	// Mock non-admin student user
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyPrincipal, &auth.Principal{
			UserID: uuid.New(),
			Email:  "student@example.com",
			Roles:  []string{"mahasiswa"},
		})
		c.Next()
	})

	router.GET("/admin-only", RequireRoles("admin"), func(c *gin.Context) {
		response.OK(c, "Should not reach here", nil, nil)
	})

	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var res response.ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Error.Code != "FORBIDDEN" {
		t.Errorf("expected error code FORBIDDEN, got %s", res.Error.Code)
	}
}

func TestRequireRoles_MultipleAllowedRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	// Mock lecturer user
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyPrincipal, &auth.Principal{
			UserID: uuid.New(),
			Email:  "dosen@example.com",
			Roles:  []string{"dosen"},
		})
		c.Next()
	})

	router.GET("/staff-or-dosen", RequireRoles("akademik", "dosen"), func(c *gin.Context) {
		response.OK(c, "Access granted", nil, nil)
	})

	req, _ := http.NewRequest(http.MethodGet, "/staff-or-dosen", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

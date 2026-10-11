package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"educore/internal/modules/auth"
)

func TestAuthorization_RBACAdminEndpoint(t *testing.T) {
	server, _, _ := setupAuthIntegrationApp(t)

	// 1. Unauthenticated request -> 401
	anonReq, _ := http.NewRequest(http.MethodGet, "/api/v1/test/admin-only", nil)
	anonRec := httptest.NewRecorder()
	server.Router().ServeHTTP(anonRec, anonReq)
	if anonRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous request, got %d", anonRec.Code)
	}

	// 2. Login as student (role: mahasiswa)
	studentLogin := auth.LoginRequest{
		Identifier: "student@example.com",
		Password:   "StudentSecret123!",
	}
	studentBody, _ := json.Marshal(studentLogin)
	studentLoginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(studentBody))
	studentLoginReq.Header.Set("Content-Type", "application/json")
	studentLoginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(studentLoginRec, studentLoginReq)

	var studentRes struct {
		Data auth.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(studentLoginRec.Body.Bytes(), &studentRes)
	studentToken := studentRes.Data.AccessToken

	// Access admin endpoint as student -> 403 Forbidden
	forbiddenReq, _ := http.NewRequest(http.MethodGet, "/api/v1/test/admin-only", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+studentToken)
	forbiddenRec := httptest.NewRecorder()
	server.Router().ServeHTTP(forbiddenRec, forbiddenReq)

	if forbiddenRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for student accessing admin endpoint, got %d: %s", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	// 3. Login as admin (role: admin)
	adminLogin := auth.LoginRequest{
		Identifier: "admin@example.com",
		Password:   "AdminSecret123!",
	}
	adminBody, _ := json.Marshal(adminLogin)
	adminLoginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(adminBody))
	adminLoginReq.Header.Set("Content-Type", "application/json")
	adminLoginRec := httptest.NewRecorder()
	server.Router().ServeHTTP(adminLoginRec, adminLoginReq)

	var adminRes struct {
		Data auth.LoginResponse `json:"data"`
	}
	_ = json.Unmarshal(adminLoginRec.Body.Bytes(), &adminRes)
	adminToken := adminRes.Data.AccessToken

	// Access admin endpoint as admin -> 200 OK
	adminReq, _ := http.NewRequest(http.MethodGet, "/api/v1/test/admin-only", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminToken)
	adminRec := httptest.NewRecorder()
	server.Router().ServeHTTP(adminRec, adminReq)

	if adminRec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin accessing admin endpoint, got %d: %s", adminRec.Code, adminRec.Body.String())
	}
}

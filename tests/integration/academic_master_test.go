package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"educore/internal/modules/auth"
	"educore/internal/platform/database"
	"educore/internal/platform/httpserver"
	"educore/internal/platform/logger"
	"educore/internal/shared/response"

	"github.com/google/uuid"
)

type academicTestHarness struct {
	server       *httpserver.Server
	db           *database.DB
	tokenMgr     *auth.TokenManager
	adminToken   string
	studentToken string
}

func setupAcademicTestApp(t *testing.T) *academicTestHarness {
	cfg := getTestDBConfig()
	log := logger.New("error", "testing")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Ensure migrations are up to version 3
	_ = database.MigrateUp("../../migrations", cfg.DSN())

	tokenMgr := auth.NewTokenManager(cfg)

	// Clean academic tables in reverse dependency order
	_, _ = db.Pool.Exec(ctx, "DELETE FROM class_schedules;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM class_lecturers;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM classes;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM course_offerings;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM academic_periods;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM course_prerequisites;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM curriculum_courses;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM curricula;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM courses;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM students;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM lecturers;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM study_programs;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM faculties;")

	// Create tokens for admin and student roles
	adminToken, err := tokenMgr.GenerateAccessToken(&auth.User{
		ID:    uuid.New(),
		Roles: []string{"admin"},
		Email: "admin@educore.ac.id",
		Name:  "Admin Test",
	})
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	studentToken, err := tokenMgr.GenerateAccessToken(&auth.User{
		ID:    uuid.New(),
		Roles: []string{"mahasiswa"},
		Email: "mhs@educore.ac.id",
		Name:  "Mahasiswa Test",
	})
	if err != nil {
		t.Fatalf("failed to generate student token: %v", err)
	}

	server := httpserver.New(cfg, log, db.Pool)

	return &academicTestHarness{
		server:       server,
		db:           db,
		tokenMgr:     tokenMgr,
		adminToken:   adminToken,
		studentToken: studentToken,
	}
}

func TestAcademicMaster_IntegrationLifecycle(t *testing.T) {
	h := setupAcademicTestApp(t)
	defer h.db.Close()

	var facultyID string
	var studyProgramID string
	var lecturerID string
	var studentID string

	// 1. Create Faculty (Admin) -> 201 Created
	t.Run("Create Faculty - Success", func(t *testing.T) {
		body := map[string]any{
			"code":        "FT",
			"name":        "Fakultas Teknik",
			"description": "Fakultas teknik dan rekayasa",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/faculties", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		facultyID = dataMap["id"].(string)
		if dataMap["code"] != "FT" {
			t.Errorf("expected code FT, got %v", dataMap["code"])
		}
	})

	// 2. Reject Duplicate Faculty Code -> 409 Conflict
	t.Run("Create Faculty - Duplicate Code", func(t *testing.T) {
		body := map[string]any{
			"code": "ft",
			"name": "Fakultas Teknologi",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/faculties", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 3. Read Faculty List & Detail
	t.Run("Get Faculty Detail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/faculties/"+facultyID, nil)
		req.Header.Set("Authorization", "Bearer "+h.studentToken) // Read accessible by student
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 4. Create Study Program (Admin) -> 201 Created
	t.Run("Create Study Program - Success", func(t *testing.T) {
		body := map[string]any{
			"faculty_id":  facultyID,
			"code":        "TIF",
			"name":        "Teknik Informatika",
			"level":       "S1",
			"description": "Program studi Sarjana Teknik Informatika",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/study-programs", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		studyProgramID = dataMap["id"].(string)
		if dataMap["code"] != "TIF" {
			t.Errorf("expected code TIF, got %v", dataMap["code"])
		}
	})

	// 5. Reject Study Program with Non-existent Faculty -> 404 Not Found
	t.Run("Create Study Program - Unknown Faculty", func(t *testing.T) {
		body := map[string]any{
			"faculty_id": uuid.New().String(),
			"code":       "SI",
			"name":       "Sistem Informasi",
			"level":      "S1",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/study-programs", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 6. Create Lecturer -> 201 Created
	t.Run("Create Lecturer - Success", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": studyProgramID,
			"nidn":             "0012345678",
			"name":             "Prof. Dr. Ir. Joko Susilo, M.T.",
			"email":            "joko@educore.ac.id",
			"phone":            "081234567890",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/lecturers", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		lecturerID = dataMap["id"].(string)
		if dataMap["nidn"] != "0012345678" {
			t.Errorf("expected NIDN 0012345678, got %v", dataMap["nidn"])
		}
	})

	// 7. Reject Duplicate Lecturer NIDN -> 409 Conflict
	t.Run("Create Lecturer - Duplicate NIDN", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": studyProgramID,
			"nidn":             "0012345678",
			"name":             "Dosen Lain",
			"email":            "lain@educore.ac.id",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/lecturers", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 8. Create Student -> 201 Created
	t.Run("Create Student - Success", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": studyProgramID,
			"nim":              "2026101001",
			"name":             "Ananda Putri",
			"email":            "ananda@student.educore.ac.id",
			"gender":           "FEMALE",
			"birth_date":       "2004-08-20",
			"admission_year":   2026,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		studentID = dataMap["id"].(string)
		if dataMap["nim"] != "2026101001" {
			t.Errorf("expected NIM 2026101001, got %v", dataMap["nim"])
		}
	})

	// 9. Reject Duplicate Student NIM -> 409 Conflict
	t.Run("Create Student - Duplicate NIM", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": studyProgramID,
			"nim":              "2026101001",
			"name":             "Mahasiswa Kembar",
			"email":            "kembar@student.educore.ac.id",
			"admission_year":   2026,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 10. Role Protection: Student role cannot mutate faculties -> 403 Forbidden
	t.Run("Role Protection - Student Forbidden from Mutation", func(t *testing.T) {
		body := map[string]any{
			"code": "FK",
			"name": "Fakultas Kedokteran",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/faculties", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.studentToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	// 11. Anonymous Protection: Request without token -> 401 Unauthorized
	t.Run("Anonymous Protection - No Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	// 12. Filtering & Pagination
	t.Run("Filtering and Pagination on Students", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/students?search=Ananda&admission_year=2026&page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataList := resp.Data.([]any)
		if len(dataList) != 1 {
			t.Errorf("expected 1 student matching filter, got %d", len(dataList))
		}
	})

	// 13. Update Status Lifecycle
	t.Run("Update Student Status", func(t *testing.T) {
		body := map[string]any{"status": "SUSPENDED"}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/students/%s/status", studentID), bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	_ = lecturerID
}

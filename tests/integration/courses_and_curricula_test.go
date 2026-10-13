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

type academicCoursesTestHarness struct {
	server       *httpserver.Server
	db           *database.DB
	tokenMgr     *auth.TokenManager
	adminToken   string
	studentToken string
}

func setupCoursesAndCurriculaTestApp(t *testing.T) *academicCoursesTestHarness {
	cfg := getTestDBConfig()
	log := logger.New("error", "testing")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Ensure migrations are up to latest
	_ = database.MigrateUp("../../migrations", cfg.DSN())

	tokenMgr := auth.NewTokenManager(cfg)

	// Clean tables in reverse dependency order
	_, _ = db.Pool.Exec(ctx, "DELETE FROM enrollments;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM study_plans;")
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
		Email: "student@educore.ac.id",
		Name:  "Student Test",
	})
	if err != nil {
		t.Fatalf("failed to generate student token: %v", err)
	}

	server := httpserver.New(cfg, log, db.Pool)

	return &academicCoursesTestHarness{
		server:       server,
		db:           db,
		tokenMgr:     tokenMgr,
		adminToken:   adminToken,
		studentToken: studentToken,
	}
}

func TestCoursesAndCurricula_IntegrationLifecycle(t *testing.T) {
	h := setupCoursesAndCurriculaTestApp(t)
	defer h.db.Close()
	ctx := context.Background()

	// Setup active Faculty and Study Program directly in database
	var facultyID, studyProgramID string
	_ = h.db.Pool.QueryRow(ctx, `
		INSERT INTO faculties (code, name, is_active)
		VALUES ('FT', 'Fakultas Teknik', true)
		RETURNING id;
	`).Scan(&facultyID)

	_ = h.db.Pool.QueryRow(ctx, `
		INSERT INTO study_programs (faculty_id, code, name, level, is_active)
		VALUES ($1, 'TIF', 'Teknik Informatika', 'S1', true)
		RETURNING id;
	`, facultyID).Scan(&studyProgramID)

	var course1ID, course2ID string
	var curriculumID string

	// 1. Create Course 1 (Algoritma dan Pemrograman - 3 SKS) -> 201 Created
	t.Run("Create Course 1 - Success", func(t *testing.T) {
		body := map[string]any{
			"code":        "tif101",
			"name":        "Algoritma dan Pemrograman",
			"credits":     3,
			"description": "Dasar-dasar logika dan sintaks pemrograman",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewReader(jsonBytes))
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
		course1ID = dataMap["id"].(string)
		if dataMap["code"] != "TIF101" {
			t.Errorf("expected uppercase code TIF101, got %v", dataMap["code"])
		}
		if dataMap["credits"] != float64(3) {
			t.Errorf("expected 3 credits, got %v", dataMap["credits"])
		}
	})

	// 2. Reject Duplicate Course Code -> 409 Conflict
	t.Run("Create Course - Duplicate Code", func(t *testing.T) {
		body := map[string]any{
			"code":    "TIF101",
			"name":    "Alpro Lain",
			"credits": 4,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 3. Reject Invalid Credits (0 or > 8) -> 400 Bad Request
	t.Run("Create Course - Invalid Credits", func(t *testing.T) {
		body := map[string]any{
			"code":    "TIF999",
			"name":    "Mata Kuliah Aneh",
			"credits": 0,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	// 4. Create Course 2 (Struktur Data - 4 SKS) -> 201 Created
	t.Run("Create Course 2 - Success", func(t *testing.T) {
		body := map[string]any{
			"code":    "TIF102",
			"name":    "Struktur Data dan Algoritma",
			"credits": 4,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewReader(jsonBytes))
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
		course2ID = dataMap["id"].(string)
	})

	// 5. Add Prerequisite: TIF101 is prerequisite of TIF102 -> 201 Created
	t.Run("Add Prerequisite - Success", func(t *testing.T) {
		body := map[string]any{
			"prerequisite_course_id": course1ID,
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/courses/%s/prerequisites", course2ID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 6. Reject Self Prerequisite -> 400 Bad Request
	t.Run("Reject Self Prerequisite", func(t *testing.T) {
		body := map[string]any{
			"prerequisite_course_id": course1ID,
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/courses/%s/prerequisites", course1ID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	// 7. Reject Circular Prerequisite -> 409 Conflict
	t.Run("Reject Circular Prerequisite", func(t *testing.T) {
		body := map[string]any{
			"prerequisite_course_id": course2ID,
		}
		jsonBytes, _ := json.Marshal(body)

		// TIF101 cannot have TIF102 as prerequisite because TIF101 is already prerequisite of TIF102
		url := fmt.Sprintf("/api/v1/courses/%s/prerequisites", course1ID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 8. Create Curriculum -> 201 Created
	t.Run("Create Curriculum - Success", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": studyProgramID,
			"code":             "KUR-TIF-2024",
			"name":             "Kurikulum Berbasis OBE Teknik Informatika 2024",
			"year":             2024,
			"description":      "Kurikulum nasional 2024",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/curricula", bytes.NewReader(jsonBytes))
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
		curriculumID = dataMap["id"].(string)
		if dataMap["code"] != "KUR-TIF-2024" {
			t.Errorf("expected KUR-TIF-2024, got %v", dataMap["code"])
		}
		if dataMap["status"] != "DRAFT" {
			t.Errorf("expected DRAFT status, got %v", dataMap["status"])
		}
	})

	// 9. Reject Curriculum with Non-existent Study Program -> 404 Not Found
	t.Run("Create Curriculum - Unknown Study Program", func(t *testing.T) {
		body := map[string]any{
			"study_program_id": uuid.New().String(),
			"code":             "KUR-UNKNOWN",
			"name":             "Kurikulum Fiktif",
			"year":             2024,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/curricula", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})

	// 10. Associate Course 1 (REQUIRED) to Curriculum -> 201 Created
	t.Run("Add Course 1 to Curriculum - Success", func(t *testing.T) {
		body := map[string]any{
			"course_id": course1ID,
			"semester":  1,
			"category":  "REQUIRED",
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/curricula/%s/courses", curriculumID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 11. Associate Course 2 (ELECTIVE) to Curriculum -> 201 Created
	t.Run("Add Course 2 to Curriculum - Success", func(t *testing.T) {
		body := map[string]any{
			"course_id": course2ID,
			"semester":  2,
			"category":  "ELECTIVE",
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/curricula/%s/courses", curriculumID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 12. Reject Duplicate Course in Curriculum -> 409 Conflict
	t.Run("Reject Duplicate Course in Curriculum", func(t *testing.T) {
		body := map[string]any{
			"course_id": course1ID,
			"semester":  3,
			"category":  "REQUIRED",
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/curricula/%s/courses", curriculumID)
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d", w.Code)
		}
	})

	// 13. Verify Total Credits and Stats Aggregation -> 200 OK
	t.Run("Verify Curriculum Credits Aggregation", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/curricula/%s", curriculumID)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+h.studentToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		stats := dataMap["stats"].(map[string]any)

		// Course 1 (3 SKS) + Course 2 (4 SKS) = 7 SKS total
		if stats["total_courses"] != float64(2) {
			t.Errorf("expected 2 courses, got %v", stats["total_courses"])
		}
		if stats["total_credits"] != float64(7) {
			t.Errorf("expected 7 total credits, got %v", stats["total_credits"])
		}
		if stats["required_credits"] != float64(3) {
			t.Errorf("expected 3 required credits, got %v", stats["required_credits"])
		}
		if stats["elective_credits"] != float64(4) {
			t.Errorf("expected 4 elective credits, got %v", stats["elective_credits"])
		}
	})

	// 14. Update Curriculum Status (DRAFT -> ACTIVE) -> 200 OK
	t.Run("Update Curriculum Status to ACTIVE", func(t *testing.T) {
		body := map[string]any{
			"status": "ACTIVE",
		}
		jsonBytes, _ := json.Marshal(body)

		url := fmt.Sprintf("/api/v1/curricula/%s/status", curriculumID)
		req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 15. Role Protection: Student role cannot mutate courses -> 403 Forbidden
	t.Run("Role Protection - Student Denied Mutation", func(t *testing.T) {
		body := map[string]any{
			"code":    "TIF201",
			"name":    "Basis Data",
			"credits": 3,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.studentToken)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	// 16. Anonymous Protection: Unauthenticated request rejected -> 401 Unauthorized
	t.Run("Anonymous Protection - No Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
		w := httptest.NewRecorder()
		h.server.Engine().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}

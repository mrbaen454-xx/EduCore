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

type classesTestHarness struct {
	server       *httpserver.Server
	db           *database.DB
	tokenMgr     *auth.TokenManager
	adminToken   string
	studentToken string
}

func setupClassesAndSchedulesTestApp(t *testing.T) *classesTestHarness {
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

	srv := httpserver.New(cfg, log, db.Pool)

	return &classesTestHarness{
		server:       srv,
		db:           db,
		tokenMgr:     tokenMgr,
		adminToken:   adminToken,
		studentToken: studentToken,
	}
}

func (h *classesTestHarness) executeRequest(method, url, token string, body any) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.server.Engine().ServeHTTP(rec, req)
	return rec
}

func TestClassesAndSchedules_IntegrationLifecycle(t *testing.T) {
	h := setupClassesAndSchedulesTestApp(t)
	defer h.db.Close()
	ctx := context.Background()

	// Seed Faculty, Study Program, Lecturer, and Course
	facultyID := uuid.New()
	_, err := h.db.Pool.Exec(ctx, `
		INSERT INTO faculties (id, code, name, is_active)
		VALUES ($1, 'FT', 'Fakultas Teknik', true);
	`, facultyID)
	if err != nil {
		t.Fatalf("failed to seed faculty: %v", err)
	}

	prodiID := uuid.New()
	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO study_programs (id, faculty_id, code, name, level, is_active)
		VALUES ($1, $2, 'TIF', 'Teknik Informatika', 'S1', true);
	`, prodiID, facultyID)
	if err != nil {
		t.Fatalf("failed to seed study program: %v", err)
	}

	lecturerID := uuid.New()
	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO lecturers (id, study_program_id, nidn, name, email, status)
		VALUES ($1, $2, '0011223344', 'Dr. Budi Santoso', 'budi@educore.ac.id', 'ACTIVE');
	`, lecturerID, prodiID)
	if err != nil {
		t.Fatalf("failed to seed lecturer: %v", err)
	}

	courseID := uuid.New()
	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO courses (id, code, name, credits, is_active)
		VALUES ($1, 'TIF201', 'Pemrograman Berorientasi Objek', 3, true);
	`, courseID)
	if err != nil {
		t.Fatalf("failed to seed course: %v", err)
	}

	var periodID string
	var offeringID string
	var classAID string
	var classBID string

	// 1. Create Academic Period
	t.Run("Create Academic Period - Success", func(t *testing.T) {
		body := map[string]any{
			"code":          "20241",
			"name":          "Semester Ganjil 2024/2025",
			"semester_type": "ODD",
			"start_date":    "2024-09-01",
			"end_date":      "2025-01-31",
			"is_active":     true,
		}
		rec := h.executeRequest("POST", "/api/v1/academic-periods", h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		periodID = dataMap["id"].(string)
	})

	// 2. Reject Invalid Dates on Academic Period (start > end)
	t.Run("Reject Invalid Dates on Academic Period", func(t *testing.T) {
		body := map[string]any{
			"code":          "20242",
			"name":          "Semester Genap Terbalik",
			"semester_type": "EVEN",
			"start_date":    "2025-06-01",
			"end_date":      "2025-02-01",
		}
		rec := h.executeRequest("POST", "/api/v1/academic-periods", h.adminToken, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 3. Create Course Offering
	t.Run("Create Course Offering - Success", func(t *testing.T) {
		body := map[string]any{
			"academic_period_id": periodID,
			"course_id":          courseID.String(),
			"study_program_id":   prodiID.String(),
		}
		rec := h.executeRequest("POST", "/api/v1/course-offerings", h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		offeringID = dataMap["id"].(string)
		if dataMap["status"] != "PLANNED" {
			t.Errorf("expected PLANNED status, got %v", dataMap["status"])
		}
	})

	// 4. Reject Duplicate Course Offering
	t.Run("Reject Duplicate Course Offering", func(t *testing.T) {
		body := map[string]any{
			"academic_period_id": periodID,
			"course_id":          courseID.String(),
			"study_program_id":   prodiID.String(),
		}
		rec := h.executeRequest("POST", "/api/v1/course-offerings", h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 5. Create Class A
	t.Run("Create Class A - Success", func(t *testing.T) {
		body := map[string]any{
			"course_offering_id": offeringID,
			"code":               "A",
			"name":               "Kelas A - Reguler",
			"capacity":           40,
		}
		rec := h.executeRequest("POST", "/api/v1/classes", h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		classAID = dataMap["id"].(string)
		if dataMap["code"] != "A" || dataMap["capacity"] != float64(40) {
			t.Errorf("unexpected class attributes: %v", dataMap)
		}
	})

	// 6. Reject Class with Zero/Negative Capacity
	t.Run("Reject Class with Zero Capacity", func(t *testing.T) {
		body := map[string]any{
			"course_offering_id": offeringID,
			"code":               "B",
			"name":               "Kelas B",
			"capacity":           0,
		}
		rec := h.executeRequest("POST", "/api/v1/classes", h.adminToken, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 7. Reject Duplicate Class Code under Same Offering
	t.Run("Reject Duplicate Class Code", func(t *testing.T) {
		body := map[string]any{
			"course_offering_id": offeringID,
			"code":               "A",
			"name":               "Kelas A Duplikat",
			"capacity":           30,
		}
		rec := h.executeRequest("POST", "/api/v1/classes", h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 8. Assign Lecturer to Class A
	t.Run("Assign Lecturer to Class A - Success", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/lecturers", classAID)
		body := map[string]any{
			"lecturer_id": lecturerID.String(),
			"role":        "COORDINATOR",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 9. Reject Duplicate Lecturer Assignment on Same Class
	t.Run("Reject Duplicate Lecturer Assignment", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/lecturers", classAID)
		body := map[string]any{
			"lecturer_id": lecturerID.String(),
			"role":        "TEAM_TEACHING",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 10. Create Schedule 1 for Class A (Monday 08:00 - 10:00, Room LAB-1)
	t.Run("Create Schedule for Class A - Success", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classAID)
		body := map[string]any{
			"day_of_week":   1, // Monday
			"start_time":    "08:00",
			"end_time":      "10:00",
			"room":          "LAB-1",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 11. Reject Schedule with Start Time >= End Time
	t.Run("Reject Schedule with Start Time >= End Time", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classAID)
		body := map[string]any{
			"day_of_week":   2,
			"start_time":    "14:00",
			"end_time":      "12:00",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 12. Reject Self Overlap on Same Class (Monday 09:00 - 11:00 overlaps 08:00 - 10:00)
	t.Run("Reject Schedule Overlap on Same Class", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classAID)
		body := map[string]any{
			"day_of_week":   1,
			"start_time":    "09:00",
			"end_time":      "11:00",
			"room":          "LAB-2",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 13. Create Class B under Same Offering
	t.Run("Create Class B - Success", func(t *testing.T) {
		body := map[string]any{
			"course_offering_id": offeringID,
			"code":               "B",
			"name":               "Kelas B - Reguler",
			"capacity":           35,
		}
		rec := h.executeRequest("POST", "/api/v1/classes", h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp response.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		dataMap := resp.Data.(map[string]any)
		classBID = dataMap["id"].(string)

		// Assign same lecturer to Class B
		assignURL := fmt.Sprintf("/api/v1/classes/%s/lecturers", classBID)
		recAssign := h.executeRequest("POST", assignURL, h.adminToken, map[string]any{
			"lecturer_id": lecturerID.String(),
			"role":        "TEAM_TEACHING",
		})
		if recAssign.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for lecturer assign on Class B: %s", recAssign.Body.String())
		}
	})

	// 14. Reject Lecturer Schedule Conflict across Classes in Same Period
	// Class A has schedule Monday 08:00 - 10:00 with this lecturer.
	// Class B attempts Monday 09:30 - 11:30 with same lecturer.
	t.Run("Reject Lecturer Schedule Conflict Across Classes", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classBID)
		body := map[string]any{
			"day_of_week":   1,
			"start_time":    "09:30",
			"end_time":      "11:30",
			"room":          "LAB-3",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 15. Reject Room Schedule Conflict in Same Period
	// Class A has Monday 08:00 - 10:00 in Room LAB-1.
	// Class B attempts Monday 08:30 - 10:30 in Room LAB-1.
	t.Run("Reject Room Schedule Conflict in Same Period", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classBID)
		body := map[string]any{
			"day_of_week":   1,
			"start_time":    "08:30",
			"end_time":      "10:30",
			"room":          "LAB-1",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 16. Allow Consecutive Schedule (Monday 10:00 - 12:00 in same room)
	// Half-open interval [08:00, 10:00) and [10:00, 12:00) do NOT overlap.
	t.Run("Allow Consecutive Schedule on Same Room", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/classes/%s/schedules", classBID)
		body := map[string]any{
			"day_of_week":   1,
			"start_time":    "10:00",
			"end_time":      "12:00",
			"room":          "LAB-1",
			"delivery_mode": "OFFLINE",
		}
		rec := h.executeRequest("POST", url, h.adminToken, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for consecutive schedule, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 17. Role Protection: Student Denied from Creating Classes and Schedules
	t.Run("Role Protection - Student Denied from Mutation", func(t *testing.T) {
		body := map[string]any{
			"course_offering_id": offeringID,
			"code":               "C",
			"name":               "Kelas C",
			"capacity":           30,
		}
		rec := h.executeRequest("POST", "/api/v1/classes", h.studentToken, body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 18. Anonymous Protection: Request without Token Rejected
	t.Run("Anonymous Protection - No Token", func(t *testing.T) {
		rec := h.executeRequest("GET", "/api/v1/classes", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

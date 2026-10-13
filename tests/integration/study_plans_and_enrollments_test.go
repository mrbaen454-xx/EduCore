package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"educore/internal/modules/auth"
	"educore/internal/platform/database"
	"educore/internal/platform/httpserver"
	"educore/internal/platform/logger"

	"github.com/google/uuid"
)

type studyPlansTestHarness struct {
	server       *httpserver.Server
	db           *database.DB
	tokenMgr     *auth.TokenManager
	adminToken   string
	student1     *testStudent
	student2     *testStudent
	inactiveStud *testStudent
	activePeriod uuid.UUID
	facultyID    uuid.UUID
	prodiID      uuid.UUID
}

type testStudent struct {
	id     uuid.UUID
	userID uuid.UUID
	token  string
	nim    string
	status string
}

func setupStudyPlansTestApp(t *testing.T) *studyPlansTestHarness {
	cfg := getTestDBConfig()
	log := logger.New("error", "testing")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	_ = database.MigrateUp("../../migrations", cfg.DSN())

	tokenMgr := auth.NewTokenManager(cfg)

	// Clean tables in reverse dependency order
	_, _ = db.Pool.Exec(ctx, "DELETE FROM attendances;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM class_sessions;")
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
	_, _ = db.Pool.Exec(ctx, "DELETE FROM user_roles;")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE email LIKE '%@test.educore.ac.id';")

	// 1. Setup Admin Token
	adminUserID := uuid.New()
	_, _ = db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, is_active)
		VALUES ($1, 'admin@test.educore.ac.id', 'dummyhash', 'Admin Tester', true)
	`, adminUserID)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 'admin')`, adminUserID)

	adminToken, err := tokenMgr.GenerateAccessToken(&auth.User{
		ID:    adminUserID,
		Roles: []string{"admin"},
		Email: "admin@test.educore.ac.id",
		Name:  "Admin Tester",
	})
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	// 2. Setup Faculty and Study Program
	var facultyID, prodiID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO faculties (code, name) VALUES ('FTI', 'Fakultas Teknologi Informasi') RETURNING id
	`).Scan(&facultyID)
	if err != nil {
		t.Fatalf("failed to insert faculty: %v", err)
	}

	err = db.Pool.QueryRow(ctx, `
		INSERT INTO study_programs (faculty_id, code, name, level)
		VALUES ($1, 'IF', 'Informatika', 'S1') RETURNING id
	`, facultyID).Scan(&prodiID)
	if err != nil {
		t.Fatalf("failed to insert study program: %v", err)
	}

	// 3. Setup Active Academic Period
	var activePeriodID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO academic_periods (code, name, semester_type, start_date, end_date, is_active)
		VALUES ('20241', 'Ganjil 2024/2025', 'ODD', '2024-09-01', '2025-01-31', true)
		RETURNING id
	`).Scan(&activePeriodID)
	if err != nil {
		t.Fatalf("failed to insert active period: %v", err)
	}

	// Helper to create student with linked user and token
	createStudent := func(nim, name, email, status string) *testStudent {
		userID := uuid.New()
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, name, is_active)
			VALUES ($1, $2, 'dummyhash', $3, true)
		`, userID, email, name)
		if err != nil {
			t.Fatalf("failed to insert user %s: %v", email, err)
		}

		_, _ = db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 'mahasiswa')`, userID)

		var sID uuid.UUID
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO students (user_id, study_program_id, nim, name, email, admission_year, status)
			VALUES ($1, $2, $3, $4, $5, 2024, $6)
			RETURNING id
		`, userID, prodiID, nim, name, email, status).Scan(&sID)
		if err != nil {
			t.Fatalf("failed to insert student %s: %v", nim, err)
		}

		tok, err := tokenMgr.GenerateAccessToken(&auth.User{
			ID:    userID,
			Roles: []string{"mahasiswa"},
			Email: email,
			Name:  name,
		})
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		return &testStudent{
			id:     sID,
			userID: userID,
			token:  tok,
			nim:    nim,
			status: status,
		}
	}

	student1 := createStudent("24001", "Budi Santoso", "budi@test.educore.ac.id", "ACTIVE")
	student2 := createStudent("24002", "Siti Rahma", "siti@test.educore.ac.id", "ACTIVE")
	inactiveStud := createStudent("24003", "Andi Cuti", "andi@test.educore.ac.id", "INACTIVE")

	server := httpserver.New(cfg, log, db.Pool)

	return &studyPlansTestHarness{
		server:       server,
		db:           db,
		tokenMgr:     tokenMgr,
		adminToken:   adminToken,
		student1:     student1,
		student2:     student2,
		inactiveStud: inactiveStud,
		activePeriod: activePeriodID,
		facultyID:    facultyID,
		prodiID:      prodiID,
	}
}

func (h *studyPlansTestHarness) createCourseAndClass(t *testing.T, courseCode, courseName string, credits int, classCode, className string, capacity int) (courseID, offeringID, classID uuid.UUID) {
	ctx := context.Background()

	err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO courses (code, name, credits, is_active)
		VALUES ($1, $2, $3, true) RETURNING id
	`, courseCode, courseName, credits).Scan(&courseID)
	if err != nil {
		t.Fatalf("failed to insert course: %v", err)
	}

	err = h.db.Pool.QueryRow(ctx, `
		INSERT INTO course_offerings (academic_period_id, course_id, study_program_id, status)
		VALUES ($1, $2, $3, 'OPEN') RETURNING id
	`, h.activePeriod, courseID, h.prodiID).Scan(&offeringID)
	if err != nil {
		t.Fatalf("failed to insert offering: %v", err)
	}

	err = h.db.Pool.QueryRow(ctx, `
		INSERT INTO classes (course_offering_id, code, name, capacity, status)
		VALUES ($1, $2, $3, $4, 'OPEN') RETURNING id
	`, offeringID, classCode, className, capacity).Scan(&classID)
	if err != nil {
		t.Fatalf("failed to insert class: %v", err)
	}

	return
}

func (h *studyPlansTestHarness) addScheduleToClass(t *testing.T, classID uuid.UUID, day int, start, end, room string) uuid.UUID {
	ctx := context.Background()
	var scheduleID uuid.UUID
	err := h.db.Pool.QueryRow(ctx, `
		INSERT INTO class_schedules (class_id, day_of_week, start_time, end_time, room, delivery_mode)
		VALUES ($1, $2, $3::time, $4::time, $5, 'OFFLINE') RETURNING id
	`, classID, day, start, end, room).Scan(&scheduleID)
	if err != nil {
		t.Fatalf("failed to add schedule: %v", err)
	}
	return scheduleID
}

func TestStudyPlans_EndToEndLifecycle(t *testing.T) {
	h := setupStudyPlansTestApp(t)

	_, algoOfferingID, algoClassAID := h.createCourseAndClass(t, "IF101", "Algoritma & Pemrograman", 4, "A", "Kelas Algoritma A", 30)
	h.addScheduleToClass(t, algoClassAID, 1, "08:00", "10:30", "Lab Komputer 1")

	// Class B under the same course offering
	var algoClassBID uuid.UUID
	err := h.db.Pool.QueryRow(context.Background(), `
		INSERT INTO classes (course_offering_id, code, name, capacity, status)
		VALUES ($1, 'B', 'Kelas Algoritma B', 30, 'OPEN') RETURNING id
	`, algoOfferingID).Scan(&algoClassBID)
	if err != nil {
		t.Fatalf("failed to insert class B: %v", err)
	}

	_, _, matdisClassID := h.createCourseAndClass(t, "IF102", "Matematika Diskrit", 3, "A", "Kelas Matdis A", 30)
	h.addScheduleToClass(t, matdisClassID, 1, "10:30", "13:00", "Ruang 201") // Selesai 10:30, mulai 10:30 -> TIDAK BENTROK

	_, _, bentrokClassID := h.createCourseAndClass(t, "IF103", "Struktur Data", 3, "A", "Kelas Struktur Data A", 30)
	h.addScheduleToClass(t, bentrokClassID, 1, "09:00", "11:30", "Ruang 202") // 09:00-11:30 BENTROK dengan 08:00-10:30!

	// 1. Student browsing available classes
	t.Run("1. Browse Available Classes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/student/course-offerings", nil)
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data []map[string]any `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Data) < 3 {
			t.Errorf("expected at least 3 available classes, got %d", len(resp.Data))
		}
	})

	// 2. Inactive student rejected from creating study plan
	t.Run("2. Inactive Student Rejected", func(t *testing.T) {
		payload := map[string]any{"academic_period_id": h.activePeriod}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/student/study-plans", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.inactiveStud.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for inactive student, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Active student creates study plan successfully
	var studyPlan1ID uuid.UUID
	t.Run("3. Create Study Plan Successfully", func(t *testing.T) {
		payload := map[string]any{"academic_period_id": h.activePeriod}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/student/study-plans", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID           uuid.UUID `json:"id"`
				TotalCredits int       `json:"total_credits"`
				Status       string    `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		studyPlan1ID = resp.Data.ID
		if studyPlan1ID == uuid.Nil {
			t.Fatalf("expected valid study plan ID")
		}
		if resp.Data.TotalCredits != 0 {
			t.Errorf("expected initial 0 credits, got %d", resp.Data.TotalCredits)
		}
	})

	// 4. Idempotency: creating again returns existing study plan
	t.Run("4. Idempotent Create Study Plan", func(t *testing.T) {
		payload := map[string]any{"academic_period_id": h.activePeriod}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/student/study-plans", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for existing study plan, got %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Data struct {
				ID uuid.UUID `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.ID != studyPlan1ID {
			t.Errorf("expected existing ID %v, got %v", studyPlan1ID, resp.Data.ID)
		}
	})

	// 5. Ownership: student 2 cannot view or modify student 1's study plan
	t.Run("5. Ownership Enforcement", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/student/study-plans/%s", studyPlan1ID), nil)
		req.Header.Set("Authorization", "Bearer "+h.student2.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cross-student access, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Enroll class into study plan
	var enrollment1ID uuid.UUID
	t.Run("6. Enroll Class Successfully", func(t *testing.T) {
		payload := map[string]any{"class_id": algoClassAID}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", studyPlan1ID), bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID            uuid.UUID `json:"id"`
				CourseCredits int       `json:"course_credits"`
				Status        string    `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		enrollment1ID = resp.Data.ID
		if enrollment1ID == uuid.Nil {
			t.Fatalf("expected valid enrollment ID")
		}
		if resp.Data.CourseCredits != 4 {
			t.Errorf("expected 4 credits, got %d", resp.Data.CourseCredits)
		}

		// Verify study plan total credits updated in DB
		var totalCredits int
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT total_credits FROM study_plans WHERE id = $1", studyPlan1ID).Scan(&totalCredits)
		if totalCredits != 4 {
			t.Errorf("expected total credits 4, got %d", totalCredits)
		}
	})

	// 7. Prevent duplicate enrollment in the same class and same course
	t.Run("7. Prevent Duplicate Class and Course Enrollment", func(t *testing.T) {
		// 7a. Duplicate exact class
		payload := map[string]any{"class_id": algoClassAID}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", studyPlan1ID), bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for duplicate class, got %d: %s", w.Code, w.Body.String())
		}

		// 7b. Duplicate same course via Class B
		payloadB := map[string]any{"class_id": algoClassBID}
		bB, _ := json.Marshal(payloadB)
		reqB := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", studyPlan1ID), bytes.NewReader(bB))
		reqB.Header.Set("Authorization", "Bearer "+h.student1.token)
		reqB.Header.Set("Content-Type", "application/json")
		wB := httptest.NewRecorder()
		h.server.Router().ServeHTTP(wB, reqB)

		if wB.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for duplicate course, got %d: %s", wB.Code, wB.Body.String())
		}
	})

	// 8. Enroll non-conflicting consecutive class
	t.Run("8. Enroll Consecutive Non-Conflicting Class", func(t *testing.T) {
		payload := map[string]any{"class_id": matdisClassID}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", studyPlan1ID), bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for consecutive class, got %d: %s", w.Code, w.Body.String())
		}

		var totalCredits int
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT total_credits FROM study_plans WHERE id = $1", studyPlan1ID).Scan(&totalCredits)
		if totalCredits != 7 { // 4 + 3 = 7
			t.Errorf("expected total credits 7, got %d", totalCredits)
		}
	})

	// 9. Reject schedule conflict
	t.Run("9. Reject Schedule Conflict", func(t *testing.T) {
		payload := map[string]any{"class_id": bentrokClassID}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", studyPlan1ID), bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for schedule clash, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 10. Drop enrollment successfully
	t.Run("10. Drop Enrollment Successfully", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments/%s", studyPlan1ID, enrollment1ID), nil)
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		// Verify status is DROPPED in DB
		var status string
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT status FROM enrollments WHERE id = $1", enrollment1ID).Scan(&status)
		if status != "DROPPED" {
			t.Errorf("expected status DROPPED, got %s", status)
		}

		// Verify credits decremented
		var totalCredits int
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT total_credits FROM study_plans WHERE id = $1", studyPlan1ID).Scan(&totalCredits)
		if totalCredits != 3 { // 7 - 4 = 3
			t.Errorf("expected total credits 3 after drop, got %d", totalCredits)
		}
	})

	// 11. Reject dropping already dropped enrollment
	t.Run("11. Reject Double Drop", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments/%s", studyPlan1ID, enrollment1ID), nil)
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for double drop, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 12. Admin study plans endpoints
	t.Run("12. Admin Study Plans Management", func(t *testing.T) {
		// List all study plans
		req := httptest.NewRequest("GET", "/api/v1/study-plans", nil)
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		// Admin create study plan for student 2
		payload := map[string]any{
			"student_id":         h.student2.id,
			"academic_period_id": h.activePeriod,
		}
		b, _ := json.Marshal(payload)
		reqCreate := httptest.NewRequest("POST", "/api/v1/study-plans", bytes.NewReader(b))
		reqCreate.Header.Set("Authorization", "Bearer "+h.adminToken)
		reqCreate.Header.Set("Content-Type", "application/json")
		wCreate := httptest.NewRecorder()
		h.server.Router().ServeHTTP(wCreate, reqCreate)

		if wCreate.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for admin create, got %d: %s", wCreate.Code, wCreate.Body.String())
		}
	})
}

func TestStudyPlans_ConcurrencyRaceConditions(t *testing.T) {
	h := setupStudyPlansTestApp(t)

	// Skenario 1: Kapasitas 1 Kursi diperebutkan oleh 2 Mahasiswa Secara Bersamaan
	t.Run("Race Condition: 1 Slot Capacity Contended by 2 Concurrent Students", func(t *testing.T) {
		_, _, tightClassID := h.createCourseAndClass(t, "TIGHT01", "Seminar Riset Terbatas", 2, "A", "Kelas Seminar", 1) // Capacity = 1

		// Create study plan for student 1 & student 2
		createSP := func(tok string) uuid.UUID {
			payload := map[string]any{"academic_period_id": h.activePeriod}
			b, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/v1/student/study-plans", bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+tok)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.server.Router().ServeHTTP(w, req)
			var resp struct {
				Data struct {
					ID uuid.UUID `json:"id"`
				} `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			return resp.Data.ID
		}

		sp1ID := createSP(h.student1.token)
		sp2ID := createSP(h.student2.token)

		var wg sync.WaitGroup
		wg.Add(2)

		results := make([]int, 2)

		enrollFunc := func(idx int, spID uuid.UUID, tok string) {
			defer wg.Done()
			payload := map[string]any{"class_id": tightClassID}
			b, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/student/study-plans/%s/enrollments", spID), bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+tok)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.server.Router().ServeHTTP(w, req)
			results[idx] = w.Code
		}

		go enrollFunc(0, sp1ID, h.student1.token)
		go enrollFunc(1, sp2ID, h.student2.token)

		wg.Wait()

		successCount := 0
		conflictCount := 0
		for _, code := range results {
			if code == http.StatusCreated {
				successCount++
			} else if code == http.StatusConflict {
				conflictCount++
			}
		}

		if successCount != 1 || conflictCount != 1 {
			t.Fatalf("expected exactly 1 success (201) and 1 conflict (409), got results: %v", results)
		}

		// Verify database state: exactly 1 active enrollment
		var enrolledCount int
		_ = h.db.Pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM enrollments WHERE class_id = $1 AND status = 'ENROLLED'
		`, tightClassID).Scan(&enrolledCount)

		if enrolledCount != 1 {
			t.Errorf("expected database enrolled count = 1, got %d", enrolledCount)
		}
	})
}

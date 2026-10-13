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

type attendanceTestHarness struct {
	server       *httpserver.Server
	db           *database.DB
	tokenMgr     *auth.TokenManager
	adminToken   string
	lecturer1    testLecturer // assigned to class
	lecturer2    testLecturer // not assigned to class
	student1     testStudent  // enrolled in class
	student2     testStudent  // enrolled in class
	student3     testStudent  // not enrolled in class
	classID      uuid.UUID
	activePeriod uuid.UUID
}

type testLecturer struct {
	id     uuid.UUID
	userID uuid.UUID
	token  string
	name   string
}

func setupAttendanceTestApp(t *testing.T) *attendanceTestHarness {
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

	// 1. Setup Admin
	adminUserID := uuid.New()
	_, _ = db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, is_active)
		VALUES ($1, 'admin@test.educore.ac.id', 'dummyhash', 'Admin Tester', true)
	`, adminUserID)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 'admin')`, adminUserID)
	adminToken, _ := tokenMgr.GenerateAccessToken(&auth.User{
		ID:    adminUserID,
		Roles: []string{"admin"},
		Email: "admin@test.educore.ac.id",
		Name:  "Admin Tester",
	})

	// 2. Setup Faculty, Prodi, Period
	var facultyID, prodiID, periodID uuid.UUID
	_ = db.Pool.QueryRow(ctx, `INSERT INTO faculties (code, name) VALUES ('FTI', 'Fakultas TI') RETURNING id`).Scan(&facultyID)
	_ = db.Pool.QueryRow(ctx, `INSERT INTO study_programs (faculty_id, code, name, level) VALUES ($1, 'IF', 'Informatika', 'S1') RETURNING id`, facultyID).Scan(&prodiID)
	_ = db.Pool.QueryRow(ctx, `
		INSERT INTO academic_periods (code, name, semester_type, start_date, end_date, is_active)
		VALUES ('20261', 'Ganjil 2026/2027', 'ODD', '2026-09-01', '2027-01-31', true)
		RETURNING id
	`).Scan(&periodID)

	// 3. Setup Course, Offering, Class
	var courseID, offeringID, classID uuid.UUID
	_ = db.Pool.QueryRow(ctx, `INSERT INTO courses (code, name, credits, is_active) VALUES ('IF201', 'Struktur Data', 3, true) RETURNING id`).Scan(&courseID)
	_ = db.Pool.QueryRow(ctx, `INSERT INTO course_offerings (academic_period_id, course_id, study_program_id, status) VALUES ($1, $2, $3, 'OPEN') RETURNING id`, periodID, courseID, prodiID).Scan(&offeringID)
	_ = db.Pool.QueryRow(ctx, `INSERT INTO classes (course_offering_id, code, name, capacity, status) VALUES ($1, 'A', 'Struktur Data A', 40, 'OPEN') RETURNING id`, offeringID).Scan(&classID)

	// 4. Setup Lecturers
	createLecturer := func(name, email, nidn string) testLecturer {
		uID := uuid.New()
		_, err := db.Pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, name, is_active) VALUES ($1, $2, 'dummyhash', $3, true)`, uID, email, name)
		if err != nil {
			t.Fatalf("failed to insert user %s: %v", email, err)
		}
		_, _ = db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 'dosen')`, uID)

		var lecID uuid.UUID
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO lecturers (user_id, study_program_id, nidn, name, email, status)
			VALUES ($1, $2, $3, $4, $5, 'ACTIVE') RETURNING id
		`, uID, prodiID, nidn, name, email).Scan(&lecID)
		if err != nil {
			t.Fatalf("failed to insert lecturer %s: %v", name, err)
		}

		tok, _ := tokenMgr.GenerateAccessToken(&auth.User{
			ID:    uID,
			Roles: []string{"dosen"},
			Email: email,
			Name:  name,
		})
		return testLecturer{id: lecID, userID: uID, token: tok, name: name}
	}

	lec1 := createLecturer("Dr. Budi Santoso", "budi@test.educore.ac.id", "00112233")
	lec2 := createLecturer("Dr. Siti Rahma", "siti@test.educore.ac.id", "00445566")

	// Assign Lecturer 1 to Class A
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO class_lecturers (class_id, lecturer_id, role, status)
		VALUES ($1, $2, 'COORDINATOR', 'ACTIVE')
	`, classID, lec1.id)
	if err != nil {
		t.Fatalf("failed to assign lecturer to class: %v", err)
	}

	// 5. Setup Students
	createStudent := func(nim, name, email string) testStudent {
		uID := uuid.New()
		_, err := db.Pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, name, is_active) VALUES ($1, $2, 'dummyhash', $3, true)`, uID, email, name)
		if err != nil {
			t.Fatalf("failed to insert user %s: %v", email, err)
		}
		_, _ = db.Pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 'mahasiswa')`, uID)

		var sID uuid.UUID
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO students (user_id, study_program_id, nim, name, email, admission_year, status)
			VALUES ($1, $2, $3, $4, $5, 2024, 'ACTIVE') RETURNING id
		`, uID, prodiID, nim, name, email).Scan(&sID)
		if err != nil {
			t.Fatalf("failed to insert student %s: %v", name, err)
		}

		tok, _ := tokenMgr.GenerateAccessToken(&auth.User{
			ID:    uID,
			Roles: []string{"mahasiswa"},
			Email: email,
			Name:  name,
		})
		return testStudent{id: sID, userID: uID, token: tok, nim: nim, status: "ACTIVE"}
	}

	stud1 := createStudent("24001", "Mahasiswa Satu", "mhs1@test.educore.ac.id")
	stud2 := createStudent("24002", "Mahasiswa Dua", "mhs2@test.educore.ac.id")
	stud3 := createStudent("24003", "Mahasiswa Tiga", "mhs3@test.educore.ac.id") // Not enrolled

	// Enroll student 1 and student 2 to Class A
	for _, st := range []testStudent{stud1, stud2} {
		var spID uuid.UUID
		err := db.Pool.QueryRow(ctx, `
			INSERT INTO study_plans (student_id, academic_period_id, status)
			VALUES ($1, $2, 'ACTIVE') RETURNING id
		`, st.id, periodID).Scan(&spID)
		if err != nil {
			t.Fatalf("failed to insert study plan for %s: %v", st.nim, err)
		}

		_, err = db.Pool.Exec(ctx, `
			INSERT INTO enrollments (study_plan_id, student_id, class_id, status)
			VALUES ($1, $2, $3, 'ENROLLED')
		`, spID, st.id, classID)
		if err != nil {
			t.Fatalf("failed to insert enrollment for %s: %v", st.nim, err)
		}
	}

	server := httpserver.New(cfg, log, db.Pool)

	return &attendanceTestHarness{
		server:       server,
		db:           db,
		tokenMgr:     tokenMgr,
		adminToken:   adminToken,
		lecturer1:    lec1,
		lecturer2:    lec2,
		student1:     stud1,
		student2:     stud2,
		student3:     stud3,
		classID:      classID,
		activePeriod: periodID,
	}
}

func TestAttendance_SessionLifecycleAndPermissions(t *testing.T) {
	h := setupAttendanceTestApp(t)

	var session1ID string
	var session2ID string

	// 1. Unauthorized lecturer (Lec 2 not teaching class) tries to create session -> 403
	t.Run("1. Unauthorized lecturer cannot create session", func(t *testing.T) {
		body := `{"session_number": 1, "date": "2026-10-15", "start_time": "08:00", "end_time": "09:40", "topic": "Array & LinkedList"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer2.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 2. Authorized lecturer (Lec 1) creates Session 1 -> 201 Created
	t.Run("2. Authorized lecturer creates session", func(t *testing.T) {
		body := `{"session_number": 1, "date": "2026-10-15", "start_time": "08:00", "end_time": "09:40", "topic": "Array & LinkedList", "status": "OPEN"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID            string `json:"id"`
				SessionNumber int    `json:"session_number"`
				Topic         string `json:"topic"`
				Status        string `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		session1ID = resp.Data.ID
		if resp.Data.SessionNumber != 1 || resp.Data.Status != "OPEN" {
			t.Fatalf("unexpected session data: %+v", resp.Data)
		}
	})

	// 3. Admin creates Session 2 -> 201 Created (Bypass)
	t.Run("3. Admin creates session with bypass", func(t *testing.T) {
		body := `{"session_number": 2, "date": "2026-10-22", "start_time": "08:00", "end_time": "09:40", "topic": "Stack & Queue", "status": "OPEN"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.adminToken)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		session2ID = resp.Data.ID
	})

	// 4. Duplicate session number -> 409 Conflict
	t.Run("4. Duplicate session number rejected", func(t *testing.T) {
		body := `{"session_number": 1, "date": "2026-10-29", "start_time": "08:00", "end_time": "09:40", "topic": "Tree & Graph"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Enrolled student views sessions list -> 200 OK
	t.Run("5. Enrolled student can list class sessions", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), nil)
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 6. Non-enrolled student views sessions list -> 403 Forbidden
	t.Run("6. Non-enrolled student forbidden from listing class sessions", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), nil)
		req.Header.Set("Authorization", "Bearer "+h.student3.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 7. Lecturer views session detail -> 200 OK
	t.Run("7. Lecturer views session detail", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/sessions/%s", session1ID), nil)
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 8. Lecturer updates session metadata -> 200 OK
	t.Run("8. Lecturer updates session metadata", func(t *testing.T) {
		body := `{"topic": "Array, Pointer, & LinkedList"}`
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/sessions/%s", session1ID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 9. Close session -> 200 OK
	t.Run("9. Close session", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/close", session2ID), nil)
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		// Closing again should return 409
		reqAgain := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/close", session2ID), nil)
		reqAgain.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		wAgain := httptest.NewRecorder()
		h.server.Router().ServeHTTP(wAgain, reqAgain)

		if wAgain.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict when closing already closed session, got %d: %s", wAgain.Code, wAgain.Body.String())
		}
	})
}

func TestAttendance_RecordingAndReporting(t *testing.T) {
	h := setupAttendanceTestApp(t)

	// Create a session first
	var sessionID string
	{
		body := `{"session_number": 1, "date": "2026-10-15", "start_time": "08:00", "end_time": "09:40", "topic": "Pertemuan 1", "status": "OPEN"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create session: %d %s", w.Code, w.Body.String())
		}
		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		sessionID = resp.Data.ID
	}

	// 1. Get Session Roster
	t.Run("1. Get Session Roster", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/sessions/%s/students", sessionID), nil)
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				TotalStudents int `json:"total_students"`
				Students      []struct {
					StudentID string `json:"student_id"`
				} `json:"students"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.TotalStudents != 2 {
			t.Fatalf("expected 2 enrolled students in roster, got %d", resp.Data.TotalStudents)
		}
	})

	// 2. Batch attendance with duplicate student ID in payload -> 409 Conflict
	t.Run("2. Duplicate student ID in batch payload rejected", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"items": [
				{"student_id": "%s", "status": "PRESENT"},
				{"student_id": "%s", "status": "LATE"}
			]
		}`, h.student1.id, h.student1.id)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/attendance", sessionID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 3. Batch attendance with non-enrolled student -> 409 Conflict (Atomic Rollback)
	t.Run("3. Non-enrolled student causes atomic batch failure", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"items": [
				{"student_id": "%s", "status": "PRESENT"},
				{"student_id": "%s", "status": "PRESENT"}
			]
		}`, h.student1.id, h.student3.id)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/attendance", sessionID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}

		// Ensure nothing was written for Student 1 due to rollback
		var count int
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM attendances WHERE session_id = $1", sessionID).Scan(&count)
		if count != 0 {
			t.Fatalf("expected 0 attendance records after rollback, got %d", count)
		}
	})

	// 4. Batch attendance valid recording -> 200 OK
	t.Run("4. Record batch attendance successfully", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"items": [
				{"student_id": "%s", "status": "PRESENT", "remarks": "Tepat waktu"},
				{"student_id": "%s", "status": "LATE", "remarks": "Terlambat 10 menit"}
			]
		}`, h.student1.id, h.student2.id)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/attendance", sessionID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Read session attendances
	var student2AttendanceID string
	t.Run("5. Read session attendances & summary", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/sessions/%s/attendance", sessionID), nil)
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				Attendances []struct {
					ID        string `json:"id"`
					StudentID string `json:"student_id"`
					Status    string `json:"status"`
				} `json:"attendances"`
				Summary struct {
					TotalEnrolled int `json:"total_enrolled"`
					TotalRecorded int `json:"total_recorded"`
					Present       int `json:"present"`
					Late          int `json:"late"`
				} `json:"summary"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.Data.Summary.TotalRecorded != 2 || resp.Data.Summary.Present != 1 || resp.Data.Summary.Late != 1 {
			t.Fatalf("unexpected summary: %+v", resp.Data.Summary)
		}

		for _, a := range resp.Data.Attendances {
			if a.StudentID == h.student2.id.String() {
				student2AttendanceID = a.ID
			}
		}
	})

	// 6. Update/Correct individual attendance record
	t.Run("6. Correct attendance record", func(t *testing.T) {
		if student2AttendanceID == "" {
			t.Fatal("student2 attendance ID empty")
		}

		body := `{"status": "SICK", "remarks": "Kirim surat keterangan dokter"}`
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/sessions/%s/attendance/%s", sessionID, student2AttendanceID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		// Verify database value updated
		var status string
		_ = h.db.Pool.QueryRow(context.Background(), "SELECT status FROM attendances WHERE id = $1", student2AttendanceID).Scan(&status)
		if status != "SICK" {
			t.Fatalf("expected status 'SICK', got %s", status)
		}
	})

	// 7. Student 1 views personal attendance history
	t.Run("7. Student views personal attendance history", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/student/attendance", nil)
		req.Header.Set("Authorization", "Bearer "+h.student1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				Attendances []struct {
					Status string `json:"status"`
				} `json:"attendances"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Data.Attendances) != 1 || resp.Data.Attendances[0].Status != "PRESENT" {
			t.Fatalf("unexpected personal history: %+v", resp.Data)
		}
	})

	// 8. Class attendance cumulative summary report
	t.Run("8. Lecturer views class cumulative attendance summary", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/classes/%s/attendance/summary", h.classID), nil)
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				TotalSessions int `json:"total_sessions"`
				Students      []struct {
					StudentID            string  `json:"student_id"`
					Present              int     `json:"present"`
					Sick                 int     `json:"sick"`
					AttendancePercentage float64 `json:"attendance_percentage"`
				} `json:"students"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.Data.TotalSessions != 1 || len(resp.Data.Students) != 2 {
			t.Fatalf("unexpected class summary: %+v", resp.Data)
		}
	})
}

func TestAttendance_ConcurrencySimultaneousBatch(t *testing.T) {
	h := setupAttendanceTestApp(t)

	// Create session
	var sessionID string
	{
		body := `{"session_number": 1, "date": "2026-10-15", "start_time": "08:00", "end_time": "09:40", "topic": "Concurrency Session", "status": "OPEN"}`
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/classes/%s/sessions", h.classID), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
		w := httptest.NewRecorder()
		h.server.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create session: %d %s", w.Code, w.Body.String())
		}
		var resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		sessionID = resp.Data.ID
	}

	// Concurrently send 5 batch attendance requests for Student 1 and Student 2 with different presence statuses
	concurrency := 5
	var wg sync.WaitGroup
	wg.Add(concurrency)

	statusCodes := make([]int, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			status := "PRESENT"
			if idx%2 == 1 {
				status = "LATE"
			}
			body := fmt.Sprintf(`{
				"items": [
					{"student_id": "%s", "status": "%s"},
					{"student_id": "%s", "status": "%s"}
				]
			}`, h.student1.id, status, h.student2.id, status)

			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/attendance", sessionID), bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+h.lecturer1.token)
			w := httptest.NewRecorder()
			h.server.Router().ServeHTTP(w, req)
			statusCodes[idx] = w.Code
		}(i)
	}

	wg.Wait()

	// Every request should succeed (200 OK) because row locking & ON CONFLICT safely upsert without duplicate error
	for idx, code := range statusCodes {
		if code != http.StatusOK {
			t.Errorf("request %d failed with code %d", idx, code)
		}
	}

	// Verify exactly 2 records exist in database (no duplicates created)
	var count int
	_ = h.db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM attendances WHERE session_id = $1", sessionID).Scan(&count)
	if count != 2 {
		t.Fatalf("expected exactly 2 records in database, got %d", count)
	}
}

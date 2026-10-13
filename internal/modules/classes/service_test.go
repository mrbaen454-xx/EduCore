package classes_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"educore/internal/modules/classes"
	"educore/internal/shared/pagination"

	"github.com/google/uuid"
)

type mockClassRepo struct {
	classes   map[uuid.UUID]*classes.Class
	lecturers map[uuid.UUID][]classes.ClassLecturerDetail
	schedules map[uuid.UUID][]classes.ClassSchedule

	// period mapping: classID -> periodID
	classPeriods map[uuid.UUID]uuid.UUID
}

func newMockClassRepo() *mockClassRepo {
	return &mockClassRepo{
		classes:      make(map[uuid.UUID]*classes.Class),
		lecturers:    make(map[uuid.UUID][]classes.ClassLecturerDetail),
		schedules:    make(map[uuid.UUID][]classes.ClassSchedule),
		classPeriods: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockClassRepo) CreateClass(ctx context.Context, c *classes.Class) error {
	for _, existing := range m.classes {
		if existing.CourseOfferingID == c.CourseOfferingID && existing.Code == c.Code {
			return classes.ErrDuplicateClassCode
		}
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	m.classes[c.ID] = c
	return nil
}

func (m *mockClassRepo) GetClassByID(ctx context.Context, id uuid.UUID) (*classes.ClassDetail, error) {
	c, ok := m.classes[id]
	if !ok {
		return nil, classes.ErrClassNotFound
	}
	periodID := m.classPeriods[id]
	return &classes.ClassDetail{
		Class:            *c,
		CourseCode:       "CS101",
		CourseName:       "Pemrograman",
		AcademicPeriodID: periodID,
		Lecturers:        m.lecturers[id],
		Schedules:        m.schedules[id],
	}, nil
}

func (m *mockClassRepo) ListClasses(ctx context.Context, filter classes.ClassFilter, p pagination.Params) ([]classes.ClassDetail, int, error) {
	var list []classes.ClassDetail
	for _, c := range m.classes {
		list = append(list, classes.ClassDetail{
			Class: *c,
		})
	}
	return list, len(list), nil
}

func (m *mockClassRepo) UpdateClass(ctx context.Context, c *classes.Class) error {
	existing, ok := m.classes[c.ID]
	if !ok {
		return classes.ErrClassNotFound
	}
	existing.Name = c.Name
	existing.Capacity = c.Capacity
	existing.Remarks = c.Remarks
	existing.UpdatedAt = time.Now()
	return nil
}

func (m *mockClassRepo) UpdateClassStatus(ctx context.Context, id uuid.UUID, status string) error {
	c, ok := m.classes[id]
	if !ok {
		return classes.ErrClassNotFound
	}
	c.Status = status
	return nil
}

func (m *mockClassRepo) AssignLecturer(ctx context.Context, cl *classes.ClassLecturer) error {
	for _, existing := range m.lecturers[cl.ClassID] {
		if existing.LecturerID == cl.LecturerID {
			return classes.ErrDuplicateLecturer
		}
	}
	cl.ID = uuid.New()
	m.lecturers[cl.ClassID] = append(m.lecturers[cl.ClassID], classes.ClassLecturerDetail{
		ClassLecturer: *cl,
		LecturerName:  "Dosen Uji",
	})
	return nil
}

func (m *mockClassRepo) GetClassLecturers(ctx context.Context, classID uuid.UUID) ([]classes.ClassLecturerDetail, error) {
	return m.lecturers[classID], nil
}

func (m *mockClassRepo) RemoveLecturer(ctx context.Context, classID, lecturerID uuid.UUID) error {
	current := m.lecturers[classID]
	var updated []classes.ClassLecturerDetail
	found := false
	for _, item := range current {
		if item.LecturerID == lecturerID {
			found = true
			continue
		}
		updated = append(updated, item)
	}
	if !found {
		return classes.ErrLecturerNotFound
	}
	m.lecturers[classID] = updated
	return nil
}

func (m *mockClassRepo) CreateSchedule(ctx context.Context, cs *classes.ClassSchedule) error {
	cs.ID = uuid.New()
	cs.CreatedAt = time.Now()
	cs.UpdatedAt = time.Now()
	m.schedules[cs.ClassID] = append(m.schedules[cs.ClassID], *cs)
	return nil
}

func (m *mockClassRepo) GetScheduleByID(ctx context.Context, id uuid.UUID) (*classes.ClassSchedule, error) {
	for _, list := range m.schedules {
		for _, s := range list {
			if s.ID == id {
				return &s, nil
			}
		}
	}
	return nil, classes.ErrScheduleNotFound
}

func (m *mockClassRepo) GetClassSchedules(ctx context.Context, classID uuid.UUID) ([]classes.ClassSchedule, error) {
	return m.schedules[classID], nil
}

func (m *mockClassRepo) UpdateSchedule(ctx context.Context, cs *classes.ClassSchedule) error {
	list := m.schedules[cs.ClassID]
	for i, s := range list {
		if s.ID == cs.ID {
			list[i] = *cs
			m.schedules[cs.ClassID] = list
			return nil
		}
	}
	return classes.ErrScheduleNotFound
}

func (m *mockClassRepo) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	for classID, list := range m.schedules {
		for i, s := range list {
			if s.ID == id {
				m.schedules[classID] = append(list[:i], list[i+1:]...)
				return nil
			}
		}
	}
	return classes.ErrScheduleNotFound
}

// Helper: check if two intervals [s1, e1) and [s2, e2) overlap
func intervalsOverlap(s1, e1, s2, e2 string) bool {
	return s1 < e2 && s2 < e1
}

func (m *mockClassRepo) CheckClassSelfConflict(ctx context.Context, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, error) {
	for _, s := range m.schedules[classID] {
		if s.ID == excludeScheduleID {
			continue
		}
		if s.DayOfWeek == dayOfWeek && intervalsOverlap(s.StartTime, s.EndTime, startTime, endTime) {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockClassRepo) CheckLecturerScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime string) (bool, string, error) {
	// Lecturers in this class
	currentLecturers := m.lecturers[classID]
	for otherClassID, pID := range m.classPeriods {
		if otherClassID == classID || pID != periodID {
			continue
		}
		// check if other class shares lecturer
		for _, otherL := range m.lecturers[otherClassID] {
			for _, curL := range currentLecturers {
				if otherL.LecturerID == curL.LecturerID {
					// check other class schedules
					for _, s := range m.schedules[otherClassID] {
						if s.ID == excludeScheduleID {
							continue
						}
						if s.DayOfWeek == dayOfWeek && intervalsOverlap(s.StartTime, s.EndTime, startTime, endTime) {
							return true, fmt.Sprintf("Dosen %s memiliki jadwal bentrok", curL.LecturerName), nil
						}
					}
				}
			}
		}
	}
	return false, "", nil
}

func (m *mockClassRepo) CheckRoomScheduleConflict(ctx context.Context, periodID, classID, excludeScheduleID uuid.UUID, dayOfWeek int, startTime, endTime, room string) (bool, string, error) {
	if room == "" {
		return false, "", nil
	}
	for otherClassID, pID := range m.classPeriods {
		if otherClassID == classID || pID != periodID {
			continue
		}
		for _, s := range m.schedules[otherClassID] {
			if s.ID == excludeScheduleID {
				continue
			}
			if s.Room != nil && *s.Room == room && s.DayOfWeek == dayOfWeek && intervalsOverlap(s.StartTime, s.EndTime, startTime, endTime) {
				return true, fmt.Sprintf("Ruangan %s sudah digunakan pada waktu yang sama", room), nil
			}
		}
	}
	return false, "", nil
}

func TestClassService(t *testing.T) {
	repo := newMockClassRepo()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	svc := classes.NewService(repo, logger)
	ctx := context.Background()

	periodID := uuid.New()
	offeringID := uuid.New()

	t.Run("Create Class Success", func(t *testing.T) {
		res, err := svc.CreateClass(ctx, classes.CreateClassRequest{
			CourseOfferingID: offeringID,
			Code:             "A",
			Name:             "Kelas A",
			Capacity:         40,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Code != "A" || res.Capacity != 40 {
			t.Errorf("unexpected class attributes")
		}
		repo.classPeriods[res.ID] = periodID
	})

	t.Run("Reject Zero or Negative Capacity", func(t *testing.T) {
		_, err := svc.CreateClass(ctx, classes.CreateClassRequest{
			CourseOfferingID: offeringID,
			Code:             "B",
			Name:             "Kelas B",
			Capacity:         0,
		})
		if err == nil {
			t.Fatalf("expected error for zero capacity")
		}
	})

	t.Run("Assign Lecturer and Schedule", func(t *testing.T) {
		classA, _ := svc.CreateClass(ctx, classes.CreateClassRequest{
			CourseOfferingID: offeringID,
			Code:             "K01",
			Name:             "Kelas K01",
			Capacity:         35,
		})
		repo.classPeriods[classA.ID] = periodID

		lecturerID := uuid.New()
		_, err := svc.AssignLecturer(ctx, classA.ID, classes.AssignLecturerRequest{
			LecturerID: lecturerID,
			Role:       "COORDINATOR",
		})
		if err != nil {
			t.Fatalf("unexpected error assigning lecturer: %v", err)
		}

		room101 := "R-101"
		// Create schedule 08:00 - 10:00 on Monday
		sched1, err := svc.CreateSchedule(ctx, classA.ID, classes.CreateScheduleRequest{
			DayOfWeek:    1,
			StartTime:    "08:00",
			EndTime:      "10:00",
			Room:         &room101,
			DeliveryMode: "OFFLINE",
		})
		if err != nil {
			t.Fatalf("unexpected error creating schedule: %v", err)
		}
		if sched1.StartTime != "08:00:00" {
			t.Errorf("expected 08:00:00, got %s", sched1.StartTime)
		}

		// 1. Same class self-overlap on same day (09:00 - 11:00 overlaps 08:00 - 10:00)
		_, err = svc.CreateSchedule(ctx, classA.ID, classes.CreateScheduleRequest{
			DayOfWeek:    1,
			StartTime:    "09:00",
			EndTime:      "11:00",
			DeliveryMode: "OFFLINE",
		})
		if err == nil {
			t.Fatalf("expected conflict error for class self-overlap")
		}

		// 2. Consecutive schedule (10:00 - 12:00) should be ALLOWED because intervals [08:00, 10:00) and [10:00, 12:00) do not overlap
		sched2, err := svc.CreateSchedule(ctx, classA.ID, classes.CreateScheduleRequest{
			DayOfWeek:    1,
			StartTime:    "10:00",
			EndTime:      "12:00",
			DeliveryMode: "OFFLINE",
		})
		if err != nil {
			t.Fatalf("unexpected error for consecutive schedule: %v", err)
		}
		if sched2 == nil {
			t.Fatalf("expected sched2 to be created")
		}

		// Create Class B in same period
		classB, _ := svc.CreateClass(ctx, classes.CreateClassRequest{
			CourseOfferingID: offeringID,
			Code:             "K02",
			Name:             "Kelas K02",
			Capacity:         35,
		})
		repo.classPeriods[classB.ID] = periodID

		// Assign same lecturer to Class B
		_, _ = svc.AssignLecturer(ctx, classB.ID, classes.AssignLecturerRequest{
			LecturerID: lecturerID,
			Role:       "TEAM_TEACHING",
		})

		// 3. Lecturer schedule conflict: Class B tries to schedule at 08:30 - 10:30 on Monday
		_, err = svc.CreateSchedule(ctx, classB.ID, classes.CreateScheduleRequest{
			DayOfWeek:    1,
			StartTime:    "08:30",
			EndTime:      "10:30",
			DeliveryMode: "OFFLINE",
		})
		if err == nil {
			t.Fatalf("expected conflict error for lecturer collision")
		}

		// 4. Room conflict: Class B tries to book R-101 at 08:00 - 10:00 on Monday
		_, err = svc.CreateSchedule(ctx, classB.ID, classes.CreateScheduleRequest{
			DayOfWeek:    1,
			StartTime:    "08:00",
			EndTime:      "10:00",
			Room:         &room101,
			DeliveryMode: "OFFLINE",
		})
		if err == nil {
			t.Fatalf("expected conflict error for room collision")
		}
	})
}

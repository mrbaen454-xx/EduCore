-- Migration 000005: Academic Periods, Course Offerings, Classes, Class Lecturers, and Class Schedules
-- Enforces UUID, strict foreign keys, unique identifiers, status checks, non-negative capacity, and schedule integrity.

-- 1. Academic Periods Table (Periode Akademik)
CREATE TABLE IF NOT EXISTS academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    semester_type VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_academic_periods_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_academic_periods_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_academic_periods_semester_type CHECK (semester_type IN ('ODD', 'EVEN', 'SHORT')),
    CONSTRAINT chk_academic_periods_dates CHECK (start_date <= end_date)
);

CREATE INDEX IF NOT EXISTS idx_academic_periods_code ON academic_periods(code);
CREATE INDEX IF NOT EXISTS idx_academic_periods_is_active ON academic_periods(is_active);
CREATE INDEX IF NOT EXISTS idx_academic_periods_dates ON academic_periods(start_date, end_date);

-- 2. Course Offerings Table (Penawaran Mata Kuliah)
CREATE TABLE IF NOT EXISTS course_offerings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_period_id UUID NOT NULL REFERENCES academic_periods(id) ON DELETE RESTRICT,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    curriculum_id UUID REFERENCES curricula(id) ON DELETE RESTRICT,
    study_program_id UUID NOT NULL REFERENCES study_programs(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'PLANNED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_course_offerings_period_course_prodi UNIQUE (academic_period_id, course_id, study_program_id),
    CONSTRAINT chk_course_offerings_status CHECK (status IN ('PLANNED', 'OPEN', 'CLOSED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_course_offerings_period_id ON course_offerings(academic_period_id);
CREATE INDEX IF NOT EXISTS idx_course_offerings_course_id ON course_offerings(course_id);
CREATE INDEX IF NOT EXISTS idx_course_offerings_curriculum_id ON course_offerings(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_course_offerings_study_program_id ON course_offerings(study_program_id);
CREATE INDEX IF NOT EXISTS idx_course_offerings_status ON course_offerings(status);

-- 3. Classes Table (Kelas Perkuliahan)
CREATE TABLE IF NOT EXISTS classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_offering_id UUID NOT NULL REFERENCES course_offerings(id) ON DELETE RESTRICT,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    capacity INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    remarks TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_classes_offering_code UNIQUE (course_offering_id, code),
    CONSTRAINT chk_classes_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_classes_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_classes_capacity CHECK (capacity > 0),
    CONSTRAINT chk_classes_status CHECK (status IN ('DRAFT', 'OPEN', 'IN_PROGRESS', 'CLOSED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_classes_offering_id ON classes(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_classes_code ON classes(code);
CREATE INDEX IF NOT EXISTS idx_classes_status ON classes(status);

-- 4. Class Lecturers Table (Penugasan Dosen pada Kelas)
CREATE TABLE IF NOT EXISTS class_lecturers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE RESTRICT,
    lecturer_id UUID NOT NULL REFERENCES lecturers(id) ON DELETE RESTRICT,
    role VARCHAR(30) NOT NULL DEFAULT 'COORDINATOR',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_class_lecturers UNIQUE (class_id, lecturer_id),
    CONSTRAINT chk_class_lecturers_role CHECK (role IN ('COORDINATOR', 'TEAM_TEACHING')),
    CONSTRAINT chk_class_lecturers_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE INDEX IF NOT EXISTS idx_class_lecturers_class_id ON class_lecturers(class_id);
CREATE INDEX IF NOT EXISTS idx_class_lecturers_lecturer_id ON class_lecturers(lecturer_id);

-- 5. Class Schedules Table (Jadwal Perkuliahan)
CREATE TABLE IF NOT EXISTS class_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE RESTRICT,
    day_of_week SMALLINT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    room VARCHAR(100),
    delivery_mode VARCHAR(20) NOT NULL DEFAULT 'OFFLINE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_class_schedules_day CHECK (day_of_week >= 1 AND day_of_week <= 7),
    CONSTRAINT chk_class_schedules_time CHECK (start_time < end_time),
    CONSTRAINT chk_class_schedules_delivery_mode CHECK (delivery_mode IN ('OFFLINE', 'ONLINE', 'HYBRID'))
);

CREATE INDEX IF NOT EXISTS idx_class_schedules_class_id ON class_schedules(class_id);
CREATE INDEX IF NOT EXISTS idx_class_schedules_day_time ON class_schedules(day_of_week, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_class_schedules_room ON class_schedules(room);

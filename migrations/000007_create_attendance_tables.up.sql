-- Migration 000007: Class Sessions and Attendances Tables
-- Enforces UUID, strict foreign keys, unique session numbers per class,
-- unique attendance records per student per session, and validated attendance statuses.

-- 1. Class Sessions Table (Sesi Perkuliahan / Pertemuan Kelas)
CREATE TABLE IF NOT EXISTS class_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE RESTRICT,
    session_number INT NOT NULL,
    date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    topic VARCHAR(255) NOT NULL,
    remarks TEXT,
    lecturer_id UUID REFERENCES lecturers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_class_session_number UNIQUE (class_id, session_number),
    CONSTRAINT chk_class_sessions_number CHECK (session_number > 0),
    CONSTRAINT chk_class_sessions_time CHECK (start_time < end_time),
    CONSTRAINT chk_class_sessions_topic CHECK (length(trim(topic)) > 0),
    CONSTRAINT chk_class_sessions_status CHECK (status IN ('SCHEDULED', 'OPEN', 'COMPLETED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_class_sessions_class_id ON class_sessions(class_id);
CREATE INDEX IF NOT EXISTS idx_class_sessions_date ON class_sessions(date);
CREATE INDEX IF NOT EXISTS idx_class_sessions_status ON class_sessions(status);
CREATE INDEX IF NOT EXISTS idx_class_sessions_lecturer_id ON class_sessions(lecturer_id);

-- 2. Attendances Table (Catatan Kehadiran Mahasiswa)
CREATE TABLE IF NOT EXISTS attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES class_sessions(id) ON DELETE RESTRICT,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'PRESENT',
    remarks TEXT,
    recorded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_attendances_session_student UNIQUE (session_id, student_id),
    CONSTRAINT uq_attendances_session_enrollment UNIQUE (session_id, enrollment_id),
    CONSTRAINT chk_attendances_status CHECK (status IN ('PRESENT', 'LATE', 'EXCUSED', 'SICK', 'ABSENT'))
);

CREATE INDEX IF NOT EXISTS idx_attendances_session_id ON attendances(session_id);
CREATE INDEX IF NOT EXISTS idx_attendances_student_id ON attendances(student_id);
CREATE INDEX IF NOT EXISTS idx_attendances_enrollment_id ON attendances(enrollment_id);
CREATE INDEX IF NOT EXISTS idx_attendances_status ON attendances(status);

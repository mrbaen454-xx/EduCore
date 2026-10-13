-- Migration 000006: Study Plans (KRS) and Course Enrollments Tables
-- Enforces UUID, strict foreign keys, unique student study plans per period, credit boundaries,
-- and race-condition free course registration with audit timestamps.

-- 1. Study Plans Table (Kartu Rencana Studi / KRS)
CREATE TABLE IF NOT EXISTS study_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    academic_period_id UUID NOT NULL REFERENCES academic_periods(id) ON DELETE RESTRICT,
    total_credits INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_study_plans_student_period UNIQUE (student_id, academic_period_id),
    CONSTRAINT chk_study_plans_credits CHECK (total_credits >= 0 AND total_credits <= 24),
    CONSTRAINT chk_study_plans_status CHECK (status IN ('DRAFT', 'ACTIVE', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_study_plans_student_id ON study_plans(student_id);
CREATE INDEX IF NOT EXISTS idx_study_plans_period_id ON study_plans(academic_period_id);
CREATE INDEX IF NOT EXISTS idx_study_plans_status ON study_plans(status);

-- 2. Course Enrollments Table (Registrasi Kelas Perkuliahan / Detail KRS)
CREATE TABLE IF NOT EXISTS enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_plan_id UUID NOT NULL REFERENCES study_plans(id) ON DELETE RESTRICT,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'ENROLLED',
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dropped_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_enrollments_status CHECK (status IN ('ENROLLED', 'DROPPED', 'CANCELLED'))
);

-- Ensure a student can only have ONE active enrollment for a given class
CREATE UNIQUE INDEX IF NOT EXISTS uq_enrollments_active_student_class 
    ON enrollments(student_id, class_id) 
    WHERE status = 'ENROLLED';

-- Ensure a study plan can only have ONE active enrollment for a given class
CREATE UNIQUE INDEX IF NOT EXISTS uq_enrollments_active_study_plan_class 
    ON enrollments(study_plan_id, class_id) 
    WHERE status = 'ENROLLED';

CREATE INDEX IF NOT EXISTS idx_enrollments_study_plan_id ON enrollments(study_plan_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_student_id ON enrollments(student_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_class_id ON enrollments(class_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_status ON enrollments(status);

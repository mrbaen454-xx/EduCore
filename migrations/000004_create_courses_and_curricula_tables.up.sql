-- Migration 000004: Courses, Curricula, Curriculum Courses, and Prerequisites
-- Enforces UUID, strict foreign keys, unique identifiers, status checks, and query indexes.

-- 1. Courses Table (Katalog Mata Kuliah)
CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    credits INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_courses_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_courses_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_courses_credits CHECK (credits >= 1 AND credits <= 8)
);

CREATE INDEX IF NOT EXISTS idx_courses_code ON courses(code);
CREATE INDEX IF NOT EXISTS idx_courses_name ON courses(name);
CREATE INDEX IF NOT EXISTS idx_courses_credits ON courses(credits);
CREATE INDEX IF NOT EXISTS idx_courses_is_active ON courses(is_active);

-- 2. Curricula Table (Kurikulum Program Studi)
CREATE TABLE IF NOT EXISTS curricula (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_program_id UUID NOT NULL REFERENCES study_programs(id) ON DELETE RESTRICT,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    year INT NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_curricula_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_curricula_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_curricula_year CHECK (year >= 1970 AND year <= 2100),
    CONSTRAINT chk_curricula_status CHECK (status IN ('DRAFT', 'ACTIVE', 'INACTIVE', 'ARCHIVED'))
);

CREATE INDEX IF NOT EXISTS idx_curricula_study_program_id ON curricula(study_program_id);
CREATE INDEX IF NOT EXISTS idx_curricula_code ON curricula(code);
CREATE INDEX IF NOT EXISTS idx_curricula_year ON curricula(year);
CREATE INDEX IF NOT EXISTS idx_curricula_status ON curricula(status);

-- 3. Curriculum Courses Table (Relasi Mata Kuliah dalam Kurikulum)
CREATE TABLE IF NOT EXISTS curriculum_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    curriculum_id UUID NOT NULL REFERENCES curricula(id) ON DELETE RESTRICT,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    semester INT NOT NULL,
    category VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_curriculum_courses UNIQUE (curriculum_id, course_id),
    CONSTRAINT chk_curriculum_courses_semester CHECK (semester >= 1 AND semester <= 14),
    CONSTRAINT chk_curriculum_courses_category CHECK (category IN ('REQUIRED', 'ELECTIVE'))
);

CREATE INDEX IF NOT EXISTS idx_curriculum_courses_curriculum_id ON curriculum_courses(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_course_id ON curriculum_courses(course_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_semester ON curriculum_courses(semester);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_category ON curriculum_courses(category);

-- 4. Course Prerequisites Table (Prasyarat Mata Kuliah)
CREATE TABLE IF NOT EXISTS course_prerequisites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    prerequisite_course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_course_prerequisites UNIQUE (course_id, prerequisite_course_id),
    CONSTRAINT chk_prerequisite_not_self CHECK (course_id != prerequisite_course_id)
);

CREATE INDEX IF NOT EXISTS idx_course_prerequisites_course_id ON course_prerequisites(course_id);
CREATE INDEX IF NOT EXISTS idx_course_prerequisites_prereq_id ON course_prerequisites(prerequisite_course_id);

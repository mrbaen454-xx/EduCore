-- Migration 000003: Academic Master Tables (Faculties, Study Programs, Lecturers, Students)
-- Enforces UUID, strict foreign keys, unique identifiers, status checks, and query indexes.

-- 1. Faculties Table
CREATE TABLE IF NOT EXISTS faculties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_faculties_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_faculties_name_non_empty CHECK (length(trim(name)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_faculties_code ON faculties(code);
CREATE INDEX IF NOT EXISTS idx_faculties_is_active ON faculties(is_active);

-- 2. Study Programs Table
CREATE TABLE IF NOT EXISTS study_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    faculty_id UUID NOT NULL REFERENCES faculties(id) ON DELETE RESTRICT,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    level VARCHAR(20) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_study_programs_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT chk_study_programs_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_study_programs_level CHECK (level IN ('D3', 'D4', 'S1', 'S2', 'S3', 'Profesi'))
);

CREATE INDEX IF NOT EXISTS idx_study_programs_faculty_id ON study_programs(faculty_id);
CREATE INDEX IF NOT EXISTS idx_study_programs_code ON study_programs(code);
CREATE INDEX IF NOT EXISTS idx_study_programs_level ON study_programs(level);
CREATE INDEX IF NOT EXISTS idx_study_programs_is_active ON study_programs(is_active);

-- 3. Lecturers Table
CREATE TABLE IF NOT EXISTS lecturers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    study_program_id UUID NOT NULL REFERENCES study_programs(id) ON DELETE RESTRICT,
    nidn VARCHAR(30) NOT NULL UNIQUE,
    nip VARCHAR(30),
    name VARCHAR(150) NOT NULL,
    title_prefix VARCHAR(50),
    title_suffix VARCHAR(50),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(25),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_lecturers_nidn_non_empty CHECK (length(trim(nidn)) > 0),
    CONSTRAINT chk_lecturers_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_lecturers_email_non_empty CHECK (length(trim(email)) > 0),
    CONSTRAINT chk_lecturers_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'LEAVE'))
);

CREATE INDEX IF NOT EXISTS idx_lecturers_study_program_id ON lecturers(study_program_id);
CREATE INDEX IF NOT EXISTS idx_lecturers_user_id ON lecturers(user_id);
CREATE INDEX IF NOT EXISTS idx_lecturers_nidn ON lecturers(nidn);
CREATE INDEX IF NOT EXISTS idx_lecturers_status ON lecturers(status);

-- 4. Students Table
CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    study_program_id UUID NOT NULL REFERENCES study_programs(id) ON DELETE RESTRICT,
    nim VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(25),
    gender VARCHAR(10),
    birth_date DATE,
    admission_year INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_students_nim_non_empty CHECK (length(trim(nim)) > 0),
    CONSTRAINT chk_students_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_students_email_non_empty CHECK (length(trim(email)) > 0),
    CONSTRAINT chk_students_gender CHECK (gender IS NULL OR gender IN ('MALE', 'FEMALE')),
    CONSTRAINT chk_students_admission_year CHECK (admission_year >= 1970 AND admission_year <= 2100),
    CONSTRAINT chk_students_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'PROBATION', 'GRADUATED', 'DROPOUT', 'SUSPENDED'))
);

CREATE INDEX IF NOT EXISTS idx_students_study_program_id ON students(study_program_id);
CREATE INDEX IF NOT EXISTS idx_students_user_id ON students(user_id);
CREATE INDEX IF NOT EXISTS idx_students_nim ON students(nim);
CREATE INDEX IF NOT EXISTS idx_students_admission_year ON students(admission_year);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status);

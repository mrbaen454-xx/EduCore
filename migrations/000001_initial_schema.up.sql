-- Baseline Schema: Authentication & RBAC Foundation (Fase 01 - 04)
-- Enforces UUID, multi-role relationship, unique email, and audit timestamps.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_users_email_non_empty CHECK (length(trim(email)) > 0),
    CONSTRAINT chk_users_name_non_empty CHECK (length(trim(name)) > 0)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(50) NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- Indexes for performance and query optimization
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- Baseline system roles
INSERT INTO roles (id, name, description) VALUES
    ('admin', 'Administrator', 'Pengelola sistem dengan akses administratif penuh'),
    ('dosen', 'Dosen Pengajar', 'Tenaga pendidik pengampu kelas dan penilai akademik'),
    ('mahasiswa', 'Mahasiswa', 'Peserta didik aktif perguruan tinggi'),
    ('akademik', 'Staf Akademik', 'Pengelola administrasi kurikulum dan jadwal perkuliahan')
ON CONFLICT (id) DO NOTHING;

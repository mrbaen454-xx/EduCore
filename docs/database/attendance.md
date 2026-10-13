# Skema Database & Integritas: Presensi dan Sesi Perkuliahan (Fase 10)
## EduCore — Academic Management System

Dokumen ini mendokumentasikan skema database PostgreSQL, constraint, indeks, serta mekanisme transaksi dan concurrency control yang diimplementasikan pada Fase 10.

---

## 1. File Migrasi

- **Migrasi Naik (Up)**: `migrations/000007_create_attendance_tables.up.sql`
- **Migrasi Turun (Down)**: `migrations/000007_create_attendance_tables.down.sql`
- **Versi Database Terkini**: `7`

---

## 2. Struktur Tabel

### 2.1. Tabel `class_sessions`

Menyimpan data pertemuan atau sesi perkuliahan dari kelas yang ditawarkan.

| Kolom | Tipe Data | Nullable | Keterangan |
|---|---|---|---|
| `id` | `UUID` | NOT NULL | Primary key default `gen_random_uuid()` |
| `class_id` | `UUID` | NOT NULL | Foreign key ke `classes(id)` ON DELETE RESTRICT |
| `session_number` | `INT` | NOT NULL | Nomor pertemuan (1, 2, 3, dst.) |
| `date` | `DATE` | NOT NULL | Tanggal sesi perkuliahan |
| `start_time` | `TIME` | NOT NULL | Waktu mulai perkuliahan |
| `end_time` | `TIME` | NOT NULL | Waktu selesai perkuliahan |
| `topic` | `VARCHAR(255)` | NOT NULL | Materi atau pokok bahasan perkuliahan |
| `remarks` | `TEXT` | NULL | Catatan tambahan |
| `lecturer_id` | `UUID` | NULL | Foreign key ke `lecturers(id)` ON DELETE SET NULL |
| `status` | `VARCHAR(20)` | NOT NULL | Status sesi (`SCHEDULED`, `OPEN`, `COMPLETED`, `CANCELLED`) |
| `created_by` | `UUID` | NULL | Foreign key ke `users(id)` ON DELETE SET NULL |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | Waktu pembuatan default `CURRENT_TIMESTAMP` |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | Waktu pembaruan default `CURRENT_TIMESTAMP` |

**Constraints**:
- `uq_class_sessions_class_number`: `UNIQUE (class_id, session_number)`
- `chk_class_sessions_number`: `CHECK (session_number >= 1)`
- `chk_class_sessions_time`: `CHECK (end_time > start_time)`
- `chk_class_sessions_status`: `CHECK (status IN ('SCHEDULED', 'OPEN', 'COMPLETED', 'CANCELLED'))`
- `chk_class_sessions_topic_non_empty`: `CHECK (length(trim(topic)) > 0)`

**Indeks**:
- `idx_class_sessions_class_id`: B-Tree pada `class_id`
- `idx_class_sessions_date`: B-Tree pada `date`
- `idx_class_sessions_status`: B-Tree pada `status`

---

### 2.2. Tabel `attendances`

Menyimpan catatan kehadiran mahasiswa pada suatu sesi perkuliahan.

| Kolom | Tipe Data | Nullable | Keterangan |
|---|---|---|---|
| `id` | `UUID` | NOT NULL | Primary key default `gen_random_uuid()` |
| `session_id` | `UUID` | NOT NULL | Foreign key ke `class_sessions(id)` ON DELETE CASCADE |
| `student_id` | `UUID` | NOT NULL | Foreign key ke `students(id)` ON DELETE RESTRICT |
| `enrollment_id` | `UUID` | NOT NULL | Foreign key ke `enrollments(id)` ON DELETE RESTRICT |
| `status` | `VARCHAR(20)` | NOT NULL | Status kehadiran (`PRESENT`, `LATE`, `EXCUSED`, `SICK`, `ABSENT`) |
| `remarks` | `TEXT` | NULL | Alasan atau catatan keterangan presensi |
| `recorded_by` | `UUID` | NULL | Foreign key ke `users(id)` ON DELETE SET NULL |
| `recorded_at` | `TIMESTAMPTZ` | NOT NULL | Waktu pencatatan presensi default `CURRENT_TIMESTAMP` |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | Waktu pembuatan record default `CURRENT_TIMESTAMP` |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | Waktu pembaruan record default `CURRENT_TIMESTAMP` |

**Constraints**:
- `uq_attendances_session_student`: `UNIQUE (session_id, student_id)` — Mencegah presensi ganda
- `chk_attendances_status`: `CHECK (status IN ('PRESENT', 'LATE', 'EXCUSED', 'SICK', 'ABSENT'))`

**Indeks**:
- `idx_attendances_session_id`: B-Tree pada `session_id`
- `idx_attendances_student_id`: B-Tree pada `student_id`
- `idx_attendances_enrollment_id`: B-Tree pada `enrollment_id`
- `idx_attendances_status`: B-Tree pada `status`

---

## 3. Strategi Transaksi dan Concurrency Control

1. **Row-Level Locking pada Sesi**:
   - Pada operasi pencatatan batch presensi (`RecordBatchAttendanceTx`) dan pembaruan presensi (`UpdateAttendanceTx`), sesi perkuliahan dikunci terlebih dahulu menggunakan:
     ```sql
     SELECT class_id, status FROM class_sessions WHERE id = $1 FOR UPDATE
     ```
   - Hal ini mencegah *race condition* di mana sesi ditutup atau dibatalkan saat batch presensi sedang diproses.
2. **Atomisitas Batch Presensi**:
   - Seluruh daftar mahasiswa dalam payload batch diverifikasi terhadap tabel `enrollments` dengan filter `class_id = classID AND status = 'ENROLLED'`.
   - Jika satu mahasiswa tidak terdaftar atau tidak aktif, transaksi di-rollback secara penuh (`tx.Rollback()`) sehingga tidak ada mutasi data parsial.
3. **Pencegahan Konflik Bersamaan (ON CONFLICT)**:
   - Operasi insert batch presensi menggunakan klausa:
     ```sql
     INSERT INTO attendances (session_id, student_id, enrollment_id, status, remarks, recorded_by, recorded_at, created_at, updated_at)
     VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
     ON CONFLICT (session_id, student_id) DO UPDATE
     SET status = EXCLUDED.status,
         remarks = EXCLUDED.remarks,
         recorded_by = EXCLUDED.recorded_by,
         recorded_at = CURRENT_TIMESTAMP,
         updated_at = CURRENT_TIMESTAMP
     ```
   - Hal ini menjamin bahwa dua request simultan tidak menghasilkan duplicate key error atau data korup.

# Skema Database: KRS (Study Plans) dan Registrasi Mahasiswa (Fase 09)

## 1. Ikhtisar Arsitektur Data

Modul KRS dan Registrasi Mahasiswa (Fase 09) mengelola data pengisian rencana studi dan pendaftaran kelas perkuliahan oleh mahasiswa. Skema ini dirancang menggunakan relasi PostgreSQL yang ketat dengan penegakan integritas data melalui *Foreign Keys*, *Unique Constraints*, *Check Constraints*, serta indeks performa.

---

## 2. Diagram Hubungan Entitas (ERD)

```text
+-----------------------+             +-------------------------+
|        students       |             |    academic_periods     |
+-----------------------+             +-------------------------+
       |           |                               |
       |           +-----------------------+       |
       |                                   |       |
       | 1                                 v       v
       |                                +---------------------+
       |                                |     study_plans     |
       |                                +---------------------+
       |                                           | 1
       |                                           |
       |                                           v N
       |                                +---------------------+
       +------------------------------> |     enrollments     |
                                        +---------------------+
                                                   | N
                                                   |
                                                   v 1
                                        +---------------------+
                                        |       classes       |
                                        +---------------------+
```

---

## 3. Spesifikasi Tabel

### 3.1. Tabel `study_plans` (Kartu Rencana Studi / KRS)

Menyimpan record induk KRS untuk seorang mahasiswa pada satu periode akademik tertentu.

| Kolom | Tipe Data | Nullable | Default | Keterangan & Constraints |
| :--- | :--- | :---: | :--- | :--- |
| `id` | `UUID` | No | `gen_random_uuid()` | Primary Key |
| `student_id` | `UUID` | No | - | Foreign Key ke `students(id)` `ON DELETE RESTRICT` |
| `academic_period_id` | `UUID` | No | - | Foreign Key ke `academic_periods(id)` `ON DELETE RESTRICT` |
| `total_credits` | `INT` | No | `0` | Total beban SKS aktif (`CHECK (total_credits >= 0 AND total_credits <= 24)`) |
| `status` | `VARCHAR(20)` | No | `'ACTIVE'` | Status KRS: `DRAFT`, `ACTIVE`, `CANCELLED` |
| `created_at` | `TIMESTAMPTZ` | No | `CURRENT_TIMESTAMP` | Waktu pembuatan |
| `updated_at` | `TIMESTAMPTZ` | No | `CURRENT_TIMESTAMP` | Waktu pembaruan terakhir |

**Constraints & Indeks:**
- `CONSTRAINT uq_study_plans_student_period UNIQUE (student_id, academic_period_id)`: Menjamin satu mahasiswa hanya memiliki maksimal 1 KRS per periode akademik.
- `idx_study_plans_student_id ON study_plans(student_id)`
- `idx_study_plans_period_id ON study_plans(academic_period_id)`
- `idx_study_plans_status ON study_plans(status)`

---

### 3.2. Tabel `enrollments` (Detail Registrasi Kelas Perkuliahan)

Menyimpan pendaftaran mahasiswa ke unit rombongan belajar kelas perkuliahan.

| Kolom | Tipe Data | Nullable | Default | Keterangan & Constraints |
| :--- | :--- | :---: | :--- | :--- |
| `id` | `UUID` | No | `gen_random_uuid()` | Primary Key |
| `study_plan_id` | `UUID` | No | - | Foreign Key ke `study_plans(id)` `ON DELETE RESTRICT` |
| `student_id` | `UUID` | No | - | Foreign Key ke `students(id)` `ON DELETE RESTRICT` |
| `class_id` | `UUID` | No | - | Foreign Key ke `classes(id)` `ON DELETE RESTRICT` |
| `status` | `VARCHAR(20)` | No | `'ENROLLED'` | Status: `ENROLLED`, `DROPPED`, `CANCELLED` |
| `enrolled_at` | `TIMESTAMPTZ` | No | `CURRENT_TIMESTAMP` | Timestamp registrasi kelas |
| `dropped_at` | `TIMESTAMPTZ` | Yes | `NULL` | Timestamp pembatalan kelas |
| `created_at` | `TIMESTAMPTZ` | No | `CURRENT_TIMESTAMP` | Waktu pembuatan |
| `updated_at` | `TIMESTAMPTZ` | No | `CURRENT_TIMESTAMP` | Waktu pembaruan |

**Constraints & Indeks Parsial:**
- `CONSTRAINT chk_enrollments_status CHECK (status IN ('ENROLLED', 'DROPPED', 'CANCELLED'))`
- `CREATE UNIQUE INDEX uq_enrollments_active_student_class ON enrollments(student_id, class_id) WHERE status = 'ENROLLED'`: Menjamin seorang mahasiswa tidak dapat terdaftar ganda secara aktif pada kelas yang sama.
- `CREATE UNIQUE INDEX uq_enrollments_active_study_plan_class ON enrollments(study_plan_id, class_id) WHERE status = 'ENROLLED'`
- `idx_enrollments_study_plan_id ON enrollments(study_plan_id)`
- `idx_enrollments_student_id ON enrollments(student_id)`
- `idx_enrollments_class_id ON enrollments(class_id)`
- `idx_enrollments_status ON enrollments(status)`

---

## 4. Strategi Transaksi dan Pencegahan Race Condition

### 4.1. Row-Level Locking pada Kapasitas Kelas
Untuk mencegah *overbooking* saat banyak mahasiswa mendaftar ke kelas yang tersisa sedikit kursi secara bersamaan:
1. Operasi pendaftaran dijalankan dalam transaksi PostgreSQL eksplisit (`BEGIN`).
2. Baris kelas dikunci menggunakan `SELECT ... FROM classes WHERE id = $1 FOR UPDATE OF c`. Transaksi konkuren yang menargetkan kelas yang sama akan mengantre secara serial.
3. Kueri kapasitas aktif dievaluasi langsung di bawah *lock*:
   `SELECT COUNT(*) FROM enrollments WHERE class_id = $1 AND status = 'ENROLLED'`.
4. Jika `active_count >= capacity`, transaksi di-rollback seketika dan mengembalikan `409 Conflict: Kapasitas kelas telah penuh`.
5. Jika kuota mencukupi, baris enrollment baru di-insert dan transaksi di-commit.

### 4.2. Row-Level Locking pada KRS Mahasiswa
1. Baris `study_plans` dikunci menggunakan `SELECT ... FROM study_plans WHERE id = $1 FOR UPDATE`.
2. Validasi batas SKS (`total_credits + course.credits <= 24`) dievaluasi di bawah lock, mencegah transaksi paralel dari mahasiswa yang sama melampaui batas maksimum 24 SKS.
3. Total SKS diperbarui secara atomik dalam transaksi yang sama.

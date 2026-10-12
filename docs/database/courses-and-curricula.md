# Dokumentasi Database: Modul Mata Kuliah dan Kurikulum (Fase 07)

## 1. Ringkasan Skema

Modul Mata Kuliah dan Kurikulum memperkenalkan 4 tabel relasional baru pada skema PostgreSQL:
1. `courses`: Katalog mata kuliah induk perguruan tinggi.
2. `course_prerequisites`: Relasi adjacency graph prasyarat antarmata kuliah.
3. `curricula`: Rancangan kurikulum per program studi beserta status lifecycle.
4. `curriculum_courses`: Relasi mata kuliah dalam struktur kurikulum program studi.

Migration: `migrations/000004_create_courses_and_curricula_tables.up.sql` dan `.down.sql`.

---

## 2. Definisi Tabel

### 2.1. Tabel `courses`
Katalog mata kuliah akademik yang menjadi sumber kebenaran data SKS dan kode mata kuliah.

| Kolom | Tipe | Constraint | Keterangan |
|---|---|---|---|
| `id` | UUID | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Pengenal unik mata kuliah |
| `code` | VARCHAR(20) | NOT NULL, UNIQUE | Kode mata kuliah (disimpan uppercase, e.g. `TIF101`) |
| `name` | VARCHAR(255) | NOT NULL | Nama resmi mata kuliah |
| `description` | TEXT | NULL | Deskripsi atau silabus ringkas |
| `credits` | SMALLINT | NOT NULL, CHECK (`credits >= 1 AND credits <= 8`) | Beban SKS mata kuliah (sumber kebenaran tunggal) |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT `true` | Status keaktifan mata kuliah |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu pembuatan |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu pembaruan terakhir |

**Index:**
- `idx_courses_code`: UNIQUE (`code`)
- `idx_courses_is_active`: B-Tree (`is_active`)
- `idx_courses_credits`: B-Tree (`credits`)

---

### 2.2. Tabel `course_prerequisites`
Tabel asosiasi penentu prasyarat mata kuliah. Menghubungkan suatu mata kuliah dengan mata kuliah prasyaratnya.

| Kolom | Tipe | Constraint | Keterangan |
|---|---|---|---|
| `id` | UUID | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Pengenal relasi prasyarat |
| `course_id` | UUID | NOT NULL, FK -> `courses(id)` ON DELETE RESTRICT | Mata kuliah target yang mensyaratkan |
| `prerequisite_course_id` | UUID | NOT NULL, FK -> `courses(id)` ON DELETE RESTRICT | Mata kuliah yang menjadi syarat |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu penambahan relasi |

**Constraint Tambahan:**
- `chk_no_self_prereq`: `CHECK (course_id != prerequisite_course_id)` (mencegah self-prerequisite)
- `uq_course_prerequisite`: `UNIQUE (course_id, prerequisite_course_id)` (mencegah duplikasi relasi)

**Index:**
- `idx_course_prereq_course_id`: B-Tree (`course_id`)
- `idx_course_prereq_prereq_id`: B-Tree (`prerequisite_course_id`)

---

### 2.3. Tabel `curricula`
Master kurikulum per program studi dengan pelacakan lifecycle status.

| Kolom | Tipe | Constraint | Keterangan |
|---|---|---|---|
| `id` | UUID | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Pengenal unik kurikulum |
| `study_program_id` | UUID | NOT NULL, FK -> `study_programs(id)` ON DELETE RESTRICT | Relasi ke program studi penyelenggara |
| `code` | VARCHAR(30) | NOT NULL, UNIQUE | Kode unik kurikulum (e.g. `KUR-IF-2024`) |
| `name` | VARCHAR(255) | NOT NULL | Nama kurikulum (e.g. `Kurikulum OBE Teknik Informatika 2024`) |
| `year` | SMALLINT | NOT NULL, CHECK (`year >= 1970 AND year <= 2100`) | Tahun angkatan penerapan kurikulum |
| `description` | TEXT | NULL | Penjelasan latar belakang / profil lulusan |
| `status` | VARCHAR(20) | NOT NULL, DEFAULT `'DRAFT'`, CHECK (`status IN ('DRAFT', 'ACTIVE', 'INACTIVE', 'ARCHIVED')`) | Status lifecycle kurikulum |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu pembuatan |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu pembaruan terakhir |

**Index:**
- `idx_curricula_code`: UNIQUE (`code`)
- `idx_curricula_study_program_id`: B-Tree (`study_program_id`)
- `idx_curricula_status`: B-Tree (`status`)
- `idx_curricula_year`: B-Tree (`year`)

---

### 2.4. Tabel `curriculum_courses`
Relasi mata kuliah yang diajarkan pada suatu kurikulum, menentukan semester rekomendasi dan kategori kelulusan.

| Kolom | Tipe | Constraint | Keterangan |
|---|---|---|---|
| `id` | UUID | PRIMARY KEY, DEFAULT `gen_random_uuid()` | Pengenal relasi kurikulum-mata kuliah |
| `curriculum_id` | UUID | NOT NULL, FK -> `curricula(id)` ON DELETE RESTRICT | Kurikulum yang memuat mata kuliah |
| `course_id` | UUID | NOT NULL, FK -> `courses(id)` ON DELETE RESTRICT | Mata kuliah yang dimasukkan |
| `recommended_semester` | SMALLINT | NOT NULL, CHECK (`recommended_semester >= 1 AND recommended_semester <= 14`) | Semester rekomendasi pengambilan (1-14) |
| `category` | VARCHAR(20) | NOT NULL, DEFAULT `'REQUIRED'`, CHECK (`category IN ('REQUIRED', 'ELECTIVE')`) | Kategori mata kuliah (Wajib / Pilihan) |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu penambahan |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT `NOW()` | Waktu pembaruan |

**Constraint Tambahan:**
- `uq_curriculum_course`: `UNIQUE (curriculum_id, course_id)` (mencegah mata kuliah ganda dalam 1 kurikulum)

**Index:**
- `idx_curr_courses_curriculum_id`: B-Tree (`curriculum_id`)
- `idx_curr_courses_course_id`: B-Tree (`course_id`)
- `idx_curr_courses_semester`: B-Tree (`recommended_semester`)
- `idx_curr_courses_category`: B-Tree (`category`)

---

## 3. Sumber Kebenaran SKS dan Agregasi Dinamis

1. Nilai SKS **tidak diduplikasi** pada tabel `curricula` maupun `curriculum_courses`.
2. Sumber kebenaran SKS hanya berada di `courses.credits`.
3. Total SKS kurikulum dihitung secara dinamis melalui query agregasi:
```sql
SELECT 
    COUNT(cc.id) AS total_courses,
    COALESCE(SUM(c.credits), 0) AS total_credits,
    COALESCE(SUM(CASE WHEN cc.category = 'REQUIRED' THEN c.credits ELSE 0 END), 0) AS required_credits,
    COALESCE(SUM(CASE WHEN cc.category = 'ELECTIVE' THEN c.credits ELSE 0 END), 0) AS elective_credits
FROM curriculum_courses cc
JOIN courses c ON c.id = cc.course_id
WHERE cc.curriculum_id = $1
```
Hal ini menjamin integritas data selalu konsisten tanpa risiko desinkronisasi kolom counter/cache.

---

## 4. Integritas Referensial dan Proteksi Penghapusan

- Semua foreign key menggunakan klausul `ON DELETE RESTRICT`.
- Program studi tidak dapat dihapus jika masih direferensikan kurikulum.
- Mata kuliah tidak dapat dihapus jika masih tercatat dalam kurikulum atau menjadi prasyarat mata kuliah lain.
- Kurikulum tidak dapat dihapus jika masih memiliki relasi mata kuliah.

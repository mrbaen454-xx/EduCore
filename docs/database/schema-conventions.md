# Konvensi Skema Database EduCore (Fase 04)

Dokumen ini mendefinisikan aturan konvensi skema, tipe data, indeks, constraint, dan standar penambahan tabel untuk **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Aturan Penamaan (*Naming Conventions*)

Semua identifier database wajib menggunakan huruf kecil dan garis bawah (**`snake_case`**).

| Objek | Format / Pola | Contoh |
|---|---|---|
| **Tabel** | Jamak (*plural*), kata benda | `users`, `roles`, `students`, `courses` |
| **Tabel Pivot (M-to-N)** | Kombinasi nama tabel terkait | `user_roles`, `curriculum_courses` |
| **Kolom** | Tunggal (*singular*), deskriptif | `username`, `email`, `created_at` |
| **Primary Key (PK)** | Kolom tunggal `id` | `id` |
| **Foreign Key (FK)** | `<entitas_target>_id` | `role_id`, `faculty_id`, `student_id` |
| **Constraint PK** | `pk_<tabel>` | `pk_users`, `pk_roles` |
| **Constraint FK** | `fk_<tabel_asal>_<tabel_tujuan>` | `fk_user_roles_users`, `fk_user_roles_roles` |
| **Constraint Unique** | `uq_<tabel>_<kolom>` | `uq_users_email`, `uq_roles_name` |
| **Constraint Check** | `chk_<tabel>_<kondisi>` | `chk_users_email_format`, `chk_users_status` |
| **Indeks Biasa** | `idx_<tabel>_<kolom>` | `idx_users_status`, `idx_user_roles_role_id` |

---

## 2. Standar Tipe Data PostgreSQL

| Data | Tipe Rekomendasi | Alasan & Catatan |
|---|---|---|
| **Identitas (PK)** | `UUID` (v4) | Menggunakan `gen_random_uuid()` dari ekstensi `pgcrypto`. Mencegah enumerasi ID numerik pada endpoint publik. |
| **Teks Pendek (Berbatas)** | `VARCHAR(N)` | Digunakan ketika panjang karakter dibatasi oleh standar/aturan bisnis (misal: kode role, NIM, NIDN). |
| **Teks Bebas / Deskripsi** | `TEXT` | Deskripsi, konten keterangan, catatan akademik. |
| **Boolean** | `BOOLEAN` | Nilai biner ya/tidak (`is_active`, `is_verified`). Default harus didefinisikan (`DEFAULT true/false`). |
| **Tanggal Kalender** | `DATE` | Tanggal lahir, tanggal mulai semester tanpa komponen jam. |
| **Waktu / Kejadian** | `TIMESTAMPTZ` | Timestamp waktu kejadian (`created_at`, `updated_at`, `submitted_at`). Selalu simpan dengan zona waktu UTC. |
| **Nilai Desimal / Keuangan** | `NUMERIC(p, s)` | IPK / SKS / Pembayaran UKT (`NUMERIC(4, 2)` atau `NUMERIC(12, 2)`). Jangan gunakan `FLOAT` karena masalah presisi. |
| **Bilangan Bulat** | `INTEGER` / `SMALLINT` | Kuota kelas, jumlah SKS, semester ke-n. |

---

## 3. Kebijakan Tanggal dan Waktu

1. **Waktu Kejadian**:
   - Selalu gunakan `TIMESTAMPTZ`.
   - Gunakan `DEFAULT CURRENT_TIMESTAMP` untuk `created_at` dan `updated_at`.
2. **Zona Waktu Aplikasi**:
   - Konfigurasi aplikasi default menggunakan `Asia/Jakarta` (WIB / UTC+7).
   - Di level PostgreSQL, data disimpan dalam representasi UTC dan dikonversi saat serialisasi JSON di API layer.
3. **Tanggal Akademik Murni**:
   - Tanggal kalender tanpa jam (misal: batas akhir pembayaran, rentang waktu pengisian KRS) menggunakan `DATE` untuk menghindari kerancuan pergeseran hari akibat konversi timezone.

---

## 4. Kebijakan Integritas dan Constraint

1. **Primary Key**:
   - Wajib ada pada setiap tabel.
   - Tabel pivot dapat menggunakan *composite primary key* jika kombinasi relasi bersifat unik:
     ```sql
     CONSTRAINT pk_user_roles PRIMARY KEY (user_id, role_id)
     ```
2. **Foreign Key**:
   - Wajib ada untuk setiap kolom relasi antar tabel.
   - **Kebijakan `ON DELETE`**:
     - **`ON DELETE CASCADE`**: Hanya diperbolehkan untuk tabel junction/pivot atau log dependen yang tidak memiliki makna mandiri jika induknya dihapus (misal: `user_roles`).
     - **`ON DELETE RESTRICT` / `NO ACTION` (Default)**: Wajib untuk seluruh entitas akademik (Mahasiswa, Dosen, Mata Kuliah, KRS, Nilai) guna mencegah penghapusan data bersejarah secara berantai tanpa izin.
3. **Check Constraint**:
   - Wajib digunakan untuk memvalidasi batas nilai stabil di level basis data, misalnya status user (`chk_users_status CHECK (status IN ('active', 'inactive', 'suspended'))`).

---

## 5. Indeks (*Indexing*)

1. **Indeks Otomatis**: PostgreSQL secara otomatis membuat *b-tree index* untuk kolom `PRIMARY KEY` dan `UNIQUE`.
2. **Indeks Kolom Foreign Key**: Wajib ditambahkan secara manual pada kolom FK yang sering digunakan untuk *JOIN* atau filter, misalnya:
   ```sql
   CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
   ```
3. **Indeks Filter**: Tambahkan indeks pada kolom yang menjadi predikat filter frekuensi tinggi, misalnya:
   ```sql
   CREATE INDEX idx_users_status ON users(status);
   ```
4. **Hindari Over-Indexing**: Jangan menambahkan indeks pada kolom dengan kardinalitas rendah jika tabel jarang difilter, atau pada kolom yang jarang dibaca namun sering di-update.

---

## 6. Baseline Skema Fase 04

Skema baseline yang telah diterapkan pada migrasi `000001_initial_schema.up.sql`:

```text
+---------------------+           +------------------------+           +---------------------+
|        roles        |           |       user_roles       |           |        users        |
+---------------------+           +------------------------+           +---------------------+
| id (UUID, PK)       |<----------| user_id (UUID, PK, FK) |---------->| id (UUID, PK)       |
| name (VARCHAR, UQ)  |           | role_id (UUID, PK, FK) |           | email (VARCHAR, UQ) |
| description (TEXT)  |           | created_at (TZ)        |           | username (VRCHR, UQ)|
| created_at (TZ)     |           +------------------------+           | password_hash (TEXT)|
| updated_at (TZ)     |                                                | full_name (VARCHAR) |
+---------------------+                                                | status (VARCHAR)    |
                                                                       | created_at (TZ)     |
                                                                       | updated_at (TZ)     |
                                                                       +---------------------+
```

### Seed Data Baseline:
Role baku sistem:
- `superadmin`: Pengelola infrastruktur dan sistem EduCore.
- `academic_admin`: Staf BAAK / administrasi akademik.
- `faculty_admin`: Staf tata usaha fakultas.
- `program_admin`: Pengelola program studi / departemen.
- `lecturer`: Dosen pengampu dan dosen pembimbing akademik.
- `student`: Mahasiswa aktif.

---

## 7. Roadmap Skema Akademik (Fase Selanjutnya)

Penambahan tabel akademik pada fase berikutnya akan mengikuti konvensi dokumen ini:

1. **Struktur Organisasi**:
   - `faculties` (Fakultas: id, code, name, description)
   - `study_programs` (Program Studi: id, faculty_id, code, name, degree_level)
2. **Profil Akademik**:
   - `students` (Mahasiswa: id, user_id, study_program_id, nim, enrollment_year, academic_status)
   - `lecturers` (Dosen: id, user_id, nidn, nip, title_prefix, title_suffix)
3. **Kurikulum & Pembelajaran**:
   - `academic_periods` (Periode Akademik: id, name, semester_type, start_date, end_date, is_active)
   - `courses` (Mata Kuliah: id, study_program_id, code, name, credit_units)
   - `classes` (Kelas Perkuliahan: id, course_id, academic_period_id, lecturer_id, class_name, capacity)
4. **Transaksi & Hasil Studi**:
   - `enrollments` (IRS/KRS: id, student_id, class_id, status)
   - `grades` (Nilai: id, enrollment_id, numerical_grade, letter_grade, grade_point)
   - `attendances` (Presensi Perkuliahan: id, class_id, student_id, session_number, status)

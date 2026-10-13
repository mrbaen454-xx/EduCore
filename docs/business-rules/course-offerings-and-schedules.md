# Aturan Bisnis & Integritas: Penawaran Kelas dan Jadwal Perkuliahan (Fase 08)

## 1. Aturan Bisnis Periode Akademik (`academic_periods`)

1. **Kode & Nama Periode**:
   - Kode wajib diisi (`length > 0`), maksimal 20 karakter, unik di seluruh sistem.
   - Contoh kode standar: `20241` (Ganjil 2024/2025), `20242` (Genap 2024/2025), `20243` (Semester Antara/Pendek 2024/2025).
2. **Jenis Semester**:
   - Nilai enum yang diizinkan: `ODD` (Ganjil), `EVEN` (Genap), `SHORT` (Semester Antara/Pendek).
3. **Validitas Tanggal**:
   - `start_date` dan `end_date` harus berupa format tanggal kalender yang valid (`YYYY-MM-DD`).
   - Tanggal mulai harus lebih awal atau sama dengan tanggal selesai (`start_date <= end_date`). Pelanggaran akan ditolak dengan error `400 Bad Request`.
4. **Status Keaktifan (`is_active`)**:
   - Hanya periode akademik aktif yang umumnya digunakan untuk pembukaan pendaftaran KRS dan perkuliahan berjalan.

---

## 2. Aturan Bisnis Penawaran Mata Kuliah (`course_offerings`)

1. **Integritas Entitas Rujukan**:
   - Periode akademik (`academic_period_id`), mata kuliah (`course_id`), dan program studi (`study_program_id`) harus terdaftar di sistem.
   - Mata kuliah yang berstatus nonaktif (`is_active = false`) dilarang ditawarkan (`409 Conflict`).
   - Program studi yang berstatus nonaktif (`is_active = false`) dilarang membuat penawaran baru (`409 Conflict`).
   - Jika kurikulum (`curriculum_id`) disertakan, kurikulum tersebut harus valid dan terdaftar di bawah program studi yang bersangkutan.
2. **Keunikan Penawaran**:
   - Kombinasi `(academic_period_id, course_id, study_program_id)` dilindungi oleh *unique constraint*. Upaya menawarkan mata kuliah yang sama dua kali pada periode dan prodi yang sama menghasilkan `409 Conflict`.
3. **Lifecycle Status Penawaran**:
   - Status yang didukung: `PLANNED` (Direncanakan), `OPEN` (Dibuka untuk pendaftaran/KRS), `CLOSED` (Ditutup), `CANCELLED` (Dibatalkan).
   - Penawaran yang telah berstatus `CANCELLED` tidak dapat diaktifkan kembali.

---

## 3. Aturan Bisnis Kelas Perkuliahan (`classes`)

1. **Kode & Identitas Kelas**:
   - Kode kelas (contoh: `A`, `B`, `K01`, `IF-01`) wajib unik di bawah penawaran mata kuliah yang sama (`UNIQUE(course_offering_id, code)`).
   - Upaya membuat kelas dengan kode yang sama di penawaran yang sama menghasilkan `409 Conflict`.
2. **Kapasitas Kelas**:
   - Kapasitas wajib berupa bilangan bulat positif (`capacity > 0`). Nilai nol atau negatif ditolak dengan `400 Bad Request`.
3. **Lifecycle Status Kelas**:
   - Status: `DRAFT`, `OPEN`, `IN_PROGRESS`, `CLOSED`, `CANCELLED`.
   - Perubahan status diverifikasi sesuai status operasional.

---

## 4. Aturan Penugasan Dosen Pengajar (`class_lecturers`)

1. **Integritas Dosen**:
   - Dosen yang ditugaskan harus terdaftar pada tabel `lecturers` dan berstatus aktif (`ACTIVE`).
2. **Keunikan Penugasan**:
   - Satu dosen hanya dapat ditugaskan satu kali pada kelas yang sama (`UNIQUE(class_id, lecturer_id)`). Upaya penugasan ganda menghasilkan `409 Conflict`.
3. **Peran Pengajar**:
   - Nilai peran yang diizinkan: `COORDINATOR` (Dosen Koordinator Pengampu) dan `TEAM_TEACHING` (Dosen Anggota Team Teaching).

---

## 5. Model Waktu & Algoritma Deteksi Bentrok Jadwal (`class_schedules`)

1. **Model Interval Waktu Setengah Terbuka $[start, end)$**:
   - Slot waktu menggunakan interval setengah terbuka $[start, end)$.
   - Dua interval $[s_A, e_A)$ dan $[s_B, e_B)$ dianggap **bentrok / bertabrakan** jika dan hanya jika:
     $$s_A < e_B \quad \text{dan} \quad s_B < e_A$$
   - **Perkuliahan Berurutan Diperbolehkan**: Jika Kelas 1 selesai pada pukul `10:00:00` dan Kelas 2 dimulai tepat pada pukul `10:00:00` di ruangan atau dengan dosen yang sama, keduanya **tidak bertabrakan**.
2. **Aturan Hari & Jam**:
   - Hari dalam minggu dinyatakan dengan integer 1 sampai 7 (1 = Senin, ..., 7 = Minggu).
   - Jam mulai harus lebih awal dari jam selesai (`start_time < end_time`).
3. **Cakupan Pemeriksaan Bentrok**:
   - **Bentrok Internal Kelas**: Suatu kelas tidak boleh memiliki lebih dari satu jadwal pada hari yang sama dengan interval waktu yang bertabrakan.
   - **Bentrok Jadwal Dosen**: Seluruh dosen yang ditugaskan pada kelas diverifikasi agar tidak mengajar di kelas lain pada periode akademik yang sama di hari yang sama dengan interval yang bertabrakan.
   - **Bentrok Jadwal Ruangan**: Jika kolom `room` diisi (non-empty), ruangan tersebut diverifikasi agar tidak digunakan oleh kelas lain pada periode akademik yang sama di hari yang sama dengan interval yang bertabrakan.

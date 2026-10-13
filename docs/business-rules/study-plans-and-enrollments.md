# Aturan Bisnis & Integritas Akademik: KRS dan Registrasi Mahasiswa (Fase 09)

## 1. Aturan Validasi Mahasiswa

1. **Kelayakan Status Akademik**:
   - Hanya mahasiswa dengan status `ACTIVE` yang diizinkan untuk membuat KRS dan mendaftarkan kelas perkuliahan.
   - Mahasiswa dengan status `INACTIVE`, `PROBATION`, `GRADUATED`, `DROPOUT`, atau `SUSPENDED` dilarang melakukan transaksi KRS dan ditolak dengan respons `409 Conflict: Mahasiswa berstatus <STATUS> tidak memenuhi syarat untuk pengisian KRS`.
2. **Keterikatan Akun Pengguna (`user_id`)**:
   - Untuk endpoint mahasiswa (`/api/v1/student/...`), profil mahasiswa diambil dari `principal.UserID` pada token autentikasi JWT.
   - Mahasiswa tidak dapat mengubah ID pemilik KRS atau mendaftarkan kelas atas nama mahasiswa lain.

---

## 2. Aturan Periode Akademik & KRS

1. **Keaktifan Periode Akademik**:
   - Pengisian dan pembaruan KRS hanya dapat dilakukan pada periode akademik yang berstatus aktif (`is_active = true`).
   - Upaya membuat KRS atau menambah kelas pada periode nonaktif ditolak dengan `409 Conflict: Periode akademik tidak aktif untuk pengisian KRS`.
2. **Keunikan KRS per Semester**:
   - Mahasiswa hanya memiliki maksimal 1 KRS pada satu periode akademik (`UNIQUE(student_id, academic_period_id)`).
   - Pemanggilan ulang endpoint pembuatan KRS pada periode yang sama bersifat idempoten: mengembalikan data KRS yang sudah ada dengan respons `200 OK`.
3. **Kesesuaian Periode Penawaran**:
   - Kelas yang didaftarkan harus berasal dari penawaran mata kuliah pada periode akademik yang sama persis dengan KRS (`course_offering.academic_period_id == study_plan.academic_period_id`). Pelanggaran ditolak dengan `409 Conflict`.

---

## 3. Aturan Registrasi Kelas Perkuliahan

1. **Status Kesiapan Kelas & Penawaran**:
   - Penawaran mata kuliah (`course_offerings`) harus berstatus `OPEN`.
   - Kelas perkuliahan (`classes`) harus berstatus `OPEN`.
   - Kelas dengan status `DRAFT`, `IN_PROGRESS`, `CLOSED`, atau `CANCELLED` tidak dapat didaftarkan.
2. **Batas Beban SKS Semester (Credit Limit)**:
   - Batas beban studi per semester adalah **maksimum 24 SKS** (sesuai dokumen kebutuhan Fase 01).
   - Penambahan kelas yang menyebabkan $\text{total\_credits} + \text{course.credits} > 24$ ditolak dengan `409 Conflict: Total SKS melebihi batas maksimum 24 SKS per semester`.
3. **Pencegahan Registrasi Ganda (Duplicate Prevention)**:
   - **Pencegahan Kelas Sama**: Mahasiswa tidak dapat mendaftar dua kali ke kelas yang sama (`409 Conflict`).
   - **Pencegahan Mata Kuliah Sama**: Mahasiswa dilarang mendaftar ke kelas lain untuk mata kuliah yang sama dalam satu semester (misalnya mengambil Kelas Algoritma A dan Algoritma B secara bersamaan). Pelanggaran ditolak dengan `409 Conflict`.
4. **Kapasitas dan Kuota Kelas**:
   - Pendaftaran hanya diizinkan jika jumlah peserta aktif (`status = 'ENROLLED'`) masih di bawah kapasitas kelas (`active_count < capacity`).
   - Jika kuota penuh, transaksi ditolak dengan `409 Conflict: Kapasitas kelas telah penuh`.
5. **Algoritma Deteksi Bentrok Jadwal (Schedule Clash Prevention)**:
   - Waktu perkuliahan menggunakan model interval setengah terbuka $[start, end)$.
   - Dua jadwal $A$ dan $B$ bentrok jika dan hanya jika pada hari yang sama berlaku:
     $$\text{start}_A < \text{end}_B \quad \text{AND} \quad \text{start}_B < \text{end}_A$$
   - Jadwal berurutan (misal Kelas 1 selesai pukul 10:00 dan Kelas 2 mulai tepat pukul 10:00) **tidak bentrok**.
   - Jika bentrok terdeteksi, pendaftaran ditolak dengan `409 Conflict` disertai informasi nama mata kuliah dan rentang jam yang bertubrukan.

---

## 4. Aturan Pembatalan Kelas (Drop Enrollment)

1. **Pelepasan Kuota dan Penyesuaian SKS**:
   - Pembatalan kelas mengubah status pendaftaran menjadi `DROPPED` dan mencatat timestamp `dropped_at`.
   - Kursi kelas dilepaskan sehingga mahasiswa lain dapat mendaftar (perhitungan kuota aktif hanya menjumlahkan `status = 'ENROLLED'`).
   - Beban SKS pada KRS berkurang sesuai SKS mata kuliah yang dibatalkan.
2. **Pencegahan Pembatalan Ganda**:
   - Pendaftaran yang sudah berstatus `DROPPED` tidak dapat dibatalkan ulang (`409 Conflict`).

---

## 5. Kebijakan Prasyarat Mata Kuliah (Prerequisites)

- Sesuai batasan ruang lingkup Fase 09, modul penilaian mahasiswa (*grades*), transkrip nilai, dan Indeks Prestasi (IP/IPK) berada pada fase berikutnya (Fase 10+).
- Evaluasi prasyarat kelulusan mata kuliah lampau dipisahkan dan ditunda hingga data riwayat nilai akademik sah tersedia, untuk mencegah pemalsuan atau asumsi nilai kelulusan palsu.

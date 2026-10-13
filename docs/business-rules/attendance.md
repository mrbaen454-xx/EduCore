# Aturan Bisnis & Integritas Presensi Mahasiswa (Fase 10)
## EduCore — Academic Management System

Dokumen ini mendefinisikan aturan bisnis, batasan integritas data, kewenangan akses, serta kebijakan akademik yang diterapkan pada modul Presensi dan Kehadiran Mahasiswa.

---

## 1. Kewenangan Akses & Otorisasi Pengguna

1. **Dosen Pengampu Kelas**:
   - Dosen hanya berwenang mengelola sesi perkuliahan dan mencatat presensi untuk kelas yang diampunya (`class_lecturers.class_id = classID AND lecturers.user_id = principal.UserID AND class_lecturers.status = 'ACTIVE'`).
   - Dosen yang tidak ditugaskan pada kelas bersangkutan dilarang membuat sesi, mengubah sesi, melihat roster peserta, mencatat presensi, maupun mengoreksi presensi (`403 Forbidden: Anda bukan dosen yang ditugaskan pada kelas ini`).
2. **Hak Administratif (Admin & Akademik)**:
   - Pengguna dengan peran `admin` atau `akademik` memiliki izin administratif untuk mengelola sesi perkuliahan dan presensi seluruh kelas (*bypass class lecturer assignment check*).
3. **Mahasiswa**:
   - Mahasiswa hanya memiliki akses baca (*read-only*) terhadap riwayat presensinya sendiri (`/api/v1/student/attendance` dan `/api/v1/student/attendance/classes/:id`).
   - Identitas mahasiswa wajib diekstraksi dari token autentikasi JWT (`principal.UserID`), dilarang mempercayai ID mahasiswa dari client input.
   - Mahasiswa hanya dapat melihat daftar sesi dari kelas yang diikutinya secara aktif (`enrollments.status = 'ENROLLED'`).
   - Mahasiswa dilarang melihat presensi mahasiswa lain atau mengubah status presensinya sendiri (`403 Forbidden`).

---

## 2. Aturan Sesi Perkuliahan (Class Sessions)

1. **Keterikatan Kelas**:
   - Setiap sesi perkuliahan wajib merujuk ke penawaran kelas yang valid dan aktif (`classes.id`).
2. **Keunikan Nomor Pertemuan**:
   - Nomor pertemuan (`session_number`) bersifat unik dalam lingkup kelas yang sama (`UNIQUE(class_id, session_number)`).
   - Upaya membuat atau memperbarui sesi dengan nomor pertemuan yang sudah ada pada kelas tersebut ditolak dengan `409 Conflict`.
3. **Validasi Waktu Perkuliahan**:
   - Format tanggal wajib `YYYY-MM-DD`.
   - Format jam mulai (`start_time`) dan jam selesai (`end_time`) wajib `HH:MM` atau `HH:MM:SS`.
   - Jam selesai perkuliahan harus setelah jam mulai perkuliahan (`end_time > start_time`). Pelanggaran ditolak dengan `400 Bad Request` (Validation Error).
4. **Siklus Hidup & Status Sesi**:
   - Status sesi yang didukung:
     - `SCHEDULED`: Sesi terjadwal, belum dimulai.
     - `OPEN`: Sesi sedang berlangsung, presensi dapat dicatat.
     - `COMPLETED`: Sesi perkuliahan telah ditutup / selesai.
     - `CANCELLED`: Sesi perkuliahan dibatalkan.
   - Sesi yang berstatus `CANCELLED` tidak dapat diisi presensi maupun diperbarui (`409 Conflict: Sesi perkuliahan telah dibatalkan`).
   - Penutupan sesi (`POST /api/v1/sessions/:id/close`) mengubah status menjadi `COMPLETED`. Penutupan ulang pada sesi yang sudah `COMPLETED` ditolak dengan `409 Conflict`.

---

## 3. Aturan Peserta dan Pencatatan Presensi

1. **Validitas Peserta Kelas (KRS / Enrollment Integration)**:
   - Presensi mahasiswa **hanya boleh dicatat** jika mahasiswa terdaftar aktif pada kelas tersebut (`enrollments.class_id = classID AND enrollments.status = 'ENROLLED'`).
   - Mahasiswa yang belum mendaftar KRS, atau yang sudah membatalkan kelas (`status = 'DROPPED'`), dilarang diberi presensi.
   - Sistem mencatat referensi `enrollment_id` secara langsung pada tabel `attendances` untuk menjaga ketertelusuran data registrasi akademik.
2. **Status Kehadiran Resmi**:
   - Status kehadiran mahasiswa dibatasi secara ketat oleh enum domain berikut:
     - `PRESENT`: Mahasiswa hadir pada sesi perkuliahan.
     - `LATE`: Mahasiswa hadir terlambat.
     - `EXCUSED`: Mahasiswa izin dengan alasan akademik/resmi.
     - `SICK`: Mahasiswa tidak hadir karena sakit.
     - `ABSENT`: Mahasiswa tidak hadir tanpa keterangan (alpa).
   - Status selain kelima nilai di atas ditolak dengan `400 Bad Request`.
3. **Pencatatan Presensi Batch (Atomic Transaction)**:
   - Dosen dapat mencatat presensi beberapa mahasiswa sekaligus dalam satu request (`POST /api/v1/sessions/:id/attendance`).
   - Batas maksimum batch per request adalah 200 mahasiswa.
   - Seluruh payload diverifikasi: jika terdapat mahasiswa yang bukan peserta aktif kelas, atau terdapat duplikasi `student_id` dalam payload, seluruh transaksi dibatalkan (*all-or-nothing rollback*).
   - Sistem tidak mengizinkan keberhasilan parsial tanpa pelaporan kegagalan.
4. **Pencegahan Presensi Ganda (Duplicate Prevention)**:
   - Satu mahasiswa hanya dapat memiliki maksimal 1 catatan presensi per sesi perkuliahan (`CONSTRAINT uq_attendances_session_student UNIQUE(session_id, student_id)`).
   - Pada tingkat database, operasi batch menggunakan `ON CONFLICT (session_id, student_id) DO UPDATE` dengan penguncian baris (`FOR UPDATE`) pada sesi, menjamin idempotensi dan konsistensi pada kondisi konkurensi tinggi.
5. **Koreksi Presensi**:
   - Presensi yang telah dicatat dapat dikoreksi oleh dosen pengampu atau staf akademik melalui endpoint `PATCH /api/v1/sessions/:id/attendance/:attendanceId`.
   - Perubahan status mencatat identitas pelaku perubahan (`recorded_by`) dan waktu pembaruan (`updated_at`).

---

## 4. Rekapitulasi dan Statistik Kehadiran

1. **Rekap Sesi**:
   - Menghitung total mahasiswa terdaftar aktif, total presensi tercatat, serta rincian per status (`present`, `late`, `excused`, `sick`, `absent`).
   - Mahasiswa yang belum dicatat presensinya **tidak otomatis dianggap alpa** sebelum ada pencatatan eksplisit dari dosen/staf.
2. **Rekapitulasi Kelas**:
   - Menyajikan rekapitulasi kumulatif per mahasiswa pada kelas tersebut:
     - Total sesi kelas yang diselenggarakan.
     - Rincian kehadiran per status.
     - Persentase kehadiran:
       $$\text{Persentase} = \frac{\text{Present} + \text{Late}}{\text{Total Sessions}} \times 100\%$$
       *(Catatan: Rumus di atas adalah agregasi kalkulasi netral. Sistem tidak menentukan batas kelulusan atau syarat ujian secara sepihak pada fase ini).*

---

## 5. Batasan Akademik yang Belum Ditentukan (Scope Boundaries)

Sesuai instruksi proyek, batasan kebijakan berikut **belum disahkan** dan sengaja **tidak dikarang atau diimplementasikan**:
1. **Persentase Minimum Kehadiran untuk Syarat Ujian (UTS/UAS)**: Belum ada keputusan resmi senat akademik mengenai batas minimal 75% atau 80%. Sistem menyajikan persentase netral tanpa memblokir ujian.
2. **Bobot Kehadiran terhadap Nilai Akhir**: Pengelolaan komponen nilai akhir merupakan bagian dari Fase 11 (Penilaian).
3. **Mekanisme Check-in Mandiri Mahasiswa (QR Code, GPS, Geofencing, PIN)**: Belum disetujui. Seluruh presensi saat ini dicatat oleh dosen atau petugas akademik resmi.
4. **Masa Sanggah / Batas Waktu Koreksi Presensi**: Belum ada batasan hari (misal 7 hari setelah sesi). Koreksi dapat dilakukan selama sesi valid.

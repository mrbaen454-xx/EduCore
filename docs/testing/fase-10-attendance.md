# Laporan Pengujian dan Verifikasi: Fase 10 (Presensi Mahasiswa)
## EduCore — Academic Management System

Dokumen ini mencatat metodologi, prasyarat pengujian, perintah yang dijalankan, serta bukti hasil pengujian nyata untuk Fase 10.

---

## 1. Lingkungan Pengujian

- **Sistem Operasi**: Windows 11
- **Go Version**: `go version go1.24.2 windows/amd64` (or active Go toolchain)
- **Database**: PostgreSQL 16 (Port 5432)
- **Database Test**: `educore_test`
- **Migration Version**: `7` (`migrations/000007_create_attendance_tables.up.sql`)

---

## 2. Rincian Pengujian Unit (Unit Testing)

File pengujian: `internal/modules/attendance/service_test.go`  
Perintah eksekusi:
```bash
go test -v ./internal/modules/attendance/...
```

Hasil pengujian:
- `TestCreateSession_Success`: **PASS** (Dosen pengampu membuat sesi perkuliahan valid).
- `TestCreateSession_UnauthorizedLecturer`: **PASS** (Dosen non-pengampu ditolak dengan `ErrUnauthorizedLecturer`).
- `TestCreateSession_AdminBypass`: **PASS** (Admin dapat membuat sesi tanpa penugasan dosen).
- `TestCreateSession_InvalidTimeRange`: **PASS** (Validasi `end_time <= start_time` menghasilkan error validasi).
- `TestCreateSession_InvalidDateFormat`: **PASS** (Format tanggal selain `YYYY-MM-DD` ditolak).
- `TestCreateSession_DuplicateSessionNumber`: **PASS** (Duplikasi nomor pertemuan mengembalikan conflict).
- `TestGetSessionDetail_StudentAccess`: **PASS** (Mahasiswa terdaftar dapat membaca, mahasiswa non-terdaftar ditolak).
- `TestCloseSession_StatusChecks`: **PASS** (Penutupan sesi berjalan, penutupan ulang sesi completed/cancelled ditolak).
- `TestRecordBatchAttendance_Validation`: **PASS** (Duplikasi ID dalam payload ditolak, status presensi tidak valid ditolak, batch valid sukses).
- `TestUpdateAttendance_Validation`: **PASS** (Validasi status presensi pada koreksi individual).

---

## 3. Rincian Pengujian Integrasi PostgreSQL

File pengujian: `tests/integration/attendance_test.go` & `tests/integration/migration_test.go`  
Perintah eksekusi:
```bash
go test -v -run "TestAttendance_|TestMigration_" ./tests/integration/...
```

Hasil pengujian:
- `TestMigration_Lifecycle`: **PASS**
  - Migrasi skema hingga versi 7 berhasil diterapkan (`dirty: false`).
  - Verifikasi keberadaan tabel `class_sessions` dan `attendances`.
  - Rollback langkah 1 berhasil menghapus tabel `attendances` dan `class_sessions`.
  - Rollback berurutan hingga skema awal dan `MigrateUp` ulang sukses.
- `TestAttendance_SessionLifecycleAndPermissions`: **PASS**
  - Dosen non-pengampu ditolak membuat sesi (403).
  - Dosen pengampu membuat Sesi 1 (201).
  - Admin membuat Sesi 2 (201).
  - Duplikasi nomor pertemuan ditolak constraint database (409).
  - Mahasiswa terdaftar dapat melihat sesi (200).
  - Mahasiswa non-terdaftar ditolak melihat sesi (403).
  - Update sesi dan close sesi berfungsi normal.
- `TestAttendance_RecordingAndReporting`: **PASS**
  - Roster peserta kelas mengambil mahasiswa terdaftar aktif (200).
  - Duplikasi ID mahasiswa dalam payload batch ditolak (409).
  - Mahasiswa non-terdaftar membatalkan seluruh transaksi batch secara atomik (409).
  - Pencatatan presensi batch berhasil disimpan di database (200).
  - Rekap sesi dan ringkasan kehadiran akurat.
  - Koreksi presensi memperbarui status di database (200).
  - Riwayat presensi mahasiswa pribadi terisolasi sesuai principal login.
  - Rekapitulasi kehadiran kelas kumulatif menghitung persentase secara tepat.
- `TestAttendance_ConcurrencySimultaneousBatch`: **PASS**
  - 5 request simultan mengirimkan pencatatan presensi mahasiswa bersamaan.
  - Seluruh request selesai dengan status 200 OK tanpa error deadlock atau inkonsistensi.
  - Tepat 2 catatan presensi tersimpan di database tanpa duplikasi record.

---

## 4. Pengujian Regresi & Seluruh Workspace

Perintah eksekusi:
```bash
go test ./...
go test -race ./internal/modules/attendance/...
go vet ./...
gofmt -w .
```

Hasil:
- Seluruh unit test dan integration test pada seluruh modul (Fase 01–10) lulus (**PASS**).
- Tidak ada data race yang terdeteksi oleh `-race`.
- `go vet` bersih tanpa peringatan (**PASS**).
- `gofmt` terformat rapi (**PASS**).

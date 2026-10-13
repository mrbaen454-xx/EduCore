# Laporan Pengujian dan QA: Penawaran Kelas dan Jadwal Perkuliahan (Fase 08)

## 1. Lingkup Pengujian

Pengujian Fase 08 memvalidasi modul Periode Akademik, Penawaran Mata Kuliah, Kelas Perkuliahan, Penugasan Dosen, serta Jadwal Perkuliahan dan Deteksi Bentrok melalui pengujian unit dan pengujian integrasi berbasis database PostgreSQL nyata.

---

## 2. Hasil Verifikasi Otomatis

| Pengujian / Perintah | Status | Bukti / Keterangan |
|---|---|---|
| Formatter (`gofmt -w .`) | **PASS** | Semua file terformat rapi sesuai idiom standar Go |
| Static Analysis (`go vet ./...`) | **PASS** | Lolos tanpa issue atau peringatan compiler |
| Unit Test Modules (`academic_periods`, `course_offerings`, `classes`) | **PASS** | Seluruh skenario logika bisnis (kapasitas, kode unik, self-overlap, lecturer collision, room collision) lulus |
| Integration Test Lifecycle (`TestClassesAndSchedules_IntegrationLifecycle`) | **PASS** | 18 skenario integrasi end-to-end terverifikasi di PostgreSQL |
| Database Migration Lifecycle (`TestMigration_Lifecycle`) | **PASS** | Migrasi versi 5 diaplikasikan dan diuji rollback 5 tahap |
| Full Test Suite (`go test ./...`) | **PASS** | Seluruh test suite dari Fase 01 hingga Fase 08 lulus |
| Concurrency / Race Test (`go test -race ./...`) | **PASS** | Tidak ditemukan data race condition |
| Build Check (`go build ./...`) | **PASS** | Binary terkompilasi bersih |

---

## 3. Matriks Skenario Integration Test (`tests/integration/classes_and_schedules_test.go`)

| No | Skenario Pengujian | Hasil | Keterangan |
|---|---|---|---|
| 1 | `Create Academic Period - Success` | **PASS** | Periode akademik berhasil dibuat |
| 2 | `Reject Invalid Dates on Academic Period` | **PASS** | Penolakan start_date > end_date (400 Bad Request) |
| 3 | `Create Course Offering - Success` | **PASS** | Penawaran mata kuliah berstatus PLANNED dibuat |
| 4 | `Reject Duplicate Course Offering` | **PASS** | Penolakan penawaran ganda pada periode & prodi sama (409 Conflict) |
| 5 | `Create Class A - Success` | **PASS** | Kelas dengan kapasitas 40 berhasil dibuka |
| 6 | `Reject Class with Zero Capacity` | **PASS** | Penolakan kapasitas 0 (400 Bad Request) |
| 7 | `Reject Duplicate Class Code` | **PASS** | Penolakan kode kelas ganda di penawaran yang sama (409 Conflict) |
| 8 | `Assign Lecturer to Class A - Success` | **PASS** | Dosen berhasil ditugaskan sebagai COORDINATOR |
| 9 | `Reject Duplicate Lecturer Assignment` | **PASS** | Penolakan penugasan ganda dosen pada kelas sama (409 Conflict) |
| 10 | `Create Schedule for Class A - Success` | **PASS** | Jadwal Senin 08:00 - 10:00 di LAB-1 berhasil dibuat |
| 11 | `Reject Schedule with Start Time >= End Time` | **PASS** | Penolakan jam terbalik (400 Bad Request) |
| 12 | `Reject Schedule Overlap on Same Class` | **PASS** | Penolakan bentrok jadwal internal kelas (409 Conflict) |
| 13 | `Create Class B - Success` | **PASS** | Kelas B berhasil dibuat dan ditugaskan dosen yang sama |
| 14 | `Reject Lecturer Schedule Conflict Across Classes` | **PASS** | Penolakan dosen mengajar di 2 kelas bersamaan (409 Conflict) |
| 15 | `Reject Room Schedule Conflict in Same Period` | **PASS** | Penolakan ruangan LAB-1 dipakai 2 kelas bersamaan (409 Conflict) |
| 16 | `Allow Consecutive Schedule on Same Room` | **PASS** | Jadwal berurutan (10:00 - 12:00) pada ruangan sama diizinkan ($[s, e)$) |
| 17 | `Role Protection - Student Denied from Mutation` | **PASS** | Mahasiswa ditolak melakukan mutasi kelas (403 Forbidden) |
| 18 | `Anonymous Protection - No Token` | **PASS** | Request tanpa token ditolak (401 Unauthorized) |

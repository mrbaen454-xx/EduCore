# Laporan Pengujian: KRS dan Registrasi Mahasiswa (Fase 09)

## 1. Lingkup Pengujian

Pengujian mencakup:
1. **Unit Test Service**: Validasi status keaktifan mahasiswa, idempoten pembuatan KRS, penolakan akses lintas mahasiswa (*ownership check*), pencegahan duplikasi pendaftaran, batas maksimum 24 SKS, kapasitas kelas penuh, dan pembatalan registrasi.
2. **Integration Test PostgreSQL**: Skenario end-to-end dengan database PostgreSQL nyata, mencakup lifecycle lengkap mahasiswa melihat penawaran kelas, membuat KRS, mendaftar kelas, mendeteksi bentrok jadwal $[start, end)$, membatalkan kelas (status `DROPPED` & pelepasan kuota), serta endpoint manajemen admin.
3. **Concurrency Test**: Uji konkurensi perebutan 1 slot kapasitas terakhir oleh 2 mahasiswa secara paralel untuk membuktikan eliminasi race condition via row locking `FOR UPDATE`.
4. **Database Migration Lifecycle**: Uji siklus migrasi version 6 (MigrateUp, MigrateDown 6 langkah, dan MigrateUp kembali).
5. **OpenAPI v1.5.0 Specification Test**: Verifikasi kepatuhan skema OpenAPI.

---

## 2. Hasil Eksekusi Pengujian

| Komponen Pengujian | Perintah | Status | Keterangan |
| :--- | :--- | :---: | :--- |
| **Pemeriksaan Format Kode** | `gofmt -s -l .` | **PASS** | Format kode Go bersih |
| **Analisis Statis Kode** | `go vet ./...` | **PASS** | Bebas dari error statis |
| **Unit Test Suite** | `go test -v ./internal/modules/study_plans/...` | **PASS** | 100% unit tests berhasil |
| **Race Detector** | `go test -race ./internal/modules/study_plans/...` | **PASS** | Bebas dari *data race* |
| **Kompilasi Binary** | `go build ./...` | **PASS** | Kompilasi bersih |
| **Database Migrations** | `go test -v -run TestMigration_Lifecycle tests/integration/migration_test.go` | **PASS** | Siklus migrasi up/down versi 1-6 lulus |
| **Integration Test Suite** | `go test -v tests/integration/study_plans_and_enrollments_test.go tests/integration/database_test.go` | **PASS** | 12 skenario end-to-end lulus |
| **Test Konkurensi Kapasitas** | `TestStudyPlans_ConcurrencyRaceConditions` | **PASS** | 2 request paralel perebutan 1 slot: tepat 1 sukses (201) dan 1 konflik (409) |
| **OpenAPI Specification** | `go test -v -run TestOpenAPISpecificationValid tests/integration/api_test.go tests/integration/database_test.go` | **PASS** | OpenAPI v1.5.0 valid |
| **Regresi Keseluruhan Proyek** | `go test ./...` | **PASS** | Seluruh modul Fase 01–09 lulus tanpa regresi |

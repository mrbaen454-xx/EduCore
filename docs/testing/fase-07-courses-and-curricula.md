# Laporan Pengujian dan QA: Modul Mata Kuliah dan Kurikulum (Fase 07)

## 1. Lingkup Pengujian

Fase 07 menguji seluruh fungsionalitas modul Mata Kuliah, Kurikulum, Relasi Kurikulum-Mata Kuliah, serta Prasyarat Mata Kuliah melalui pengujian unit dan pengujian integrasi berbasis PostgreSQL nyata.

---

## 2. Hasil Eksekusi Pengujian Otomatis

### 2.1. Ringkasan Eksekusi

| Pemeriksaan | Perintah / Alat | Status | Keterangan / Bukti |
|---|---|---|---|
| Code Formatter | `gofmt -w .` | **PASS** | Semua file Go diformat rapi sesuai standar resmi |
| Static Analysis | `go vet ./...` | **PASS** | Tidak ditemukan issue atau peringatan dari compiler |
| Unit Tests | `go test -v ./internal/modules/courses/... ./internal/modules/curricula/...` | **PASS** | Semua 7 skenario pengujian unit lulus (0.00s execution) |
| Integration Tests | `go test -v ./tests/integration/...` | **PASS** | 16 skenario integrasi end-to-end lulus pada PostgreSQL `educore_test` |
| Full Project Tests | `go test ./...` | **PASS** | Seluruh test suite paket project berhasil |
| Race Condition Detector | `go test -race ./internal/modules/courses/... ./internal/modules/curricula/...` | **PASS** | Tidak ditemukan data race pada eksekusi konkurensi |
| Application Build | `go build ./...` | **PASS** | Binary terkompilasi bersih tanpa error atau circular dependencies |
| Database Migration | `go run ./cmd/migrate up` & `TestMigration_Lifecycle` | **PASS** | Migrasi versi 4 diaplikasikan dan diuji siklus up/down |

---

## 3. Matriks Skenario Integration Test (`tests/integration/courses_and_curricula_test.go`)

| No | Nama Skenario Test | Status | Verifikasi |
|---|---|---|---|
| 1 | `Create_Course_1_-_Success` | **PASS** | Pembuatan mata kuliah dengan kode uppercase dan SKS valid |
| 2 | `Create_Course_-_Duplicate_Code` | **PASS** | Penolakan kode mata kuliah ganda dengan error 409 Conflict |
| 3 | `Create_Course_-_Invalid_Credits` | **PASS** | Penolakan SKS 0 atau > 8 dengan error 400 Bad Request |
| 4 | `Create_Course_2_-_Success` | **PASS** | Pembuatan mata kuliah kedua untuk pengujian relasi |
| 5 | `Add_Prerequisite_-_Success` | **PASS** | Penambahan relasi prasyarat antarmata kuliah |
| 6 | `Reject_Self_Prerequisite` | **PASS** | Penolakan mata kuliah menjadi prasyarat dirinya sendiri (400) |
| 7 | `Reject_Circular_Prerequisite` | **PASS** | Deteksi dan penolakan siklus prasyarat A -> B -> A (409) |
| 8 | `Create_Curriculum_-_Success` | **PASS** | Pembuatan kurikulum untuk prodi aktif dengan status DRAFT |
| 9 | `Create_Curriculum_-_Unknown_Study_Program` | **PASS** | Penolakan kurikulum dengan program studi tidak ada (404) |
| 10 | `Add_Course_1_to_Curriculum_-_Success` | **PASS** | Penambahan mata kuliah wajib ke kurikulum semester 1 |
| 11 | `Add_Course_2_to_Curriculum_-_Success` | **PASS** | Penambahan mata kuliah pilihan ke kurikulum semester 2 |
| 12 | `Reject_Duplicate_Course_in_Curriculum` | **PASS** | Penolakan mata kuliah yang sama dua kali dalam 1 kurikulum (409) |
| 13 | `Verify_Curriculum_Credits_Aggregation` | **PASS** | Verifikasi agregasi dinamis total SKS (3 wajib + 3 pilihan = 6 total) |
| 14 | `Update_Curriculum_Status_to_ACTIVE` | **PASS** | Transisi status kurikulum dari DRAFT ke ACTIVE |
| 15 | `Role_Protection_-_Student_Denied_Mutation` | **PASS** | Otorisasi RBAC: role mahasiswa ditolak mutasi data (403 Forbidden) |
| 16 | `Anonymous_Protection_-_No_Token` | **PASS** | Endpoint tanpa JWT ditolak (401 Unauthorized) |

---

## 4. Cara Menjalankan Uji Lingkungan Lokal

```bash
# 1. Jalankan migrasi database test
$env:DB_NAME="educore_test"
go run ./cmd/migrate up

# 2. Jalankan test integrasi
go test -v ./tests/integration/... -run TestCoursesAndCurricula_IntegrationLifecycle

# 3. Jalankan pengujian race condition
go test -race ./internal/modules/courses/... ./internal/modules/curricula/...
```

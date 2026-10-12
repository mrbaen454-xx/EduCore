# Laporan & Panduan Pengujian: Data Master Akademik (Fase 06)

Dokumen ini mendokumentasikan strategi pengujian, skenario uji integrasi, dan hasil verifikasi aktual untuk modul Data Master Akademik **EduCore**.

---

## 1. Strategi Pengujian Berlapis

1. **Unit Testing**:
   - `internal/modules/faculties/service_test.go`: Pengujian aturan pembuatan fakultas, kode duplikat, pangkas spasi, dan status toggle.
   - `internal/modules/study_programs/service_test.go`: Pengujian validasi jenjang prodi, penolakan fakultas induk tidak aktif, dan kode prodi duplikat.
   - `internal/modules/lecturers/service_test.go`: Pengujian format gelar, validasi NIDN, email unik, dan status dosen.
   - `internal/modules/students/service_test.go`: Pengujian format NIM, tahun angkatan, format tanggal lahir, dan status mahasiswa.
2. **Integration Testing**:
   - `tests/integration/academic_master_test.go`: Pengujian *end-to-end* di atas PostgreSQL nyata melalui Gin router dan HTTP pipeline.

---

## 2. Matriks Skenario Uji Integrasi (`academic_master_test.go`)

| No | Skenario Uji | Aktor / Token | Ekspektasi HTTP | Verifikasi Assert |
|---|---|---|---|---|
| 1 | Create Faculty - Success | `admin` | `201 Created` | Data ID UUID terbit, code `FT` tersimpan UPPERCASE |
| 2 | Create Faculty - Duplicate Code | `admin` | `409 Conflict` | Kode `ft` ditolak karena konflik keunikan |
| 3 | Get Faculty Detail | `mahasiswa` | `200 OK` | Endpoint baca dapat diakses oleh authenticated non-admin |
| 4 | Create Study Program - Success | `admin` | `201 Created` | Terhubung ke `faculty_id` yang valid |
| 5 | Create Study Program - Unknown Faculty | `admin` | `404 Not Found` | Ditolak karena fakultas induk fiktif |
| 6 | Create Lecturer - Success | `admin` | `201 Created` | Dosen berhasil terdaftar di bawah prodi |
| 7 | Create Lecturer - Duplicate NIDN | `admin` | `409 Conflict` | NIDN duplikat ditolak oleh constraint DB |
| 8 | Create Student - Success | `admin` | `201 Created` | Mahasiswa berhasil terdaftar dengan data biodata lengkap |
| 9 | Create Student - Duplicate NIM | `admin` | `409 Conflict` | NIM duplikat ditolak oleh constraint DB |
| 10 | Role Protection - Mutation Denied | `mahasiswa` | `403 Forbidden` | Role mahasiswa dilarang menambah fakultas |
| 11 | Anonymous Protection | Tanpa Token | `401 Unauthorized` | Request tanpa bearer token ditolak middleware |
| 12 | Filtering & Pagination | `admin` | `200 OK` | Filter nama `Ananda` dan angkatan `2026` menghasilkan 1 record |
| 13 | Update Student Status | `admin` | `200 OK` | Status mahasiswa berhasil diubah menjadi `SUSPENDED` |

---

## 3. Instruksi Menjalankan Pengujian

Jalankan rangkaian perintah berikut dari root repositori:

```bash
# 1. Format kode
gofmt -w .

# 2. Analisis statis Go vet
go vet ./...

# 3. Eksekusi seluruh unit & integration test
go test -v ./...

# 4. Pengujian race condition
go test -race ./...

# 5. Build binary
go build -o bin/educore.exe cmd/api/main.go
```

---

## 4. Hasil Verifikasi Aktual

| Pengujian | Status | Bukti / Catatan Eksekusi |
|---|---|---|
| `gofmt -w .` | **PASS** | Seluruh file Go terformat rapi sesuai idiom standar Go |
| `go vet ./...` | **PASS** | Exit code 0, tidak ditemukan kesalahan variabel / import |
| `go test ./...` | **PASS** | Seluruh unit test dan integration test pada DB PostgreSQL lulus |
| `go test -race ./...` | **PASS** | Exit code 0, tidak terdeteksi race condition pada konkurensi |
| `go build ./...` | **PASS** | Kompilasi seluruh package dan binary sukses tanpa error |
| Database Migration Up | **PASS** | Migrasi `000003_create_academic_master_tables.up.sql` berhasil diterapkan |
| Database Migration Rollback | **PASS** | Terverifikasi pada `TestMigration_Lifecycle` (up -> step down -> restore) |

# EduCore - Academic Management System (SIAKAD)

EduCore adalah backend Sistem Informasi Akademik berbasis Go dengan arsitektur **Modular Monolith**, REST API, dan PostgreSQL.

---

## 1. Prasyarat

- Go 1.22 atau lebih baru (teruji pada Go 1.24)
- PostgreSQL 16+ (atau Docker / Docker Compose)
- Git
- Make (Opsional, untuk menjalankan target di Makefile)

---

## 2. Menyiapkan Database dan Menjalankan Aplikasi

### 2.1 Menyiapkan PostgreSQL

#### Opsi A: Menggunakan Docker Compose
```bash
docker compose up -d postgres
```

#### Opsi B: Menggunakan PostgreSQL Lokal
Pastikan service PostgreSQL aktif dan buat database berikut:
```sql
CREATE DATABASE educore;
CREATE DATABASE educore_test;
```

### 2.2 Menyiapkan Konfigurasi Lingkungan
Salin konfigurasi environment dari template:
- Windows PowerShell:
  ```powershell
  copy .env.example .env
  ```
- Linux / macOS:
  ```bash
  cp .env.example .env
  ```
Sesuaikan nilai `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, dan `DB_NAME` pada file `.env`.

### 2.3 Menjalankan Migrasi Skema
Sebelum menjalankan aplikasi, terapkan migrasi skema baseline (`roles`, `users`, `user_roles`):
```bash
# Menggunakan Go CLI:
go run ./cmd/migrate up

# Atau menggunakan Makefile:
make db-up
```

Untuk memeriksa versi migrasi saat ini:
```bash
go run ./cmd/migrate version
```

### 2.4 Menjalankan Server Backend
Jalankan aplikasi:
- Menggunakan Make:
  ```bash
  make run
  ```
- Menggunakan Go CLI secara langsung:
  ```bash
  go run ./cmd/api
  ```
Server akan mendengarkan request HTTP di port `8080` (sesuai konfigurasi `APP_PORT`).

---

## 3. Konfigurasi Environment (`.env.example`)

| Variabel | Tipe | Default | Deskripsi |
|---|---|---|---|
| `APP_ENV` | string | `development` | Lingkungan aplikasi (`development`, `production`, `testing`) |
| `APP_PORT` | int | `8080` | Port listen HTTP server (1 - 65535) |
| `APP_TIMEZONE` | string | `Asia/Jakarta` | Zona waktu aplikasi |
| `LOG_LEVEL` | string | `info` | Tingkat logging (`debug`, `info`, `warn`, `error`) |
| `HTTP_READ_TIMEOUT` | duration | `15s` | Timeout pembacaan request HTTP |
| `HTTP_WRITE_TIMEOUT` | duration | `15s` | Timeout penulisan response HTTP |
| `HTTP_IDLE_TIMEOUT` | duration | `60s` | Timeout koneksi idle HTTP keep-alive |
| `HTTP_SHUTDOWN_TIMEOUT`| duration | `10s` | Batas waktu graceful shutdown server |
| `DB_HOST` | string | `localhost` | Host database PostgreSQL |
| `DB_PORT` | int | `5432` | Port database PostgreSQL |
| `DB_NAME` | string | `educore` | Nama basis data aplikasi |
| `DB_USER` | string | `educore_app` | Username akun PostgreSQL |
| `DB_PASSWORD` | string | `""` | Kata sandi database (wajib diisi sesuai server) |
| `DB_SSLMODE` | string | `disable` | Mode SSL PostgreSQL (`disable`, `require`, `verify-full`) |
| `DB_MAX_CONNS` | int | `10` | Batas maksimum koneksi pool |
| `DB_MIN_CONNS` | int | `2` | Jumlah koneksi idle minimum pool |
| `DB_MAX_CONN_LIFETIME` | duration | `30m` | Waktu hidup maksimum koneksi pool |
| `DB_MAX_CONN_IDLE_TIME` | duration | `5m` | Waktu idle maksimum koneksi pool |
| `DB_CONNECT_TIMEOUT` | duration | `5s` | Timeout koneksi awal dan ping startup |
| `DB_HEALTH_TIMEOUT` | duration | `2s` | Timeout ping database pada readiness check |

---

## 4. Endpoint Tersedia

### 4.1 Probe Infrastruktur
- `GET /health/live`: Mengecek apakah proses server berjalan (*liveness probe*).
- `GET /health/ready`: Mengecek kesiapan server dan konektivitas aktif ke pool PostgreSQL (*readiness probe*). Mengembalikan HTTP 200 jika database `up` atau 503 jika `down`.

### 4.2 Endpoint Diagnostik API
- `GET /api/v1/health`: Mengecek status operasional modul REST API v1.

---

## 5. Contoh Pengujian Endpoint (`curl`)

### A. Liveness Check
```bash
curl -i http://localhost:8080/health/live
```

### B. Readiness Check (Terhubung ke PostgreSQL)
```bash
curl -i http://localhost:8080/health/ready
```
*Respons:*
```json
{
  "status": "ready",
  "timestamp": "2026-10-11T23:00:00Z",
  "checks": {
    "database": "up"
  }
}
```

### C. API v1 Health Check (Dengan Custom Request ID)
```bash
curl -i -H "X-Request-ID: test-custom-req-001" http://localhost:8080/api/v1/health
```

---

## 6. Pengujian dan Kualitas Kode

Jalankan rangkaian pengujian otomatis (membutuhkan database `educore_test` yang aktif):

```bash
# Format kode sesuai standar Go
gofmt -w .

# Analisis statis
go vet ./...

# Menjalankan seluruh unit test dan integration test
go test -v ./...

# Pengujian dengan deteksi race condition
go test -race ./...

# Build binary aplikasi
go build -o bin/educore.exe cmd/api/main.go
```

Atau melalui shortcut Makefile:
```bash
make fmt
make vet
make test
make test-race
make build
make db-up
make db-down
make db-version
```

---

## 7. Dokumentasi Lengkap Proyek

- **Database**:
  - [Arsitektur Database](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/database-architecture.md)
  - [Konvensi Skema & Data Types](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/schema-conventions.md)
  - [Panduan Migrasi](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/migrations.md)
  - [Panduan Operasional & Backup](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/operations.md)
- **REST API**:
  - [Spesifikasi OpenAPI 3.0](file:///d:/share/SARONI/PROJECTS/EduCore/api/openapi.yaml)
  - [Konvensi REST API](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/rest-api-conventions.md)
  - [Kamus Kode Error](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/error-codes.md)
- **Arsitektur & Bisnis**:
  - [Panduan Modul Bisnis](file:///d:/share/SARONI/PROJECTS/EduCore/internal/modules/README.md)
  - [Ikhtisar Bisnis SIAKAD](file:///d:/share/SARONI/PROJECTS/EduCore/docs/requirements/business-overview.md)

---

## 8. Batasan Fitur pada Fase 04

Fase 04 berfokus pada **Fondasi Database PostgreSQL**. Fitur-fitur berikut dijadwalkan pada fase berikutnya:
- **Fase 05**: Autentikasi Pengguna & Otorisasi RBAC (Registrasi, Login, Password Hashing bcrypt, JWT, Refresh Token, Middleware Auth).
- **Fase 06+**: Modul Bisnis Akademik (Fakultas, Prodi, Mahasiswa, Dosen, Kurikulum, KRS/IRS, Jadwal, Presensi, Penilaian, IP/IPK, Transkrip).


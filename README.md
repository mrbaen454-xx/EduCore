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
Sesuaikan nilai `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, dan `JWT_SECRET` pada file `.env`.

### 2.3 Menjalankan Migrasi Skema
Sebelum menjalankan aplikasi, terapkan migrasi skema database:
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
| `DB_USER` | string | `postgres` | Username akun PostgreSQL |
| `DB_PASSWORD` | string | `""` | Kata sandi database |
| `DB_SSLMODE` | string | `disable` | Mode SSL PostgreSQL (`disable`, `require`, `verify-full`) |
| `DB_MAX_CONNS` | int | `10` | Batas maksimum koneksi pool |
| `DB_MIN_CONNS` | int | `2` | Jumlah koneksi idle minimum pool |
| `DB_MAX_CONN_LIFETIME` | duration | `30m` | Waktu hidup maksimum koneksi pool |
| `DB_MAX_CONN_IDLE_TIME` | duration | `5m` | Waktu idle maksimum koneksi pool |
| `DB_CONNECT_TIMEOUT` | duration | `5s` | Timeout koneksi awal dan ping startup |
| `DB_HEALTH_TIMEOUT` | duration | `2s` | Timeout ping database pada readiness check |
| `JWT_SECRET` | string | *wajib* | Secret key signing HS256 (min 32 karakter) |
| `JWT_ISSUER` | string | `educore-auth` | Penerbit klaim `iss` pada JWT |
| `JWT_AUDIENCE` | string | `educore-api` | Target penerima klaim `aud` pada JWT |
| `JWT_ACCESS_TOKEN_TTL`| duration | `15m` | Masa berlaku JWT Access Token |
| `JWT_REFRESH_TOKEN_TTL`| duration | `168h` | Masa berlaku Refresh Token (7 hari) |
| `ARGON2_MEMORY` | uint32 | `65536` | Alokasi memori hashing Argon2id (KiB) |
| `ARGON2_ITERATIONS` | uint32 | `3` | Jumlah iterasi/time cost Argon2id |
| `ARGON2_PARALLELISM` | uint8 | `4` | Tingkat paralelisme thread Argon2id |
| `ARGON2_SALT_LENGTH` | uint32 | `16` | Panjang random salt (bytes) |
| `ARGON2_KEY_LENGTH` | uint32 | `32` | Panjang output hash key (bytes) |

---

## 4. Endpoint Tersedia

### 4.1 Probe Infrastruktur & Diagnostik
- `GET /health/live`: Mengecek apakah proses server berjalan (*liveness probe*).
- `GET /health/ready`: Mengecek kesiapan server dan konektivitas aktif ke pool PostgreSQL (*readiness probe*).
- `GET /api/v1/health`: Mengecek status operasional modul REST API v1.

### 4.2 Autentikasi & Pengguna (Fase 05)
- `POST /api/v1/auth/login`: Login pengguna dengan identifier & password; menghasilkan Access Token & Refresh Token.
- `POST /api/v1/auth/refresh`: Memperbarui Access Token dengan mekanisme rotasi Refresh Token otomatis.
- `POST /api/v1/auth/logout`: Mencabut refresh token aktif untuk mengakhiri sesi pengguna.
- `GET /api/v1/users/me`: Mengambil profil dan roles pengguna yang sedang login (membutuhkan header `Authorization: Bearer <token>`).
- `GET /api/v1/test/admin-only`: Endpoint pengujian otorisasi RBAC (hanya dapat diakses oleh role `ADMIN`).

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

### C. Login Pengguna
```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"admin@educore.ac.id","password":"YourSecurePassword"}'
```

### D. Mengakses Profil Pengguna Terautentikasi
```bash
curl -i http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

### E. Merotasi Refresh Token
```bash
curl -i -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<REFRESH_TOKEN>"}'
```

### F. Logout / Pencabutan Sesi
```bash
curl -i -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<REFRESH_TOKEN>"}'
```

---

## 6. Pengujian dan Kualitas Kode

Jalankan seluruh suite pengujian otomatis (membutuhkan database `educore_test` yang aktif):

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

- **Keamanan & Autentikasi (Fase 05)**:
  - [Dokumentasi Autentikasi](file:///d:/share/SARONI/PROJECTS/EduCore/docs/security/authentication.md)
  - [Dokumentasi Otorisasi & RBAC](file:///d:/share/SARONI/PROJECTS/EduCore/docs/security/authorization.md)
  - [Siklus Hidup Token & Sesi](file:///d:/share/SARONI/PROJECTS/EduCore/docs/security/token-lifecycle.md)
- **Database (Fase 04)**:
  - [Arsitektur Database](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/database-architecture.md)
  - [Konvensi Skema & Data Types](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/schema-conventions.md)
  - [Panduan Migrasi](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/migrations.md)
  - [Panduan Operasional & Backup](file:///d:/share/SARONI/PROJECTS/EduCore/docs/database/operations.md)
- **REST API (Fase 03)**:
  - [Spesifikasi OpenAPI 3.0](file:///d:/share/SARONI/PROJECTS/EduCore/api/openapi.yaml)
  - [Konvensi REST API](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/rest-api-conventions.md)
  - [Kamus Kode Error](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/error-codes.md)
- **Arsitektur & Bisnis (Fase 01 & 02)**:
  - [Panduan Modul Bisnis](file:///d:/share/SARONI/PROJECTS/EduCore/internal/modules/README.md)
  - [Ikhtisar Bisnis SIAKAD](file:///d:/share/SARONI/PROJECTS/EduCore/docs/requirements/business-overview.md)

---

## 8. Status Roadmap & Batasan Fitur

- **Fase 01**: Kebutuhan & Desain Arsitektur SIAKAD (Selesai)
- **Fase 02**: Kerangka Fondasi Backend & Graceful Shutdown (Selesai)
- **Fase 03**: Fondasi REST API, Middleware, Error Handling, Routing (Selesai)
- **Fase 04**: Database PostgreSQL, Connection Pool, Transaksi, Migrasi (Selesai)
- **Fase 05**: Autentikasi & Otorisasi RBAC, Argon2id, JWT, Refresh Token Rotation (Selesai)
- **Fase 06+**: Modul Bisnis Akademik (Fakultas, Prodi, Mahasiswa, Dosen, Kurikulum, KRS/IRS, Jadwal, Presensi, Penilaian, IP/IPK, Transkrip) *(Belum dimulai)*

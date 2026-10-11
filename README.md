# EduCore - Academic Management System (SIAKAD)

EduCore adalah backend Sistem Informasi Akademik berbasis Go dengan arsitektur **Modular Monolith** dan REST API.

---

## 1. Prasyarat

- Go 1.27 atau lebih baru
- Git
- Make (Opsional, untuk menjalankan target di Makefile)

---

## 2. Menjalankan Aplikasi Lokal

1. Salin konfigurasi environment dari template:
   - Windows PowerShell / Command Prompt:
     ```powershell
     copy .env.example .env
     ```
   - Linux / macOS:
     ```bash
     cp .env.example .env
     ```
2. Sesuaikan konfigurasi pada file `.env` jika diperlukan.
3. Jalankan aplikasi:
   - Menggunakan Make:
     ```bash
     make run
     ```
   - Menggunakan Go CLI secara langsung:
     ```bash
     go run cmd/api/main.go
     ```
4. Server akan mendengarkan request HTTP di port `8080` (sesuai konfigurasi `APP_PORT`).

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

---

## 4. Endpoint Tersedia (Fase 03: Fondasi REST API)

### 4.1 Probe Infrastruktur
- `GET /health/live`: Mengecek apakah proses server berjalan (*liveness probe*).
- `GET /health/ready`: Mengecek kesiapan server melayani traffic HTTP (*readiness probe*).

### 4.2 Endpoint Diagnostik API
- `GET /api/v1/health`: Mengecek status operasional modul REST API v1.

---

## 5. Contoh Pengujian Endpoint (`curl`)

### A. Liveness Check
```bash
curl -i http://localhost:8080/health/live
```

### B. Readiness Check
```bash
curl -i http://localhost:8080/health/ready
```

### C. API v1 Health Check (Dengan Custom Request ID)
```bash
curl -i -H "X-Request-ID: test-custom-req-001" http://localhost:8080/api/v1/health
```

### D. Route Tidak Ditemukan (404 JSON)
```bash
curl -i http://localhost:8080/api/v1/route-yang-tidak-ada
```

### E. Metode HTTP Tidak Didukung (405 JSON)
```bash
curl -i -X DELETE http://localhost:8080/api/v1/health
```

---

## 6. Pengujian dan Kualitas Kode

Jalankan rangkaian pengujian otomatis:

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
```

---

## 7. Dokumentasi API & Arsitektur

- **Spesifikasi OpenAPI 3.0**: [`api/openapi.yaml`](file:///d:/share/SARONI/PROJECTS/EduCore/api/openapi.yaml)
- **Konvensi REST API**: [`docs/api/rest-api-conventions.md`](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/rest-api-conventions.md)
- **Kamus Kode Error**: [`docs/api/error-codes.md`](file:///d:/share/SARONI/PROJECTS/EduCore/docs/api/error-codes.md)
- **Panduan Modul Bisnis**: [`internal/modules/README.md`](file:///d:/share/SARONI/PROJECTS/EduCore/internal/modules/README.md)
- **Ikhtisar Bisnis SIAKAD**: [`docs/requirements/business-overview.md`](file:///d:/share/SARONI/PROJECTS/EduCore/docs/requirements/business-overview.md)

---

## 8. Batasan Fitur pada Fase 03

Fase 03 berfokus murni pada **Fondasi REST API**. Fitur-fitur berikut belum diimplementasikan pada fase ini dan dijadwalkan pada fase berikutnya:
- Autentikasi pengguna & otorisasi RBAC (JWT, refresh token, password hashing).
- Koneksi dan migrasi database PostgreSQL (pgxpool).
- Modul bisnis akademik (Mahasiswa, Dosen, Mata Kuliah, KRS, Presensi, Penilaian, IP/IPK).
- Antarmuka pengguna (Frontend web atau mobile).

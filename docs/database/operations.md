# Panduan Operasional Database EduCore (Fase 04)

Dokumen ini menyediakan panduan operasional harian, konfigurasi lingkungan, prosedur backup/pemulihan, dan panduan pemecahan masalah (*troubleshooting*) database PostgreSQL untuk **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Persiapan Lingkungan PostgreSQL

Pengembangan EduCore dapat menggunakan salah satu dari dua pendekatan berikut:

### Opsi A: PostgreSQL Lokal / Native Service (Windows / Linux / macOS)
1. Pastikan PostgreSQL 16+ telah terinstal dan servicenya berjalan pada port `5432`.
2. Buat database aplikasi dan database pengujian:
   ```sql
   CREATE DATABASE educore;
   CREATE DATABASE educore_test;
   ```
3. Buat role pengguna aplikasi:
   ```sql
   CREATE USER educore_app WITH PASSWORD 'your_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE educore TO educore_app;
   GRANT ALL PRIVILEGES ON DATABASE educore_test TO educore_app;
   ```

### Opsi B: Docker Compose
Jalankan container PostgreSQL terisolasi menggunakan berkas `compose.yaml`:
```bash
# Menjalankan database di latar belakang
docker compose up -d postgres

# Memeriksa status kesehatan container
docker compose ps

# Melihat log container
docker compose logs -f postgres

# Menghentikan container (data volume tetap aman)
docker compose down
```

> [!CAUTION]
> Jangan menjalankan `docker compose down -v` kecuali Anda memang berniat menghapus seluruh data pengembangan lokal pada volume `postgres_data`.

---

## 2. Konfigurasi Environment Variables

Aplikasi memuat konfigurasi dari file `.env` di root direktori proyek:

| Variabel | Tipe / Default | Wajib? | Keterangan |
|---|---|---|---|
| `DB_HOST` | String (`localhost`) | Ya | Host server PostgreSQL |
| `DB_PORT` | Int (`5432`) | Ya | Port PostgreSQL (1 - 65535) |
| `DB_NAME` | String (`educore`) | Ya | Nama basis data |
| `DB_USER` | String (`educore_app`) | Ya | Nama pengguna database |
| `DB_PASSWORD` | String | Tidak* | Kata sandi database (*wajib di staging/prod) |
| `DB_SSLMODE` | String (`disable`) | Ya | Mode SSL: `disable`, `require`, `verify-ca`, `verify-full` |
| `DB_MAX_CONNS` | Int (`10`) | Ya | Jumlah koneksi maksimum pool (min 1) |
| `DB_MIN_CONNS` | Int (`2`) | Ya | Jumlah koneksi idle minimum pool (>= 0, <= MaxConns) |
| `DB_MAX_CONN_LIFETIME` | Durasi (`30m`) | Ya | Masa hidup koneksi sebelum di-recycle |
| `DB_MAX_CONN_IDLE_TIME` | Durasi (`5m`) | Ya | Waktu idle maksimum koneksi |
| `DB_CONNECT_TIMEOUT` | Durasi (`5s`) | Ya | Batas waktu timeout saat initial ping/connect |
| `DB_HEALTH_TIMEOUT` | Durasi (`2s`) | Ya | Batas waktu ping saat endpoint `/health/ready` dipanggil |

### Perbedaan Antar Lingkungan:
- **Local Development**: `DB_SSLMODE=disable`, `DB_MAX_CONNS=10`.
- **Testing**: Menggunakan `DB_NAME=educore_test` yang terisolasi.
- **Staging / Production**: Wajib `DB_SSLMODE=require` atau `verify-full`, `DB_MAX_CONNS` disesuaikan dengan kapasitas RAM PostgreSQL dan jumlah replica aplikasi.

---

## 3. Menjalankan Aplikasi dan Migrasi

1. Salin template konfigurasi:
   ```bash
   cp .env.example .env
   ```
2. Sesuaikan kredensial di `.env`.
3. Jalankan migrasi skema awal:
   ```bash
   go run ./cmd/migrate up
   ```
4. Jalankan server backend:
   ```bash
   go run ./cmd/api
   ```
5. Periksa kesiapan aplikasi (*Readiness Health Check*):
   ```bash
   curl -i http://localhost:8080/health/ready
   ```
   *Respons yang diharapkan (HTTP 200 OK):*
   ```json
   {
     "status": "ready",
     "timestamp": "2026-10-11T23:00:00Z",
     "checks": {
       "database": "up"
     }
   }
   ```

---

## 4. Prosedur Backup dan Pemulihan (Backup & Recovery)

### A. Prosedur Pencadangan (Backup)
Gunakan utilitas resmi PostgreSQL `pg_dump` dengan format *custom archive* (`-Fc`) untuk fleksibilitas kompresi dan seleksi objek:

```bash
# Backup database pengembangan / produksi:
pg_dump -h localhost -p 5432 -U educore_app -d educore -Fc -f educore_backup_$(date +%Y%m%d_%H%M%S).dump
```

Untuk skrip otomatisasi di server, hindari menulis password di flag terminal. Gunakan variabel lingkungan `PGPASSWORD` atau file konfigurasi `~/.pgpass`.

### B. Prosedur Pemulihan (Recovery)
> [!CAUTION]
> Pastikan tidak melakukan restore ke database produksi yang sedang aktif melayani transaksi tanpa koordinasi *maintenance window*.

1. Buat database baru atau bersihkan objek target:
   ```bash
   createdb -h localhost -p 5432 -U educore_app educore_restored
   ```
2. Pulihkan data menggunakan `pg_restore`:
   ```bash
   pg_restore -h localhost -p 5432 -U educore_app -d educore_restored --clean --if-exists educore_backup_20261011_000000.dump
   ```
3. Verifikasi konsistensi tabel dan data setelah pemulihan selesai:
   ```sql
   SELECT count(*) FROM users;
   SELECT count(*) FROM roles;
   ```

---

## 5. Pemecahan Masalah (*Troubleshooting*)

### Masalah 1: Startup Aplikasi Gagal dengan Pesan `failed to connect to database`
- **Penyebab**: Server PostgreSQL mati, port terblokir firewall, atau kredensial/port salah.
- **Tindakan**:
  1. Periksa apakah PostgreSQL aktif (`pg_isready -h localhost -p 5432`).
  2. Periksa kecocokan kredensial di `.env`. Log aplikasi tidak membocorkan password demi keamanan.

### Masalah 2: Endpoint `/health/ready` Mengembalikan `503 Service Unavailable`
- **Penyebab**: Koneksi database terputus di tengah jalan atau beban CPU server PostgreSQL 100% sehingga ping melebihi `DB_HEALTH_TIMEOUT` (2 detik).
- **Format Respons Error**:
  ```json
  {
    "code": "DEPENDENCY_UNAVAILABLE",
    "message": "Database is not available",
    "status": 503
  }
  ```
- **Tindakan**: Periksa log backend untuk melihat rincian error internal (misal: `context deadline exceeded`). Periksa kapasitas connection pool dan throughput database server.

### Masalah 3: Migrasi Gagal dengan Status `Dirty: true`
- **Penyebab**: Terjadi sintaks DDL yang salah atau terputusnya koneksi di tengah-tengah eksekusi file migrasi `.up.sql`.
- **Tindakan**:
  1. Buka PostgreSQL console.
  2. Periksa tabel `schema_migrations`:
     ```sql
     SELECT * FROM schema_migrations;
     ```
  3. Perbaiki error DDL yang gagal dieksekusi.
  4. Reset status dirty secara manual:
     ```sql
     UPDATE schema_migrations SET dirty = false WHERE version = <failed_version>;
     ```
  5. Jalankan kembali migrasi naik (`go run ./cmd/migrate up`).

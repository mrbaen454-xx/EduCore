# Panduan Migrasi Database EduCore (Fase 04)

Dokumen ini menjelaskan alur kerja, pengoperasian, aturan keamanan, dan strategi rollback untuk sistem migrasi skema database **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Tool Migrasi yang Dipilih

EduCore menggunakan pustaka standar industri:
- **Library**: `github.com/golang-migrate/migrate/v4`
- **Driver Database**: `github.com/golang-migrate/migrate/v4/database/pgx/v5`
- **Driver Source**: `github.com/golang-migrate/migrate/v4/source/file`
- **CLI Khusus**: `cmd/migrate/main.go`

### Alasan Pemilihan `golang-migrate`:
1. **Dukungan Native pgx/v5**: Terintegrasi langsung dengan ekosistem pgx tanpa memerlukan driver database generik `database/sql` lib/pq yang sudah usang.
2. **Ketat & Terdokumentasi**: Menggunakan tabel audit versi `schema_migrations` dengan status `version` dan flag `dirty` untuk mencegah migrasi setengah jalan dijalankan ulang tanpa penanganan teknis.
3. **Format Standar SQL**: Tidak mengunci pengembang ke dalam DSL/syntax proprietary; menggunakan SQL murni dengan file berpasangan `.up.sql` dan `.down.sql`.

---

## 2. Struktur File Migrasi

Semua berkas migrasi ditempatkan pada direktori `migrations/`:

```text
migrations/
├── 000001_initial_schema.up.sql
└── 000001_initial_schema.down.sql
```

### Konvensi Penamaan:
`{VERSION}_{DESCRIPTION}.{up|down}.sql`
- `{VERSION}`: 6 digit angka berurutan (`000001`, `000002`, dst).
- `{DESCRIPTION}`: Keterangan singkat dalam format `snake_case`.
- `.up.sql`: Perintah SQL untuk mengaplikasikan skema baru.
- `.down.sql`: Perintah SQL untuk membatalkan skema secara simetris dan aman.

---

## 3. Cara Menjalankan Migrasi

### Melalui Perintah Go CLI Langsung:
1. **Menjalankan Seluruh Migrasi Naik (`up`)**:
   ```bash
   go run ./cmd/migrate up
   ```
   *Atau dengan flags eksplisit:*
   ```bash
   go run ./cmd/migrate -dir migrations -dsn "postgres://educore_app:secret@localhost:5432/educore?sslmode=disable" up
   ```

2. **Memeriksa Status Versi Migrasi Saat Ini (`version`)**:
   ```bash
   go run ./cmd/migrate version
   ```
   *Contoh Output:*
   ```text
   Current migration version: 1 (dirty: false)
   ```

3. **Melakukan Rollback (`down`)**:
   ```bash
   # Rollback 1 tahap migrasi terakhir (default):
   go run ./cmd/migrate down -steps 1

   # Rollback 2 tahap:
   go run ./cmd/migrate down -steps 2
   ```

### Melalui Makefile (Jika Lingkungan Mendukung Make):
```bash
make db-up
make db-down
make db-version
```

---

## 4. Tabel Metadata `schema_migrations`

`golang-migrate` secara otomatis membuat dan mengelola tabel internal:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL PRIMARY KEY,
    dirty boolean NOT NULL
);
```

- `version`: Nomor versi migrasi aktif terakhir yang berhasil diterapkan.
- `dirty`:
  - `false`: Migrasi berjalan sukses tanpa error.
  - `true`: Terjadi kegagalan saat proses SQL berjalan. Jika status `dirty: true`, runner migrasi akan menolak menjalankan migrasi baru sampai DBA/developer memperbaiki database secara manual dan mereset status flag dirty.

---

## 5. Cara Membuat Migrasi Baru

1. Tentukan nomor urut berikutnya (misal: `000002`).
2. Buat berkas baru berpasangan:
   - `migrations/000002_create_faculties_table.up.sql`
   - `migrations/000002_create_faculties_table.down.sql`
3. Tulis DDL SQL yang valid, aman, dan eksplisit.
4. Uji migrasi pada database lokal atau testing:
   ```bash
   go run ./cmd/migrate up
   go run ./cmd/migrate version
   go run ./cmd/migrate down -steps 1
   go run ./cmd/migrate up
   ```

---

## 6. Kebijakan Keamanan dan Risiko Rollback

> [!WARNING]
> **Rollback (`down.sql`) berpotensi menghilangkan data produksi secara permanen!**

1. **Prinsip Immutability**:
   - File migrasi yang sudah pernah di-commit dan dijalankan di lingkungan staging/produksi **TIDAK BOLEH DIUBAH ATAU DIHAPUS**.
   - Setiap perbaikan atau perubahan skema wajib dibuat sebagai file migrasi baru (`000002`, `000003`, dst).
2. **Jangan Menggunakan Rollback Otomatis Saat Startup**:
   - Aplikasi EduCore **tidak** menjalankan migrasi `down` secara otomatis jika aplikasi gagal booting.
3. **Pemisahan Jalur Eksekusi**:
   - Migrasi dijalankan sebagai proses operasional terpisah (misal: init container, deployment pipeline CI/CD, atau perintah CLI eksplisit), bukan dijalankan secara paralel oleh setiap replika instance HTTP server.
4. **Prosedur Sebelum Rollback**:
   - Lakukan backup database lengkap (`pg_dump`) sebelum mengeksekusi migrasi `down` di lingkungan staging atau produksi.

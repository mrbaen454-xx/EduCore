# Arsitektur Database EduCore (Fase 04)

Dokumen ini menjelaskan arsitektur koneksi, manajemen connection pool, siklus hidup (lifecycle), alur transaksi, dan integrasi database PostgreSQL pada **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Peran PostgreSQL

PostgreSQL 18+ dipilih sebagai *primary datastore* (RDBMS) tunggal untuk EduCore dengan pertimbangan:
- **Integritas Relasional Kuat**: Menjamin konsistensi data akademik antar entitas (mahasiswa, dosen, program studi, kurikulum, IRS/KRS, nilai) melalui *Foreign Keys*, *Unique Constraints*, dan *Check Constraints*.
- **Kepatuhan ACID Penuh**: Menjamin keandalan transaksi akademik multi-tabel (misal: pendaftaran mata kuliah/KRS dengan kuota kelas, pembatalan IRS, dan penginputan nilai).
- **Dukungan Tipe Data Modern**: Mendukung `UUID` native, `TIMESTAMPTZ` untuk pencatatan waktu lintas zona, `JSONB` untuk audit/metadata dinamis, serta ekstensi kriptografi (`pgcrypto`).

---

## 2. Arsitektur Koneksi dan Connection Pool

EduCore menggunakan driver **pgx v5** (`github.com/jackc/pgx/v5`) dan connection pool resmi **pgxpool** (`github.com/jackc/pgx/v5/pgxpool`).

```text
+-------------------------------------------------------------+
|                      EduCore Application                    |
|                                                             |
|  +--------------------+         +------------------------+  |
|  | HTTP Handlers      |         | Health Check Handler   |  |
|  | (REST API Request) |         | (/health/ready)        |  |
|  +---------+----------+         +-----------+------------+  |
|            |                                |               |
|            | Acquire conn / RunInTx         | CheckHealth() |
|            v                                v               |
|  +-------------------------------------------------------+  |
|  |             *pgxpool.Pool (Single Instance)           |  |
|  |                                                       |  |
|  |   MaxConns: 10              MinConns: 2               |  |
|  |   MaxConnLifetime: 30m      MaxConnIdleTime: 5m       |  |
|  |   ConnectTimeout: 5s        HealthTimeout: 2s         |  |
|  +---------------------------+---------------------------+  |
+------------------------------|------------------------------+
                               |
                   TCP / TLS Connection Pool
                               |
                               v
               +-------------------------------+
               |     PostgreSQL 18 Database    |
               |      (educore / educore_test) |
               +-------------------------------+
```

### Karakteristik Pool:
1. **Single Shared Instance**: Pool dibuat satu kali saat startup aplikasi (`internal/app/app.go`) dan diinjeksikan secara eksplisit (*Dependency Injection*) ke HTTP Server dan service yang membutuhkan.
2. **Tidak Ada Pool Per-Request**: Request HTTP tidak membuka koneksi fisik baru, melainkan meminjam (*acquire*) koneksi dari pool lalu mengembalikannya (*release*) secara otomatis setelah query/transaksi selesai.
3. **Konfigurasi Adaptif**:
   - `DB_MAX_CONNS`: Batas atas koneksi simultan untuk mencegah kejenuhan koneksi PostgreSQL (*connection exhaustion*).
   - `DB_MIN_CONNS`: Koneksi idle minimum yang selalu siap melayani request tanpa latensi pembuatan koneksi TCP baru (*cold start*).
   - `DB_MAX_CONN_LIFETIME`: Masa hidup maksimum koneksi sebelum di-recycle guna merilis memori backend PostgreSQL.
   - `DB_MAX_CONN_IDLE_TIME`: Waktu idle maksimum sebelum koneksi non-minimum ditutup.

---

## 3. Siklus Hidup Database (Lifecycle)

Siklus hidup koneksi database terintegrasi penuh dengan lifecycle aplikasi di `internal/app/app.go`:

```text
       Application Startup
               |
               v
      Load Configuration (.env)
               |
               v
    Validate DB Configuration
               |
               v
    Create *pgxpool.Pool
               |
               v
    Ping Database (Timeout 5s)  -------- (Gagal) --------> Fatal Error & Exit
               | (Sukses)
               v
    Inject Pool to HTTPServer
               |
               v
     Start HTTP Listener
               |
        [Runtime Phase]
      Serve HTTP Requests
      /health/ready (ping)
               |
               v
   Receive SIGINT / SIGTERM
               |
               v
     Graceful Shutdown HTTP
    (Tunggu in-flight request)
               |
               v
       dbPool.Close()
 (Tutup semua koneksi database)
               |
               v
       Aplikasi Berhenti Bersih
```

### Aturan Shutdown:
1. HTTP Server berhenti menerima request baru (`Shutdown()`).
2. Request yang sedang berjalan diberikan waktu toleransi (*shutdown timeout*) untuk menyelesaikan transaksi database.
3. `dbPool.Close()` dipanggil **setelah** server HTTP selesai berhenti, memastikan tidak ada koneksi database yang ditutup prematur saat query sedang berlangsung.

---

## 4. Alur Transaksi (`RunInTx`)

Untuk menjamin eksekusi multi-query aman, paket `internal/platform/database` menyediakan abstraksi helper:

```go
func RunInTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error
```

### Karakteristik Eksekusi:
1. **Eksplisit**: Transaksi dimulai dengan `pool.BeginTx(ctx, pgx.TxOptions{})`.
2. **Jaminan Rollback**: Menggunakan `defer tx.Rollback(ctx)` yang otomatis aman jika commit sudah berhasil (error `pgx.ErrTxClosed` diabaikan).
3. **Commit Terkendali**: `tx.Commit(ctx)` hanya dieksekusi jika callback `fn(tx)` mengembalikan `nil`.
4. **Preservasi Error**: Jika eksekusi fungsi gagal dan rollback juga mengalami kendala, error bisnis utama tetap diprioritaskan agar tidak tertutupi oleh error rollback jaringan.

---

## 5. Hubungan dengan Layer Aplikasi

Arsitektur EduCore mengikuti pola **Modular Monolith**:
- **Layer Platform** (`internal/platform/database`): Mengelola driver, pool, health check, parser error, dan runner migrasi. Tidak mengandung domain logic.
- **Layer Application** (`internal/app`): Mengorkestrasi pembuatan pool dan menyalurkannya via struct dependency.
- **Layer Transport / API** (`internal/platform/httpserver`): Menerima pool hanya untuk keperluan health check readiness (`/health/ready`).
- **Layer Repository / Service (Fase Selanjutnya)**: Modul bisnis (misal: `internal/modules/auth`, `internal/modules/academic`) akan menerima `*pgxpool.Pool` untuk mengeksekusi query dan transaksi domain.

---

## 6. Batasan Implementasi Fase 04

1. **Fokus Fondasi**: Fase 04 hanya mencakup infrastruktur database, pool, health check, baseline tabel (`roles`, `users`, `user_roles`), dan error handling.
2. **Belum Ada Domain Query**: Belum ada repository atau service akademik (Mahasiswa, KRS, Nilai) pada fase ini.
3. **Tanpa ORM**: Digunakan pendekatan SQL murni / pgx native untuk kontrol performa dan integritas constraint yang presisi.

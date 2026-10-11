# Panduan Autentikasi EduCore (Fase 05)

Dokumen ini menjelaskan arsitektur autentikasi, mekanisme login, kebijakan password hashing, dan mitigasi keamanan identitas pada **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Mekanisme Login

Endpoint login publik tersedia pada:
```http
POST /api/v1/auth/login
Content-Type: application/json
```

### Format Payload Request:
```json
{
  "identifier": "admin@example.com",
  "password": "AdminSecurePassword123!"
}
```

### Alur Eksekusi Login:
1. **Normalisasi Identifier**: String identifier dinormalisasi menggunakan `strings.ToLower(strings.TrimSpace(identifier))` sebelum dievaluasi.
2. **Pencarian Akun Otoritatif**: Akun dicari di tabel `users` database PostgreSQL.
3. **Mitigasi Serangan Enumerasi Akun & Timing Attacks**:
   - Jika identifier tidak terdaftar di database, sistem **tidak** langsung mengembalikan error.
   - Sistem melakukan verifikasi sandi tiruan (*dummy Argon2id verification*) terhadap hash konstan sehingga waktu eksekusi komputasi tetap identik antara identifier yang terdaftar dan tidak terdaftar.
4. **Verifikasi Hash Sandi**: Sandi diverifikasi menggunakan pustaka Argon2id dengan perbandingan waktu-konstan (`subtle.ConstantTimeCompare`).
5. **Pemeriksaan Status Akun**: Kolom `is_active` dievaluasi. Jika `false`, login ditolak.
6. **Penerbitan Token**:
   - Menerbitkan **Access Token (JWT)** berumur pendek (default 15 menit).
   - Menerbitkan **Refresh Token** bernilai kriptografis acak (default 7 hari) dan menyimpan hash SHA-256 token tersebut ke database.
7. **Pesan Respons Gagal**: Jika kredensial salah, pengguna tidak terdaftar, atau akun nonaktif, respons yang dikembalikan selalu pesan generik yang seragam:
   ```json
   {
     "success": false,
     "message": "Identifier atau password tidak valid",
     "error": {
       "code": "INVALID_CREDENTIALS",
       "details": []
     },
     "meta": {
       "request_id": "7d9e18b8f36c4b6ea32f1a8c9b20e111"
     }
   }
   ```

---

## 2. Password Hashing (Argon2id)

EduCore menggunakan algoritma **Argon2id** (pemenang Password Hashing Competition / PHC dan rekomendasi resmi OWASP).

### Parameter Algoritma (Dikonfigurasi via Environment):
- **Memory (`ARGON2_MEMORY`)**: `65536 KiB` (64 MB)
- **Iterations / Time (`ARGON2_ITERATIONS`)**: `3` rounds
- **Parallelism / Threads (`ARGON2_PARALLELISM`)**: `2` threads
- **Salt Length (`ARGON2_SALT_LENGTH`)**: `16` bytes (kriptografis acak via `crypto/rand`)
- **Key Length (`ARGON2_KEY_LENGTH`)**: `32` bytes (256-bit output)

### Format Penyimpanan Hash (PHC String Format):
```text
$argon2id$v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>
```

Hash sandi tidak pernah dikembalikan dalam endpoint mana pun, tidak pernah dicatat pada log aplikasi, dan terlindungi dari serangan *dictionary* dan *GPU/ASIC cracking*.

---

## 3. Kebijakan Access Token (JWT)

- **Algoritma**: HMAC-SHA256 (`HS256`) dengan secret minimum 32 karakter.
- **Header Token**: Disematkan melalui header standar `Authorization: Bearer <access_token>`.
- **Klaim Standar (Registered Claims)**:
  - `sub`: User ID (UUID string).
  - `iss`: Issuer (`educore-api`).
  - `aud`: Audience (`educore-app`).
  - `iat`: Timestamp penerbitan token.
  - `exp`: Timestamp kedaluwarsa token (default: `iat + 15m`).
  - `jti`: Identifier unik token (UUID v4) untuk mencegah replay.
- **Klaim Kustom**:
  - `roles`: Array string role pengguna aktif (misal: `["admin"]`, `["dosen"]`).
  - `email`: Alamat email terdaftar.
  - `name`: Nama lengkap pengguna.

---

## 4. Perlindungan dan Batasan Saat Ini

1. **Brute-Force & Credential Stuffing**: Rate limiting terdistribusi belum diimplementasikan pada level aplikasi. Pada lingkungan produksi, disarankan memasang reverse proxy (Nginx, Cloudflare, Traefik) untuk membatasi laju request per IP ke `/api/v1/auth/login`.
2. **Ketiadaan Fitur Registrasi Publik**: Sesuai desain SIAKAD perguruan tinggi, akun pengguna diprovisi secara terpusat oleh administrator institusi atau sinkronisasi data kepegawaian/akademik, bukan registrasi publik.
3. **MFA / 2FA**: Multi-Factor Authentication belum termasuk dalam ruang lingkup Fase 05 dan dijadwalkan pada roadmap penguatan keamanan lanjutan.

# Siklus Hidup Token (Token Lifecycle) & Manajemen Sesi EduCore

Dokumen ini menjelaskan arsitektur, siklus hidup token (Access Token & Refresh Token), mekanisme rotasi, deteksi pencurian token (*token reuse detection*), pencabutan sesi (*revocation*), dan pedoman penyimpanan pada sisi klien untuk sistem **EduCore — Academic Management System**.

---

## 1. Arsitektur Dual-Token

EduCore menggunakan strategi **Dual-Token** untuk menyeimbangkan performa, skalabilitas, dan keamanan:

| Karakteristik | Access Token (JWT) | Refresh Token (Opaque Token) |
|---|---|---|
| **Format** | JSON Web Token (RFC 7519) | 32-byte cryptographically secure random (Hex-encoded) |
| **Masa Berlaku** | Pendek (Default: 15 Menit) | Panjang (Default: 7 Hari) |
| **Penyimpanan Server** | *Stateless* (tidak disimpan di database) | *Stateful* (Disimpan sebagai SHA-256 hash di PostgreSQL) |
| **Tujuan** | Otorisasi request ke API terlindungi | Penerbitan access token baru tanpa meminta kredensial ulang |
| **Algoritma / Hash** | HMAC-SHA256 (`HS256`) dengan secret >= 32 byte | SHA-256 hashing sebelum disimpan ke database |

```
+--------+                               +---------------+
|        |--(A)- POST /auth/login ------>|               |
|        |<-(B)- Access + Refresh Token -|               |
|        |                               |    EduCore    |
|        |--(C)- Request + AccessToken ->|   API Server  |
| Client |<-(D)- Resource / Data --------|       &       |
|        |                               |   PostgreSQL  |
|        |--(E)- POST /auth/refresh ---->|               |
|        |<-(F)- New Access + New Refresh|               |
|        |                               |               |
|        |--(G)- POST /auth/logout ----->|               |
|        |<-(H)- 200 OK (Revoked) -------|               |
+--------+                               +---------------+
```

---

## 2. Siklus Hidup Access Token

1. **Penerbitan (*Issuance*)**:
   - Diterbitkan saat `POST /api/v1/auth/login` berhasil atau melalui rotasi `POST /api/v1/auth/refresh`.
   - Menghasilkan payload JWT dengan klaim standar (`sub`, `iss`, `aud`, `iat`, `exp`, `jti`) serta klaim otorisasi (`roles`, `email`, `name`).
2. **Penggunaan (*Usage*)**:
   - Klien menyertakan token pada header HTTP `Authorization: Bearer <access_token>`.
   - Middleware `Authenticate` memverifikasi tanda tangan kriptografis, batas waktu (`exp`), `nbf`, penerbit (`iss`), dan audiens (`aud`) secara lokal dalam memori tanpa query database pada setiap request.
3. **Kedaluwarsa (*Expiration*)**:
   - Setelah masa berlaku habis (misalnya 15 menit), middleware menolak request dengan status `HTTP 401 Unauthorized` dan kode error `UNAUTHORIZED`.
   - Klien menangkap error ini dan secara otomatis memicu proses perpanjangan sesi menggunakan refresh token.

---

## 3. Siklus Hidup & Rotasi Refresh Token (*Token Rotation*)

Untuk mencegah ancaman pencurian refresh token jangka panjang, EduCore menerapkan **Refresh Token Rotation (RTR)** dengan **Token Reuse Detection**.

### 3.1 Alur Rotasi Normal

1. Klien mengirim `POST /api/v1/auth/refresh` membawa token mentah (`refresh_token`).
2. Server menghitung hash SHA-256 dari token mentah dan mencari record di tabel `refresh_tokens`.
3. Server memvalidasi:
   - Token ditemukan.
   - Token belum dicabut (`revoked_at IS NULL`).
   - Token belum kedaluwarsa (`expires_at > NOW()`).
   - Akun pengguna berstatus `ACTIVE`.
4. Dalam satu transaksi PostgreSQL atomik (`SERIALIZABLE` / `READ COMMITTED` dengan baris terkunci):
   - Server menghasilkan pasangan token baru: Access Token baru & Refresh Token baru.
   - Hash Refresh Token baru disimpan ke tabel `refresh_tokens`.
   - Refresh Token lama ditandai telah dicabut (`revoked_at = NOW()`) dan dicatat referensi token penggantinya (`replaced_by = new_token_id`).
5. Server mengembalikan pasangan token baru ke klien.

### 3.2 Deteksi Penggunaan Kembali (*Token Reuse Detection*)

Jika pihak ketiga (penyerang) berhasil mencuri refresh token yang telah digunakan (atau salinan token yang telah dirotasi):

1. Penyerang mencoba mengirim refresh token lama yang sudah memiliki `revoked_at != NULL` atau `replaced_by != NULL`.
2. Server mendeteksi bahwa token yang sudah dicabut sedang dicoba kembali (*reuse detected*).
3. **Tindakan Keamanan Otomatis**:
   - Server membatalkan/mencabut **seluruh** refresh token aktif milik pengguna tersebut (`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`).
   - Server mencatat peristiwa keamanan (*security audit log*) bahwa indikasi kompromi sesi terjadi pada akun tersebut.
   - Server mengembalikan `HTTP 401 Unauthorized`.
   - Pengguna sah maupun penyerang dipaksa login ulang menggunakan kredensial lengkap.

---

## 4. Mekanisme Logout & Pencabutan Sesi (*Revocation*)

EduCore menyediakan endpoint logout eksplisit:

`POST /api/v1/auth/logout`

Request Body:
```json
{
  "refresh_token": "string-refresh-token-mentah"
}
```

Alur:
1. Klien mengirimkan refresh token sesi yang ingin diakhiri.
2. Server menghitung SHA-256 hash dan mencabut sesi terkait (`UPDATE refresh_tokens SET revoked_at = NOW()`).
3. Respons `HTTP 200 OK` dikembalikan.
4. Klien menghapus Access Token dan Refresh Token dari penyimpanan lokalnya.

### Pencabutan Massal (Admin/Perubahan Status Akun)

Jika status akun pengguna diubah menjadi `INACTIVE` atau `SUSPENDED` oleh Administrator, atau terjadi pergantian password:
- Fungsi `RevokeAllUserTokens(ctx, userID)` dapat dipanggil untuk mencabut seluruh sesi aktif pengguna tersebut dari database.
- Upaya refresh token berikutnya akan langsung ditolak karena status akun divalidasi pada setiap permohonan refresh.

---

## 5. Penyimpanan Token pada Sisi Klien (*Client-Side Storage*)

Penyimpanan token yang tepat sangat bergantung pada jenis klien yang mengakses API:

### 5.1 Aplikasi Web (Single Page Applications - React/Vue/Svelte)

| Komponen | Rekomendasi Penyimpanan | Rationale |
|---|---|---|
| **Access Token** | Memory Klien (Variabel JavaScript / State Store) | Menghindari persistensi token di `localStorage` yang rentan serangan Cross-Site Scripting (XSS). Hilang saat reload, langsung diisi ulang via refresh token. |
| **Refresh Token** | Secure Cookie (`HttpOnly`, `Secure`, `SameSite=Strict`) atau Secure Storage | `HttpOnly` mencegah script JavaScript jahat membaca token secara langsung saat terjadi XSS. |

*Catatan: Apabila Refresh Token dikirim via JSON body (arsitektur saat ini), klien web disarankan menyimpan Access Token hanya di memory runtime dan mengisolasi penyimpanan Refresh Token dengan proteksi XSS ketat (misalnya Content Security Policy - CSP yang ketat).*

### 5.2 Aplikasi Mobile (Android / iOS)

- **Access Token**: Keystore / Memory Runtime.
- **Refresh Token**: Android Keystore / EncryptedSharedPreferences (Android) atau iOS Keychain Services (iOS).
- Jangan pernah menyimpan token di berkas teks biasa atau shared preferences tanpa enkripsi.

---

## 6. Rotasi Kunci Penandatanganan (Signing Key Rotation)

Untuk Access Token yang menggunakan algoritma `HS256`:
- `JWT_SECRET` dikelola melalui variabel lingkungan (Environment Variables) atau Secret Manager (HashiCorp Vault, AWS Secrets Manager).
- Panjang minimal secret adalah 32 karakter (256 bit entropy).
- **Prosedur Rotasi**:
  1. Siapkan secret baru.
  2. Karena masa berlaku access token EduCore singkat (15 menit), rotasi dapat dijadwalkan pada waktu pemeliharaan (*maintenance window*), atau menggunakan strategi *dual-key verification* jika masa transisi tanpa gangguan diperlukan.
  3. Saat kunci dirotasi, pengguna yang sedang aktif akan diminta refresh sesi; jika verifikasi token lama gagal, permohonan refresh akan menerbitkan access token baru yang ditandatangani dengan kunci baru.

---

## 7. Analisis Ancaman & Mitigasi

| Vektor Serangan | Risiko | Mitigasi EduCore |
|---|---|---|
| **JWT Header Alg Spoofing** | Penyerang mengirim token dengan `"alg": "none"` atau beralih algoritma | Validator secara eksplisit mengunci parser hanya pada `jwt.SigningMethodHS256`. |
| **Database Compromise** | Kebocoran data tabel refresh token | Database hanya menyimpan hash SHA-256 (`token_hash`), token mentah tidak pernah disimpan. |
| **Brute-force Refresh Token** | Penyerang menebak token | Token memiliki entropi 256-bit (32 byte `crypto/rand`), kemungkinan collision/tebakan secara matematis mustahil. |
| **Replay Attack / Stolen Refresh** | Token dicuri dan digunakan berulang | Refresh Token Rotation: token langsung dicabut setelah digunakan satu kali; deteksi penggunaan ulang membatalkan seluruh sesi korban. |
| **Timing Attack pada Enumerasi** | Perbedaan waktu respon login membocorkan keberadaan username | `auth.Service` menjalankan *dummy password verification* dengan Argon2id jika akun tidak ditemukan. |

# Panduan Otorisasi dan Kontrol Akses (Fase 05)

Dokumen ini mendefinisikan model otorisasi, hierarki role, middleware proteksi, dan aturan kontrol akses pada **EduCore — Academic Management System (SIAKAD)**.

---

## 1. Model Role-Based Access Control (RBAC)

EduCore menerapkan otorisasi berbasis peran (**Role-Based Access Control**). Struktur data mendukung **Multi-Role**, di mana satu akun pengguna dapat memiliki lebih dari satu peran (misal: seorang `dosen` yang sekaligus menjabat sebagai staf `akademik`).

### Daftar Role Baku Sistem:
| Role ID | Nama Role | Deskripsi & Hak Akses Umum |
|---|---|---|
| `admin` | Administrator | Pengelolaan teknis sistem, audit, konfigurasi sistem, dan manajemen hak akses global. |
| `akademik` | Staf Akademik | Pengelolaan kurikulum, jadwal perkuliahan, penawaran kelas, dan administrasi akademik institusi. |
| `dosen` | Dosen Pengajar | Pengelolaan kelas perkuliahan yang diampu, absensi mahasiswa, dan penginputan nilai. |
| `mahasiswa` | Mahasiswa | Pengisian KRS/IRS, melihat jadwal perkuliahan pribadi, melihat presensi, dan melihat transkrip/KHS. |

---

## 2. Prinsip Keamanan Otorisasi

1. **Deny-by-Default**:
   - Seluruh endpoint yang tidak secara eksplisit ditandai publik mewajibkan autentikasi.
   - Jika hak akses atau role belum ditentukan, akses otomatis ditolak.
2. **Pemisahan Autentikasi vs Otorisasi**:
   - **Autentikasi (401 Unauthorized)**: Menjawab *siapa Anda?* Terjadi saat token tidak ada, rusak, salah tandatangan, atau telah kedaluwarsa.
   - **Otorisasi (403 Forbidden)**: Menjawab *apa yang boleh Anda lakukan?* Terjadi saat identitas pengguna valid, namun pengguna tidak memiliki role yang diizinkan untuk resource/aksi tersebut.
3. **Validasi Status Akun Real-Time**:
   - Endpoint profil pengguna (`GET /api/v1/users/me`) membaca data akun secara langsung dari database otoritatif untuk memastikan akun yang dibekukan (`is_active = false`) segera kehilangan akses.

---

## 3. Middleware Otorisasi

### 1. `middleware.Authenticate(tokenMgr)`
- Membaca header `Authorization: Bearer <token>`.
- Memvalidasi tanda tangan HS256, issuer, audience, dan masa kedaluwarsa JWT.
- Menyimpan objek `*auth.Principal` dan `uuid.UUID` pengguna ke dalam context request Gin.

### 2. `middleware.RequireRoles(allowedRoles ...string)`
- Mengekstrak `Principal` dari context.
- Memeriksa apakah pengguna memiliki setidaknya salah satu dari `allowedRoles` yang diizinkan.
- Jika pengguna tidak memiliki role yang sesuai:
  ```json
  {
    "success": false,
    "message": "Akses ditolak: Anda tidak memiliki role yang diizinkan untuk resource ini",
    "error": {
      "code": "FORBIDDEN",
      "details": []
    },
    "meta": {
      "request_id": "9f3c7e4a1b2d8e6f0c4a7b9e1d3f5a8b"
    }
  }
  ```

---

## 4. Endpoint Pengujian Otorisasi

Untuk keperluan verifikasi dan pengujian otomatis, sistem menyediakan endpoint uji role:
```http
GET /api/v1/test/admin-only
Authorization: Bearer <TOKEN>
```
- Menghasilkan HTTP `200 OK` jika token memiliki role `admin`.
- Menghasilkan HTTP `403 Forbidden` jika token memiliki role non-admin (misal: `mahasiswa`).
- Menghasilkan HTTP `401 Unauthorized` jika diakses tanpa token.

---

## 5. Batasan dan Roadmap Otorisasi Sumber Daya (Resource Ownership)

Pada Fase 05, fondasi otorisasi berbasis peran (Role-level RBAC) telah selesai. Pada fase-fase berikutnya saat modul bisnis dibangun:
- **Otorisasi Kepemilikan (Ownership Check)**: Mahasiswa hanya boleh melihat dan memodifikasi KRS miliknya sendiri (`student_id == principal.UserID`).
- **Otorisasi Dosen Pengampu**: Dosen hanya boleh menginput nilai untuk kelas perkuliahan yang ditugaskan kepadanya.
- Pemeriksaan kepemilikan tersebut akan diimplementasikan pada lapisan service masing-masing modul domain.

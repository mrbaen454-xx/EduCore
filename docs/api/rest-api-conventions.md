# Konvensi REST API EduCore

Dokumen ini mendefinisikan standar desain dan arsitektur REST API yang berlaku secara konsisten untuk seluruh endpoint EduCore.

---

## 1. Prefix dan Versi API

- Seluruh endpoint modul bisnis akademik dikelompokkan dengan prefix versi: `/api/v1`.
- Health check infrastruktur berada di luar prefix versi:
  - `GET /health/live`: Mengecek apakah proses server hidup.
  - `GET /health/ready`: Mengecek apakah server siap menerima lalu lintas HTTP.
- Endpoint diagnostik API berada pada:
  - `GET /api/v1/health`: Mengecek ketersediaan API v1.

---

## 2. Konvensi Metode HTTP

| Metode | Penggunaan | Status Berhasil Default | Response Body |
|---|---|---|---|
| `GET` | Mengambil satu atau sekumpulan resource | `200 OK` | Ada (`data`) |
| `POST` | Membuat resource baru atau memproses aksi | `201 Created` / `200 OK` | Ada (`data`) |
| `PUT` | Mengganti keseluruhan resource | `200 OK` | Ada (`data`) |
| `PATCH` | Memperbarui sebagian field resource | `200 OK` | Ada (`data`) |
| `DELETE` | Menghapus resource | `204 No Content` / `200 OK` | Tidak ada untuk 204 |

---

## 3. Konvensi Header & Content-Type

1. **Content-Type**:
   - Seluruh request yang mengirim body wajib menyertakan:
     ```http
     Content-Type: application/json
     ```
   - Server selalu mengembalikan respons dengan `Content-Type: application/json`.
2. **X-Request-ID**:
   - Client dapat menyertakan header `X-Request-ID` untuk tracing terdistribusi.
   - Aturan validasi: Panjang 1–64 karakter, hanya karakter alfanumerik, tanda hubung (`-`), dan garis bawah (`_`).
   - Jika client tidak mengirimkan atau nilainya tidak valid, server akan otomatis membuat ID 32-karakter acak aman (`crypto/rand`).
   - Server selalu menyertakan `X-Request-ID` pada response header dan field `meta.request_id` pada respons error.

---

## 4. Standar Format Respons

### 4.1 Format Respons Sukses

Setiap respons sukses dibungkus dalam struktur JSON:

```json
{
  "success": true,
  "message": "Pesan deskriptif keberhasilan",
  "data": { ... },
  "meta": null
}
```

**Ketentuan:**
- `success`: Bernilai boolean `true`.
- `message`: Kalimat informatif yang aman dan mudah dipahami pengguna.
- `data`: Berisi payload hasil query atau operasi. Jika endpoint tidak mengembalikan data, gunakan `null` secara konsisten.
- `meta`: Metadata opsional (seperti pagination). Bernilai `null` jika tidak digunakan.

#### Contoh 200 OK (Single Object)
```json
{
  "success": true,
  "message": "Data mahasiswa berhasil diambil",
  "data": {
    "id": "mhs-001",
    "nim": "2024001001",
    "name": "Budi Santoso",
    "status": "aktif"
  },
  "meta": null
}
```

#### Contoh 204 No Content
Untuk penghapusan resource dengan status `204 No Content`, server **tidak** mengirim response body (panjang konten = 0).

---

### 4.2 Format Respons Error

Setiap respons error dibungkus dalam struktur standar:

```json
{
  "success": false,
  "message": "Pesan ringkas kegagalan",
  "error": {
    "code": "KODE_ERROR_STABIL",
    "details": []
  },
  "meta": {
    "request_id": "9f3c7e4a1b2d8e6f0c4a7b9e1d3f5a8b"
  }
}
```

**Ketentuan:**
- `success`: Bernilai boolean `false`.
- `message`: Ringkasan error publik yang ramah pengguna.
- `error.code`: Kode error mesin yang stabil (lihat `error-codes.md`).
- `error.details`: Array objek kesalahan spesifik (misalnya validasi field). Jika tidak ada detail tambahan, wajib berupa array kosong `[]` (bukan `null`).
- `meta.request_id`: Request ID untuk mempermudah penelusuran log internal.

#### Contoh 400 Bad Request (Validasi Gagal)
```json
{
  "success": false,
  "message": "Validasi request gagal",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": [
      {
        "field": "email",
        "message": "Format 'email' harus berupa alamat email yang valid"
      },
      {
        "field": "semester",
        "message": "Nilai 'semester' maksimal 14"
      }
    ]
  },
  "meta": {
    "request_id": "b188c6e39f374719bd699f056ec102ee"
  }
}
```

#### Contoh 404 Not Found (Route)
```json
{
  "success": false,
  "message": "Endpoint tidak ditemukan",
  "error": {
    "code": "ROUTE_NOT_FOUND",
    "details": []
  },
  "meta": {
    "request_id": "c287a911a37c4be59c3e9812df0a99ff"
  }
}
```

#### Contoh 405 Method Not Allowed
```json
{
  "success": false,
  "message": "Metode HTTP tidak didukung untuk endpoint ini",
  "error": {
    "code": "METHOD_NOT_ALLOWED",
    "details": []
  },
  "meta": {
    "request_id": "a199cf0218b74c8ea5bf90d3d5f12e11"
  }
}
```

---

## 5. Validasi Request

Aturan validasi input:
1. **Penolakan Field Tidak Dikenal (`DisallowUnknownFields`)**:
   Request body yang menyertakan atribut di luar kontrak struct DTO akan ditolak dengan error 400.
2. **Satu Dokumen JSON**:
   Hanya satu dokumen JSON per request yang diterima (tidak boleh ada trailing payload).
3. **Ukuran Maksimum Body (Payload Limit)**:
   Ukuran body request dibatasi maksimum 2 MB secara default untuk mencegah serangan DoS. Request melebihi batas menghasilkan `413 CONTENT_TOO_LARGE`.
4. **Mapping Error Ramah**:
   Tag validasi Go (`required`, `min`, `max`, `email`, `oneof`, dsb.) otomatis dipetakan ke pesan dalam Bahasa Indonesia yang informatif.

---

## 6. Pagination, Sorting, dan Filtering

### 6.1 Query Parameters
Endpoint daftar data menggunakan query parameter standar:
- `page`: Nomor halaman (integer positif >= 1, default: `1`).
- `limit`: Jumlah data per halaman (integer positif 1–100, default: `20`).
- `sort`: Nama kolom pengurutan (wajib terdaftar dalam allowlist modul).
- `order`: Arah pengurutan (`asc` atau `desc`, default: `asc`).

Contoh request:
```http
GET /api/v1/students?page=2&limit=25&sort=name&order=asc
```

### 6.2 Format Metadata Pagination
```json
{
  "success": true,
  "message": "Daftar mahasiswa berhasil diambil",
  "data": [
    { ... }
  ],
  "meta": {
    "page": 2,
    "limit": 25,
    "total_items": 120,
    "total_pages": 5
  }
}
```

### 6.3 Pencegahan SQL Injection pada Sorting
Modul backend **wajib** memvalidasi parameter `sort` terhadap *allowlist* menggunakan helper `params.ValidateSort(allowedColumns)` sebelum diteruskan ke query database.

---

## 7. Penanganan Error Internal & Keamanan

1. **Prinsip Masking**:
   Error internal sistem (misalnya koneksi database terputus, query error, panic runtime) dicatat secara lengkap pada level internal logger (`log/slog`), tetapi respons ke client dimasking menjadi pesan umum: `"Terjadi kesalahan internal pada server"` dengan kode `INTERNAL_ERROR`.
2. **Kerahasiaan Data**:
   Stack trace, string koneksi, kredensial, query SQL mentah, dan token rahasia **dilarang keras** dikembalikan dalam respons HTTP publik.
3. **Sanitasi Logging**:
   Nilai dari client (URL path, query parameter) disanitasi dari karakter newline (`\r`, `\n`) untuk mencegah log injection. Header sensitif seperti `Authorization` dan `Cookie` tidak pernah dicatat dalam log request.

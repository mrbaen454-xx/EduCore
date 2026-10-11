# Kamus Kode Error EduCore REST API

Dokumen ini berisi daftar kode error mesin (*machine-readable error codes*) yang digunakan pada seluruh endpoint EduCore, beserta pemetaan HTTP Status Code dan panduan penggunaannya.

---

## Tabel Kode Error Standar

| Kode Error | HTTP Status | Kondisi Penggunaan | Pesan Publik Aman |
|---|---|---|---|
| `VALIDATION_ERROR` | `400 Bad Request` | Satu atau lebih field pada request body/param tidak memenuhi aturan validasi. Menyertakan array `details`. | `Validasi request gagal` |
| `BAD_REQUEST` | `400 Bad Request` | Format request rusak, parameter query tidak valid, field tidak dikenal, atau multi-JSON payload. | `Format request tidak valid atau rusak` |
| `UNAUTHORIZED` | `401 Unauthorized` | Kredensial autentikasi belum diberikan, kadaluarsa, atau format token tidak valid. | `Autentikasi diperlukan untuk mengakses resource ini` |
| `FORBIDDEN` | `403 Forbidden` | Identitas pengguna terverifikasi, tetapi tidak memiliki hak akses/otorisasi terhadap resource atau aksi terkait. | `Akses ditolak: Anda tidak memiliki izin untuk resource ini` |
| `RESOURCE_NOT_FOUND` | `404 Not Found` | Data atau entitas spesifik yang diminta (misal: ID mahasiswa, kelas, mata kuliah) tidak ditemukan di sistem. | `Resource yang diminta tidak ditemukan` |
| `ROUTE_NOT_FOUND` | `404 Not Found` | Alamat URL endpoint tidak terdaftar pada router server. | `Endpoint tidak ditemukan` |
| `METHOD_NOT_ALLOWED` | `405 Method Not Allowed` | Path endpoint terdaftar, tetapi metode HTTP yang dikirim tidak didukung (misal: DELETE pada endpoint GET-only). | `Metode HTTP tidak didukung untuk endpoint ini` |
| `CONFLICT` | `409 Conflict` | Terjadi pelanggaran konsistensi data atau constraint unik (misal: NIM duplikat, jadwal kuliah bentrok). | `Terjadi konflik pada status resource atau data duplikat` |
| `CONTENT_TOO_LARGE` | `413 Content Too Large` | Ukuran body request melebihi ambang batas yang diizinkan (maksimal 2 MB secara default). | `Ukuran body request melebihi batas maksimum yang diizinkan` |
| `UNSUPPORTED_MEDIA_TYPE` | `415 Unsupported Media Type` | Header `Content-Type` yang dikirim bukan `application/json` pada request yang mewajibkan body JSON. | `Content-Type request tidak didukung, gunakan application/json` |
| `DEPENDENCY_UNAVAILABLE` | `503 Service Unavailable` | Layanan pihak ketiga atau dependensi vital (database, storage) sedang tidak dapat dijangkau. | `Layanan dependensi sedang tidak tersedia` |
| `INTERNAL_ERROR` | `500 Internal Server Error` | Terjadi kesalahan runtime tak terduga atau kepanikan (*panic*) pada server. Seluruh detail teknis disembunyikan dari klien. | `Terjadi kesalahan internal pada server` |

---

## Struktur Detail Validasi (`details`)

Ketika kode error adalah `VALIDATION_ERROR`, array `details` menyediakan rincian per-field dengan struktur:

```json
{
  "field": "string (nama properti JSON)",
  "message": "string (penjelasan kegagalan aturan dalam Bahasa Indonesia)"
}
```

Contoh:
```json
{
  "success": false,
  "message": "Validasi request gagal",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": [
      {
        "field": "nim",
        "message": "Field 'nim' wajib diisi"
      },
      {
        "field": "semester",
        "message": "Nilai 'semester' minimal 1"
      }
    ]
  },
  "meta": {
    "request_id": "7d9e18b8f36c4b6ea32f1a8c9b20e111"
  }
}
```

---

## Prinsip Keamanan & Error Handling

1. **Anti-Leakage**: Klien tidak boleh menerima cuplikan stack trace, pesan error driver SQL mentah, query database, atau informasi sensitif infrastruktur.
2. **Korelasi Request ID**: Setiap error menyertakan `meta.request_id` yang cocok dengan header `X-Request-ID` dan log server internal, memungkinkan teknisi melacak akar masalah dengan cepat tanpa membocorkannya ke publik.
3. **Kekebalan Panic**: Panic yang terjadi dalam siklus request akan ditangkap oleh middleware recovery dan diubah menjadi respons `INTERNAL_ERROR` 500 yang aman.

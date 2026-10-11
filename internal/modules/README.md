# EduCore Business Modules

Direktori ini dipersiapkan sebagai tempat implementasi modul-modul fungsional akademik EduCore pada fase berikutnya (seperti autentikasi, data master, kurikulum, KRS, presensi, penilaian, dan transkrip).

## Panduan Implementasi Modul

Setiap modul yang akan dibangun pada fase selanjutnya diharapkan mengikuti konvensi fondasi REST API (Fase 03):

1. **Request DTO & Validasi**:
   Gunakan struct request DTO khusus dengan tag `json` dan tag `validate` (misalnya `validate:"required,min=3"`). Lakukan binding menggunakan `validation.BindJSON(c, &dto)`. Jangan menggunakan model database/domain entity secara langsung sebagai DTO input.

2. **Error Handling**:
   Gunakan package `educore/internal/shared/errors`:
   - Jika data tidak ditemukan: `errors.ErrNotFound("Mahasiswa dengan ID tersebut tidak ditemukan")`
   - Jika aturan bisnis dilanggar: `errors.ErrBadRequest("Batas maksimum SKS terlampaui")` atau `errors.ErrConflict(...)`
   - Jika terjadi error internal tak terduga (misal database failure): bungkus dengan `errors.ErrInternal(err)` agar detail teknis tersimpan di logger internal namun respons ke client tetap aman.

3. **Format Respons**:
   Kembalikan data melalui helper `educore/internal/shared/response`:
   - `response.OK(c, "Data berhasil diambil", data, meta)`
   - `response.Created(c, "Data berhasil dibuat", data, nil)`
   - `response.NoContent(c)`
   - Untuk error: `response.RenderFromError(c, err, middleware.GetRequestID(c))`

4. **Pagination & Sorting**:
   Untuk endpoint list/daftar, gunakan `pagination.Parse(c)` dan validasi kolom pengurutan menggunakan `params.ValidateSort(allowedColumns)`.

# Kebijakan dan Aturan Bisnis: Mata Kuliah dan Kurikulum (Fase 07)

## 1. Aturan Bisnis Mata Kuliah (`courses`)

1. **Format Kode Mata Kuliah**:
   - Kode wajib diisi (`required`), panjang maksimal 20 karakter.
   - Kode dinormalisasi menjadi **UPPERCASE** dan di-trim whitespace.
   - Kode harus unik secara global pada tabel katalog `courses`. Upaya membuat kode yang sama menghasilkan respons `409 Conflict`.
2. **Nama Mata Kuliah**:
   - Wajib diisi, minimal 2 karakter, maksimal 255 karakter, tidak boleh hanya terdiri dari spasi.
3. **Beban SKS**:
   - Nilai SKS harus berupa bilangan bulat positif dalam rentang 1 sampai 8 (`1 <= credits <= 8`).
   - SKS bernilai 0, negatif, atau melebihi 8 akan ditolak dengan respons `422 Unprocessable Entity` atau `400 Bad Request`.
4. **Keaktifan (`is_active`)**:
   - Nilai default saat pembuatan adalah `true`.
   - Mata kuliah yang tidak aktif (`is_active = false`) tidak dapat ditambahkan ke dalam kurikulum baru (`409 Conflict`).
5. **Penghapusan / Proteksi Data**:
   - Mata kuliah yang masih direferensikan oleh kurikulum (`curriculum_courses`) atau menjadi prasyarat bagi mata kuliah lain (`course_prerequisites`) tidak dapat dihapus (`ON DELETE RESTRICT`).

---

## 2. Aturan Bisnis Prasyarat Mata Kuliah (`course_prerequisites`)

1. **Mata Kuliah Prasyarat Harus Valid**:
   - Target mata kuliah dan prasyarat mata kuliah harus sudah terdaftar pada katalog `courses`.
2. **Self-Prerequisite Dilarang**:
   - Suatu mata kuliah tidak boleh menjadi prasyarat bagi dirinya sendiri (`course_id != prerequisite_course_id`).
   - Ditangani di level service validation dan database check constraint.
3. **Duplikasi Prasyarat Dilarang**:
   - Kombinasi `(course_id, prerequisite_course_id)` dilindungi oleh unique constraint. Upaya duplikasi mengembalikan `409 Conflict`.
4. **Pencegahan Ketergantungan Siklus (Circular Dependency Prevention)**:
   - Sistem melakukan penelusuran graf prasyarat (graph traversal / cycle detection) sebelum relasi disimpan.
   - Jika mata kuliah A mensyaratkan B, maka B dilarang mensyaratkan A (secara langsung maupun transitif). Pelanggaran mengembalikan `409 Conflict` dengan pesan *"penambahan prasyarat akan menimbulkan dependensi siklik (circular dependency)"*.

---

## 3. Aturan Bisnis Kurikulum (`curricula`)

1. **Relasi Program Studi**:
   - Setiap kurikulum wajib memiliki referensi program studi (`study_program_id`) yang valid dan berstatus aktif (`is_active = true`).
   - Jika program studi tidak ditemukan, mengembalikan `404 Not Found`. Jika program studi berstatus nonaktif, mengembalikan `409 Conflict`.
2. **Kode dan Tahun Kurikulum**:
   - Kode kurikulum wajib unik, dinormalisasi uppercase, maksimal 30 karakter.
   - Tahun penerapan wajib bernilai antara 1970 hingga 2100.
3. **Lifecycle Status Kurikulum**:
   - Status kurikulum yang didukung:
     - `DRAFT`: Kurikulum sedang dalam perancangan. Mata kuliah dapat ditambah atau diubah.
     - `ACTIVE`: Kurikulum telah diberlakukan secara resmi untuk mahasiswa.
     - `INACTIVE`: Kurikulum sudah tidak digunakan untuk angkatan baru.
     - `ARCHIVED`: Kurikulum telah diarsipkan untuk arsip historis.
   - **Aturan Transisi Status**:
     - `DRAFT` dapat berpindah ke `ACTIVE` atau `INACTIVE`.
     - `ACTIVE` dapat berpindah ke `INACTIVE` atau `ARCHIVED`.
     - `INACTIVE` dapat diaktifkan kembali (`ACTIVE`) atau diarsipkan (`ARCHIVED`).
     - `ARCHIVED` bersifat final. Kurikulum berstatus `ARCHIVED` tidak dapat diubah metadatanya atau ditambah mata kuliahnya (`409 Conflict`).
4. **Proteksi Referensi**:
   - Kurikulum yang memuat relasi mata kuliah tidak dapat dihapus secara fisik (`ON DELETE RESTRICT`).

---

## 4. Aturan Bisnis Mata Kuliah dalam Kurikulum (`curriculum_courses`)

1. **Relasi Kurikulum & Mata Kuliah**:
   - Kurikulum target harus ada dan tidak berstatus `ARCHIVED`.
   - Mata kuliah yang diasosiasikan harus berstatus aktif (`is_active = true`).
2. **Keunikan dalam Kurikulum**:
   - Satu mata kuliah hanya boleh muncul satu kali dalam kurikulum yang sama (`UNIQUE(curriculum_id, course_id)`). Upaya duplikasi mengembalikan `409 Conflict`.
3. **Semester Rekomendasi**:
   - Nilai semester rekomendasi harus berada dalam rentang semester perkuliahan formal: **1 sampai 14**.
4. **Kategori Kelulusan**:
   - Nilai kategori yang diizinkan adalah:
     - `REQUIRED`: Mata kuliah wajib lulus.
     - `ELECTIVE`: Mata kuliah pilihan.
5. **Perhitungan Total SKS Dinamis**:
   - SKS total kurikulum dihitung berdasarkan relasi mata kuliah aktif dalam kurikulum tersebut.
   - Mengembalikan ringkasan: `total_courses`, `total_credits`, `required_credits`, `elective_credits`.

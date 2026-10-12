# Aturan Bisnis & Integritas: Data Master Akademik (Fase 06)

Dokumen ini mendefinisikan aturan bisnis resmi (*business rules*), siklus hidup (*lifecycle*), validasi data, serta penjaminan integritas transaksi untuk modul Data Master Akademik di sistem **EduCore**.

---

## 1. Aturan Bisnis Fakultas

1. **Format & Normalisasi Kode**:
   - Kode fakultas wajib diisi, tidak boleh hanya spasi, dan memiliki panjang 2–20 karakter.
   - Kode dinormalisasi secara otomatis menjadi huruf kapital (*UPPERCASE*) dan dipangkas spasi (*trimmed*).
   - Kode fakultas harus unik secara global di seluruh sistem.
2. **Ketergantungan Operasional**:
   - Fakultas nonaktif (`is_active = false`) tidak dapat menerima pembuatan program studi baru. Upaya membuat program studi di bawah fakultas nonaktif akan ditolak dengan error `409 Conflict`.
   - Menonaktifkan fakultas **tidak** otomatis mengubah status program studi di bawahnya secara serentak (mencegah *cascading side-effect* tanpa persetujuan administratif).
3. **Penghapusan Fisik**:
   - Penghapusan fisik data fakultas dilarang jika terdapat program studi yang merujuk kepadanya (`ON DELETE RESTRICT`). Sistem menyarankan penggunaan toggle status nonaktif.

---

## 2. Aturan Bisnis Program Studi

1. **Relasi Fakultas Induk**:
   - Setiap program studi wajib merujuk ke fakultas yang sah (*valid foreign key*).
   - Fakultas induk wajib berada dalam status aktif saat prodi dibuat atau dihubungkan.
2. **Keunikan & Jenjang**:
   - Kode program studi wajib unik secara global institusi (contoh: `TIF`, `SI`, `TI`).
   - Jenjang pendidikan (*education level*) dibatasi secara ketat pada nilai baku:
     `D3`, `D4`, `S1`, `S2`, `S3`, `Profesi`.
3. **Integritas Relasi**:
   - Program studi tidak dapat dihapus secara fisik jika telah memiliki referensi dosen atau mahasiswa terdaftar.

---

## 3. Aturan Bisnis Dosen

1. **Identitas & Legalitas**:
   - NIDN (Nomor Induk Dosen Nasional) bersifat wajib, unik, dan tidak boleh terduplikasi.
   - Email dosen wajib unik dan memiliki format email valid.
   - Dosen wajib bernaung pada homebase program studi yang berstatus aktif.
2. **Format Gelar & Nama**:
   - Nama asli disimpan terpisah dari gelar depan (`title_prefix`) dan gelar belakang (`title_suffix`).
   - Tampilan nama lengkap dosen diagregasikan melalui helper domain `FullNameWithTitles()` (contoh: `Prof. Dr. Ir. Joko Susilo, M.T., IPU`).
3. **Status Kepegawaian Dosen**:
   - Nilai status dibatasi pada:
     - `ACTIVE`: Dosen aktif menjalankan tridharma perguruan tinggi.
     - `INACTIVE`: Dosen tidak aktif / pensiun / purna tugas.
     - `LEAVE`: Dosen sedang cuti tugas belajar / sabatikal.
4. **Keterkaitan Akun Pengguna**:
   - Profil dosen dapat dihubungkan ke akun pengguna (`user_id`).
   - Menonaktifkan profil dosen tidak otomatis menghapus akun pengguna, dan sebaliknya.

---

## 4. Aturan Bisnis Mahasiswa

1. **NIM & Identitas Akademik**:
   - NIM (Nomor Induk Mahasiswa) wajib unik di seluruh institusi.
   - Mahasiswa wajib terdaftar pada program studi yang aktif.
   - Tahun angkatan (*admission year*) harus berupa tahun valid dalam rentang `1900` hingga `2100`.
2. **Biodata & Privasi**:
   - Tanggal lahir bersifat opsional namun jika diisi harus berformat `YYYY-MM-DD` dan merepresentasikan tanggal valid.
   - Jenis kelamin dibatasi pada `MALE` dan `FEMALE`.
3. **Siklus Hidup Status Mahasiswa**:
   - Nilai status akademik mahasiswa:
     - `ACTIVE`: Terdaftar aktif dan berhak mengisi KRS.
     - `INACTIVE`: Mahasiswa mangkir / belum registrasi.
     - `PROBATION`: Mahasiswa dalam masa evaluasi akademik / percobaan.
     - `GRADUATED`: Mahasiswa telah lulus yudisium.
     - `DROPOUT`: Mahasiswa mengundurkan diri atau diberhentikan.
     - `SUSPENDED`: Mahasiswa diskors karena sanksi akademik atau disiplin.
4. **Proteksi Riwayat Transkrip**:
   - Mahasiswa yang telah memiliki data akademik (KRS, nilai, transkrip) dilarang keras dihapus secara fisik.

---

## 5. Penanganan Konkurensi & Integritas Basis Data

1. **Pencegahan Race Condition pada Validasi Keunikan**:
   - EduCore tidak hanya mengandalkan pola `SELECT` sebelum `INSERT`.
   - Seluruh aturan keunikan (kode fakultas, kode prodi, NIDN, NIM, email) dijaga melalui *Unique Constraint* PostgreSQL.
   - Driver dan repository memetakan kode error PostgreSQL `23505` (*unique_violation*) menjadi respons terstruktur `409 Conflict`.
2. **Penanganan Integritas Foreign Key**:
   - Pelanggaran referensi foreign key (`23503` - *foreign_key_violation*) dipetakan secara terstruktur menjadi error `404 Not Found` (jika induk tidak ditemukan) atau `409 Conflict` (jika data terikat menolak penghapusan).
   - Pengguna API tidak akan pernah menerima pesan galat SQL mentah (*raw database error leak*).

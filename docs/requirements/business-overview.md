# Business Overview: EduCore

## 1. Executive Summary
EduCore adalah Sistem Informasi Akademik (SIAKAD) untuk perguruan tinggi yang dirancang sebagai sistem terpusat (Modular Monolith) berbasis Go dan PostgreSQL. Sistem ini mengelola operasi akademik mulai dari data master, kurikulum, registrasi mahasiswa (KRS), presensi, penilaian, hingga penerbitan transkrip (IP/IPK), dengan mengedepankan integritas data, keamanan, dan auditabilitas.

## 2. In-Scope
- Autentikasi dan Manajemen Akun (RBAC, JWT)
- Data Master (Fakultas, Prodi, Dosen, Mahasiswa)
- Akademik Dasar (Periode, Kurikulum, Mata Kuliah)
- Operasional Kelas & KRS
- Presensi & Penilaian (Grades)
- Transkrip Akademik (Perhitungan IP/IPK)
- Audit log untuk operasi kritis
- Laporan (Dashboard spesifik per role)

## 3. Out-of-Scope (Fase Awal)
- Billing / Pembayaran UKT Mahasiswa
- Penerimaan Mahasiswa Baru (PMB)
- Integrasi ke layanan pihak ketiga di luar konteks akademik murni
- Sistem Mobile App Native

## 4. Asumsi Utama & Konfirmasi (Decisions Needed)
- Kredensial Login: Email (Unik) & Password.
- Role: Satu pengguna dapat memiliki banyak role (Multi-role support).
- Skala Penilaian: Default menggunakan A (4) - E (0).
- KRS Approval: KRS disetujui otomatis jika memenuhi prasyarat (Untuk MVP awal, approval eksplisit oleh dosen pembimbing dapat dimatikan/dijadikan opsional).
- Retake Policy: Menggunakan nilai tertinggi (Best Grade) untuk IPK.
- Batasan SKS: Min 0, Max 24 SKS per semester.
- Kehadiran: Pencatatan manual oleh dosen (tidak ada hardware/sensor luar).

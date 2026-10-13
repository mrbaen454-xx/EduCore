# Dokumentasi REST API: Presensi dan Kehadiran Mahasiswa (Fase 10)
## EduCore — Academic Management System

Prefix Base URL: `/api/v1`  
Header Wajib: `Authorization: Bearer <token>` dan `Content-Type: application/json`

---

## 1. Daftar Endpoint Sesi Perkuliahan

### 1.1. Membuat Sesi Perkuliahan
- **Method**: `POST`
- **Path**: `/api/v1/classes/:id/sessions`
- **Role Izin**: `dosen` (pengampu kelas), `admin`, `akademik`
- **Request Body**:
  ```json
  {
    "session_number": 1,
    "date": "2026-10-15",
    "start_time": "08:00",
    "end_time": "09:40",
    "topic": "Pengantar Algoritma & Kompleksitas",
    "remarks": "Membawa laptop masing-masing",
    "status": "OPEN"
  }
  ```
- **Responses**:
  - `201 Created`: Sesi berhasil dibuat.
  - `400 Bad Request`: Format input atau validasi waktu salah (`end_time <= start_time`).
  - `403 Forbidden`: Dosen bukan pengampu kelas bersangkutan.
  - `409 Conflict`: Nomor pertemuan sudah ada pada kelas ini.

---

### 1.2. Melihat Daftar Sesi Perkuliahan Kelas
- **Method**: `GET`
- **Path**: `/api/v1/classes/:id/sessions?page=1&limit=20`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`, `mahasiswa` (terdaftar di kelas)
- **Responses**:
  - `200 OK`: Mengembalikan daftar sesi dengan pagination metadata.
  - `403 Forbidden`: Mahasiswa tidak terdaftar pada kelas atau dosen bukan pengampu.

---

### 1.3. Melihat Detail Sesi Perkuliahan
- **Method**: `GET`
- **Path**: `/api/v1/sessions/:id`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`, `mahasiswa` (terdaftar di kelas)
- **Responses**:
  - `200 OK`: Mengembalikan data lengkap sesi, metadata dosen, dan statistik kehadiran saat ini.
  - `404 Not Found`: Sesi tidak ditemukan.

---

### 1.4. Memperbarui Metadata Sesi Perkuliahan
- **Method**: `PATCH`
- **Path**: `/api/v1/sessions/:id`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Request Body** (opsional per field):
  ```json
  {
    "topic": "Pengantar Algoritma Lanjutan",
    "remarks": "Ruang pindah ke Lab Komputer 2"
  }
  ```
- **Responses**:
  - `200 OK`: Sesi berhasil diperbarui.
  - `409 Conflict`: Sesi telah dibatalkan (`CANCELLED`).

---

### 1.5. Menutup Sesi Perkuliahan
- **Method**: `POST`
- **Path**: `/api/v1/sessions/:id/close`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Responses**:
  - `200 OK`: Sesi berhasil ditutup (`status = 'COMPLETED'`).
  - `409 Conflict`: Sesi sudah berstatus `COMPLETED` atau `CANCELLED`.

---

## 2. Daftar Endpoint Presensi Dosen

### 2.1. Melihat Daftar Peserta Sesi (Roster)
- **Method**: `GET`
- **Path**: `/api/v1/sessions/:id/students`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Responses**:
  - `200 OK`: Mengembalikan daftar mahasiswa terdaftar aktif beserta status presensi saat ini (jika sudah dicatat).

---

### 2.2. Melihat Presensi Satu Sesi
- **Method**: `GET`
- **Path**: `/api/v1/sessions/:id/attendance`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Responses**:
  - `200 OK`: Mengembalikan daftar presensi yang sudah tercatat dan ringkasan statistik kehadiran (`AttendanceSummary`).

---

### 2.3. Mencatat Presensi Batch
- **Method**: `POST`
- **Path**: `/api/v1/sessions/:id/attendance`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Request Body**:
  ```json
  {
    "items": [
      {
        "student_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "status": "PRESENT",
        "remarks": "Hadir tepat waktu"
      },
      {
        "student_id": "f1e2d3c4-b5a6-7890-abcd-ef0987654321",
        "status": "LATE",
        "remarks": "Terlambat 15 menit"
      }
    ]
  }
  ```
- **Responses**:
  - `200 OK`: Seluruh presensi berhasil dicatat secara atomik.
  - `400 Bad Request`: Status kehadiran tidak valid atau items kosong.
  - `409 Conflict`: Terdapat mahasiswa yang bukan peserta aktif kelas, duplikasi ID dalam payload, atau sesi telah dibatalkan.

---

### 2.4. Mengoreksi Catatan Presensi
- **Method**: `PATCH`
- **Path**: `/api/v1/sessions/:id/attendance/:attendanceId`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Request Body**:
  ```json
  {
    "status": "SICK",
    "remarks": "Surat keterangan sakit diserahkan"
  }
  ```
- **Responses**:
  - `200 OK`: Presensi berhasil diperbarui.
  - `404 Not Found`: Catatan presensi tidak ditemukan.

---

### 2.5. Melihat Rekapitulasi Kehadiran Kelas
- **Method**: `GET`
- **Path**: `/api/v1/classes/:id/attendance/summary`
- **Role Izin**: `dosen` (pengampu), `admin`, `akademik`
- **Responses**:
  - `200 OK`: Mengembalikan rekapitulasi kumulatif per mahasiswa pada kelas tersebut (total sesi, rincian per status, persentase kehadiran).

---

## 3. Daftar Endpoint Presensi Mahasiswa

### 3.1. Melihat Riwayat Presensi Pribadi
- **Method**: `GET`
- **Path**: `/api/v1/student/attendance?page=1&limit=20`
- **Role Izin**: `mahasiswa`
- **Responses**:
  - `200 OK`: Mengembalikan daftar seluruh riwayat presensi mahasiswa login dan ringkasan kumulatif.

---

### 3.2. Melihat Riwayat Presensi Berdasarkan Kelas
- **Method**: `GET`
- **Path**: `/api/v1/student/attendance/classes/:id?page=1&limit=20`
- **Role Izin**: `mahasiswa`
- **Responses**:
  - `200 OK`: Mengembalikan riwayat presensi mahasiswa pada kelas tertentu.
  - `403 Forbidden`: Mahasiswa tidak terdaftar pada kelas tersebut.

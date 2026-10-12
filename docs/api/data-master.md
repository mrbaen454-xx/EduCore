# Dokumentasi REST API: Data Master Akademik (Fase 06)

Dokumen ini merangkum seluruh endpoint REST API untuk modul Data Master Akademik EduCore: **Fakultas**, **Program Studi**, **Dosen**, dan **Mahasiswa**.

Base URL: `/api/v1`

---

## 1. Matriks Otorisasi dan Hak Akses

| Endpoint | Method | Path | Akses Role | Keterangan |
|---|---|---|---|---|
| Create Faculty | POST | `/api/v1/faculties` | `admin`, `akademik` | Menambah data fakultas baru |
| List Faculties | GET | `/api/v1/faculties` | Authenticated (`admin`, `akademik`, `dosen`, `mahasiswa`) | Menampilkan daftar fakultas terpaginasi |
| Detail Faculty | GET | `/api/v1/faculties/:id` | Authenticated | Detail fakultas dan jumlah prodi |
| Update Faculty | PATCH | `/api/v1/faculties/:id` | `admin`, `akademik` | Memperbarui nama/deskripsi |
| Update Faculty Status | PATCH | `/api/v1/faculties/:id/status` | `admin`, `akademik` | Mengaktifkan/menonaktifkan fakultas |
| Create Study Program | POST | `/api/v1/study-programs` | `admin`, `akademik` | Menambah program studi di bawah fakultas |
| List Study Programs | GET | `/api/v1/study-programs` | Authenticated | Daftar prodi dengan filter |
| Detail Study Program | GET | `/api/v1/study-programs/:id` | Authenticated | Detail prodi beserta data fakultas induk |
| Update Study Program | PATCH | `/api/v1/study-programs/:id` | `admin`, `akademik` | Memperbarui prodi |
| Update Study Program Status | PATCH | `/api/v1/study-programs/:id/status` | `admin`, `akademik` | Mengaktifkan/menonaktifkan prodi |
| Create Lecturer | POST | `/api/v1/lecturers` | `admin`, `akademik` | Mendaftarkan profil dosen |
| List Lecturers | GET | `/api/v1/lecturers` | Authenticated | Daftar dosen dengan filter prodi/status |
| Detail Lecturer | GET | `/api/v1/lecturers/:id` | Authenticated | Detail profil dosen lengkap |
| Update Lecturer | PATCH | `/api/v1/lecturers/:id` | `admin`, `akademik` | Memperbarui biodata dosen |
| Update Lecturer Status | PATCH | `/api/v1/lecturers/:id/status` | `admin`, `akademik` | Mengubah status: `ACTIVE`, `INACTIVE`, `LEAVE` |
| Create Student | POST | `/api/v1/students` | `admin`, `akademik` | Mendaftarkan profil mahasiswa |
| List Students | GET | `/api/v1/students` | Authenticated | Daftar mahasiswa dengan filter angkatan/status |
| Detail Student | GET | `/api/v1/students/:id` | Authenticated | Detail profil mahasiswa lengkap |
| Update Student | PATCH | `/api/v1/students/:id` | `admin`, `akademik` | Memperbarui biodata mahasiswa |
| Update Student Status | PATCH | `/api/v1/students/:id/status` | `admin`, `akademik` | Mengubah status mahasiswa |

---

## 2. Format Respons Baku

### Format Sukses (`200 OK` / `201 Created`)
```json
{
  "success": true,
  "message": "Fakultas berhasil dibuat",
  "data": {
    "id": "1b08f4c2-0051-4ae9-b221-caea66c72d67",
    "code": "FT",
    "name": "Fakultas Teknik",
    "description": "Fakultas teknik dan rekayasa",
    "is_active": true,
    "created_at": "2026-10-12T22:30:00Z",
    "updated_at": "2026-10-12T22:30:00Z"
  },
  "request_id": "4b684cb3-000c-4fa2-bf58-963d8650ad88"
}
```

### Format Error Baku (`400`, `401`, `403`, `404`, `409`)
```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Kode fakultas 'FT' sudah digunakan"
  },
  "request_id": "8fce9328-360d-4bb8-8684-26937e2ab692"
}
```

---

## 3. Rincian Endpoint & Query Parameter

### 3.1. Fakultas (`/api/v1/faculties`)

#### `POST /api/v1/faculties`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "code": "FT",
    "name": "Fakultas Teknik",
    "description": "Fakultas teknik dan rekayasa"
  }
  ```
- **Response**: `201 Created`

#### `GET /api/v1/faculties`
- **Query Params**:
  - `search` (string): Mencari kecocokan pada kode atau nama fakultas
  - `is_active` (bool): Memfilter status aktif (`true` / `false`)
  - `page` (int, default: 1)
  - `limit` (int, default: 20, max: 100)
  - `sort_by` (string, whitelist: `name`, `code`, `created_at`, default: `created_at`)
  - `order` (string: `asc` / `desc`, default: `desc`)

---

### 3.2. Program Studi (`/api/v1/study-programs`)

#### `POST /api/v1/study-programs`
- **Request Body**:
  ```json
  {
    "faculty_id": "1b08f4c2-0051-4ae9-b221-caea66c72d67",
    "code": "TIF",
    "name": "Teknik Informatika",
    "level": "S1",
    "description": "Program studi Sarjana Teknik Informatika"
  }
  ```
- **Response**: `201 Created`

#### `GET /api/v1/study-programs`
- **Query Params**:
  - `search` (string): Nama atau kode program studi
  - `faculty_id` (UUID): Filter program studi berdasarkan fakultas induk
  - `level` (string: `D3`, `D4`, `S1`, `S2`, `S3`, `Profesi`)
  - `is_active` (bool)
  - `sort_by` (whitelist: `name`, `code`, `level`, `created_at`)

---

### 3.3. Dosen (`/api/v1/lecturers`)

#### `POST /api/v1/lecturers`
- **Request Body**:
  ```json
  {
    "study_program_id": "52f6b3cf-c205-4f3b-ba2d-4d22beea04f5",
    "nidn": "0012345678",
    "name": "Joko Susilo",
    "title_prefix": "Prof. Dr. Ir.",
    "title_suffix": "M.T., IPU",
    "email": "joko.susilo@educore.ac.id",
    "phone": "081234567890"
  }
  ```
- **Response**: `201 Created`

#### `PATCH /api/v1/lecturers/:id/status`
- **Request Body**:
  ```json
  {
    "status": "LEAVE"
  }
  ```
  Nilai status yang valid: `ACTIVE`, `INACTIVE`, `LEAVE`.

---

### 3.4. Mahasiswa (`/api/v1/students`)

#### `POST /api/v1/students`
- **Request Body**:
  ```json
  {
    "study_program_id": "52f6b3cf-c205-4f3b-ba2d-4d22beea04f5",
    "nim": "2026101001",
    "name": "Ananda Putri",
    "email": "ananda@student.educore.ac.id",
    "phone": "081987654321",
    "gender": "FEMALE",
    "birth_date": "2004-08-20",
    "admission_year": 2026
  }
  ```
- **Response**: `201 Created`

#### `PATCH /api/v1/students/:id/status`
- **Request Body**:
  ```json
  {
    "status": "SUSPENDED"
  }
  ```
  Nilai status yang valid: `ACTIVE`, `INACTIVE`, `PROBATION`, `GRADUATED`, `DROPOUT`, `SUSPENDED`.

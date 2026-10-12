# Dokumentasi API: Modul Mata Kuliah dan Kurikulum (Fase 07)

Base URL: `/api/v1`  
Autentikasi: Bearer JWT (`Authorization: Bearer <token>`)

---

## 1. Endpoints Mata Kuliah (`/api/v1/courses`)

### 1.1. `POST /api/v1/courses`
Membuat mata kuliah baru di katalog akademik.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "code": "TIF101",
  "name": "Algoritma dan Pemrograman",
  "description": "Dasar-dasar logika pemrograman dan struktur kontrol",
  "credits": 3
}
```
- **Response 201 Created**:
```json
{
  "success": true,
  "message": "Mata kuliah berhasil dibuat",
  "data": {
    "id": "c1f72744-8cb3-4876-8809-5c4d0a3ad871",
    "code": "TIF101",
    "name": "Algoritma dan Pemrograman",
    "description": "Dasar-dasar logika pemrograman dan struktur kontrol",
    "credits": 3,
    "is_active": true,
    "created_at": "2026-10-12T16:00:00Z",
    "updated_at": "2026-10-12T16:00:00Z"
  }
}
```

### 1.2. `GET /api/v1/courses`
Mendapatkan daftar mata kuliah dengan pagination, filtering, dan sorting.
- **Permission**: `admin`, `akademik`, `dosen`, `mahasiswa`
- **Query Parameters**:
  - `page` (int, default: 1)
  - `limit` (int, default: 10, max: 100)
  - `search` (string, filter kode atau nama)
  - `is_active` (boolean)
  - `credits` (int)
  - `sort` (string, contoh: `code`, `-credits`, `created_at`)

### 1.3. `GET /api/v1/courses/{id}`
Mendapatkan detail mata kuliah beserta daftar mata kuliah prasyaratnya.
- **Permission**: `admin`, `akademik`, `dosen`, `mahasiswa`

### 1.4. `PATCH /api/v1/courses/{id}`
Memperbarui metadata mata kuliah (nama, deskripsi, SKS, status aktif).
- **Permission**: `admin`, `akademik`

### 1.5. `POST /api/v1/courses/{id}/prerequisites`
Menambahkan prasyarat ke mata kuliah.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "prerequisite_course_id": "8b0821ec-ec51-40be-bd3d-0453c07624c4"
}
```

### 1.6. `DELETE /api/v1/courses/{id}/prerequisites/{prerequisiteId}`
Menghapus relasi prasyarat antarmata kuliah.
- **Permission**: `admin`, `akademik`

---

## 2. Endpoints Kurikulum (`/api/v1/curricula`)

### 2.1. `POST /api/v1/curricula`
Membuat master kurikulum baru untuk program studi aktif.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "study_program_id": "5863c09b-640b-466c-a496-e17926ce4054",
  "code": "KUR-IF-2024",
  "name": "Kurikulum OBE Informatika 2024",
  "year": 2024,
  "description": "Kurikulum berbasis capaian pembelajaran lulusan (OBE)"
}
```

### 2.2. `GET /api/v1/curricula`
Mendapatkan daftar kurikulum dengan pagination dan filtering.
- **Permission**: `admin`, `akademik`, `dosen`, `mahasiswa`
- **Query Parameters**:
  - `study_program_id` (uuid)
  - `status` (`DRAFT`, `ACTIVE`, `INACTIVE`, `ARCHIVED`)
  - `year` (int)
  - `search` (string)

### 2.3. `GET /api/v1/curricula/{id}`
Mendapatkan detail kurikulum, program studi, agregasi statistik SKS (`total_courses`, `total_credits`, `required_credits`, `elective_credits`), serta daftar mata kuliah di dalamnya.
- **Permission**: `admin`, `akademik`, `dosen`, `mahasiswa`

### 2.4. `PATCH /api/v1/curricula/{id}`
Memperbarui nama, tahun, atau deskripsi kurikulum.
- **Permission**: `admin`, `akademik`

### 2.5. `PATCH /api/v1/curricula/{id}/status`
Memperbarui lifecycle status kurikulum (`DRAFT` -> `ACTIVE` -> `INACTIVE` -> `ARCHIVED`).
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "status": "ACTIVE"
}
```

---

## 3. Endpoints Mata Kuliah dalam Kurikulum (`/api/v1/curricula/{id}/courses`)

### 3.1. `POST /api/v1/curricula/{id}/courses`
Menambahkan mata kuliah ke dalam susunan kurikulum.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "course_id": "c1f72744-8cb3-4876-8809-5c4d0a3ad871",
  "recommended_semester": 1,
  "category": "REQUIRED"
}
```

### 3.2. `GET /api/v1/curricula/{id}/courses`
Mendapatkan daftar mata kuliah dalam kurikulum terurut per semester rekomendasi.
- **Permission**: `admin`, `akademik`, `dosen`, `mahasiswa`
- **Query Parameters**:
  - `semester` (int, 1-14)
  - `category` (`REQUIRED`, `ELECTIVE`)

### 3.3. `PATCH /api/v1/curricula/{id}/courses/{courseId}`
Memperbarui semester rekomendasi atau kategori mata kuliah pada kurikulum.
- **Permission**: `admin`, `akademik`

### 3.4. `DELETE /api/v1/curricula/{id}/courses/{courseId}`
Menghapus mata kuliah dari kurikulum jika aman (belum memiliki referensi kelas).
- **Permission**: `admin`, `akademik`

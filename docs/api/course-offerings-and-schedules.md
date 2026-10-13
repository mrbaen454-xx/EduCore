# Dokumentasi API: Penawaran Kelas dan Jadwal Perkuliahan (Fase 08)

Base URL: `/api/v1`  
Header Wajib: `Authorization: Bearer <access_token>`

---

## 1. Endpoints Periode Akademik (`/api/v1/academic-periods`)

### 1.1. `POST /api/v1/academic-periods`
Membuat periode akademik baru.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "code": "20241",
  "name": "Semester Ganjil 2024/2025",
  "semester_type": "ODD",
  "start_date": "2024-09-01",
  "end_date": "2025-01-31",
  "is_active": true
}
```
- **Response 201 Created**:
```json
{
  "success": true,
  "message": "Periode akademik berhasil dibuat",
  "data": {
    "id": "e6a21d55-6b58-45be-9a3d-4c577033ea3e",
    "code": "20241",
    "name": "Semester Ganjil 2024/2025",
    "semester_type": "ODD",
    "start_date": "2024-09-01",
    "end_date": "2025-01-31",
    "is_active": true,
    "created_at": "2026-10-13T00:00:00Z",
    "updated_at": "2026-10-13T00:00:00Z"
  }
}
```

### 1.2. `GET /api/v1/academic-periods`
Menampilkan daftar periode akademik dengan pagination dan filter.
- **Permission**: Seluruh pengguna terotentikasi
- **Query Parameters**:
  - `page` (int, default: 1)
  - `limit` (int, default: 20)
  - `search` (string)
  - `semester_type` (`ODD`, `EVEN`, `SHORT`)
  - `is_active` (boolean)
  - `sort` (string, contoh: `code`, `-start_date`)

### 1.3. `GET /api/v1/academic-periods/{id}`
Mendapatkan detail periode akademik berdasarkan ID.
- **Permission**: Seluruh pengguna terotentikasi

### 1.4. `PATCH /api/v1/academic-periods/{id}`
Memperbarui metadata periode akademik.
- **Permission**: `admin`, `akademik`

---

## 2. Endpoints Penawaran Mata Kuliah (`/api/v1/course-offerings`)

### 2.1. `POST /api/v1/course-offerings`
Membuat penawaran mata kuliah baru pada periode dan prodi tertentu.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "academic_period_id": "e6a21d55-6b58-45be-9a3d-4c577033ea3e",
  "course_id": "c1f72744-8cb3-4876-8809-5c4d0a3ad871",
  "study_program_id": "5863c09b-640b-466c-a496-e17926ce4054",
  "curriculum_id": "f2e96ca7-f273-4ea2-9cb5-397a6e147134"
}
```
- **Response 201 Created**:
```json
{
  "success": true,
  "message": "Penawaran mata kuliah berhasil dibuat",
  "data": {
    "id": "7b822d10-8f92-4fa0-82a8-1e4277b1028f",
    "academic_period_id": "e6a21d55-6b58-45be-9a3d-4c577033ea3e",
    "academic_period_code": "20241",
    "course_id": "c1f72744-8cb3-4876-8809-5c4d0a3ad871",
    "course_code": "TIF201",
    "course_name": "Pemrograman Berorientasi Objek",
    "course_credits": 3,
    "study_program_id": "5863c09b-640b-466c-a496-e17926ce4054",
    "study_program_name": "Teknik Informatika",
    "status": "PLANNED",
    "total_classes": 0
  }
}
```

### 2.2. `GET /api/v1/course-offerings`
Menampilkan daftar penawaran mata kuliah.
- **Permission**: Seluruh pengguna terotentikasi
- **Query Parameters**:
  - `academic_period_id` (uuid)
  - `study_program_id` (uuid)
  - `course_id` (uuid)
  - `status` (`PLANNED`, `OPEN`, `CLOSED`, `CANCELLED`)
  - `search` (string)
  - `page`, `limit`, `sort`

### 2.3. `GET /api/v1/course-offerings/{id}`
Mendapatkan detail penawaran mata kuliah.
- **Permission**: Seluruh pengguna terotentikasi

### 2.4. `PATCH /api/v1/course-offerings/{id}/status`
Memperbarui lifecycle status penawaran mata kuliah.
- **Permission**: `admin`, `akademik`

---

## 3. Endpoints Kelas Perkuliahan (`/api/v1/classes`)

### 3.1. `POST /api/v1/classes`
Membuka kelas perkuliahan baru di bawah penawaran tertentu.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "course_offering_id": "7b822d10-8f92-4fa0-82a8-1e4277b1028f",
  "code": "A",
  "name": "Kelas A - Reguler Pagi",
  "capacity": 40,
  "remarks": "Prioritas mahasiswa angkatan 2024"
}
```

### 3.2. `GET /api/v1/classes`
Menampilkan daftar kelas perkuliahan.
- **Permission**: Seluruh pengguna terotentikasi

### 3.3. `GET /api/v1/classes/{id}`
Menampilkan detail kelas, daftar dosen pengajar, dan jadwal pelaksanaan.
- **Permission**: Seluruh pengguna terotentikasi

### 3.4. `PATCH /api/v1/classes/{id}`
Memperbarui nama, kapasitas, atau catatan kelas.
- **Permission**: `admin`, `akademik`

### 3.5. `PATCH /api/v1/classes/{id}/status`
Memperbarui lifecycle status kelas (`DRAFT`, `OPEN`, `IN_PROGRESS`, `CLOSED`, `CANCELLED`).
- **Permission**: `admin`, `akademik`

---

## 4. Endpoints Penugasan Dosen (`/api/v1/classes/{id}/lecturers`)

### 4.1. `POST /api/v1/classes/{id}/lecturers`
Menugaskan dosen pada kelas perkuliahan.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "lecturer_id": "4b62e49c-f86a-49a7-80f4-5f16d729e120",
  "role": "COORDINATOR"
}
```

### 4.2. `GET /api/v1/classes/{id}/lecturers`
Menampilkan daftar dosen pengajar pada kelas.
- **Permission**: Seluruh pengguna terotentikasi

### 4.3. `DELETE /api/v1/classes/{id}/lecturers/{lecturerId}`
Menghapus penugasan dosen dari kelas.
- **Permission**: `admin`, `akademik`

---

## 5. Endpoints Jadwal Perkuliahan (`/api/v1/classes/{id}/schedules`)

### 5.1. `POST /api/v1/classes/{id}/schedules`
Membuat jadwal waktu perkuliahan. Dilengkapi validasi bentrok jadwal internal kelas, bentrok jadwal dosen, dan bentrok jadwal ruangan.
- **Permission**: `admin`, `akademik`
- **Request Body**:
```json
{
  "day_of_week": 1,
  "start_time": "08:00",
  "end_time": "10:00",
  "room": "LAB-1",
  "delivery_mode": "OFFLINE"
}
```

### 5.2. `GET /api/v1/classes/{id}/schedules`
Menampilkan daftar jadwal perkuliahan kelas.
- **Permission**: Seluruh pengguna terotentikasi

### 5.3. `PATCH /api/v1/classes/{id}/schedules/{scheduleId}`
Memperbarui jadwal perkuliahan dengan re-validasi bentrok.
- **Permission**: `admin`, `akademik`

### 5.4. `DELETE /api/v1/classes/{id}/schedules/{scheduleId}`
Menghapus jadwal perkuliahan.
- **Permission**: `admin`, `akademik`

# Katalog Endpoint REST API: KRS dan Registrasi Mahasiswa (Fase 09)

## 1. Ikhtisar Endpoint

Seluruh endpoint bernaung di bawah prefix `/api/v1` dan dilindungi oleh otentikasi JWT Bearer Token serta otorisasi berbasis Role (RBAC).

---

## 2. Matriks Akses dan Endpoint

| Method | Path | Akses Role | Deskripsi |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/student/course-offerings` | `mahasiswa` | Melihat penawaran kelas berstatus OPEN untuk KRS |
| `GET` | `/api/v1/student/study-plans` | `mahasiswa` | Melihat riwayat KRS milik sendiri |
| `POST` | `/api/v1/student/study-plans` | `mahasiswa` | Membuat atau mengambil KRS untuk periode aktif |
| `GET` | `/api/v1/student/study-plans/:id` | `mahasiswa` | Melihat detail KRS sendiri dan daftar kelas yang diambil |
| `POST` | `/api/v1/student/study-plans/:id/enrollments` | `mahasiswa` | Mendaftarkan kelas perkuliahan ke KRS sendiri |
| `DELETE` | `/api/v1/student/study-plans/:id/enrollments/:enrollmentId` | `mahasiswa` | Membatalkan pendaftaran kelas dari KRS |
| `GET` | `/api/v1/study-plans` | `admin`, `akademik` | Melihat daftar seluruh KRS (filter student, periode, status) |
| `POST` | `/api/v1/study-plans` | `admin`, `akademik` | Membuat KRS atas nama mahasiswa tertentu |
| `GET` | `/api/v1/study-plans/:id` | `admin`, `akademik` | Melihat detail lengkap KRS mahasiswa mana pun |
| `POST` | `/api/v1/study-plans/:id/enrollments` | `admin`, `akademik` | Mendaftarkan kelas ke KRS atas nama mahasiswa |
| `DELETE` | `/api/v1/study-plans/:id/enrollments/:enrollmentId` | `admin`, `akademik` | Membatalkan kelas dari KRS atas nama mahasiswa |

---

## 3. Contoh Request & Response

### 3.1. Membuat KRS Mahasiswa (`POST /api/v1/student/study-plans`)
**Request:**
```http
POST /api/v1/student/study-plans HTTP/1.1
Authorization: Bearer <STUDENT_JWT_TOKEN>
Content-Type: application/json

{
  "academic_period_id": "0d6110f0-8c90-4e31-bd62-1b6c00be05b7"
}
```

**Response (`201 Created`):**
```json
{
  "success": true,
  "message": "KRS berhasil dibuat",
  "data": {
    "id": "18cfbc9f-b778-4eb7-a722-1d54f5c22501",
    "student_id": "84a3c3e2-6325-455b-bfb3-b186b51eb923",
    "student_nim": "24001",
    "student_name": "Budi Santoso",
    "study_program_id": "f5bc0302-39c0-43be-ae65-c38d3fe7ad25",
    "study_program_name": "Informatika",
    "academic_period_id": "0d6110f0-8c90-4e31-bd62-1b6c00be05b7",
    "academic_period_code": "20241",
    "academic_period_name": "Ganjil 2024/2025",
    "total_credits": 0,
    "status": "ACTIVE",
    "created_at": "2026-10-14T02:00:00Z",
    "updated_at": "2026-10-14T02:00:00Z",
    "enrollments": []
  }
}
```

---

### 3.2. Mendaftarkan Kelas ke KRS (`POST /api/v1/student/study-plans/:id/enrollments`)
**Request:**
```http
POST /api/v1/student/study-plans/18cfbc9f-b778-4eb7-a722-1d54f5c22501/enrollments HTTP/1.1
Authorization: Bearer <STUDENT_JWT_TOKEN>
Content-Type: application/json

{
  "class_id": "22bf359e-141a-466a-ba92-d6ffbbdf3881"
}
```

**Response (`201 Created`):**
```json
{
  "success": true,
  "message": "Kelas berhasil ditambahkan ke KRS",
  "data": {
    "id": "31fe1cf1-417c-47bc-ae51-d4ba11fe3729",
    "study_plan_id": "18cfbc9f-b778-4eb7-a722-1d54f5c22501",
    "student_id": "84a3c3e2-6325-455b-bfb3-b186b51eb923",
    "class_id": "22bf359e-141a-466a-ba92-d6ffbbdf3881",
    "class_code": "A",
    "class_name": "Kelas Algoritma A",
    "course_id": "aa123b34-8c82-411a-821b-cc34f0e737bc",
    "course_code": "IF101",
    "course_name": "Algoritma & Pemrograman",
    "course_credits": 4,
    "status": "ENROLLED",
    "enrolled_at": "2026-10-14T02:05:00Z",
    "dropped_at": null,
    "schedules": [
      {
        "id": "84cc11ff-124b-46e2-9b93-84b23fe492aa",
        "class_id": "22bf359e-141a-466a-ba92-d6ffbbdf3881",
        "day_of_week": 1,
        "start_time": "08:00:00",
        "end_time": "10:30:00",
        "room": "Lab Komputer 1",
        "delivery_mode": "OFFLINE"
      }
    ]
  }
}
```

---

### 3.3. Membatalkan Kelas (`DELETE /api/v1/student/study-plans/:id/enrollments/:enrollmentId`)
**Request:**
```http
DELETE /api/v1/student/study-plans/18cfbc9f-b778-4eb7-a722-1d54f5c22501/enrollments/31fe1cf1-417c-47bc-ae51-d4ba11fe3729 HTTP/1.1
Authorization: Bearer <STUDENT_JWT_TOKEN>
```

**Response (`200 OK`):**
```json
{
  "success": true,
  "message": "Registrasi kelas berhasil dibatalkan",
  "data": {
    "study_plan_id": "18cfbc9f-b778-4eb7-a722-1d54f5c22501",
    "enrollment_id": "31fe1cf1-417c-47bc-ae51-d4ba11fe3729",
    "status": "DROPPED"
  }
}
```

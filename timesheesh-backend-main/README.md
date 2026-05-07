# Timesheesh Backend

Backend API untuk aplikasi Timesheesh dengan sistem autentikasi berbasis role.

## Fitur

- ✅ Autentikasi dengan JWT
- ✅ Role-based access control (Admin, Project Manager, Employee, Management)
- ✅ Employee types (Fulltime, Parttime, Freelance)
- ✅ PostgreSQL database

## Setup

### Prerequisites

- Go 1.25.5 atau lebih baru
- Docker & Docker Compose (untuk database PostgreSQL)

### Installation

1. Clone repository
```bash
git clone <repository-url>
cd timesheesh-backend
```

2. Install dependencies
```bash
go mod download
```

3. Setup environment variables
```bash
# Copy file contoh
cp env.example .env

# Edit .env sesuai kebutuhan (pastikan DB_USER, DB_PASSWORD, dan DB_NAME sudah di-set)
```

4. Jalankan database dengan Docker Compose
```bash
docker-compose up -d
```

5. Jalankan aplikasi
```bash
go run main.go
```

Aplikasi akan berjalan di `http://localhost:8080`

## Database

### PostgreSQL dengan Docker Compose

Database PostgreSQL akan otomatis berjalan dengan docker-compose. Pastikan file `.env` sudah dibuat dengan kredensial database:

```bash
# Start database
docker-compose up -d

# Stop database
docker-compose down

# Stop dan hapus data
docker-compose down -v
```

Kredensial database diatur di file `.env`:
- `DB_HOST`: Host database (default: localhost)
- `DB_PORT`: Port database (default: 5432)
- `DB_USER`: Username database
- `DB_PASSWORD`: Password database
- `DB_NAME`: Nama database
- `DB_SSLMODE`: SSL mode (default: disable)

## API Endpoints

### Public Endpoints

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

### Protected Endpoints

Semua endpoint di bawah memerlukan header:
```
Authorization: Bearer <token>
```

#### Get Profile
```http
GET /api/profile
```

#### Admin Dashboard
```http
GET /api/admin/dashboard
```

#### Project Manager Dashboard
```http
GET /api/project-manager/dashboard
```

#### Employee Dashboard
```http
GET /api/employee/dashboard
```

#### Management Dashboard
```http
GET /api/management/dashboard
```

### Admin Only Endpoints

Semua endpoint di bawah memerlukan:
- Header: `Authorization: Bearer <admin_token>`
- Role: `admin`

#### Register New User (Admin Only)
```http
POST /api/admin/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "full_name": "John Doe",
  "role": "employee",
  "employee_type": "fulltime"
}
```

#### Get All Users
```http
GET /api/admin/users?page=1&limit=10
```

#### Get User by ID
```http
GET /api/admin/users/:id
```

#### Create User
```http
POST /api/admin/users
Content-Type: application/json

{
  "email": "newuser@example.com",
  "password": "password123",
  "full_name": "New User",
  "role": "employee",
  "employee_type": "fulltime"
}
```

#### Update User
```http
PUT /api/admin/users/:id
Content-Type: application/json

{
  "full_name": "Updated Name",
  "role": "projectmanager"
}
```

#### Delete User
```http
DELETE /api/admin/users/:id
```

## Roles

- **admin**: Akses penuh ke semua fitur
- **projectmanager**: Mengelola proyek
- **employee**: Karyawan (memerlukan employee_type)
- **management**: Manajemen perusahaan

## Employee Types

Hanya berlaku untuk role `employee`:
- **fulltime**: Karyawan tetap
- **parttime**: Karyawan paruh waktu
- **freelance**: Karyawan freelance

## Environment Variables

File `.env` harus berisi:

- `DB_HOST`: Host PostgreSQL (default: localhost)
- `DB_PORT`: Port PostgreSQL (default: 5432)
- `DB_USER`: Username PostgreSQL (required)
- `DB_PASSWORD`: Password PostgreSQL (required)
- `DB_NAME`: Nama database PostgreSQL (required)
- `DB_SSLMODE`: SSL mode (default: disable)
- `JWT_SECRET`: Secret key untuk JWT token (wajib di production)
- `PORT`: Port server (default: 8080)

## Development

```bash
# Run dengan hot reload (jika menggunakan air)
air

# Build
go build

# Run tests
go test ./...
```


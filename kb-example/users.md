# Users & Authentication

## Tabel

### users
| Kolom | Tipe | Deskripsi |
|-------|------|-----------|
| id | INT (PK) | ID user |
| username | VARCHAR | Username login |
| email | VARCHAR | Email |
| department_id | INT (FK) | Relasi ke department |
| is_active | BOOLEAN | Status aktif |

### departments
| Kolom | Tipe | Deskripsi |
|-------|------|-----------|
| id | INT (PK) | ID department |
| name | VARCHAR | Nama department |

## Relasi
- `users.department_id` → `departments.id`

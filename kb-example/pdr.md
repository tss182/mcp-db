# PDR (Purchase Delivery Receipt)

## Tabel Utama

### prdr_list
Tabel utama untuk daftar PDR.

| Kolom | Tipe | Deskripsi |
|-------|------|-----------|
| id | INT (PK) | ID unik PDR |
| form_id | INT (FK) | Relasi ke tabel `form` |
| pdr_number | VARCHAR | Nomor PDR |
| supplier_id | INT | ID supplier |
| status | VARCHAR | Status: draft, approved, received |
| created_at | DATETIME | Tanggal dibuat |

### form
Tabel master form/dokumen.

| Kolom | Tipe | Deskripsi |
|-------|------|-----------|
| id | INT (PK) | ID unik form |
| form_name | VARCHAR | Nama form |
| form_type | VARCHAR | Tipe form |
| department | VARCHAR | Departemen pemilik |

## Relasi

- `prdr_list.form_id` → `form.id` (Many-to-One)
- Satu form bisa memiliki banyak PDR

## Contoh Query

```sql
-- Ambil semua PDR beserta nama form-nya
SELECT p.pdr_number, p.status, f.form_name, f.department
FROM prdr_list p
JOIN form f ON p.form_id = f.id
WHERE p.status = 'approved';
```

# MCP-DB

MCP Server untuk mengeksekusi query database (MySQL & PostgreSQL) melalui protokol Model Context Protocol (MCP). Server ini memungkinkan AI assistant mengakses database dengan kontrol hak akses yang granular.

## Fitur

- Mendukung **MySQL** dan **PostgreSQL**
- Kontrol hak akses granular (READ, CREATE, UPDATE, DELETE)
- **Knowledge Base** — tambahkan konteks skema, relasi tabel, dan bisnis logic agar AI lebih paham struktur database
- Komunikasi via STDIO (standar MCP)
- Output hasil query dalam format JSON
- Validasi jenis query otomatis berdasarkan konfigurasi
- DDL selalu ditolak (CREATE TABLE, ALTER, DROP, dll) untuk keamanan

## Instalasi

### Prasyarat

- Go 1.21 atau lebih baru
- MySQL atau PostgreSQL yang sudah berjalan

### Install

```bash
go install github.com/tss182/mcp-db@latest
```

Binary akan tersedia di `$GOPATH/bin/mcp-db`.

## Konfigurasi

Server dikonfigurasi melalui environment variable:

| Variable | Deskripsi | Wajib |
|----------|-----------|-------|
| `DB_DRIVER` | Driver database (`mysql` atau `postgres`) | Ya |
| `DB_DSN` | Data Source Name / Connection String | Ya |
| `ACTION_READ` | Izinkan query SELECT/SHOW/DESCRIBE/EXPLAIN (`true`/`false`) | Tidak (default: `false`) |
| `ACTION_CREATE` | Izinkan query INSERT (`true`/`false`) | Tidak (default: `false`) |
| `ACTION_UPDATE` | Izinkan query UPDATE (`true`/`false`) | Tidak (default: `false`) |
| `ACTION_DELETE` | Izinkan query DELETE (`true`/`false`) | Tidak (default: `false`) |
| `ACTION_DDL` | Izinkan query DDL — CREATE TABLE, ALTER, DROP, TRUNCATE, RENAME (`true`/`false`) | Tidak (default: `false`) |
| `KB_PATH` | Path ke direktori knowledge base (berisi file `.md`/`.txt`) | Tidak |

### Format DSN

**MySQL:**
```
user:password@tcp(host:port)/dbname
```

**PostgreSQL:**
```
host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable
```

## Contoh Pemakaian

### Konfigurasi MCP dengan MySQL

File `mcp.json`:

```json
{
  "mcpServers": {
    "mysql-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "mysql",
        "DB_DSN": "root:password@tcp(127.0.0.1:3306)/myapp",
        "ACTION_READ": "true",
        "ACTION_CREATE": "true",
        "ACTION_UPDATE": "true",
        "ACTION_DELETE": "false"
      }
    }
  }
}
```

### Konfigurasi MCP dengan PostgreSQL

File `mcp.json`:

```json
{
  "mcpServers": {
    "postgres-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "postgres",
        "DB_DSN": "host=localhost port=5432 user=postgres password=secret dbname=myapp sslmode=disable",
        "ACTION_READ": "true",
        "ACTION_CREATE": "true",
        "ACTION_UPDATE": "false",
        "ACTION_DELETE": "false"
      }
    }
  }
}
```

### Contoh: Read-Only Access (Aman untuk Produksi)

```json
{
  "mcpServers": {
    "prod-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "postgres",
        "DB_DSN": "host=prod-server port=5432 user=readonly password=secret dbname=app sslmode=require",
        "ACTION_READ": "true",
        "ACTION_CREATE": "false",
        "ACTION_UPDATE": "false",
        "ACTION_DELETE": "false"
      }
    }
  }
}
```

### Contoh: Full Access (Development)

```json
{
  "mcpServers": {
    "dev-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "mysql",
        "DB_DSN": "root:root@tcp(localhost:3306)/dev_db",
        "ACTION_READ": "true",
        "ACTION_CREATE": "true",
        "ACTION_UPDATE": "true",
        "ACTION_DELETE": "true"
      }
    }
  }
}
```

### Contoh: Dengan Knowledge Base

```json
{
  "mcpServers": {
    "my-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "mysql",
        "DB_DSN": "root:password@tcp(127.0.0.1:3306)/myapp",
        "ACTION_READ": "true",
        "ACTION_CREATE": "true",
        "ACTION_UPDATE": "true",
        "ACTION_DELETE": "false",
        "KB_PATH": "/path/to/knowledge-base"
      }
    }
  }
}
```

## Knowledge Base

Knowledge base memungkinkan kamu memberikan konteks tambahan ke AI tentang struktur database, relasi antar tabel, dan bisnis logic. Dengan ini AI akan lebih akurat dalam menulis query karena paham konteks spesifik database kamu.

### Setup

1. Buat direktori untuk menyimpan file knowledge base:

```bash
mkdir kb
```

2. Tambahkan file `.md` atau `.txt` yang menjelaskan skema database:

```bash
kb/
├── pdr.md
├── users.md
└── inventory.md
```

3. Set environment variable `KB_PATH` ke direktori tersebut.

### Format File Knowledge Base

Tulis penjelasan dalam format bebas (Markdown disarankan). Yang penting adalah menjelaskan:
- Struktur tabel (kolom, tipe data)
- Relasi antar tabel (foreign key, join)
- Konteks bisnis (apa arti data tersebut)
- Contoh query yang sering dipakai

### Contoh File: `kb/pdr.md`

```markdown
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

​```sql
-- Ambil semua PDR beserta nama form-nya
SELECT p.pdr_number, p.status, f.form_name, f.department
FROM prdr_list p
JOIN form f ON p.form_id = f.id
WHERE p.status = 'approved';
​```
```

### Cara Kerja

Ketika AI menggunakan tool `get_knowledge`, server akan:
1. Jika tanpa parameter `search` → menampilkan daftar semua file knowledge yang tersedia
2. Jika dengan parameter `search` → mencari keyword di nama file dan isi konten (case-insensitive)

AI akan otomatis memanfaatkan knowledge base sebelum menulis query, sehingga hasilnya lebih akurat dan sesuai konteks.

## Tool yang Tersedia

### `execute_db_query`

Mengeksekusi query SQL ke database yang terhubung.

**Parameter:**

| Nama | Tipe | Wajib | Deskripsi |
|------|------|-------|-----------|
| `query` | string | Ya | Query SQL yang akan dieksekusi |

**Contoh penggunaan oleh AI:**

```sql
SELECT * FROM users WHERE active = 1
```

```sql
INSERT INTO logs (message, created_at) VALUES ('test', NOW())
```

### `get_knowledge`

Mengambil informasi dari knowledge base untuk memahami struktur database sebelum menulis query.

**Parameter:**

| Nama | Tipe | Wajib | Deskripsi |
|------|------|-------|-----------|
| `search` | string | Tidak | Kata kunci pencarian. Jika kosong, menampilkan daftar topik yang tersedia |

**Contoh penggunaan oleh AI:**

- `get_knowledge({})` → Tampilkan semua topik
- `get_knowledge({"search": "pdr"})` → Cari info tentang PDR
- `get_knowledge({"search": "users"})` → Cari info tentang tabel users

## Kategorisasi Query

| Kategori | Perintah SQL | Environment Variable |
|----------|-------------|---------------------|
| READ | SELECT, SHOW, DESCRIBE, EXPLAIN | `ACTION_READ` |
| CREATE | INSERT | `ACTION_CREATE` |
| UPDATE | UPDATE | `ACTION_UPDATE` |
| DELETE | DELETE | `ACTION_DELETE` |
| DDL | CREATE TABLE, ALTER, DROP, TRUNCATE, RENAME | `ACTION_DDL` |

## DDL (Data Definition Language)

Query DDL seperti `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, `TRUNCATE TABLE`, dan `RENAME TABLE` dikontrol oleh `ACTION_DDL`.

- **Default: `false` (ditolak)** — untuk keamanan, DDL tidak diizinkan kecuali dinyatakan eksplisit.
- Set `ACTION_DDL=true` jika membutuhkan DDL, misalnya untuk development atau migrasi database.

**Peringatan:** Mengaktifkan DDL berarti AI bisa mengubah struktur database (membuat/menghapus tabel, mengubah kolom, dll). Gunakan dengan hati-hati dan hanya di lingkungan development.

### Contoh: Dengan DDL Aktif (Development/Migrasi)

```json
{
  "mcpServers": {
    "dev-db": {
      "command": "/path/to/mcp-db",
      "env": {
        "DB_DRIVER": "mysql",
        "DB_DSN": "root:root@tcp(localhost:3306)/dev_db",
        "ACTION_READ": "true",
        "ACTION_CREATE": "true",
        "ACTION_UPDATE": "true",
        "ACTION_DELETE": "true",
        "ACTION_DDL": "true"
      }
    }
  }
}
```

## Catatan Penting

- **Keamanan:** Selalu gunakan akses minimal yang diperlukan. Untuk production, aktifkan hanya `ACTION_READ`.
- **DDL Ditolak:** Semua query DDL (CREATE, ALTER, DROP, TRUNCATE, RENAME) selalu ditolak secara hardcode untuk keamanan.
- **PostgreSQL & LastInsertId:** PostgreSQL tidak mendukung `LastInsertId()`. Gunakan `RETURNING id` dalam query INSERT untuk mendapatkan ID terakhir.
- **Validasi Query:** Server memvalidasi jenis query berdasarkan kata pertama dari statement SQL. Query yang tidak dikenali akan ditolak.

## Lisensi

MIT License

Copyright (c) 2024 Triyana Suryapraja Sukmana

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

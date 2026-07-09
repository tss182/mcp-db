# MCP-DB

MCP Server untuk mengeksekusi query database (MySQL & PostgreSQL) melalui protokol Model Context Protocol (MCP). Server ini memungkinkan AI assistant mengakses database dengan kontrol hak akses yang granular.

## Fitur

- Mendukung **MySQL** dan **PostgreSQL**
- Kontrol hak akses granular (READ, CREATE, UPDATE, DELETE)
- Komunikasi via STDIO (standar MCP)
- Output hasil query dalam format JSON
- Validasi jenis query otomatis berdasarkan konfigurasi

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

## Tool yang Tersedia

### `execute_db_query`

Mengeksekusi query SQL ke database yang terhubung.

**Parameter:**

| Nama | Tipe | Wajib | Deskripsi |
|------|------|-------|-----------|
| `query` | string | Ya | Query SQL yang akan dieksekusi |

**Contoh penggunaan oleh AI:**

```
SELECT * FROM users WHERE active = 1
```

```
INSERT INTO logs (message, created_at) VALUES ('test', NOW())
```

## Kategorisasi Query

| Kategori | Perintah SQL | Environment Variable |
|----------|-------------|---------------------|
| READ | SELECT, SHOW, DESCRIBE, EXPLAIN | `ACTION_READ` |
| CREATE | INSERT | `ACTION_CREATE` |
| UPDATE | UPDATE | `ACTION_UPDATE` |
| DELETE | DELETE | `ACTION_DELETE` |
| DDL (selalu ditolak) | CREATE, ALTER, DROP, TRUNCATE, RENAME | — |

## DDL Restriction

Query DDL (Data Definition Language) seperti `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, `TRUNCATE TABLE`, dan `RENAME TABLE` **selalu ditolak secara hardcode**, terlepas dari konfigurasi environment variable. Ini untuk mencegah perubahan struktur database yang tidak disengaja melalui AI assistant.

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

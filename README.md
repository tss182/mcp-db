# MCP-DB

An MCP (Model Context Protocol) server that enables AI assistants to execute database queries on MySQL and PostgreSQL with granular access control and an optional knowledge base for schema context.

## Features

- Supports **MySQL** and **PostgreSQL**
- Granular access control (READ, CREATE, UPDATE, DELETE, DDL)
- **Knowledge Base** — provide schema definitions, table relationships, and business context so the AI understands your database structure
- Communicates via STDIO (standard MCP transport)
- Returns query results as JSON
- Automatic query type validation based on configuration
- DDL blocked by default for safety (configurable)

## Installation

### Prerequisites

- Go 1.21 or later
- A running MySQL or PostgreSQL instance

### Install

```bash
go install github.com/tss182/mcp-db@latest
```

The binary will be available at `$GOPATH/bin/mcp-db`.

## Configuration

The server is configured entirely through environment variables:

| Variable | Description | Required |
|----------|-------------|----------|
| `DB_DRIVER` | Database driver (`mysql` or `postgres`) | Yes |
| `DB_DSN` | Data Source Name / Connection String | Yes |
| `ACTION_READ` | Allow SELECT/SHOW/DESCRIBE/EXPLAIN (`true`/`false`) | No (default: `false`) |
| `ACTION_CREATE` | Allow INSERT (`true`/`false`) | No (default: `false`) |
| `ACTION_UPDATE` | Allow UPDATE (`true`/`false`) | No (default: `false`) |
| `ACTION_DELETE` | Allow DELETE (`true`/`false`) | No (default: `false`) |
| `ACTION_DDL` | Allow DDL — CREATE TABLE, ALTER, DROP, TRUNCATE, RENAME (`true`/`false`) | No (default: `false`) |
| `KB_PATH` | Path to knowledge base directory (contains `.md`/`.txt` files) | No |

### DSN Format

**MySQL:**
```
user:password@tcp(host:port)/dbname
```

**PostgreSQL:**
```
host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable
```

## Usage Examples

### MySQL Configuration

`mcp.json`:

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

### PostgreSQL Configuration

`mcp.json`:

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

### Read-Only Access (Safe for Production)

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

### Full Access with DDL (Development)

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

### With Knowledge Base

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

## Available Tools

### `execute_db_query`

Executes a SQL query against the connected database.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | The SQL query to execute |

**Examples:**

```sql
SELECT * FROM users WHERE active = 1
```

```sql
INSERT INTO logs (message, created_at) VALUES ('test', NOW())
```

### `get_knowledge`

Retrieves information from the knowledge base to understand the database structure before writing queries.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `search` | string | No | Search keyword. If empty, lists all available topics |

**Examples:**

- `get_knowledge({})` — List all topics
- `get_knowledge({"search": "pdr"})` — Search for PDR-related info
- `get_knowledge({"search": "users"})` — Search for user table info

## Query Categories

| Category | SQL Commands | Environment Variable |
|----------|-------------|---------------------|
| READ | SELECT, SHOW, DESCRIBE, EXPLAIN | `ACTION_READ` |
| CREATE | INSERT | `ACTION_CREATE` |
| UPDATE | UPDATE | `ACTION_UPDATE` |
| DELETE | DELETE | `ACTION_DELETE` |
| DDL | CREATE TABLE, ALTER, DROP, TRUNCATE, RENAME | `ACTION_DDL` |

## DDL (Data Definition Language)

DDL queries such as `CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, `TRUNCATE TABLE`, and `RENAME TABLE` are controlled by `ACTION_DDL`.

- **Default: `false` (blocked)** — DDL is not allowed unless explicitly enabled.
- Set `ACTION_DDL=true` if you need DDL, for example during development or database migrations.

**Warning:** Enabling DDL means the AI can modify your database structure (create/drop tables, alter columns, etc). Use with caution and only in development environments.

## Knowledge Base

The knowledge base allows you to provide additional context to the AI about your database schema, table relationships, and business logic. This makes the AI more accurate when writing queries because it understands the specific context of your database.

### Setup

1. Create a directory for your knowledge base files:

```bash
mkdir kb
```

2. Add `.md` or `.txt` files describing your database schema:

```
kb/
├── pdr.md
├── users.md
└── inventory.md
```

3. Set the `KB_PATH` environment variable to point to this directory.

### Knowledge Base File Format

Write your descriptions in any format (Markdown recommended). The important things to document are:

- Table structure (columns, data types)
- Table relationships (foreign keys, joins)
- Business context (what the data means)
- Common query examples

### Example File: `kb/pdr.md`

```markdown
# PDR (Purchase Delivery Receipt)

## Tables

### prdr_list
Main table for PDR records.

| Column | Type | Description |
|--------|------|-------------|
| id | INT (PK) | Unique PDR ID |
| form_id | INT (FK) | References `form` table |
| pdr_number | VARCHAR | PDR number |
| supplier_id | INT | Supplier ID |
| status | VARCHAR | Status: draft, approved, received |
| created_at | DATETIME | Creation date |

### form
Master table for forms/documents.

| Column | Type | Description |
|--------|------|-------------|
| id | INT (PK) | Unique form ID |
| form_name | VARCHAR | Form name |
| form_type | VARCHAR | Form type |
| department | VARCHAR | Owning department |

## Relationships

- `prdr_list.form_id` → `form.id` (Many-to-One)
- One form can have many PDRs

## Example Queries

```sql
-- Get all PDRs with their form names
SELECT p.pdr_number, p.status, f.form_name, f.department
FROM prdr_list p
JOIN form f ON p.form_id = f.id
WHERE p.status = 'approved';
```
```

### How It Works

When the AI uses the `get_knowledge` tool, the server will:
1. Without a `search` parameter — list all available knowledge base files
2. With a `search` parameter — search by keyword in file names and content (case-insensitive)

The AI will automatically leverage the knowledge base before writing queries, resulting in more accurate and context-aware output.

## Important Notes

- **Security:** Always use the minimum required access. For production, enable only `ACTION_READ`.
- **DDL:** Blocked by default. Only enable in development environments.
- **PostgreSQL & LastInsertId:** PostgreSQL does not support `LastInsertId()`. Use `RETURNING id` in your INSERT queries to get the last inserted ID.
- **Query validation:** The server validates query types based on the first keyword of the SQL statement. Unrecognized queries are rejected.

## Changelog

### v0.1.1
- Added knowledge base support (`KB_PATH`, `get_knowledge` tool)
- Added configurable DDL access control (`ACTION_DDL`)
- All messages and documentation in English
- Version constant added to server

### v0.1.0
- Initial release
- MySQL and PostgreSQL support
- Granular access control (READ, CREATE, UPDATE, DELETE)
- JSON output for query results
- STDIO transport

## License

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

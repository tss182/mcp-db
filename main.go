package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DBConfig menyimpan konfigurasi koneksi dan hak akses
type DBConfig struct {
	Driver      string
	DSN         string
	AllowRead   bool
	AllowCreate bool
	AllowUpdate bool
	AllowDelete bool
}

var db *sql.DB
var config DBConfig

func main() {
	// Parse Environment Variables
	config = DBConfig{
		Driver:      os.Getenv("DB_DRIVER"),
		DSN:         os.Getenv("DB_DSN"),
		AllowRead:   parseBoolEnv("ACTION_READ", false),
		AllowCreate: parseBoolEnv("ACTION_CREATE", false),
		AllowUpdate: parseBoolEnv("ACTION_UPDATE", false),
		AllowDelete: parseBoolEnv("ACTION_DELETE", false),
	}

	if config.Driver == "" || config.DSN == "" {
		log.Fatal("DB_DRIVER dan DB_DSN harus diisi.")
	}

	var err error
	db, err = sql.Open(config.Driver, config.DSN)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Gagal menghubungi database: %v", err)
	}

	// Inisialisasi MCP Server
	s := server.NewMCPServer(
		"DynamicDB-MCP",
		"1.1.0",
		server.WithToolCapabilities(true),
	)

	// Daftarkan Tool
	tool := mcp.NewTool("execute_db_query",
		mcp.WithDescription("Mengeksekusi query database. Hak akses diatur oleh server (Bisa membaca atau memodifikasi data tergantung konfigurasi)."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Query SQL yang akan dieksekusi.")),
	)
	s.AddTool(tool, executeDBQuery)

	log.Printf("Mulai MCP Server [%s]. Akses: READ=%v, CREATE=%v, UPDATE=%v, DELETE=%v",
		config.Driver, config.AllowRead, config.AllowCreate, config.AllowUpdate, config.AllowDelete)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Handler untuk eksekusi tool
func executeDBQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Arguments tidak valid"), nil
	}
	query, ok := args["query"].(string)
	if !ok {
		return mcp.NewToolResultError("Argument 'query' tidak valid"), nil
	}

	queryType, isAllowed := validateQueryAccess(query)
	if !isAllowed {
		return mcp.NewToolResultError(fmt.Sprintf("Akses ditolak untuk jenis query ini (Tipe: %s)", queryType)), nil
	}

	// Jika query adalah READ, gunakan QueryContext untuk mengambil data
	if queryType == "READ" {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error eksekusi READ: %v", err)), nil
		}
		defer rows.Close()

		jsonResult, err := rowsToJSON(rows)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error konversi data: %v", err)), nil
		}
		return mcp.NewToolResultText(jsonResult), nil
	}

	// Jika query adalah CREATE, UPDATE, atau DELETE, gunakan ExecContext
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error eksekusi %s: %v", queryType, err)), nil
	}

	rowsAffected, _ := result.RowsAffected()
	msg := fmt.Sprintf("Query %s berhasil dieksekusi. Rows affected: %d", queryType, rowsAffected)

	// Khusus CREATE (INSERT), coba ambil LastInsertId jika disupport driver (Postgres biasanya tidak support LastInsertId, melainkan via RETURNING)
	if queryType == "CREATE" && config.Driver == "mysql" {
		if lastID, err := result.LastInsertId(); err == nil && lastID > 0 {
			msg += fmt.Sprintf(", Last Insert ID: %d", lastID)
		}
	}

	return mcp.NewToolResultText(msg), nil
}

// ddlKeywords berisi keyword DDL yang selalu ditolak secara hardcode
var ddlKeywords = []string{"CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME"}

// isDDLQuery mengecek apakah query termasuk DDL berdasarkan keyword pertama
func isDDLQuery(command string) bool {
	for _, kw := range ddlKeywords {
		if command == kw {
			return true
		}
	}
	return false
}

// validateQueryAccess mengkategorikan query dan mengecek izinnya
func validateQueryAccess(query string) (string, bool) {
	q := strings.ToUpper(strings.TrimSpace(query))
	words := strings.Fields(q)
	if len(words) == 0 {
		return "UNKNOWN", false
	}

	command := words[0]

	// Hardcode restrict: query DDL selalu ditolak
	if isDDLQuery(command) {
		return "DDL", false
	}

	switch command {
	case "SELECT", "SHOW", "DESCRIBE", "EXPLAIN":
		return "READ", config.AllowRead
	case "INSERT":
		return "CREATE", config.AllowCreate
	case "UPDATE":
		return "UPDATE", config.AllowUpdate
	case "DELETE":
		return "DELETE", config.AllowDelete
	default:
		return "UNKNOWN", false
	}
}

// Helper parsing environment boolean
func parseBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

// Helper mengubah sql.Rows menjadi JSON
func rowsToJSON(rows *sql.Rows) (string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var result []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return "", err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			if b, ok := (*val).([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = *val
			}
		}
		result = append(result, rowMap)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "[]", nil
	}
	return string(jsonBytes), nil
}

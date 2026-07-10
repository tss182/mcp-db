package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	AllowDDL    bool
}

// KnowledgeEntry menyimpan satu file knowledge base
type KnowledgeEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

var db *sql.DB
var config DBConfig
var knowledgeBase []KnowledgeEntry

func main() {
	// Parse Environment Variables
	config = DBConfig{
		Driver:      os.Getenv("DB_DRIVER"),
		DSN:         os.Getenv("DB_DSN"),
		AllowRead:   parseBoolEnv("ACTION_READ", false),
		AllowCreate: parseBoolEnv("ACTION_CREATE", false),
		AllowUpdate: parseBoolEnv("ACTION_UPDATE", false),
		AllowDelete: parseBoolEnv("ACTION_DELETE", false),
		AllowDDL:    parseBoolEnv("ACTION_DDL", false),
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

	// Load Knowledge Base
	kbPath := os.Getenv("KB_PATH")
	if kbPath != "" {
		knowledgeBase, err = loadKnowledgeBase(kbPath)
		if err != nil {
			log.Printf("Warning: Gagal memuat knowledge base dari %s: %v", kbPath, err)
		} else {
			log.Printf("Knowledge base dimuat: %d file dari %s", len(knowledgeBase), kbPath)
		}
	}

	// Inisialisasi MCP Server
	s := server.NewMCPServer(
		"DynamicDB-MCP",
		"1.2.0",
		server.WithToolCapabilities(true),
	)

	// Daftarkan Tool: execute_db_query
	queryTool := mcp.NewTool("execute_db_query",
		mcp.WithDescription("Mengeksekusi query database. Hak akses diatur oleh server (Bisa membaca atau memodifikasi data tergantung konfigurasi)."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Query SQL yang akan dieksekusi.")),
	)
	s.AddTool(queryTool, executeDBQuery)

	// Daftarkan Tool: get_knowledge
	kbTool := mcp.NewTool("get_knowledge",
		mcp.WithDescription("Mengambil informasi dari knowledge base (skema database, relasi tabel, konteks bisnis). Gunakan tool ini sebelum menulis query untuk memahami struktur database."),
		mcp.WithString("search", mcp.Description("Kata kunci pencarian (opsional). Jika kosong, menampilkan daftar semua topik yang tersedia.")),
	)
	s.AddTool(kbTool, getKnowledge)

	log.Printf("Mulai MCP Server [%s]. Akses: READ=%v, CREATE=%v, UPDATE=%v, DELETE=%v, DDL=%v, KB=%d files",
		config.Driver, config.AllowRead, config.AllowCreate, config.AllowUpdate, config.AllowDelete, config.AllowDDL, len(knowledgeBase))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// loadKnowledgeBase memuat semua file .md dan .txt dari direktori yang diberikan
func loadKnowledgeBase(dirPath string) ([]KnowledgeEntry, error) {
	var entries []KnowledgeEntry

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Warning: Gagal membaca file %s: %v", path, err)
			return nil
		}

		// Gunakan relative path dari KB_PATH sebagai nama
		relPath, _ := filepath.Rel(dirPath, path)
		entries = append(entries, KnowledgeEntry{
			Name:    relPath,
			Content: string(content),
		})

		return nil
	})

	return entries, err
}

// Handler untuk get_knowledge tool
func getKnowledge(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if len(knowledgeBase) == 0 {
		return mcp.NewToolResultText("Knowledge base kosong. Set environment variable KB_PATH ke direktori yang berisi file .md atau .txt."), nil
	}

	args, _ := request.Params.Arguments.(map[string]interface{})
	search, _ := args["search"].(string)
	search = strings.TrimSpace(search)

	// Jika tidak ada keyword pencarian, tampilkan daftar topik
	if search == "" {
		var topics []string
		for _, entry := range knowledgeBase {
			topics = append(topics, fmt.Sprintf("- %s", entry.Name))
		}
		result := fmt.Sprintf("Knowledge base tersedia (%d topik):\n%s\n\nGunakan parameter 'search' untuk mencari informasi spesifik.", len(knowledgeBase), strings.Join(topics, "\n"))
		return mcp.NewToolResultText(result), nil
	}

	// Cari berdasarkan keyword (case-insensitive)
	searchLower := strings.ToLower(search)
	var matches []string

	for _, entry := range knowledgeBase {
		nameLower := strings.ToLower(entry.Name)
		contentLower := strings.ToLower(entry.Content)

		if strings.Contains(nameLower, searchLower) || strings.Contains(contentLower, searchLower) {
			matches = append(matches, fmt.Sprintf("=== %s ===\n%s", entry.Name, entry.Content))
		}
	}

	if len(matches) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("Tidak ditemukan knowledge base yang cocok dengan '%s'.", search)), nil
	}

	result := fmt.Sprintf("Ditemukan %d hasil untuk '%s':\n\n%s", len(matches), search, strings.Join(matches, "\n\n"))
	return mcp.NewToolResultText(result), nil
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

// ddlKeywords berisi keyword DDL
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

	// DDL dikontrol lewat ACTION_DDL (default: false)
	if isDDLQuery(command) {
		return "DDL", config.AllowDDL
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

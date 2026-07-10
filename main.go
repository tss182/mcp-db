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

const Version = "0.1.2"

// DBConfig holds connection and access control configuration
type DBConfig struct {
	Driver      string
	DSN         string
	AllowRead   bool
	AllowCreate bool
	AllowUpdate bool
	AllowDelete bool
	AllowDDL    bool
}

// KnowledgeEntry holds a single knowledge base file
type KnowledgeEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

var db *sql.DB
var config DBConfig
var knowledgeBase []KnowledgeEntry

func main() {
	// Parse environment variables
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
		log.Fatal("DB_DRIVER and DB_DSN are required.")
	}

	var err error
	db, err = sql.Open(config.Driver, config.DSN)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Load knowledge base
	kbPath := os.Getenv("KB_PATH")
	if kbPath != "" {
		knowledgeBase, err = loadKnowledgeBase(kbPath)
		if err != nil {
			log.Printf("Warning: Failed to load knowledge base from %s: %v", kbPath, err)
		} else {
			log.Printf("Knowledge base loaded: %d files from %s", len(knowledgeBase), kbPath)
		}
	}

	// Initialize MCP Server
	s := server.NewMCPServer(
		"MCP-DBs",
		Version,
		server.WithToolCapabilities(true),
	)

	// Register tool: execute_db_query
	queryTool := mcp.NewTool("execute_db_query",
		mcp.WithDescription("Execute a database query. Access rights are controlled by the server configuration (can read or modify data depending on settings)."),
		mcp.WithString("query", mcp.Required(), mcp.Description("The SQL query to execute.")),
	)
	s.AddTool(queryTool, executeDBQuery)

	// Register tool: get_knowledge
	kbTool := mcp.NewTool("get_knowledge",
		mcp.WithDescription("Retrieve information from the knowledge base (database schema, table relations, business context). Use this tool before writing queries to understand the database structure."),
		mcp.WithString("search", mcp.Description("Search keyword (optional). If empty, lists all available topics.")),
	)
	s.AddTool(kbTool, getKnowledge)

	log.Printf("MCP Server v%s started [%s]. Access: READ=%v, CREATE=%v, UPDATE=%v, DELETE=%v, DDL=%v, KB=%d files",
		Version, config.Driver, config.AllowRead, config.AllowCreate, config.AllowUpdate, config.AllowDelete, config.AllowDDL, len(knowledgeBase))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// loadKnowledgeBase loads all .md and .txt files from the given directory
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
			log.Printf("Warning: Failed to read file %s: %v", path, err)
			return nil
		}

		// Use relative path from KB_PATH as the entry name
		relPath, _ := filepath.Rel(dirPath, path)
		entries = append(entries, KnowledgeEntry{
			Name:    relPath,
			Content: string(content),
		})

		return nil
	})

	return entries, err
}

// getKnowledge handles the get_knowledge tool
func getKnowledge(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if len(knowledgeBase) == 0 {
		return mcp.NewToolResultText("Knowledge base is empty. Set the KB_PATH environment variable to a directory containing .md or .txt files."), nil
	}

	args, _ := request.Params.Arguments.(map[string]interface{})
	search, _ := args["search"].(string)
	search = strings.TrimSpace(search)

	// If no search keyword, list all available topics
	if search == "" {
		var topics []string
		for _, entry := range knowledgeBase {
			topics = append(topics, fmt.Sprintf("- %s", entry.Name))
		}
		result := fmt.Sprintf("Available knowledge base (%d topics):\n%s\n\nUse the 'search' parameter to find specific information.", len(knowledgeBase), strings.Join(topics, "\n"))
		return mcp.NewToolResultText(result), nil
	}

	// Search by keyword (case-insensitive)
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
		return mcp.NewToolResultText(fmt.Sprintf("No knowledge base entries found matching '%s'.", search)), nil
	}

	result := fmt.Sprintf("Found %d results for '%s':\n\n%s", len(matches), search, strings.Join(matches, "\n\n"))
	return mcp.NewToolResultText(result), nil
}

// executeDBQuery handles the execute_db_query tool
func executeDBQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments"), nil
	}
	query, ok := args["query"].(string)
	if !ok {
		return mcp.NewToolResultError("Invalid 'query' argument"), nil
	}

	queryType, isAllowed := validateQueryAccess(query)
	if !isAllowed {
		return mcp.NewToolResultError(fmt.Sprintf("Access denied for this query type (Type: %s)", queryType)), nil
	}

	// For READ queries, use QueryContext to fetch data
	if queryType == "READ" {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("READ execution error: %v", err)), nil
		}
		defer rows.Close()

		jsonResult, err := rowsToJSON(rows)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Data conversion error: %v", err)), nil
		}
		return mcp.NewToolResultText(jsonResult), nil
	}

	// For CREATE, UPDATE, DELETE, or DDL queries, use ExecContext
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("%s execution error: %v", queryType, err)), nil
	}

	rowsAffected, _ := result.RowsAffected()
	msg := fmt.Sprintf("Query %s executed successfully. Rows affected: %d", queryType, rowsAffected)

	// For INSERT on MySQL, try to get LastInsertId (PostgreSQL does not support this, use RETURNING instead)
	if queryType == "CREATE" && config.Driver == "mysql" {
		if lastID, err := result.LastInsertId(); err == nil && lastID > 0 {
			msg += fmt.Sprintf(", Last Insert ID: %d", lastID)
		}
	}

	return mcp.NewToolResultText(msg), nil
}

// ddlKeywords contains DDL keywords
var ddlKeywords = []string{"CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME"}

// isDDLQuery checks if the query is a DDL statement based on the first keyword
func isDDLQuery(command string) bool {
	for _, kw := range ddlKeywords {
		if command == kw {
			return true
		}
	}
	return false
}

// validateQueryAccess categorizes the query and checks access permissions
func validateQueryAccess(query string) (string, bool) {
	q := strings.ToUpper(strings.TrimSpace(query))
	words := strings.Fields(q)
	if len(words) == 0 {
		return "UNKNOWN", false
	}

	command := words[0]

	// DDL is controlled via ACTION_DDL (default: false)
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

// parseBoolEnv parses a boolean environment variable with a default value
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

// rowsToJSON converts sql.Rows to a JSON string
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

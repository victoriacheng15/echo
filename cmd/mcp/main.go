package main

import (
	"echo/internal/db"
	"echo/internal/mcp"
	"echo/internal/service"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	defaultDB := mcp.GetDefaultDBPath()
	dbPath := flag.String("db", defaultDB, "Path to the SQLite database file")
	flag.Parse()

	// Redirect logs to stderr to avoid corrupting MCP JSON-RPC on stdout
	log.SetOutput(os.Stderr)

	// Ensure the database directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize Database
	sqldb, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	// Initialize Service
	memorySvc := service.NewMemoryService(sqldb)

	// Create MCP Server
	s := mcp.NewServer(memorySvc)

	log.Printf("Echo MCP Server starting (DB: %s)...", *dbPath)

	// Start the server using Stdio transport
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

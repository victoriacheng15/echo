package main

import (
	"context"
	"echo/internal/db"
	"echo/internal/service"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	defaultDB := getDefaultDBPath()
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
	s := server.NewMCPServer(
		"Echo",
		"1.0.0",
	)

	// Register Tools
	registerStoreMemoryTool(s, memorySvc)
	registerRecallMemoryTool(s, memorySvc)
	registerSearchMemoriesTool(s, memorySvc)
	registerDeleteMemoryTool(s, memorySvc)

	log.Printf("Echo MCP Server starting (DB: %s)...", *dbPath)

	// Start the server using Stdio transport
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getDefaultDBPath() string {
	// 1. Respect XDG_DATA_HOME if set
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// 2. Fall back to ~/.local/share
		home, err := os.UserHomeDir()
		if err != nil {
			return "echo.db" // Final fallback to current directory
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "echo", "echo.db")
}

func registerStoreMemoryTool(s *server.MCPServer, svc *service.MemoryService) {
	tool := mcp.NewTool("store_memory",
		mcp.WithDescription("Saves or updates a contextual memory in the persistent 'brain'."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The memory content (max 8KB).")),
		mcp.WithString("context_key", mcp.Required(), mcp.Description("Context identifier (e.g., 'project:name' or 'global').")),
		mcp.WithString("entry_type",
			mcp.Required(),
			mcp.Description("Type of entry."),
			mcp.WithStringEnumItems([]string{"instruction", "snippet", "request", "sentence", "boilerplate"}),
		),
		mcp.WithString("metadata", mcp.Description("Optional JSON metadata string.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}
		contextKey, err := request.RequireString("context_key")
		if err != nil {
			return mcp.NewToolResultError("context_key is required"), nil
		}
		entryType, err := request.RequireString("entry_type")
		if err != nil {
			return mcp.NewToolResultError("entry_type is required"), nil
		}
		metadata := request.GetString("metadata", "")

		if err := svc.StoreMemory(content, contextKey, entryType, metadata); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to store memory: %v", err)), nil
		}

		return mcp.NewToolResultText("Memory stored successfully."), nil
	})
}

func registerRecallMemoryTool(s *server.MCPServer, svc *service.MemoryService) {
	tool := mcp.NewTool("recall_memory",
		mcp.WithDescription("Retrieves the most relevant memories for the current environment."),
		mcp.WithArray("context_keys",
			mcp.Required(),
			mcp.Description("List of context keys to search (e.g., ['global', 'project:echo'])."),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("limit", mcp.Description("Maximum number of memories to return (default 10).")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contextKeys, err := request.RequireStringSlice("context_keys")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("context_keys is required: %v", err)), nil
		}
		limit := request.GetInt("limit", 10)

		memories, err := svc.RecallMemory(contextKeys, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to recall memories: %v", err)), nil
		}

		data, _ := json.MarshalIndent(memories, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	})
}

func registerSearchMemoriesTool(s *server.MCPServer, svc *service.MemoryService) {
	tool := mcp.NewTool("search_memories",
		mcp.WithDescription("Full-text search across all stored memories."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Keyword to search for in memory content.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query is required"), nil
		}

		memories, err := svc.SearchMemories(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search memories: %v", err)), nil
		}

		data, _ := json.MarshalIndent(memories, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	})
}

func registerDeleteMemoryTool(s *server.MCPServer, svc *service.MemoryService) {
	tool := mcp.NewTool("delete_memory",
		mcp.WithDescription("Removes a specific memory from the persistent brain."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The content of the memory to delete.")),
		mcp.WithString("context_key", mcp.Required(), mcp.Description("The context key of the memory to delete.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}
		contextKey, err := request.RequireString("context_key")
		if err != nil {
			return mcp.NewToolResultError("context_key is required"), nil
		}

		if err := svc.DeleteMemory(content, contextKey); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete memory: %v", err)), nil
		}

		return mcp.NewToolResultText("Memory deleted successfully."), nil
	})
}

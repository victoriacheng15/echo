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

	// Load Governance Rules for tool descriptions
	governanceRules := loadGovernanceRules()

	// Create MCP Server
	s := server.NewMCPServer(
		"Echo",
		"1.0.0",
	)

	// Register Tools
	registerStoreMemoryTool(s, memorySvc, governanceRules)
	registerRecallMemoryTool(s, memorySvc, governanceRules)
	registerSearchMemoriesTool(s, memorySvc, governanceRules)
	registerDeletionTools(s, memorySvc)

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

func loadGovernanceRules() string {
	// Try to load rules/memories.md from current directory
	data, err := os.ReadFile("rules/memories.md")
	if err != nil {
		log.Printf("Warning: rules/memories.md not found. Falling back to default descriptions.")
		return ""
	}
	return "\n\nGOVERNANCE RULES:\n" + string(data)
}

func registerStoreMemoryTool(s *server.MCPServer, svc *service.MemoryService, rules string) {
	description := "Saves or updates a memory. " + rules
	tool := mcp.NewTool("store_memory",
		mcp.WithDescription(description),
		mcp.WithString("content", mcp.Required(), mcp.Description("The memory content (max 8KB).")),
		mcp.WithString("context_key", mcp.Required(), mcp.Description("Context identifier (e.g., 'project:name' or 'global').")),
		mcp.WithString("entry_type",
			mcp.Required(),
			mcp.Description("Type of entry."),
			mcp.WithStringEnumItems([]string{"directive", "artifact", "fact"}),
		),
		mcp.WithArray("tags", mcp.Description("Optional list of tags for categorization (e.g., ['security', 'go-standard'])."), mcp.WithStringItems()),
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
		tags, _ := request.RequireStringSlice("tags")

		if err := svc.StoreMemoryWithTags(content, contextKey, entryType, tags); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to store memory: %v", err)), nil
		}

		return mcp.NewToolResultText("Memory stored successfully."), nil
	})
}

func registerRecallMemoryTool(s *server.MCPServer, svc *service.MemoryService, rules string) {
	description := "Recalls memories for a given context. " + rules
	tool := mcp.NewTool("recall_memory",
		mcp.WithDescription(description),
		mcp.WithArray("context_keys",
			mcp.Required(),
			mcp.Description("List of context keys to search (e.g., ['global', 'project:echo'])."),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("limit", mcp.Description("Maximum number of memories to return (default 10).")),
		mcp.WithBoolean("verbose", mcp.Description("If true, includes audit metadata (source, created_at, importance_score) in the output.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contextKeys, err := request.RequireStringSlice("context_keys")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("context_keys is required: %v", err)), nil
		}
		limit := request.GetInt("limit", 10)
		verbose := request.GetBool("verbose", false)

		memories, err := svc.RecallMemory(contextKeys, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to recall memories: %v", err)), nil
		}

		if !verbose {
			for i := range memories {
				memories[i].ID = 0
				memories[i].ContextKey = ""
				memories[i].EntryType = ""
				memories[i].ImportanceScore = 0
				memories[i].Source = ""
				memories[i].CreatedAt = ""
				memories[i].IsActive = false
			}
		}

		data, _ := json.Marshal(memories)
		return mcp.NewToolResultText(string(data)), nil
	})
}

func registerSearchMemoriesTool(s *server.MCPServer, svc *service.MemoryService, rules string) {
	description := "Full-text search for a memory. " + rules
	tool := mcp.NewTool("search_memories",
		mcp.WithDescription(description),
		mcp.WithString("query", mcp.Required(), mcp.Description("Keyword to search for in memory content.")),
		mcp.WithBoolean("verbose", mcp.Description("If true, includes audit metadata (source, created_at, importance_score) in the output.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query is required"), nil
		}
		verbose := request.GetBool("verbose", false)

		memories, err := svc.SearchMemories(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search memories: %v", err)), nil
		}

		if !verbose {
			for i := range memories {
				memories[i].ID = 0
				memories[i].ContextKey = ""
				memories[i].EntryType = ""
				memories[i].ImportanceScore = 0
				memories[i].Source = ""
				memories[i].CreatedAt = ""
				memories[i].IsActive = false
			}
		}

		data, _ := json.Marshal(memories)
		return mcp.NewToolResultText(string(data)), nil
	})
}

func registerDeletionTools(s *server.MCPServer, svc *service.MemoryService) {
	// Step 1: Search for the memory to get its exact content
	searchTool := mcp.NewTool("search_for_deletion",
		mcp.WithDescription("Step 1 of 2: Safely search for a memory you intend to delete. You MUST show the returned results to the user and obtain their explicit confirmation before proceeding to Step 2."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Keyword to find the memory to delete.")),
	)
	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query is required"), nil
		}
		memories, err := svc.SearchMemories(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search for memory: %v", err)), nil
		}
		if len(memories) == 0 {
			return mcp.NewToolResultText("No memory found for that query."), nil
		}
		// In a real scenario, you might handle multiple matches. For now, return the first one.
		data, _ := json.Marshal(memories[0])
		return mcp.NewToolResultText(string(data)), nil
	})

	// Step 2: Delete the memory using the exact content and key
	deleteTool := mcp.NewTool("delete_memory",
		mcp.WithDescription("Step 2 of 2: Deletes a memory. You MUST ONLY call this tool after the user has explicitly confirmed the deletion of the memory returned by search_for_deletion. This is a destructive, non-reversible action."),
		mcp.WithString("content", mcp.Required(), mcp.Description("The exact content of the memory to delete.")),
		mcp.WithString("context_key", mcp.Required(), mcp.Description("The exact context key of the memory to delete.")),
	)
	s.AddTool(deleteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

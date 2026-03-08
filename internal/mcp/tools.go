package mcp

import (
	"context"
	"echo/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures a new Echo MCP Server.
func NewServer(memorySvc *service.MemoryService, analyticsSvc *service.AnalyticsService, rateSvc *service.RateService) *server.MCPServer {
	// Create MCP Server
	s := server.NewMCPServer(
		"Echo",
		"1.0.0",
	)

	// Load Governance Rules for tool descriptions
	governanceRules := loadGovernanceRules()

	// Register Tools (CRUD Order)
	registerStoreMemoryTool(s, memorySvc, governanceRules)    // Create/Reinforce
	registerRecallMemoryTool(s, memorySvc, governanceRules)   // Read (Contextual)
	registerSearchMemoriesTool(s, memorySvc, governanceRules) // Read (FTS)
	registerUpdateMemoryTool(s, memorySvc)                    // Update (Surgical)
	registerDeletionTools(s, memorySvc)                       // Delete

	// Register Analytical Tools (Phase 6.5)
	if analyticsSvc != nil && rateSvc != nil {
		registerGetAnalyticsTool(s, analyticsSvc, rateSvc)
	}

	return s
}

// --- Tool Registration: CREATE ---
// registerStoreMemoryTool registers the 'store_memory' tool.
// Inputs:
// - content (string, required): The memory content (max 8KB).
// - context_key (string, required): Context identifier (e.g., 'project:name' or 'global').
// - entry_type (string, required): Type of entry (directive, artifact, fact).
// - tags (string array, optional): Optional list of tags for categorization.
func registerStoreMemoryTool(s *server.MCPServer, svc *service.MemoryService, rules string) {
	description := "Saves a new memory or reinforces an existing one. " + rules
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

// --- Tool Registration: READ ---
// registerRecallMemoryTool registers the 'recall_memory' tool.
// Inputs:
// - context_keys (string array, required): List of context keys to search.
// - limit (number, optional): Maximum number of memories to return (default 10).
// - verbose (boolean, optional): If true, includes audit metadata in the output.
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
				memories[i] = cleanMemoryForResults(memories[i])
			}
		}

		data, _ := json.Marshal(memories)
		return mcp.NewToolResultText(string(data)), nil
	})
}

// --- Tool Registration: SEARCH ---
// registerSearchMemoriesTool registers the 'search_memories' tool.
// Inputs:
// - query (string, required): Keyword to search for in memory content.
// - verbose (boolean, optional): If true, includes audit metadata in the output.
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
				memories[i] = cleanMemoryForResults(memories[i])
			}
		}

		data, _ := json.Marshal(memories)
		return mcp.NewToolResultText(string(data)), nil
	})
}

// --- Tool Registration: UPDATE ---
// registerUpdateMemoryTool registers the 'update_memory' tool.
// Inputs:
// - id (number, required): The ID of the memory to update.
// - content (string, required): The new content for the memory.
func registerUpdateMemoryTool(s *server.MCPServer, svc *service.MemoryService) {
	tool := mcp.NewTool("update_memory",
		mcp.WithDescription("Updates the content (description) of an existing memory by its ID. Use this when the core instruction or information needs to be refined without losing its history (metadata, importance score)."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("The ID of the memory to update.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The new content (description) for the memory.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := request.GetInt("id", 0)
		if id == 0 {
			return mcp.NewToolResultError("id is required and must be non-zero"), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}

		if err := svc.UpdateMemoryContentByID(int64(id), content); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update memory: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Memory %d updated successfully.", int64(id))), nil
	})
}

// --- Tool Registration: DELETE ---
// registerDeletionTools registers 'search_for_deletion' and 'delete_memory' tools.
// search_for_deletion Inputs:
// - query (string, required): Keyword to find the memory to delete.
// delete_memory Inputs:
// - content (string, required): The exact content of the memory to delete.
// - context_key (string, required): The exact context key of the memory to delete.
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

// --- Tool Registration: ANALYTICS ---
// registerGetAnalyticsTool registers the 'get_analytics' tool.
// Inputs:
// - context_key (string, optional): Filter by context.
// - agent (string, optional): Filter by agent.
func registerGetAnalyticsTool(s *server.MCPServer, svc *service.AnalyticsService, rateSvc *service.RateService) {
	tool := mcp.NewTool("get_analytics",
		mcp.WithDescription("Retrieves analytical insights including context ROI, unit economics (FinOps), and environmental impact (GreenOps)."),
		mcp.WithString("context_key", mcp.Description("Optional filter by context_key (e.g., 'project:echo').")),
		mcp.WithString("agent", mcp.Description("Optional filter by agent identifier.")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Sync events before querying to ensure up-to-date data
		if err := svc.SyncEvents(); err != nil {
			log.Printf("Warning: failed to sync events for analytics: %v", err)
		}

		contextKey := request.GetString("context_key", "")
		agent := request.GetString("agent", "")

		impacts, err := svc.GetProjectImpact(rateSvc.Card, contextKey, agent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve analytics: %v", err)), nil
		}

		data, _ := json.Marshal(impacts)
		return mcp.NewToolResultText(string(data)), nil
	})
}

// --- Helpers ---

// loadGovernanceRules reads the memory governance rules from rules/memories.md.
func loadGovernanceRules() string {
	// Try to load rules/memories.md from current directory
	data, err := os.ReadFile("rules/memories.md")
	if err != nil {
		log.Printf("Warning: rules/memories.md not found. Falling back to default descriptions.")
		return ""
	}
	return "\n\nGOVERNANCE RULES:\n" + string(data)
}

// cleanMemoryForResults strips metadata from memories for non-verbose output.
func cleanMemoryForResults(m service.Memory) service.Memory {
	return service.Memory{
		Content: m.Content,
		Tags:    m.Tags,
	}
}

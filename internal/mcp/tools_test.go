package mcp

import (
	"context"
	"echo/internal/db"
	"echo/internal/service"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewServer(t *testing.T) {
	dbPath := "test_mcp_server.db"
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}()

	sqldb, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer sqldb.Close()

	dataDir := "test_data_mcp"
	defer os.RemoveAll(dataDir)

	analyticsSvc, _ := service.NewAnalyticsService(dataDir)
	rateSvc := &service.RateService{Card: service.RateCard{ComputeCADPerMs: 0.0001}}
	svc := service.NewMemoryService(sqldb)
	s := NewServer(svc, analyticsSvc, rateSvc)

	t.Run("VerifyToolMetadata", func(t *testing.T) {
		toolNames := []string{
			"store_memory",
			"recall_memory",
			"search_memories",
			"update_memory",
			"search_for_deletion",
			"delete_memory",
			"get_analytics",
		}

		tools := s.ListTools()
		for _, name := range toolNames {
			found := false
			for _, tool := range tools {
				if tool.Tool.Name == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Tool %s not registered", name)
			}
		}
	})

	t.Run("TestToolHandlers_TableDriven", func(t *testing.T) {
		ctx := context.Background()
		tools := s.ListTools()
		handlers := make(map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
		for _, tool := range tools {
			handlers[tool.Tool.Name] = tool.Handler
		}

		// Prepare a memory for update/deletion tests
		svc.StoreMemory("memory to be deleted", "global", "fact")
		var initialID int64
		sqldb.QueryRow("SELECT id FROM memories WHERE content = 'memory to be deleted'").Scan(&initialID)

		tests := []struct {
			name      string
			tool      string
			arguments map[string]any
			wantErr   bool
		}{
			{
				name: "store_memory_success",
				tool: "store_memory",
				arguments: map[string]any{
					"content":     "test content",
					"context_key": "global",
					"entry_type":  "directive",
					"tags":        []any{"tag1", "tag2"},
				},
				wantErr: false,
			},
			{
				name: "store_memory_missing_content",
				tool: "store_memory",
				arguments: map[string]any{
					"context_key": "global",
					"entry_type":  "directive",
				},
				wantErr: true,
			},
			{
				name: "recall_memory_success",
				tool: "recall_memory",
				arguments: map[string]any{
					"context_keys": []any{"global"},
					"limit":        5,
					"verbose":      true,
				},
				wantErr: false,
			},
			{
				name: "recall_memory_missing_keys",
				tool: "recall_memory",
				arguments: map[string]any{
					"limit": 5,
				},
				wantErr: true,
			},
			{
				name: "search_memories_success",
				tool: "search_memories",
				arguments: map[string]any{
					"query":   "test",
					"verbose": false,
				},
				wantErr: false,
			},
			{
				name: "search_memories_missing_query",
				tool: "search_memories",
				arguments: map[string]any{
					"verbose": false,
				},
				wantErr: true,
			},
			{
				name: "update_memory_success",
				tool: "update_memory",
				arguments: map[string]any{
					"id":      float64(initialID),
					"content": "updated content",
					"tags":    []any{"newtag"},
				},
				wantErr: false,
			},
			{
				name: "update_memory_invalid_id",
				tool: "update_memory",
				arguments: map[string]any{
					"id":      float64(0),
					"content": "updated content",
				},
				wantErr: true,
			},
			{
				name: "update_memory_missing_content",
				tool: "update_memory",
				arguments: map[string]any{
					"id": float64(initialID),
				},
				wantErr: true,
			},
			{
				name: "search_for_deletion_success",
				tool: "search_for_deletion",
				arguments: map[string]any{
					"query": "updated",
				},
				wantErr: false,
			},
			{
				name: "search_for_deletion_no_match",
				tool: "search_for_deletion",
				arguments: map[string]any{
					"query": "non-existent-query-string",
				},
				wantErr: false, // returns "No memory found" text, not IsError: true
			},
			{
				name:      "search_for_deletion_missing_query",
				tool:      "search_for_deletion",
				arguments: map[string]any{},
				wantErr:   true,
			},
			{
				name: "delete_memory_success",
				tool: "delete_memory",
				arguments: map[string]any{
					"id": float64(initialID),
				},
				wantErr: false,
			},
			{
				name: "delete_memory_invalid_id",
				tool: "delete_memory",
				arguments: map[string]any{
					"id": float64(0),
				},
				wantErr: true,
			},
			{
				name: "get_analytics_success",
				tool: "get_analytics",
				arguments: map[string]any{
					"context_key": "global",
				},
				wantErr: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{}
				req.Params.Arguments = tt.arguments

				res, err := handlers[tt.tool](ctx, req)
				if err != nil {
					t.Fatalf("Handler returned actual error: %v", err)
				}
				if res.IsError != tt.wantErr {
					t.Errorf("got IsError %v, want %v. Error content: %+v", res.IsError, tt.wantErr, res.Content)
				}
			})
		}
	})
}

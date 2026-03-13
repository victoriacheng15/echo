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
				},
				wantErr: false,
			},
			{
				name: "get_analytics_success",
				tool: "get_analytics",
				arguments: map[string]any{
					"context_key": "global",
				},
				wantErr: false,
			},
			{
				name: "recall_memory_success",
				tool: "recall_memory",
				arguments: map[string]any{
					"context_keys": []any{"global"},
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

func TestGetDefaultDBPath(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		old := os.Getenv("XDG_DATA_HOME")
		os.Setenv("XDG_DATA_HOME", "/tmp/xdg")
		defer os.Setenv("XDG_DATA_HOME", old)

		path := db.GetDefaultDBPath()
		expected := "/tmp/xdg/echo/echo.db"
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})
}

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

	svc := service.NewMemoryService(sqldb)
	s := NewServer(svc)

	t.Run("VerifyToolMetadata", func(t *testing.T) {
		// Just check for presence of tools
		toolNames := []string{
			"store_memory",
			"recall_memory",
			"search_memories",
			"update_memory",
			"search_for_deletion",
			"delete_memory",
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

	t.Run("TestToolHandlers", func(t *testing.T) {
		ctx := context.Background()
		tools := s.ListTools()
		handlers := make(map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
		for _, tool := range tools {
			handlers[tool.Tool.Name] = tool.Handler
		}

		t.Run("store_memory_handler", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"content":     "test content",
				"context_key": "global",
				"entry_type":  "directive",
			}

			res, err := handlers["store_memory"](ctx, req)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %+v", res.Content[0])
			}
		})
	})
}

func TestGetDefaultDBPath(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		old := os.Getenv("XDG_DATA_HOME")
		os.Setenv("XDG_DATA_HOME", "/tmp/xdg")
		defer os.Setenv("XDG_DATA_HOME", old)

		path := GetDefaultDBPath()
		expected := "/tmp/xdg/echo/echo.db"
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})
}

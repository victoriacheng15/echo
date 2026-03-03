package main

import (
	"context"
	"echo/internal/db"
	"echo/internal/service"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterTools(t *testing.T) {
	dbPath := "test_main.db"
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
	s := server.NewMCPServer("Test", "1.0.0")
	
	registerStoreMemoryTool(s, svc)
	registerRecallMemoryTool(s, svc)
	registerSearchMemoriesTool(s, svc)
	registerDeleteMemoryTool(s, svc)

	t.Run("VerifyToolMetadata", func(t *testing.T) {
		tests := []struct {
			name        string
			description string
		}{
			{
				name:        "store_memory",
				description: "Saves or updates a contextual memory in the persistent 'brain'.",
			},
			{
				name:        "recall_memory",
				description: "Retrieves the most relevant memories for the current environment.",
			},
			{
				name:        "search_memories",
				description: "Full-text search across all stored memories.",
			},
			{
				name:        "delete_memory",
				description: "Removes a specific memory from the persistent brain.",
			},
		}

		tools := s.ListTools()
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var foundTool *server.ServerTool
				for _, tool := range tools {
					if tool.Tool.Name == tt.name {
						foundTool = tool
						break
					}
				}

				if foundTool == nil {
					t.Fatalf("Tool %s not registered", tt.name)
				}

				if foundTool.Tool.Description != tt.description {
					t.Errorf("Expected description %q, got %q", tt.description, foundTool.Tool.Description)
				}
			})
		}
	})

	t.Run("TestToolHandlers", func(t *testing.T) {
		ctx := context.Background()
		tools := s.ListTools()
		handlers := make(map[string]server.ToolHandlerFunc)
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

		t.Run("store_memory_handler_errors", func(t *testing.T) {
			cases := []struct {
				name string
				args map[string]any
			}{
				{"missing content", map[string]any{"context_key": "global", "entry_type": "directive"}},
				{"missing key", map[string]any{"content": "a", "entry_type": "directive"}},
				{"missing type", map[string]any{"content": "a", "context_key": "global"}},
				{"invalid data", map[string]any{"content": "", "context_key": "global", "entry_type": "directive"}},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := mcp.CallToolRequest{}
					req.Params.Arguments = tc.args
					res, _ := handlers["store_memory"](ctx, req)
					if !res.IsError {
						t.Errorf("Expected error for %s", tc.name)
					}
				})
			}
		})

		t.Run("recall_memory_handler", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"context_keys": []string{"global"},
				"limit":        5.0,
			}

			res, err := handlers["recall_memory"](ctx, req)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %+v", res.Content[0])
			}
		})

		t.Run("recall_memory_handler_error", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"context_keys": "not-an-array",
			}

			res, _ := handlers["recall_memory"](ctx, req)
			if !res.IsError {
				t.Fatal("Expected error for invalid context_keys")
			}
		})

		t.Run("search_memories_handler", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"query": "test",
			}

			res, err := handlers["search_memories"](ctx, req)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %+v", res.Content[0])
			}
		})

		t.Run("search_memories_handler_error", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{}
			res, _ := handlers["search_memories"](ctx, req)
			if !res.IsError {
				t.Fatal("Expected error for missing query")
			}
		})

		t.Run("search_memories_db_error", func(t *testing.T) {
			sqldb.Close()
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": "test"}
			res, _ := handlers["search_memories"](ctx, req)
			if !res.IsError {
				t.Fatal("Expected error for closed DB")
			}
			sqldb, _ = db.InitDB(dbPath)
			svc = service.NewMemoryService(sqldb)
			registerDeleteMemoryTool(s, svc)
			for _, tool := range s.ListTools() {
				if tool.Tool.Name == "delete_memory" {
					handlers["delete_memory"] = tool.Handler
				}
			}
		})

		t.Run("delete_memory_handler", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"content":     "test content",
				"context_key": "global",
			}

			res, err := handlers["delete_memory"](ctx, req)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %+v", res.Content[0])
			}
		})

		t.Run("delete_memory_handler_errors", func(t *testing.T) {
			cases := []struct {
				name string
				args map[string]any
			}{
				{"missing content", map[string]any{"context_key": "global"}},
				{"missing key", map[string]any{"content": "a"}},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := mcp.CallToolRequest{}
					req.Params.Arguments = tc.args
					res, _ := handlers["delete_memory"](ctx, req)
					if !res.IsError {
						t.Errorf("Expected error for %s", tc.name)
					}
				})
			}
		})
	})
}

func TestGetDefaultDBPath(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		os.Setenv("XDG_DATA_HOME", "/tmp/xdg")
		defer os.Unsetenv("XDG_DATA_HOME")
		path := getDefaultDBPath()
		expected := "/tmp/xdg/echo/echo.db"
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})

	t.Run("XDG_DATA_HOME unset", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")
		path := getDefaultDBPath()
		if path == "" {
			t.Error("Expected a non-empty path")
		}
	})
}

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

	rules := "\n\nGOVERNANCE RULES:\ntest rules"
	registerStoreMemoryTool(s, svc, rules)
	registerRecallMemoryTool(s, svc, rules)
	registerSearchMemoriesTool(s, svc, rules)
	registerDeletionTools(s, svc)
	registerUpdateMemoryTool(s, svc)

	t.Run("VerifyToolMetadata", func(t *testing.T) {
		tests := []struct {
			name        string
			description string
		}{
			{
				name:        "store_memory",
				description: "Saves a new memory or reinforces an existing one. " + rules,
			},
			{
				name:        "recall_memory",
				description: "Recalls memories for a given context. " + rules,
			},
			{
				name:        "search_memories",
				description: "Full-text search for a memory. " + rules,
			},
			{
				name:        "update_memory",
				description: "Updates the content (description) of an existing memory by its ID. Use this when the core instruction or information needs to be refined without losing its history (metadata, importance score).",
			},
			{
				name:        "search_for_deletion",
				description: "Step 1 of 2: Safely search for a memory you intend to delete. You MUST show the returned results to the user and obtain their explicit confirmation before proceeding to Step 2.",
			},
			{
				name:        "delete_memory",
				description: "Step 2 of 2: Deletes a memory. You MUST ONLY call this tool after the user has explicitly confirmed the deletion of the memory returned by search_for_deletion. This is a destructive, non-reversible action.",
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

		t.Run("search_for_deletion_handler", func(t *testing.T) {
			// Seed a memory to find
			svc.StoreMemory("to delete", "global", "directive")

			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"query": "to delete",
			}

			res, err := handlers["search_for_deletion"](ctx, req)
			if err != nil {
				t.Fatalf("Handler error: %v", err)
			}
			if res.IsError {
				t.Fatalf("Tool returned error: %+v", res.Content[0])
			}
		})

		t.Run("delete_memory_handler", func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"content":     "to delete",
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

package cli

import (
	"echo/internal/db"
	"echo/internal/service"
	"fmt"
	"testing"
)

// setupTestDB initializes an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *service.MemoryService {
	sqldb, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
	return service.NewMemoryService(sqldb)
}

// TestDispatcher_Run_UnknownCommand verifies that unknown commands return an error.
func TestDispatcher_Run_UnknownCommand(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{"unknown-cmd"})
	if err == nil {
		t.Error("Expected error for unknown command, got nil")
	}
}

// TestDispatcher_Run_NoCommand verifies that providing no command returns an error.
func TestDispatcher_Run_NoCommand(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{})
	if err == nil {
		t.Error("Expected error for no command, got nil")
	}
}

// TestDispatcher_Run_Help verifies that the help command does not return an error.
func TestDispatcher_Run_Help(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{"help"})
	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}
}

func TestDispatcher_Store(t *testing.T) {
	memSvc := setupTestDB(t)
	d := NewDispatcher(memSvc, nil, nil, "table")

	// 1. Success case
	err := d.Run([]string{"store", "-content", "test memory", "-context", "project:test", "-tags", "tag1,tag2"})
	if err != nil {
		t.Fatalf("Store command failed: %v", err)
	}

	// Verify it was stored
	mems, _ := memSvc.RecallMemory([]string{"project:test"}, 1)
	if len(mems) != 1 || mems[0].Content != "test memory" {
		t.Errorf("Memory not stored correctly")
	}
	if len(mems[0].Tags) != 2 || mems[0].Tags[0] != "tag1" || mems[0].Tags[1] != "tag2" {
		t.Errorf("Tags not stored correctly: %v", mems[0].Tags)
	}

	// 2. Missing content
	err = d.Run([]string{"store", "-context", "project:test"})
	if err == nil {
		t.Error("Expected error for missing content, got nil")
	}
}

func TestDispatcher_Recall(t *testing.T) {
	memSvc := setupTestDB(t)
	memSvc.StoreMemory("memory 1", "project:test", "fact")
	memSvc.StoreMemory("memory 2", "global", "fact")

	d := NewDispatcher(memSvc, nil, nil, "table")

	// 1. Recall specific context
	err := d.Run([]string{"recall", "-contexts", "project:test"})
	if err != nil {
		t.Fatalf("Recall command failed: %v", err)
	}

	// 2. Recall multiple contexts
	err = d.Run([]string{"recall", "-contexts", "project:test,global", "-limit", "5"})
	if err != nil {
		t.Fatalf("Recall multiple contexts failed: %v", err)
	}
}

func TestDispatcher_Search(t *testing.T) {
	memSvc := setupTestDB(t)
	memSvc.StoreMemory("unique search term", "global", "fact")

	d := NewDispatcher(memSvc, nil, nil, "table")

	// 1. Success case
	err := d.Run([]string{"search", "-query", "unique"})
	if err != nil {
		t.Fatalf("Search command failed: %v", err)
	}

	// 2. Missing query
	err = d.Run([]string{"search"})
	if err == nil {
		t.Error("Expected error for missing query, got nil")
	}
}

func TestDispatcher_Delete(t *testing.T) {
	memSvc := setupTestDB(t)
	memSvc.StoreMemory("to be deleted", "global", "fact")
	mems, _ := memSvc.RecallMemory([]string{"global"}, 1)
	if len(mems) == 0 {
		t.Fatalf("Failed to store memory for deletion test")
	}
	id := int64(mems[0].ID)

	d := NewDispatcher(memSvc, nil, nil, "table")

	// 1. Success case
	idStr := fmt.Sprintf("%d", id)
	err := d.Run([]string{"delete", "-id", idStr})
	if err != nil {
		t.Fatalf("Delete command failed: %v", err)
	}

	// Verify it was deleted
	mems, _ = memSvc.RecallMemory([]string{"global"}, 1)
	if len(mems) != 0 {
		t.Errorf("Memory was not deleted")
	}

	// 2. Missing ID
	err = d.Run([]string{"delete"})
	if err == nil {
		t.Error("Expected error for missing id, got nil")
	}
}

func TestDispatcher_OutputFormats(t *testing.T) {
	memSvc := setupTestDB(t)
	memSvc.StoreMemory("format test", "global", "fact")

	formats := []string{"table", "json", "csv"}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			d := NewDispatcher(memSvc, nil, nil, f)
			err := d.Run([]string{"recall", "-contexts", "global"})
			if err != nil {
				t.Errorf("Recall with format %s failed: %v", f, err)
			}
		})
	}
}

func TestDispatcher_Maintain(t *testing.T) {
	memSvc := setupTestDB(t)
	d := NewDispatcher(memSvc, nil, nil, "table")
	
	// 1. Test without sync/rebuild flags should fail
	err := d.Run([]string{"maintain"})
	if err == nil {
		t.Error("Expected error for maintain without flags, got nil")
	}

	// 2. Rebuild flag (successfully calls RebuildIndex)
	err = d.Run([]string{"maintain", "-rebuild"})
	if err != nil {
		t.Errorf("Maintain rebuild failed: %v", err)
	}

	// 3. Sync flag (fails if AnalyticsSvc is nil)
	err = d.Run([]string{"maintain", "-sync"})
	if err == nil {
		t.Error("Expected error for sync when AnalyticsSvc is nil, got nil")
	}
}

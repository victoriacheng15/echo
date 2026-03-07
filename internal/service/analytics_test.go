package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyticsService(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Initialize
	as, err := NewAnalyticsService(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create AnalyticsService: %v", err)
	}
	defer as.Close()

	// Step 2: Create a sample events.jsonl
	eventsContent := `{"timestamp":"2026-03-06T10:00:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"project:echo","memory_ids":[1,2],"latency_ms":15,"is_hit":true,"joules":0.75}
{"timestamp":"2026-03-06T10:01:00Z","tool":"search","source_interface":"mcp","agent":"claude","context_key":"global","memory_ids":[],"latency_ms":50,"is_hit":false,"joules":2.5}`
	
	err = os.WriteFile(filepath.Join(tmpDir, "events.jsonl"), []byte(eventsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write sample events: %v", err)
	}

	// Step 3: Sync
	if err := as.SyncEvents(); err != nil {
		t.Fatalf("Failed to sync events: %v", err)
	}

	// Step 4: Verify View
	t.Run("QueryView", func(t *testing.T) {
		var callCount int
		err := as.db.QueryRow("SELECT call_count FROM project_finops WHERE context_key = 'project:echo' AND agent = 'claude'").Scan(&callCount)
		if err != nil {
			t.Fatalf("Failed to query view: %v", err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call for project:echo, got %d", callCount)
		}

		var totalJoules float64
		err = as.db.QueryRow("SELECT total_joules FROM project_finops WHERE context_key = 'global'").Scan(&totalJoules)
		if err != nil {
			t.Fatalf("Failed to query joules: %v", err)
		}
		if totalJoules != 2.5 {
			t.Errorf("Expected 2.5 joules for global, got %f", totalJoules)
		}
	})
}

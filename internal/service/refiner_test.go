package service

import (
	"echo/internal/db"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestKnowledgeRefiner(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refiner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Initialize Services
	sqlitePath := filepath.Join(tmpDir, "test.db")
	sqldb, _ := db.InitDB(sqlitePath)
	defer sqldb.Close()

	memorySvc := NewMemoryService(sqldb)
	analyticsSvc, _ := NewAnalyticsService(tmpDir)
	defer analyticsSvc.Close()

	// RateCard with aggressive decay for testing
	rates := &RateService{Card: RateCard{
		TargetHitRate: 0.5,
		DecayStep:     1,
	}}

	refiner := NewKnowledgeRefiner(memorySvc, analyticsSvc, rates)

	t.Run("RefineCycleDecay", func(t *testing.T) {
		// 1. Add a memory with importance 5
		memorySvc.StoreMemory("bad memory", "test:low", "fact")
		var initialID int64
		sqldb.QueryRow("SELECT id FROM memories WHERE content = 'bad memory'").Scan(&initialID)

		// 2. Mock events with 0 hit rate for this context
		eventsContent := `{"timestamp":"2026-03-06T12:00:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"test:low","memory_ids":[1],"latency_ms":10,"is_hit":false,"joules":0.5}`
		os.WriteFile(filepath.Join(tmpDir, "events.jsonl"), []byte(eventsContent), 0644)

		// 3. Run Refinement
		if err := refiner.Refine(); err != nil {
			t.Fatalf("Refinement failed: %v", err)
		}

		// 4. Verify decay in SQLite
		var importance int
		sqldb.QueryRow("SELECT importance_score FROM memories WHERE id = ?", initialID).Scan(&importance)

		// Initial was 1 (default from StoreMemory UPSERT if not exists), so 1 - 1 = 0
		if importance != 0 {
			t.Errorf("Expected importance score 0, got %d", importance)
		}

		var isActive bool
		sqldb.QueryRow("SELECT is_active FROM memories WHERE id = ?", initialID).Scan(&isActive)
		if isActive {
			t.Error("Memory should be deactivated when importance hits 0")
		}
	})
}

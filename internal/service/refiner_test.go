package service

import (
	"echo/internal/db"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestKnowledgeRefiner_TableDriven(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refiner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		initialMemory  Memory
		eventsJSONL    string // Use %d for memory ID injection
		targetHitRate  float64
		decayStep      int
		wantImportance int
		wantActive     bool
	}{
		{
			name: "DecayToZeroAndDeactivate",
			initialMemory: Memory{
				Content:    "bad memory",
				ContextKey: "test:low",
				EntryType:  "fact",
			},
			eventsJSONL:    `{"timestamp":"2026-03-06T12:00:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"test:low","memory_ids":[%d],"latency_ms":10,"is_hit":false,"joules":0.5}`,
			targetHitRate:  0.5,
			decayStep:      1,
			wantImportance: 0,
			wantActive:     false,
		},
		{
			name: "StayActiveAboveTarget",
			initialMemory: Memory{
				Content:    "good memory",
				ContextKey: "test:high",
				EntryType:  "fact",
			},
			eventsJSONL:    `{"timestamp":"2026-03-06T12:05:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"test:high","memory_ids":[%d],"latency_ms":10,"is_hit":true,"joules":0.5}`,
			targetHitRate:  0.1,
			decayStep:      1,
			wantImportance: 1,
			wantActive:     true,
		},
		{
			name: "PartialDecay",
			initialMemory: Memory{
				Content:    "fading memory",
				ContextKey: "test:fading",
				EntryType:  "fact",
			},
			eventsJSONL:    `{"timestamp":"2026-03-06T12:10:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"test:fading","memory_ids":[%d],"latency_ms":10,"is_hit":false,"joules":0.5}`,
			targetHitRate:  0.9,
			decayStep:      1,
			wantImportance: 1, // 2 - 1 = 1
			wantActive:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup for each sub-test
			subTmpDir, _ := os.MkdirTemp(tmpDir, tt.name)
			sqlitePath := filepath.Join(subTmpDir, "test.db")
			sqldb, _ := db.InitDB(sqlitePath)
			defer sqldb.Close()

			memorySvc := NewMemoryService(sqldb)
			analyticsSvc, _ := NewAnalyticsService(subTmpDir)
			defer analyticsSvc.Close()

			rates := &RateService{Card: RateCard{
				TargetHitRate: tt.targetHitRate,
				DecayStep:     tt.decayStep,
			}}

			refiner := NewKnowledgeRefiner(memorySvc, analyticsSvc, rates)

			// 1. Store memory (initial importance 1)
			memorySvc.StoreMemory(tt.initialMemory.Content, tt.initialMemory.ContextKey, tt.initialMemory.EntryType)

			// Special case for PartialDecay to have initial importance 2
			if tt.name == "PartialDecay" {
				memorySvc.StoreMemory(tt.initialMemory.Content, tt.initialMemory.ContextKey, tt.initialMemory.EntryType)
			}

			// Get the actual ID
			var memoryID int64
			sqldb.QueryRow("SELECT id FROM memories WHERE content = ?", tt.initialMemory.Content).Scan(&memoryID)

			// 2. Mock events with correct ID
			eventFile := filepath.Join(subTmpDir, "events.jsonl")
			eventData := fmt.Sprintf(tt.eventsJSONL, memoryID)
			os.WriteFile(eventFile, []byte(eventData), 0644)

			// 3. Run Refinement
			if err := refiner.Refine(); err != nil {
				t.Fatalf("Refinement failed: %v", err)
			}

			// 4. Verify
			var importance int
			var isActive bool
			err := sqldb.QueryRow("SELECT importance_score, is_active FROM memories WHERE id = ?", memoryID).Scan(&importance, &isActive)
			if err != nil {
				t.Fatalf("Failed to query memory: %v", err)
			}

			if importance != tt.wantImportance {
				t.Errorf("got importance %d, want %d", importance, tt.wantImportance)
			}
			if isActive != tt.wantActive {
				t.Errorf("got isActive %v, want %v", isActive, tt.wantActive)
			}
		})
	}
}

func TestRefine_NoEvents(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "refiner_no_events")
	defer os.RemoveAll(tmpDir)

	sqlitePath := filepath.Join(tmpDir, "test.db")
	sqldb, _ := db.InitDB(sqlitePath)
	defer sqldb.Close()

	memorySvc := NewMemoryService(sqldb)
	analyticsSvc, _ := NewAnalyticsService(tmpDir)
	defer analyticsSvc.Close()

	rates := &RateService{Card: RateCard{TargetHitRate: 0.5, DecayStep: 1}}
	refiner := NewKnowledgeRefiner(memorySvc, analyticsSvc, rates)

	if err := refiner.Refine(); err != nil {
		t.Errorf("Expected no error when events.jsonl is missing, got %v", err)
	}
}

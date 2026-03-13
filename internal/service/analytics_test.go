package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyticsService_TableDriven(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name          string
		eventsContent string
		query         string
		wantValue     interface{}
	}{
		{
			name:          "CountProjectCalls",
			eventsContent: `{"timestamp":"2026-03-06T10:00:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"project:echo","memory_ids":[1,2],"latency_ms":15,"is_hit":true,"joules":0.75}`,
			query:         "SELECT call_count FROM project_finops WHERE context_key = 'project:echo' AND agent = 'claude'",
			wantValue:     1,
		},
		{
			name:          "SumGlobalJoules",
			eventsContent: `{"timestamp":"2026-03-06T10:01:00Z","tool":"search","source_interface":"mcp","agent":"claude","context_key":"global","memory_ids":[],"latency_ms":50,"is_hit":false,"joules":2.5}`,
			query:         "SELECT total_joules FROM project_finops WHERE context_key = 'global'",
			wantValue:     2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subTmpDir, _ := os.MkdirTemp(tmpDir, tt.name)
			as, err := NewAnalyticsService(subTmpDir)
			if err != nil {
				t.Fatalf("Failed to create AnalyticsService: %v", err)
			}
			defer as.Close()

			err = os.WriteFile(filepath.Join(subTmpDir, "events.jsonl"), []byte(tt.eventsContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write sample events: %v", err)
			}

			if err := as.SyncEvents(); err != nil {
				t.Fatalf("Failed to sync events: %v", err)
			}

			var got interface{}
			switch tt.wantValue.(type) {
			case int:
				var val int
				err = as.db.QueryRow(tt.query).Scan(&val)
				got = val
			case float64:
				var val float64
				err = as.db.QueryRow(tt.query).Scan(&val)
				got = val
			}

			if err != nil {
				t.Fatalf("Failed to query view: %v", err)
			}

			if got != tt.wantValue {
				t.Errorf("got %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestGetProjectImpact_TableDriven(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "impact_test")
	defer os.RemoveAll(tmpDir)

	as, _ := NewAnalyticsService(tmpDir)
	defer as.Close()

	eventsContent := `{"timestamp":"2026-03-06T10:00:00Z","tool":"recall","source_interface":"mcp","agent":"claude","context_key":"project:echo","memory_ids":[1,2],"latency_ms":100,"is_hit":true,"joules":1.0}`
	os.WriteFile(filepath.Join(tmpDir, "events.jsonl"), []byte(eventsContent), 0644)
	as.SyncEvents()

	card := RateCard{
		ComputeCADPerMs:   0.001, // 100ms * 0.001 = 0.1
		EnergyCADPerJoule: 0.05,  // 1.0J * 0.05 = 0.05
		CarbonGPerJoule:   0.5,   // 1.0J * 0.5 = 0.5
	}

	tests := []struct {
		name       string
		contextKey string
		agent      string
		wantCost   float64
		wantCarbon float64
	}{
		{
			name:       "ProjectImpact",
			contextKey: "project:echo",
			agent:      "claude",
			wantCost:   0.15, // 0.1 + 0.05
			wantCarbon: 0.5,
		},
		{
			name:       "NoMatchContext",
			contextKey: "project:none",
			agent:      "claude",
			wantCost:   0,
			wantCarbon: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impacts, err := as.GetProjectImpact(card, tt.contextKey, tt.agent)
			if err != nil {
				t.Fatalf("GetProjectImpact failed: %v", err)
			}

			if tt.wantCost == 0 && len(impacts) == 0 {
				return
			}

			if len(impacts) != 1 {
				t.Fatalf("Expected 1 impact record, got %d", len(impacts))
			}

			const epsilon = 1e-9
			if (impacts[0].TotalCostCAD-tt.wantCost) > epsilon || (tt.wantCost-impacts[0].TotalCostCAD) > epsilon {
				t.Errorf("got cost %f, want %f", impacts[0].TotalCostCAD, tt.wantCost)
			}
			if (impacts[0].TotalCarbonG-tt.wantCarbon) > epsilon || (tt.wantCarbon-impacts[0].TotalCarbonG) > epsilon {
				t.Errorf("got carbon %f, want %f", impacts[0].TotalCarbonG, tt.wantCarbon)
			}
		})
	}
}

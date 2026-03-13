package service

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTelemetryService_TableDriven(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "telemetry_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name   string
		events []TelemetryEvent
	}{
		{
			name: "SingleEvent",
			events: []TelemetryEvent{
				{
					Tool:            "test_tool",
					SourceInterface: "test_cli",
					ContextKey:      "test:context",
					LatencyMs:       123.45,
					IsHit:           true,
					Joules:          6.1725,
				},
			},
		},
		{
			name: "MultipleEvents",
			events: []TelemetryEvent{
				{Tool: "tool1", ContextKey: "ctx1", LatencyMs: 10, IsHit: true},
				{Tool: "tool2", ContextKey: "ctx2", LatencyMs: 20, IsHit: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subTmpDir, _ := os.MkdirTemp(tmpDir, tt.name)
			ts, err := NewTelemetryService(subTmpDir, 10)
			if err != nil {
				t.Fatalf("Failed to create telemetry service: %v", err)
			}

			for _, event := range tt.events {
				ts.Emit(event)
			}

			// Give some time for background processing and close to flush
			time.Sleep(50 * time.Millisecond)
			ts.Close()

			logPath := filepath.Join(subTmpDir, "events.jsonl")
			f, err := os.Open(logPath)
			if err != nil {
				t.Fatalf("Failed to open log file: %v", err)
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			count := 0
			for scanner.Scan() {
				var decoded TelemetryEvent
				if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
					t.Fatalf("Failed to decode event %d: %v", count, err)
				}
				if decoded.Tool != tt.events[count].Tool {
					t.Errorf("Event %d: expected tool %s, got %s", count, tt.events[count].Tool, decoded.Tool)
				}
				count++
			}

			if count != len(tt.events) {
				t.Errorf("got %d events, want %d", count, len(tt.events))
			}
		})
	}
}

func TestCalculateJoules_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		tdpFactor float64
		latencyMs float64
		want      float64
	}{
		{"Standard", 0.05, 100.0, 5.0},
		{"HighTDP", 0.1, 100.0, 10.0},
		{"ZeroLatency", 0.05, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &TelemetryService{tdpFactor: tt.tdpFactor}
			got := ts.CalculateJoules(tt.latencyMs)
			if got != tt.want {
				t.Errorf("got %f joules, want %f", got, tt.want)
			}
		})
	}
}

package service

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTelemetryService(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "telemetry_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ts, err := NewTelemetryService(tmpDir, 10)
	if err != nil {
		t.Fatalf("Failed to create telemetry service: %v", err)
	}

	t.Run("EmitAndProcess", func(t *testing.T) {
		event := TelemetryEvent{
			Tool:            "test_tool",
			SourceInterface: "test_cli",
			ContextKey:      "test:context",
			LatencyMs:       123.45,
			IsHit:           true,
			Joules:          6.1725,
		}

		ts.Emit(event)

		// Give some time for background processing
		time.Sleep(100 * time.Millisecond)
		ts.Close()

		logPath := filepath.Join(tmpDir, "events.jsonl")
		f, err := os.Open(logPath)
		if err != nil {
			t.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		if !scanner.Scan() {
			t.Fatal("Expected at least one event in log file")
		}

		var decoded TelemetryEvent
		if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
			t.Fatalf("Failed to decode event: %v", err)
		}

		if decoded.Tool != event.Tool {
			t.Errorf("Expected tool %s, got %s", event.Tool, decoded.Tool)
		}
		if decoded.LatencyMs != event.LatencyMs {
			t.Errorf("Expected latency %f, got %f", event.LatencyMs, decoded.LatencyMs)
		}
	})
}

func TestCalculateJoules(t *testing.T) {
	ts := &TelemetryService{tdpFactor: 0.1}
	joules := ts.CalculateJoules(100.0)
	expected := 10.0
	if joules != expected {
		t.Errorf("Expected %f joules, got %f", expected, joules)
	}
}

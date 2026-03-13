package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TelemetryEvent represents a single analytical event.
type TelemetryEvent struct {
	Timestamp       string  `json:"timestamp"`        // ISO8601
	Tool            string  `json:"tool"`             // store, recall, search, etc.
	SourceInterface string  `json:"source_interface"` // mcp, cli, web
	ContextKey      string  `json:"context_key"`      // e.g., project:echo
	MemoryIDs       []int64 `json:"memory_ids"`       // Affected record IDs
	LatencyMs       float64 `json:"latency_ms"`       // Execution duration
	IsHit           bool    `json:"is_hit"`           // True if results found
	Joules          float64 `json:"joules"`           // Real or Synthetic
}

// TelemetryService handles asynchronous event emission to JSONL.
type TelemetryService struct {
	logPath   string
	eventChan chan TelemetryEvent
	wg        sync.WaitGroup
	quit      chan struct{}
	tdpFactor float64 // MilliJoules per millisecond
}

// NewTelemetryService creates a new TelemetryService.
func NewTelemetryService(dataDir string, bufferSize int) (*TelemetryService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create telemetry directory: %w", err)
	}

	ts := &TelemetryService{
		logPath:   filepath.Join(dataDir, "events.jsonl"),
		eventChan: make(chan TelemetryEvent, bufferSize),
		quit:      make(chan struct{}),
		tdpFactor: 0.05, // Default synthetic: 0.05 mJ/ms (exploratory value)
	}

	ts.wg.Add(1)
	go ts.processEvents()

	return ts, nil
}

// Emit sends an event to the background processor. Non-blocking unless buffer is full.
func (ts *TelemetryService) Emit(event TelemetryEvent) {
	select {
	case ts.eventChan <- event:
	default:
		log.Printf("Warning: Telemetry buffer full, dropping event: %s", event.Tool)
	}
}

// CalculateJoules returns synthetic energy consumption based on latency.
func (ts *TelemetryService) CalculateJoules(latencyMs float64) float64 {
	// Future: Check for Kepler /proc or metrics here.
	return latencyMs * ts.tdpFactor
}

// Close flushes remaining events and stops the background processor.
func (ts *TelemetryService) Close() {
	close(ts.quit)
	ts.wg.Wait()
}

func (ts *TelemetryService) processEvents() {
	defer ts.wg.Done()

	f, err := os.OpenFile(ts.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening telemetry log: %v", err)
		return
	}
	defer f.Close()

	for {
		select {
		case event := <-ts.eventChan:
			ts.writeEvent(f, event)
		case <-ts.quit:
			// Drain the channel before quitting
			for {
				select {
				case event := <-ts.eventChan:
					ts.writeEvent(f, event)
				default:
					return
				}
			}
		}
	}
}

func (ts *TelemetryService) writeEvent(f *os.File, event TelemetryEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().Format(time.RFC3339)
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling telemetry event: %v", err)
		return
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("Error writing telemetry event: %v", err)
	}
}

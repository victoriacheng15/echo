package service

import (
	"fmt"
	"log"
)

// KnowledgeRefiner orchestrates the feedback loop between analytics and state.
type KnowledgeRefiner struct {
	Memory    *MemoryService
	Analytics *AnalyticsService
	Rates     *RateService
}

// NewKnowledgeRefiner creates a new KnowledgeRefiner.
func NewKnowledgeRefiner(memory *MemoryService, analytics *AnalyticsService, rates *RateService) *KnowledgeRefiner {
	return &KnowledgeRefiner{
		Memory:    memory,
		Analytics: analytics,
		Rates:     rates,
	}
}

// Refine orchestrates a complete refinement cycle.
func (kr *KnowledgeRefiner) Refine() error {
	log.Println("Starting analytical knowledge refinement cycle...")

	// 1. Sync Telemetry to DuckDB
	if err := kr.Analytics.SyncEvents(); err != nil {
		return fmt.Errorf("failed to sync events during refinement: %w", err)
	}

	// 2. Identify Low-Signal Memories
	targetHitRate := kr.Rates.Card.TargetHitRate
	lowValueIDs, err := kr.Analytics.GetLowValueMemoryIDs(targetHitRate)
	if err != nil {
		return fmt.Errorf("failed to identify low-value memories: %w", err)
	}

	if len(lowValueIDs) == 0 {
		log.Println("No low-signal memories identified for decay.")
		return nil
	}

	log.Printf("Identified %d memories for analytical decay.", len(lowValueIDs))

	// 3. Apply Decay in SQLite
	decayStep := kr.Rates.Card.DecayStep
	if err := kr.Memory.DecayImportance(lowValueIDs, decayStep); err != nil {
		return fmt.Errorf("failed to apply importance decay: %w", err)
	}

	log.Println("Refinement cycle completed successfully.")
	return nil
}

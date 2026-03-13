package service

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

// AnalyticsService handles DuckDB-based analytical queries.
type AnalyticsService struct {
	db      *sql.DB
	dbPath  string
	dataDir string
}

// NewAnalyticsService initializes a new DuckDB analytics engine.
func NewAnalyticsService(dataDir string) (*AnalyticsService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create analytics directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "analytics.duckdb")
	// Open DuckDB with the CGO driver
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	as := &AnalyticsService{
		db:      db,
		dbPath:  dbPath,
		dataDir: dataDir,
	}

	if err := as.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize duckdb schema: %w", err)
	}

	return as, nil
}

// Close closes the DuckDB connection.
func (as *AnalyticsService) Close() error {
	return as.db.Close()
}

// SyncEvents ingests new data from events.jsonl into DuckDB.
// This uses DuckDB's native high-performance JSON reader.
func (as *AnalyticsService) SyncEvents() error {
	eventLog := filepath.Join(as.dataDir, "events.jsonl")

	// Check if file exists
	if _, err := os.Stat(eventLog); os.IsNotExist(err) {
		return nil // Nothing to sync
	}

	// Create a temporary table to load the JSONL data
	// read_json_auto is extremely fast and handles schema inference.
	query := fmt.Sprintf(`
		CREATE OR REPLACE TABLE events_staging AS 
		SELECT * FROM read_json_auto('%s');

		-- Insert only new events (based on timestamp)
		-- In a real scenario, we might want a more robust offset management.
		INSERT INTO events 
		SELECT s.* FROM events_staging s
		WHERE NOT EXISTS (
			SELECT 1 FROM events e 
			WHERE e.timestamp = s.timestamp 
			AND e.tool = s.tool 
			AND e.context_key = s.context_key
		);

		DROP TABLE events_staging;
	`, eventLog)

	_, err := as.db.Exec(query)
	return err
}

func (as *AnalyticsService) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		timestamp TIMESTAMP,
		tool VARCHAR,
		source_interface VARCHAR,
		context_key VARCHAR,
		memory_ids BIGINT[],
		latency_ms DOUBLE,
		is_hit BOOLEAN,
		joules DOUBLE
	);

	-- Analytical View for Project FinOps
	CREATE OR REPLACE VIEW project_finops AS
	SELECT 
		context_key,
		COUNT(*) as call_count,
		SUM(latency_ms) as total_latency_ms,
		SUM(joules) as total_joules,
		0.0 as total_cost_cad,
		0.0 as total_carbon_g,
		CAST(SUM(CASE WHEN is_hit THEN 1 ELSE 0 END) AS DOUBLE) / COUNT(*) as hit_rate
	FROM events
	GROUP BY context_key;
	`
	_, err := as.db.Exec(query)
	return err
}

// ProjectImpact represents the economic and environmental cost of a project's context usage.
type ProjectImpact struct {
	ContextKey     string  `json:"context_key"`
	CallCount      int     `json:"call_count"`
	TotalCostCAD   float64 `json:"total_cost_cad"`
	TotalCarbonG   float64 `json:"total_carbon_g"`
	AverageHitRate float64 `json:"average_hit_rate"`
}

// GetProjectImpact calculates the real-world impact using the provided rate card.
// Supports optional filtering by context_key.
func (as *AnalyticsService) GetProjectImpact(card RateCard, contextKey string) ([]ProjectImpact, error) {
	baseQuery := `
	SELECT 
		context_key,
		call_count,
		(total_latency_ms * ?) + (total_joules * ?) as total_cost_cad,
		(total_joules * ?) as total_carbon_g,
		hit_rate
	FROM project_finops
	WHERE 1=1
	`
	args := []interface{}{card.ComputeCADPerMs, card.EnergyCADPerJoule, card.CarbonGPerJoule}

	if contextKey != "" {
		baseQuery += " AND context_key = ?"
		args = append(args, contextKey)
	}

	rows, err := as.db.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var impacts []ProjectImpact
	for rows.Next() {
		var pi ProjectImpact
		if err := rows.Scan(&pi.ContextKey, &pi.CallCount, &pi.TotalCostCAD, &pi.TotalCarbonG, &pi.AverageHitRate); err != nil {
			return nil, err
		}
		impacts = append(impacts, pi)
	}
	return impacts, nil
}

// GetLowValueMemoryIDs identifies memory IDs that are candidates for decay.
// Criteria: Context hit rate < target OR specific memory ID hasn't appeared in a hit recently.
func (as *AnalyticsService) GetLowValueMemoryIDs(targetHitRate float64) ([]int64, error) {
	// Identify contexts that are below the performance baseline
	query := `
	WITH context_performance AS (
		SELECT context_key, hit_rate FROM project_finops
	),
	low_performance_memories AS (
		-- Get all IDs from contexts that are under-performing
		SELECT DISTINCT unnest(e.memory_ids) as id
		FROM events e
		JOIN context_performance cp ON e.context_key = cp.context_key
		WHERE cp.hit_rate < ?
	)
	SELECT id FROM low_performance_memories;
	`

	rows, err := as.db.Query(query, targetHitRate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite database, sets WAL mode, and runs migrations.
func InitDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}
	// Connect with WAL mode and busy timeout
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database at %s: %w", dbPath, err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("Database initialized at: %s", dbPath)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL CHECK (length(content) > 0 AND length(content) <= 8192),
		context_key TEXT NOT NULL CHECK (length(context_key) > 0),
		entry_type TEXT DEFAULT 'instruction' CHECK (
			entry_type IN ('instruction', 'snippet', 'request', 'sentence', 'boilerplate')
		),
		usage_count INTEGER DEFAULT 1,
		last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT CHECK (metadata IS NULL OR json_valid(metadata)),
		UNIQUE(content, context_key)
	);

	CREATE INDEX IF NOT EXISTS idx_context_relevance ON memories(context_key, usage_count DESC, last_used DESC);
	CREATE INDEX IF NOT EXISTS idx_last_used ON memories(last_used DESC);
	`

	_, err := db.Exec(query)
	return err
}

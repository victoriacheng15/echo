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
	-- v1.1.1 Schema
	CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL CHECK (length(content) > 0 AND length(content) <= 8192),
		context_key TEXT NOT NULL CHECK (length(context_key) > 0),
		entry_type TEXT DEFAULT 'directive' CHECK (entry_type IN ('directive', 'artifact', 'fact')),
		importance_score INTEGER DEFAULT 1 CHECK (importance_score BETWEEN 0 AND 10),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		source TEXT DEFAULT 'mcp' CHECK (source IN ('mcp', 'cli')),
		is_active BOOLEAN DEFAULT 1,
		tags TEXT CHECK (tags IS NULL OR json_valid(tags)),
		UNIQUE(content, context_key)
	);

	CREATE INDEX IF NOT EXISTS idx_context_relevance ON memories(context_key, importance_score DESC);
	CREATE INDEX IF NOT EXISTS idx_content ON memories(content);
	CREATE INDEX IF NOT EXISTS idx_is_active ON memories(is_active);

	-- FTS5 Virtual Table for high-speed searching
	CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		content,
		context_key,
		is_active,
		importance_score,
		content='memories',
		content_rowid='id'
	);

	-- Triggers to keep FTS index in sync
	DROP TRIGGER IF EXISTS memories_ai;
	CREATE TRIGGER memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, content, context_key, is_active, importance_score) VALUES (new.id, new.content, new.context_key, new.is_active, new.importance_score);
	END;

	DROP TRIGGER IF EXISTS memories_ad;
	CREATE TRIGGER memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content, context_key, is_active, importance_score) VALUES('delete', old.id, old.content, old.context_key, old.is_active, old.importance_score);
	END;

	DROP TRIGGER IF EXISTS memories_au;
	CREATE TRIGGER memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, content, context_key, is_active, importance_score) VALUES('delete', old.id, old.content, old.context_key, old.is_active, old.importance_score);
		INSERT INTO memories_fts(rowid, content, context_key, is_active, importance_score) VALUES (new.id, new.content, new.context_key, new.is_active, new.importance_score);
	END;

	-- Backfill FTS index with existing data if they are missing
	INSERT INTO memories_fts(rowid, content, context_key, is_active, importance_score)
	SELECT id, content, context_key, is_active, importance_score FROM memories
	WHERE id NOT IN (SELECT rowid FROM memories_fts);
	`

	_, err := db.Exec(query)
	return err
}

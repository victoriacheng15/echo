package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Valid entry types for memories.
var validEntryTypes = map[string]bool{
	"directive": true,
	"artifact":  true,
	"fact":      true,
}

var contextKeyRegex = regexp.MustCompile(`^[a-z]+:[a-z0-9-_/]+$`)

// Memory represents a stored memory.
type Memory struct {
	ID              int      `json:"id"`
	Content         string   `json:"content"`
	ContextKey      string   `json:"context_key"`
	EntryType       string   `json:"entry_type"`
	ImportanceScore int      `json:"importance_score"`
	CreatedAt       string   `json:"created_at"`
	Source          string   `json:"source"`
	IsActive        bool     `json:"is_active"`
	Tags            []string `json:"tags,omitempty"`
}

// MemoryService provides business logic for memory management.
type MemoryService struct {
	db     *sql.DB
	Source string // Default source for new memories (e.g., "mcp", "cli")
}

// NewMemoryService creates a new MemoryService.
func NewMemoryService(db *sql.DB) *MemoryService {
	return &MemoryService{
		db:     db,
		Source: "mcp",
	}
}

// ValidateMemory checks if the memory fields are valid.
func (s *MemoryService) ValidateMemory(content, contextKey, entryType string) error {
	if len(content) < 1 || len(content) > 8192 {
		return errors.New("content must be between 1 and 8,192 characters")
	}

	if contextKey != "global" && !contextKeyRegex.MatchString(contextKey) {
		return errors.New("context_key must follow 'type:identifier' format or be 'global'")
	}

	if !validEntryTypes[entryType] {
		return fmt.Errorf("invalid entry_type: %s", entryType)
	}

	return nil
}

// StoreMemory saves or updates a contextual memory.
func (s *MemoryService) StoreMemory(content, contextKey, entryType string) error {
	return s.StoreMemoryWithTags(content, contextKey, entryType, nil)
}

// StoreMemoryWithTags saves or updates a contextual memory with optional tags.
func (s *MemoryService) StoreMemoryWithTags(content, contextKey, entryType string, tags []string) error {
	if err := s.ValidateMemory(content, contextKey, entryType); err != nil {
		return err
	}

	query := `
	INSERT INTO memories (content, context_key, entry_type, source, tags)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(content, context_key) DO UPDATE SET
		entry_type = excluded.entry_type,
		tags = COALESCE(excluded.tags, memories.tags),
		importance_score = MIN(memories.importance_score + 1, 10),
		is_active = 1;
	`

	var tagsJSON sql.NullString
	if len(tags) > 0 {
		data, err := json.Marshal(tags)
		if err == nil {
			tagsJSON = sql.NullString{String: string(data), Valid: true}
		}
	}

	_, err := s.db.Exec(query, content, contextKey, entryType, s.Source, tagsJSON)
	return err
}

// RecallMemory retrieves the most relevant memories for the provided context keys.
func (s *MemoryService) RecallMemory(contextKeys []string, limit int) ([]Memory, error) {
	if len(contextKeys) == 0 {
		return nil, errors.New("context_keys cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	// Use placeholder for context keys
	placeholders := make([]string, len(contextKeys))
	args := make([]interface{}, len(contextKeys))
	for i, key := range contextKeys {
		placeholders[i] = "?"
		args[i] = key
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
	SELECT id, content, context_key, entry_type, importance_score, created_at, source, is_active, tags
	FROM memories
	WHERE context_key IN (%s) AND is_active = 1
	ORDER BY importance_score DESC, created_at DESC
	LIMIT ?;
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

func (s *MemoryService) SearchMemories(query string) ([]Memory, error) {
	var sqlQuery string
	var args []interface{}

	// For very short queries, use LIKE to ensure substring matching
	if len(query) < 3 {
		sqlQuery = `
		SELECT id, content, context_key, entry_type, importance_score, created_at, source, is_active, tags
		FROM memories
		WHERE content LIKE ? AND is_active = 1
		ORDER BY importance_score DESC, created_at DESC;
		`
		args = []interface{}{"%" + query + "%"}
	} else {
		// For longer queries, use FTS5
		// Include is_active:1 in the match to ensure FTS-native filtering speed
		ftsQuery := fmt.Sprintf("is_active:1 AND \"%s\"*", strings.ReplaceAll(query, "\"", ""))
		sqlQuery = `
		SELECT memories.id, memories.content, memories.context_key, memories.entry_type, 
		       memories.importance_score, memories.created_at, memories.source, memories.is_active, memories.tags
		FROM memories
		JOIN memories_fts ON memories.id = memories_fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY memories_fts.importance_score DESC
		LIMIT 100;
		`
		args = []interface{}{ftsQuery}
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

func (s *MemoryService) scanMemories(rows *sql.Rows) ([]Memory, error) {
	var memories []Memory
	for rows.Next() {
		var m Memory
		var tags sql.NullString
		if err := rows.Scan(
			&m.ID, &m.Content, &m.ContextKey, &m.EntryType,
			&m.ImportanceScore, &m.CreatedAt, &m.Source, &m.IsActive, &tags,
		); err != nil {
			return nil, err
		}
		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &m.Tags)
		}
		memories = append(memories, m)
	}
	return memories, nil
}

// DeleteMemory removes a specific memory from the database.
func (s *MemoryService) DeleteMemory(content, contextKey string) error {
	query := `DELETE FROM memories WHERE content = ? AND context_key = ?;`
	_, err := s.db.Exec(query, content, contextKey)
	return err
}

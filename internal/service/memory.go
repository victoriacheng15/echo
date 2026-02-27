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
	"instruction": true,
	"snippet":     true,
	"request":     true,
	"sentence":    true,
	"boilerplate": true,
}

var contextKeyRegex = regexp.MustCompile(`^[a-z]+:[a-z0-9-_/]+$`)

// Memory represents a stored memory.
type Memory struct {
	ID         int    `json:"id"`
	Content    string `json:"content"`
	ContextKey string `json:"context_key"`
	EntryType  string `json:"entry_type"`
	UsageCount int    `json:"usage_count"`
	LastUsed   string `json:"last_used"`
	Metadata   string `json:"metadata,omitempty"`
}

// MemoryService provides business logic for memory management.
type MemoryService struct {
	db *sql.DB
}

// NewMemoryService creates a new MemoryService.
func NewMemoryService(db *sql.DB) *MemoryService {
	return &MemoryService{db: db}
}

// ValidateMemory checks if the memory fields are valid.
func (s *MemoryService) ValidateMemory(content, contextKey, entryType, metadata string) error {
	if len(content) < 1 || len(content) > 8192 {
		return errors.New("content must be between 1 and 8,192 characters")
	}

	if contextKey != "global" && !contextKeyRegex.MatchString(contextKey) {
		return errors.New("context_key must follow 'type:identifier' format or be 'global'")
	}

	if !validEntryTypes[entryType] {
		return fmt.Errorf("invalid entry_type: %s", entryType)
	}

	if metadata != "" {
		var js json.RawMessage
		if err := json.Unmarshal([]byte(metadata), &js); err != nil {
			return errors.New("metadata must be a valid JSON object")
		}
	}

	return nil
}

// StoreMemory saves or updates a contextual memory.
func (s *MemoryService) StoreMemory(content, contextKey, entryType, metadata string) error {
	if err := s.ValidateMemory(content, contextKey, entryType, metadata); err != nil {
		return err
	}

	query := `
	INSERT INTO memories (content, context_key, entry_type, metadata)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(content, context_key) DO UPDATE SET
		usage_count = usage_count + 1,
		last_used = CURRENT_TIMESTAMP,
		metadata = excluded.metadata,
		entry_type = excluded.entry_type;
	`

	var meta sql.NullString
	if metadata != "" {
		meta = sql.NullString{String: metadata, Valid: true}
	}

	_, err := s.db.Exec(query, content, contextKey, entryType, meta)
	return err
}

// RecallMemory retrieves the most relevant memories for the provided context keys.
func (s *MemoryService) RecallMemory(contextKeys []string, limit int) ([]Memory, error) {
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
	SELECT id, content, context_key, entry_type, usage_count, last_used, metadata
	FROM memories
	WHERE context_key IN (%s)
	ORDER BY usage_count DESC, last_used DESC
	LIMIT ?;
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		var metadata sql.NullString
		if err := rows.Scan(&m.ID, &m.Content, &m.ContextKey, &m.EntryType, &m.UsageCount, &m.LastUsed, &metadata); err != nil {
			return nil, err
		}
		if metadata.Valid {
			m.Metadata = metadata.String
		}
		memories = append(memories, m)
	}

	return memories, nil
}

// SearchMemories performs a full-text search across all stored memories.
func (s *MemoryService) SearchMemories(query string) ([]Memory, error) {
	sqlQuery := `
	SELECT id, content, context_key, entry_type, usage_count, last_used, metadata
	FROM memories
	WHERE content LIKE ?
	ORDER BY usage_count DESC, last_used DESC;
	`

	rows, err := s.db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		var metadata sql.NullString
		if err := rows.Scan(&m.ID, &m.Content, &m.ContextKey, &m.EntryType, &m.UsageCount, &m.LastUsed, &metadata); err != nil {
			return nil, err
		}
		if metadata.Valid {
			m.Metadata = metadata.String
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

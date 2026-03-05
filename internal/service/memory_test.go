package service

import (
	"echo/internal/db"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMemoryService(t *testing.T) {
	dbPath := "test_service.db"
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}()

	sqldb, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	svc := NewMemoryService(sqldb)

	cleanup := func() {
		sqldb.Exec("DELETE FROM memories")
	}

	t.Run("ValidateMemory", func(t *testing.T) {
		tests := []struct {
			name       string
			content    string
			contextKey string
			entryType  string
			wantErr    bool
		}{
			{"valid", "hello", "project:echo", "directive", false},
			{"too_long", string(make([]byte, 8193)), "global", "directive", true},
			{"invalid_key", "hello", "foo", "directive", true},
			{"invalid_type", "hello", "global", "foo", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := svc.ValidateMemory(tt.content, tt.contextKey, tt.entryType); (err != nil) != tt.wantErr {
					t.Errorf("ValidateMemory() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("StoreMemory", func(t *testing.T) {
		cleanup()
		tests := []struct {
			name       string
			content    string
			contextKey string
			entryType  string
			wantErr    bool
		}{
			{"initial insert", "first memory", "global", "directive", false},
			{"upsert update", "first memory", "global", "directive", false},
			{"new entry", "second memory", "project:echo", "artifact", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := svc.StoreMemory(tt.content, tt.contextKey, tt.entryType); (err != nil) != tt.wantErr {
					t.Fatalf("StoreMemory() error = %v, wantErr %v", err, tt.wantErr)
				}

				if !tt.wantErr {
					var content string
					err = sqldb.QueryRow("SELECT content FROM memories WHERE content = ? AND context_key = ?", tt.content, tt.contextKey).Scan(&content)
					if err != nil {
						t.Fatalf("Failed to query memory: %v", err)
					}
				}
			})
		}

		t.Run("StoreMemory DB error", func(t *testing.T) {
			// Create a service with a closed DB
			sqldb.Close()
			defer func() {
				// Re-open for following tests
				sqldb, _ = db.InitDB(dbPath)
				svc = NewMemoryService(sqldb)
			}()
			err := svc.StoreMemory("fail", "global", "directive")
			if err == nil {
				t.Error("Expected DB error for closed connection, got nil")
			}
		})

		t.Run("Idempotency and Reinforcement", func(t *testing.T) {
			cleanup()
			baseContent := "reinforcement-target"
			baseKey := "project:echo"

			tests := []struct {
				name          string
				inputContent  string
				inputKey      string
				expectedScore int
			}{
				{"initial_store_untrimmed", baseContent + "  ", "Project:Echo", 1},
				{"reinforce_with_leading_space", "  " + baseContent, baseKey, 2},
				{"reinforce_with_case_variation", baseContent, "PROJECT:ECHO", 3},
				{"reinforce_with_mixed_noise", " " + baseContent + " ", " PrOjEcT:eChO ", 4},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					err := svc.StoreMemory(tt.inputContent, tt.inputKey, "directive")
					if err != nil {
						t.Fatalf("StoreMemory failed: %v", err)
					}

					var score int
					err = sqldb.QueryRow("SELECT importance_score FROM memories WHERE content = ? AND context_key = ?", baseContent, baseKey).Scan(&score)
					if err != nil {
						t.Fatalf("Failed to query memory: %v", err)
					}
					if score != tt.expectedScore {
						t.Errorf("Expected score %d, got %d", tt.expectedScore, score)
					}
				})
			}
		})
	})

	t.Run("UpdateMemoryContentByID table-driven", func(t *testing.T) {
		cleanup()
		initialContent := "original content"
		svc.StoreMemory(initialContent, "global", "directive")

		var id int64
		err := sqldb.QueryRow("SELECT id FROM memories WHERE content = ?", initialContent).Scan(&id)
		if err != nil {
			t.Fatalf("Failed to query memory ID: %v", err)
		}

		tests := []struct {
			name        string
			newContent  string
			wantErr     bool
			checkResult bool
		}{
			{"valid update", "updated content", false, true},
			{"empty content", "", true, false},
			{"very long content", string(make([]byte, 8193)), true, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := svc.UpdateMemoryContentByID(id, tt.newContent)
				if (err != nil) != tt.wantErr {
					t.Errorf("UpdateMemoryContentByID() error = %v, wantErr %v", err, tt.wantErr)
				}

				if tt.checkResult && !tt.wantErr {
					var updatedContent string
					err = sqldb.QueryRow("SELECT content FROM memories WHERE id = ?", id).Scan(&updatedContent)
					if err != nil {
						t.Fatalf("Failed to query updated content: %v", err)
					}
					if updatedContent != tt.newContent {
						t.Errorf("Expected content %q, got %q", tt.newContent, updatedContent)
					}
				}
			})
		}
	})

	t.Run("RecallMemory", func(t *testing.T) {
		cleanup()
		// Seed data for RecallMemory
		seedData := []struct {
			content    string
			contextKey string
		}{
			{"memory A", "project:a"},
			{"memory B", "project:b"},
			{"memory C", "global"},
		}
		for _, sd := range seedData {
			svc.StoreMemory(sd.content, sd.contextKey, "directive")
		}

		tests := []struct {
			name        string
			contextKeys []string
			limit       int
			wantCount   int
		}{
			{"single context", []string{"project:a"}, 10, 1},
			{"multiple contexts", []string{"project:a", "global"}, 10, 2},
			{"no match", []string{"project:none"}, 10, 0},
			{"limit check", []string{"project:a", "project:b", "global"}, 1, 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				memories, err := svc.RecallMemory(tt.contextKeys, tt.limit)
				if err != nil {
					t.Fatalf("RecallMemory() error = %v", err)
				}
				if len(memories) != tt.wantCount {
					t.Errorf("RecallMemory() got %d memories, want %d", len(memories), tt.wantCount)
				}
			})
		}

		t.Run("RecallMemory error", func(t *testing.T) {
			_, err := svc.RecallMemory([]string{}, 10)
			if err == nil {
				t.Error("Expected error for empty context_keys, got nil")
			}
		})

		t.Run("RecallMemory DB error", func(t *testing.T) {
			sqldb.Close()
			_, err := svc.RecallMemory([]string{"global"}, 10)
			if err == nil {
				t.Error("Expected DB error for closed connection, got nil")
			}
			sqldb, _ = db.InitDB(dbPath)
			svc = NewMemoryService(sqldb)
		})
	})

	t.Run("SearchMemories", func(t *testing.T) {
		cleanup()
		// Seed data for SearchMemories
		seedData := []string{"apple pie", "banana split", "cherry tart"}
		for _, s := range seedData {
			svc.StoreMemory(s, "global", "directive")
		}

		tests := []struct {
			name      string
			query     string
			wantCount int
		}{
			{"exact match part", "apple", 1},
			{"partial match", "a", 3}, // apple, banana, cherry all have 'a'
			{"no match", "zucchini", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				memories, err := svc.SearchMemories(tt.query)
				if err != nil {
					t.Fatalf("SearchMemories() error = %v", err)
				}
				if len(memories) != tt.wantCount {
					t.Errorf("SearchMemories() got %d memories, want %d", len(memories), tt.wantCount)
				}
			})
		}

		t.Run("SearchMemories DB error", func(t *testing.T) {
			sqldb.Close()
			_, err := svc.SearchMemories("test")
			if err == nil {
				t.Error("Expected DB error for closed connection, got nil")
			}
			sqldb, _ = db.InitDB(dbPath)
			svc = NewMemoryService(sqldb)
		})
	})

	t.Run("DeleteMemory", func(t *testing.T) {
		cleanup()
		svc.StoreMemory("delete me", "global", "directive")

		tests := []struct {
			name       string
			content    string
			contextKey string
			wantExists bool
		}{
			{"delete existing", "delete me", "global", false},
			{"delete non-existing", "non-existent", "global", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := svc.DeleteMemory(tt.content, tt.contextKey); err != nil {
					t.Fatalf("DeleteMemory() error = %v", err)
				}

				var count int
				sqldb.QueryRow("SELECT COUNT(*) FROM memories WHERE content = ? AND context_key = ?", tt.content, tt.contextKey).Scan(&count)
				if tt.wantExists && count == 0 {
					t.Errorf("Expected memory to exist, but it was deleted")
				}
				if !tt.wantExists && count != 0 {
					t.Errorf("Expected memory to be deleted, but it still exists")
				}
			})
		}

		t.Run("DeleteMemory DB error", func(t *testing.T) {
			sqldb.Close()
			err := svc.DeleteMemory("fail", "global")
			if err == nil {
				t.Error("Expected DB error for closed connection, got nil")
			}
			sqldb, _ = db.InitDB(dbPath)
			svc = NewMemoryService(sqldb)
		})
	})
}

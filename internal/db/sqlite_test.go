package db

import (
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	dbPath := "test_init.db"
	defer os.Remove(dbPath)

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	t.Run("CheckTableExistence", func(t *testing.T) {
		tests := []struct {
			tableName string
		}{
			{"memories"},
		}

		for _, tt := range tests {
			t.Run(tt.tableName, func(t *testing.T) {
				var name string
				err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tt.tableName).Scan(&name)
				if err != nil {
					t.Errorf("Table '%s' not found: %v", tt.tableName, err)
				}
			})
		}
	})

	t.Run("CheckPragmas", func(t *testing.T) {
		tests := []struct {
			pragma string
			want   string
		}{
			{"journal_mode", "wal"},
		}

		for _, tt := range tests {
			t.Run(tt.pragma, func(t *testing.T) {
				var got string
				err = db.QueryRow("PRAGMA " + tt.pragma).Scan(&got)
				if err != nil {
					t.Fatalf("Failed to query pragma %s: %v", tt.pragma, err)
				}
				if got != tt.want {
					t.Errorf("Pragma %s: got %s, want %s", tt.pragma, got, tt.want)
				}
			})
		}
	})

	t.Run("InvalidPath", func(t *testing.T) {
		_, err := InitDB("")
		if err == nil {
			t.Error("Expected error for empty path, got nil")
		}
	})

	t.Run("MigrationError", func(t *testing.T) {
		// We can test this by providing a read-only DB or a closed one
		dbPath := "readonly.db"
		os.WriteFile(dbPath, []byte("garbage"), 0400)
		defer os.Remove(dbPath)
		_, err := InitDB(dbPath)
		if err == nil {
			t.Error("Expected migration error for garbage file, got nil")
		}
	})
}

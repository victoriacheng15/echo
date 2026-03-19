package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultDataDir(t *testing.T) {
	t.Run("XDG_DATA_HOME set", func(t *testing.T) {
		old := os.Getenv("XDG_DATA_HOME")
		tempDir := t.TempDir()
		os.Setenv("XDG_DATA_HOME", tempDir)
		defer os.Setenv("XDG_DATA_HOME", old)

		path := GetDefaultDataDir()
		expected := filepath.Join(tempDir, "echo")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})

	t.Run("XDG_DATA_HOME empty", func(t *testing.T) {
		old := os.Getenv("XDG_DATA_HOME")
		os.Setenv("XDG_DATA_HOME", "")
		defer os.Setenv("XDG_DATA_HOME", old)

		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Skipping test: User home directory not available")
		}

		path := GetDefaultDataDir()
		expected := filepath.Join(home, ".local", "share", "echo")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})
}

func TestGetDefaultDBPath(t *testing.T) {
	old := os.Getenv("XDG_DATA_HOME")
	tempDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", old)

	path := GetDefaultDBPath()
	expected := filepath.Join(tempDir, "echo", "echo.db")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

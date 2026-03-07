package db

import (
	"os"
	"path/filepath"
)

// GetDefaultDataDir returns the base directory for Echo's persistent data (~/.local/share/echo).
func GetDefaultDataDir() string {
	// 1. Respect XDG_DATA_HOME if set
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// 2. Fall back to ~/.local/share
		home, err := os.UserHomeDir()
		if err != nil {
			return "." // Final fallback to current directory
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "echo")
}

// GetDefaultDBPath returns the standard path for the Echo SQLite database.
func GetDefaultDBPath() string {
	return filepath.Join(GetDefaultDataDir(), "echo.db")
}

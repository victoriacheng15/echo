package cli

import (
	"echo/internal/service"
	"testing"
)

// TestDispatcher_Run_UnknownCommand verifies that unknown commands return an error.
func TestDispatcher_Run_UnknownCommand(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{"unknown-cmd"})
	if err == nil {
		t.Error("Expected error for unknown command, got nil")
	}
}

// TestDispatcher_Run_NoCommand verifies that providing no command returns an error.
func TestDispatcher_Run_NoCommand(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{})
	if err == nil {
		t.Error("Expected error for no command, got nil")
	}
}

// TestDispatcher_Run_Help verifies that the help command does not return an error.
func TestDispatcher_Run_Help(t *testing.T) {
	d := NewDispatcher(nil, nil, nil, "table")
	err := d.Run([]string{"help"})
	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}
}

// Note: Future tests will use a mock or temporary SQLite database to verify
// that subcommands correctly invoke MemoryService methods.
func TestDispatcher_SubcommandSkeletons(t *testing.T) {
	// Initialize with nil services just to test routing logic for now
	d := NewDispatcher(&service.MemoryService{}, nil, nil, "table")
	
	commands := []string{"store", "recall", "search", "delete", "maintain"}
	
	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			err := d.Run([]string{cmd})
			if err != nil {
				t.Errorf("Subcommand %s failed unexpectedly: %v", cmd, err)
			}
		})
	}
}

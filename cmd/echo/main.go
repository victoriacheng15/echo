package main

import (
	"echo/internal/cli"
	"echo/internal/db"
	"echo/internal/mcp"
	"echo/internal/service"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	mcp_server "github.com/mark3labs/mcp-go/server"
)

func main() {
	// 1. Initialize Global Flags
	defaultDB := db.GetDefaultDBPath()
	dbPath := flag.String("db", defaultDB, "Path to the SQLite database file")
	outputFormat := flag.String("o", "table", "Output format (table, json, csv)")
	flag.StringVar(outputFormat, "output", "table", "Output format (table, json, csv)")

	// Custom usage for the unified binary (Human-facing only)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: echo [global flags] <command> [subcommand flags]\n\n")
		fmt.Fprintf(os.Stderr, "Global Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  store       CLI: Save a new memory\n")
		fmt.Fprintf(os.Stderr, "  recall      CLI: Retrieve memories\n")
		fmt.Fprintf(os.Stderr, "  search      CLI: Keyword search\n")
		fmt.Fprintf(os.Stderr, "  delete      CLI: Remove a memory\n")
		fmt.Fprintf(os.Stderr, "  maintain    CLI: DB maintenance\n")
	}

	// Parse global flags but stop at the first non-flag argument (the subcommand)
	flag.Parse()

	// 2. Bootstrap Shared Services (Shared by MCP and CLI)
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	sqldb, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	dataDir := db.GetDefaultDataDir()
	// RateService now uses embedded defaults (Zero-Config)
	rateSvc := service.NewRateService()

	analyticsSvc, err := service.NewAnalyticsService(dataDir)
	if err != nil {
		log.Printf("Warning: AnalyticsService initialization failed: %v", err)
	}
	if analyticsSvc != nil {
		defer analyticsSvc.Close()
	}

	memorySvc := service.NewMemoryService(sqldb)

	// 3. Smart TTY Routing (Separate Human and AI interfaces)
	if flag.NArg() == 0 {
		// If Stdin is a Terminal (Human), show help/error
		if isTerminal(os.Stdin) {
			fmt.Fprintf(os.Stderr, "Error: Subcommand required.\n\n")
			flag.Usage()
			os.Exit(1)
		}

		// If Stdin is NOT a terminal (AI Agent Host), start MCP Server
		log.SetOutput(os.Stderr) // Redirect logs to stderr for MCP JSON-RPC
		s := mcp.NewServer(memorySvc, analyticsSvc, rateSvc)
		log.Printf("Echo MCP Server starting (DB: %s)...", *dbPath)
		if err := mcp_server.ServeStdio(s); err != nil {
			log.Fatalf("MCP Server error: %v", err)
		}
		return
	}

	// 4. Dispatch to CLI Subcommands (Human-facing)
	command := flag.Arg(0)
	switch command {
	case "store", "recall", "search", "delete", "maintain", "help":
		// CLI Dispatcher (Human interface)
		dispatcher := cli.NewDispatcher(memorySvc, analyticsSvc, rateSvc, *outputFormat)
		if err := dispatcher.Run(flag.Args()); err != nil {
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command %q\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

// isTerminal checks if the given file is a terminal/TTY.
func isTerminal(f *os.File) bool {
	stat, _ := f.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

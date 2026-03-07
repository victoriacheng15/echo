package main

import (
	"echo/internal/db"
	"echo/internal/mcp"
	"echo/internal/service"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	defaultDB := db.GetDefaultDBPath()
	dbPath := flag.String("db", defaultDB, "Path to the SQLite database file")
	flag.Parse()

	// Redirect logs to stderr to avoid corrupting MCP JSON-RPC on stdout
	log.SetOutput(os.Stderr)

	// Ensure the database directory exists
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize Database
	sqldb, err := db.InitDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	// Initialize Service
	dataDir := db.GetDefaultDataDir()
	telemetrySvc, err := service.NewTelemetryService(dataDir, 100)
	if err != nil {
		log.Printf("Warning: Failed to initialize telemetry: %v", err)
	} else {
		defer telemetrySvc.Close()
	}

	rateSvc, err := service.NewRateService(filepath.Join("configs", "rates.yml"))
	if err != nil {
		log.Printf("Warning: Failed to initialize rate card: %v", err)
	}

	analyticsSvc, err := service.NewAnalyticsService(dataDir)
	if err != nil {
		log.Printf("Warning: Failed to initialize analytics engine: %v", err)
	} else {
		defer analyticsSvc.Close()
	}

	memorySvc := service.NewMemoryService(sqldb).WithTelemetry(telemetrySvc)

	// Create Knowledge Refiner
	if analyticsSvc != nil && rateSvc != nil {
		refiner := service.NewKnowledgeRefiner(memorySvc, analyticsSvc, rateSvc)
		// Milestone 4: Start background refiner loop (Simple check every 50 events)
		// For now, we'll trigger it once at startup for demonstration.
		go func() {
			if err := refiner.Refine(); err != nil {
				log.Printf("Knowledge Refiner error: %v", err)
			}
		}()
	}

	// Create MCP Server
	s := mcp.NewServer(memorySvc)

	log.Printf("Echo MCP Server starting (DB: %s)...", *dbPath)

	// Start the server using Stdio transport
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

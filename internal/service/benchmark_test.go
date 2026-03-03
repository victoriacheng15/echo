package service

import (
	"echo/internal/db"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func BenchmarkRecallMemory(b *testing.B) {
	dbPath := "benchmark_recall.db"
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}()

	sqldb, err := db.InitDB(dbPath)
	if err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	svc := NewMemoryService(sqldb)

	// Seed 1,000 memories to simulate a production-like load
	for i := 0; i < 1000; i++ {
		contextKey := "project:benchmark"
		if i%10 == 0 {
			contextKey = "global"
		}
		content := fmt.Sprintf("Memory content for entry %d", i)
		err := svc.StoreMemory(content, contextKey, "directive")
		if err != nil {
			b.Fatalf("Failed to seed memory: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.RecallMemory([]string{"project:benchmark", "global"}, 10)
		if err != nil {
			b.Fatalf("RecallMemory failed: %v", err)
		}
	}
}

func BenchmarkSearchMemories(b *testing.B) {
	dbPath := "benchmark_search.db"
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}()

	sqldb, err := db.InitDB(dbPath)
	if err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqldb.Close()

	svc := NewMemoryService(sqldb)

	// Seed 1,000 memories to simulate a production-like load
	for i := 0; i < 1000; i++ {
		content := fmt.Sprintf("Memory content for entry %d with special-keyword", i)
		err := svc.StoreMemory(content, "global", "directive")
		if err != nil {
			b.Fatalf("Failed to seed memory: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.SearchMemories("special-keyword")
		if err != nil {
			b.Fatalf("SearchMemories failed: %v", err)
		}
	}
}

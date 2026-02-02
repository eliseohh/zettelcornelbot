package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eliseohh/zettelcornelbot/internal/index"
)

func main() {
	testDir := "test_vault"
	dbPath := "test_index.db"

	// Cleanup
	os.RemoveAll(testDir)
	os.Remove(dbPath)

	// Setup
	os.Mkdir(testDir, 0755)

	// Create Note A
	os.WriteFile(filepath.Join(testDir, "A.md"), []byte(`# Alpha
Fecha: 2024-02-02
Tipo: idea

## Notas
Links to [[note-b]]

## Enlaces
- [[note-b]]`), 0644)

	// Create Note B
	os.WriteFile(filepath.Join(testDir, "B.md"), []byte(`# Beta
Fecha: 2024-02-02
Tipo: idea

## Notas
Back to [[note-a]]

## Enlaces
- [[note-a]]`), 0644)

	// Init DB
	db, err := index.NewDB(dbPath)
	if err != nil {
		panic(err)
	}

	schema, _ := index.ReadSchemaFile("internal/index/schema.sql")
	db.InitSchema(schema)

	// Run Sync
	idx := index.NewIndexer(db)
	if err := idx.Sync(testDir); err != nil {
		panic(err)
	}

	// Check Edges
	var count int
	db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&count)
	fmt.Printf("Initial Edge Count: %d (Expected 2)\n", count)
	if count != 2 {
		fmt.Println("❌ Initial sync failed")
		os.Exit(1)
	}

	// Modify A
	fmt.Println("Modifying A.md...")
	time.Sleep(1 * time.Second) // Ensure mod time / or just relying on hash
	os.WriteFile(filepath.Join(testDir, "A.md"), []byte(`# Alpha Modified
Fecha: 2024-02-02
Tipo: idea

## Notas
Links to [[note-b]] and [[note-c]]

## Enlaces
- [[note-b]]
- [[note-c]]`), 0644)

	if err := idx.Sync(testDir); err != nil {
		panic(err)
	}

	db.QueryRow("SELECT COUNT(*) FROM edges WHERE source_id='A'").Scan(&count)
	fmt.Printf("Modified Edge Count for A: %d (Expected 2)\n", count)
	if count != 2 {
		fmt.Println("❌ Update sync failed")
		os.Exit(1)
	}

	fmt.Println("✔ Indexer Test Passed")
}

package index

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eliseohh/zettelcornelbot/internal/markdown"
)

type Indexer struct {
	db *DB
}

func NewIndexer(db *DB) *Indexer {
	return &Indexer{db: db}
}

// Concurrency structures
type scanJob struct {
	FullPath string
	RelPath  string
}

type scanResult struct {
	RelPath string
	Hash    string
	Note    *markdown.Note // nil if skipped or error
	Err     error
	IsNew   bool
	Changed bool
}

// Sync walks the directory with a Worker Pool pattern.
func (idx *Indexer) Sync(rootDir string) error {
	fmt.Printf("Starting Sync for %s (Goroutines)...\n", rootDir)

	// Channels
	jobs := make(chan scanJob, 100)
	results := make(chan scanResult, 100)

	validPaths := make(map[string]bool)
	var wg sync.WaitGroup

	// 1. Worker Pool (4 workers)
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			idx.worker(jobs, results)
		}()
	}

	// 2. Walker (Producer)
	go func() {
		defer close(jobs)
		filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			} // Log?
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}

			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return nil
			}

			jobs <- scanJob{FullPath: path, RelPath: relPath}
			return nil
		})
	}()

	// 3. Closer Goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. Consumer (Main Thread - DB Writer)
	// SQLite single-writer preference.
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Cache current hashes to minimize Read queries inside loop?
	// Or just query map? For large DB, map is better.
	// But let's stick to simple logic first. Query inside consumer.
	// Wait, consumer is single thread, so Query is fine.

	for res := range results {
		if res.Err != nil {
			fmt.Printf("⚠️ Error processing %s: %v\n", res.RelPath, res.Err)
			continue
		}

		validPaths[res.RelPath] = true

		// DB Check logic moved to Consumer or Worker?
		// Worker computed Hash. Consumer checks DB.

		var currentHash string
		err := idx.db.QueryRow("SELECT hash FROM nodes WHERE path = ?", res.RelPath).Scan(&currentHash)

		isNew := err == sql.ErrNoRows
		isChanged := err == nil && currentHash != res.Hash

		if isNew {
			fmt.Printf("[+] New: %s\n", res.RelPath)
		} else if isChanged {
			fmt.Printf("[*] Changed: %s\n", res.RelPath)
		}

		if isNew || isChanged {
			// Do Indexing
			if res.Note == nil {
				// Failed parsing but got hash? Or skip?
				continue
			}
			if err := idx.dbUpdate(tx, res.RelPath, res.Hash, res.Note); err != nil {
				fmt.Printf("❌ DB Error %s: %v\n", res.RelPath, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return idx.prune(validPaths)
}

func (idx *Indexer) worker(jobs <-chan scanJob, results chan<- scanResult) {
	// Reusable hasher per worker?
	// sha256.New() is cheap.
	for job := range jobs {
		res := scanResult{RelPath: job.RelPath}

		// 1. Hash
		h, err := calculateHash(job.FullPath)
		if err != nil {
			res.Err = err
			results <- res
			continue
		}
		res.Hash = h

		// 2. Parse (Optimistic: Always parse, Consumer decides if needed?)
		// Optimization: If we only parse when Changed, the Worker should probably only Hash?
		// But passing "Changed" required DB access.
		// If we want parsing to be parallel, we MUST parse here.
		// Cost: CPU parsing files that haven't changed.
		// Tradeoff: If 99% files unchanged, we waste CPU?
		// Better: Worker only Hashes? No, user wants Goroutines for optimization.
		// Optimization is speeding up the "Clean Build" or "Massive Change".
		// For incremental, maybe reading DB in worker? No, idx.db sharing is thread safe for Reads.
		// Let's Check DB in Worker (Read) to decide if Parse is needed.

		var currentHash string
		err = idx.db.QueryRow("SELECT hash FROM nodes WHERE path = ?", job.RelPath).Scan(&currentHash)
		if err == nil && currentHash == h {
			// No change
			res.Changed = false
			results <- res // Empty note, consumer marks validPath
			continue
		}

		// Parse needed
		note, err := markdown.ParseFile(job.FullPath)
		if err != nil {
			res.Err = err
			results <- res
			continue
		}
		res.Note = note
		results <- res
	}
}

// dbUpdate extracts DB logic from old indexFile
func (idx *Indexer) dbUpdate(tx *sql.Tx, relPath, hash string, note *markdown.Note) error {
	id := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	title := note.Title
	if title == "" {
		title = id
	}

	_, err := tx.Exec("DELETE FROM nodes WHERE path = ?", relPath)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO nodes (id, path, hash, last_mod, title) VALUES (?, ?, ?, ?, ?)`,
		id, relPath, hash, 0, title)
	if err != nil {
		return err
	}

	if note.Type != "" {
		_, err = tx.Exec("INSERT INTO tags (node_id, tag) VALUES (?, ?)", id, note.Type)
		if err != nil {
			return err
		}
	}

	for _, targetName := range note.Links {
		_, err = tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, type) VALUES (?, ?, ?)", id, targetName, "wiki_link")
		if err != nil {
			return err
		}
	}
	return nil
}

func (idx *Indexer) prune(validPaths map[string]bool) error {
	// ... (Same logic, simple delete)
	// Re-implement brevity
	rows, err := idx.db.Query("SELECT path FROM nodes")
	if err != nil {
		return err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		if !validPaths[p] {
			toDelete = append(toDelete, p)
		}
	}

	if len(toDelete) > 0 {
		fmt.Printf("[-] Pruning %d stale files\n", len(toDelete))
		tx, err := idx.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, p := range toDelete {
			tx.Exec("DELETE FROM nodes WHERE path = ?", p)
		}
		return tx.Commit()
	}
	return nil
}

func calculateHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

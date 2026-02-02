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

	"github.com/eliseohenriquez/zettelcornelbot/internal/markdown"
)

type Indexer struct {
	db *DB
}

func NewIndexer(db *DB) *Indexer {
	return &Indexer{db: db}
}

// Sync walks the directory and updates the index to match the filesystem state.
func (idx *Indexer) Sync(rootDir string) error {
	fmt.Printf("Starting Sync for %s...\n", rootDir)

	// Track valid paths to identify deletions later
	validPaths := make(map[string]bool)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir // Skip .git, .hidden
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		validPaths[relPath] = true

		// Check if needs update
		hash, err := calculateHash(path)
		if err != nil {
			return fmt.Errorf("hash failed %s: %w", path, err)
		}

		// Check DB state
		var currentHash string
		err = idx.db.QueryRow("SELECT hash FROM nodes WHERE path = ?", relPath).Scan(&currentHash)
		if err == sql.ErrNoRows {
			// New file
			fmt.Printf("[+] New: %s\n", relPath)
			return idx.indexFile(path, relPath, hash)
		} else if err != nil {
			return err
		}

		if currentHash != hash {
			// Changed file
			fmt.Printf("[*] Changed: %s\n", relPath)
			return idx.indexFile(path, relPath, hash)
		}

		// Unchanged
		return nil
	})

	if err != nil {
		return err
	}

	return idx.prune(validPaths)
}

func (idx *Indexer) indexFile(absPath, relPath, hash string) error {
	note, err := markdown.ParseFile(absPath)
	if err != nil {
		return fmt.Errorf("parse error %s: %w", absPath, err)
	}

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Upsert Node
	// On conflict replace? Or explicit update.
	// We use REPLACE or INSERT + DELETE old data.

	// Normalize ID: use filename without extension
	id := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))

	title := note.Title
	if title == "" {
		title = id // Fallback
	}

	// Clean old data for this path if exists (might simple Replace or Delete/Insert)
	// Complication: ID might change for the same path?
	// Simplest: Delete by path, then insert.
	_, err = tx.Exec("DELETE FROM nodes WHERE path = ?", relPath)
	if err != nil {
		return err
	}

	// Note: We need to cascade delete edges/tags, but schema has ON DELETE CASCADE on id.
	// But we just deleted by path. We need to query ID first?
	// Actually, if we delete the node (by path implies looking up id?),
	// wait, `id` is primary key in nodes table?
	// Schema: `id TEXT PRIMARY KEY, path TEXT NOT NULL UNIQUE`.
	// So we should delete by path.
	// `DELETE FROM nodes WHERE path = ?` works and triggers cascades if SQLite foreign keys enabled.
	// !! IMPORTANT: Must enable foreign keys in SQLite connection.

	_, err = tx.Exec(`
		INSERT INTO nodes (id, path, hash, last_mod, title) 
		VALUES (?, ?, ?, ?, ?)
	`, id, relPath, hash, 0, title) // hash is content hash, last_mod ignored for now
	if err != nil {
		return err
	}

	// 2. Insert Tags
	// New format uses "Type" metadata, we index that as a tag
	if note.Type != "" {
		_, err = tx.Exec("INSERT INTO tags (node_id, tag) VALUES (?, ?)", id, note.Type)
		if err != nil {
			return err
		}
	}

	// 3. Insert Edges (Links)
	if len(note.Links) == 0 {
		fmt.Printf("DEBUG: No links found in %s\n", relPath)
	} else {
		fmt.Printf("DEBUG: Found links in %s: %v\n", relPath, note.Links)
	}
	for _, targetName := range note.Links {
		// ... existing comments ...
		_, err = tx.Exec("INSERT OR IGNORE INTO edges (source_id, target_id, type) VALUES (?, ?, ?)", id, targetName, "wiki_link")
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (idx *Indexer) prune(validPaths map[string]bool) error {
	// Find all paths in DB not in validPaths
	rows, err := idx.db.Query("SELECT path FROM nodes")
	if err != nil {
		return err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return err
		}
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
			_, err = tx.Exec("DELETE FROM nodes WHERE path = ?", p)
			if err != nil {
				return err
			}
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

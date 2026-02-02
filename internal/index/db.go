package index

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	// Enable Foreign Keys
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &DB{db}, nil
}

func (d *DB) InitSchema(schemaContent string) error {
	_, err := d.Exec(schemaContent)
	if err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	return nil
}

// Helper to reset the index completely (Determinism principle)
func (d *DB) Nuke() error {
	_, err := d.Exec(`
		DROP TABLE IF EXISTS edges;
		DROP TABLE IF EXISTS tags;
		DROP TABLE IF EXISTS nodes;
	`)
	return err
}

func ReadSchemaFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

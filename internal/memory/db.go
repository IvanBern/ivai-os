package memory

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// MemoryStore handles persistence using an in-memory SQLite database.
type MemoryStore struct {
	db *sql.DB
}

// NewMemoryStore initializes a new in-memory SQLite database.
func NewMemoryStore() (*MemoryStore, error) {
	// Using modernc.org/sqlite for a CGO-free SQLite experience in Go.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	ms := &MemoryStore{db: db}
	if err := ms.bootstrap(); err != nil {
		return nil, err
	}

	return ms, nil
}

// bootstrap creates the initial schema for the OS.
func (ms *MemoryStore) bootstrap() error {
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	_, err := ms.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}
	
	return nil
}

// Close closes the database connection.
func (ms *MemoryStore) Close() error {
	return ms.db.Close()
}

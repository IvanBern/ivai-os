package memory

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Store handles all persistent state for the OS
type Store struct {
	db *sql.DB
}

// Message represents a single interaction log
type Message struct {
	Role    string
	Content string
}

// NewStore initializes the SQLite database and schemas
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create the short-term memory table
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SaveMessage writes a new prompt or response to memory
func (s *Store) SaveMessage(role, content string) error {
	_, err := s.db.Exec("INSERT INTO messages (role, content) VALUES (?, ?)", role, content)
	return err
}

// GetRecentMessages retrieves the last N messages to build the LLM context window
func (s *Store) GetRecentMessages(limit int) ([]Message, error) {
	// We order by ID DESC to get the newest, then reverse in Go so chronological order is maintained
	query := `SELECT role, content FROM (SELECT id, role, content FROM messages ORDER BY id DESC LIMIT ?) ORDER BY id ASC`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

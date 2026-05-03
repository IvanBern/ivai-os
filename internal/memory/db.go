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
	Role             string
	Content          string
	ReasoningContent string
}

// NewStore initializes the SQLite database and schemas
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create the short-term memory table
	// We add reasoning_content to support models like DeepSeek-R1
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		reasoning_content TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SaveMessage writes a new prompt or response to memory
func (s *Store) SaveMessage(role, content, reasoning string) error {
	_, err := s.db.Exec("INSERT INTO messages (role, content, reasoning_content) VALUES (?, ?, ?)", role, content, reasoning)
	return err
}

// GetRecentMessages retrieves the last N messages to build the LLM context window
func (s *Store) GetRecentMessages(limit int) ([]Message, error) {
	query := `SELECT role, content, reasoning_content FROM (SELECT id, role, content, reasoning_content FROM messages ORDER BY id DESC LIMIT ?) ORDER BY id ASC`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.ReasoningContent); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// CountMessages returns the total number of stored messages.
func (s *Store) CountMessages() (int, error) {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// DashboardMessage includes metadata for the web dashboard.
type DashboardMessage struct {
	ID               int    `json:"id"`
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// GetAllMessages returns all messages with timestamps for the dashboard.
func (s *Store) GetAllMessages(limit, offset int) ([]DashboardMessage, error) {
	query := `SELECT id, role, content, reasoning_content, created_at FROM messages ORDER BY id DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []DashboardMessage
	for rows.Next() {
		var msg DashboardMessage
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.ReasoningContent, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

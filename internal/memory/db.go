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
	);

	CREATE TABLE IF NOT EXISTS task_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		instruction TEXT NOT NULL,
		model TEXT NOT NULL DEFAULT '',
		success INTEGER NOT NULL DEFAULT 0,
		response TEXT DEFAULT '',
		error_msg TEXT DEFAULT '',
		duration_ms INTEGER NOT NULL DEFAULT 0,
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

// GetRecentMessages retrieves the last N messages to build the LLM context window.
func (s *Store) GetRecentMessages(limit int) ([]Message, error) {
	return queryMessageRows(s,
		`SELECT role, content, reasoning_content FROM (SELECT id, role, content, reasoning_content FROM messages ORDER BY id DESC LIMIT ?) ORDER BY id ASC`,
		[]any{limit},
		func(msg *Message) []any { return []any{&msg.Role, &msg.Content, &msg.ReasoningContent} },
	)
}

// GetAllMessages returns all messages with timestamps for the dashboard.
func (s *Store) GetAllMessages(limit, offset int) ([]DashboardMessage, error) {
	return queryMessageRows(s,
		`SELECT id, role, content, reasoning_content, created_at FROM messages ORDER BY id DESC LIMIT ? OFFSET ?`,
		[]any{limit, offset},
		func(msg *DashboardMessage) []any { return []any{&msg.ID, &msg.Role, &msg.Content, &msg.ReasoningContent, &msg.CreatedAt} },
	)
}

func queryMessageRows[T any](s *Store, query string, args []any, fields func(*T) []any) ([]T, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		if err := rows.Scan(fields(&item)...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}

// TaskResult records the outcome of a processed task for self-evolution.
type TaskResult struct {
	ID          int    `json:"id"`
	Instruction string `json:"instruction"`
	Model       string `json:"model"`
	Success     bool   `json:"success"`
	Response    string `json:"response,omitempty"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
	CreatedAt   string `json:"created_at"`
}

// SaveTaskResult records a task outcome.
func (s *Store) SaveTaskResult(tr TaskResult) error {
	_, err := s.db.Exec(
		"INSERT INTO task_results (instruction, model, success, response, error_msg, duration_ms) VALUES (?, ?, ?, ?, ?, ?)",
		tr.Instruction, tr.Model, tr.Success, tr.Response, tr.ErrorMsg, tr.DurationMs,
	)
	return err
}

// GetTaskResults returns recent task results for analysis.
func (s *Store) GetTaskResults(limit int) ([]TaskResult, error) {
	rows, err := s.db.Query(
		"SELECT id, instruction, model, success, response, error_msg, duration_ms, created_at FROM task_results ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TaskResult
	for rows.Next() {
		var tr TaskResult
		if err := rows.Scan(&tr.ID, &tr.Instruction, &tr.Model, &tr.Success, &tr.Response, &tr.ErrorMsg, &tr.DurationMs, &tr.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, tr)
	}
	return results, nil
}

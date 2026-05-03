package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

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
	);

	CREATE TABLE IF NOT EXISTS embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL,
		embedding BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{db: db}, nil
}

// SaveMessage writes a new prompt or response to memory
func (s *Store) SaveMessage(role, content, reasoning string) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := s.db.Exec("INSERT INTO messages (role, content, reasoning_content) VALUES (?, ?, ?)", role, content, reasoning)
	return err
}

// CountMessages returns the total number of stored messages.
func (s *Store) CountMessages() (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
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
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
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
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
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

// SaveEmbedding stores a text embedding for semantic search.
func (s *Store) SaveEmbedding(source, content string, embedding []float64) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}
	data, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO embeddings (source, content, embedding) VALUES (?, ?, ?)", source, content, data)
	return err
}

// EmbeddingResult holds a search result with similarity score.
type EmbeddingResult struct {
	Source     string  `json:"source"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

// SearchSimilar finds the most semantically similar stored embeddings.
func (s *Store) SearchSimilar(queryEmbedding []float64, limit int) ([]EmbeddingResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	candidates, err := s.loadRecentEmbeddings(200)
	if err != nil {
		return nil, err
	}
	return scoreAndRank(queryEmbedding, candidates, limit), nil
}

func (s *Store) loadRecentEmbeddings(limit int) ([]candidate, error) {
	rows, err := s.db.Query("SELECT source, content, embedding FROM embeddings ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []candidate
	for rows.Next() {
		var c candidate
		var embJSON []byte
		if err := rows.Scan(&c.source, &c.content, &embJSON); err != nil {
			continue
		}
		json.Unmarshal(embJSON, &c.embedding)
		candidates = append(candidates, c)
	}
	return candidates, nil
}

type candidate struct {
	source    string
	content   string
	embedding []float64
}

type scoredResult struct {
	source     string
	content    string
	similarity float64
}

func scoreAndRank(queryEmb []float64, candidates []candidate, limit int) []EmbeddingResult {
	var scored []scoredResult
	for _, c := range candidates {
		sim := cosineSimilarity(queryEmb, c.embedding)
		if sim > 0.3 {
			scored = append(scored, scoredResult{c.source, c.content, sim})
		}
	}
	sortScoredBySimilarity(scored)
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	results := make([]EmbeddingResult, len(scored))
	for i, s := range scored {
		results[i] = EmbeddingResult{Source: s.source, Content: s.content, Similarity: s.similarity}
	}
	return results
}

func sortScoredBySimilarity(scored []scoredResult) {
	for i := 1; i < len(scored); i++ {
		for j := i; j > 0 && scored[j].similarity > scored[j-1].similarity; j-- {
			scored[j], scored[j-1] = scored[j-1], scored[j]
		}
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// CountEmbeddings returns the total number of stored embeddings.
func (s *Store) CountEmbeddings() (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// TaskStats holds aggregate task statistics.
type TaskStats struct {
	Total       int     `json:"total"`
	Successes   int     `json:"successes"`
	Failures    int     `json:"failures"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration int64   `json:"avg_duration_ms"`
}

// GetTaskStats returns aggregate statistics from task_results.
func (s *Store) GetTaskStats() (TaskStats, error) {
	if s.db == nil {
		return TaskStats{}, fmt.Errorf("database not initialized")
	}
	var stats TaskStats
	var avgDuration float64
	if err := s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN NOT success THEN 1 ELSE 0 END),0), COALESCE(AVG(duration_ms),0) FROM task_results").Scan(&stats.Total, &stats.Successes, &stats.Failures, &avgDuration); err != nil {
		return stats, err
	}
	stats.AvgDuration = int64(avgDuration)
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Successes) / float64(stats.Total) * 100
	}
	return stats, nil
}

// GetRecentEmbeddings returns the most recent embeddings with their metadata.
func (s *Store) GetRecentEmbeddings(limit int) ([]EmbeddingResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := s.db.Query("SELECT source, content FROM embeddings ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EmbeddingResult
	for rows.Next() {
		var r EmbeddingResult
		if err := rows.Scan(&r.Source, &r.Content); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

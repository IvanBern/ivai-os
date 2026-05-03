package memory

import (
	"math"
	"os"
	"sort"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := "test_memory_full.db"
	t.Cleanup(func() { os.Remove(dbPath) })
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store
}

func TestStoreSaveAndGetMessages(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveMessage("user", "hello", ""); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	if err := store.SaveMessage("assistant", "world", "thinking..."); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	msgs, err := store.GetRecentMessages(10)
	if err != nil {
		t.Fatalf("GetRecentMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "world" || msgs[1].ReasoningContent != "thinking..." {
		t.Errorf("unexpected second message: %+v", msgs[1])
	}
}

func TestCountMessages(t *testing.T) {
	store := newTestStore(t)

	count, err := store.CountMessages()
	if err != nil {
		t.Fatalf("CountMessages failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	store.SaveMessage("user", "a", "")
	store.SaveMessage("user", "b", "")
	count, _ = store.CountMessages()
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestGetAllMessages(t *testing.T) {
	store := newTestStore(t)
	store.SaveMessage("user", "first", "")
	store.SaveMessage("assistant", "second", "")

	msgs, err := store.GetAllMessages(10, 0)
	if err != nil {
		t.Fatalf("GetAllMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// Order is DESC by id, so second message comes first
	if msgs[0].Role != "assistant" || msgs[0].Content != "second" {
		t.Errorf("expected assistant/second, got %+v", msgs[0])
	}
	if msgs[0].ID <= 0 || msgs[0].CreatedAt == "" {
		t.Errorf("missing id or created_at: %+v", msgs[0])
	}

	// Pagination: offset 1
	msgs, err = store.GetAllMessages(10, 1)
	if err != nil {
		t.Fatalf("GetAllMessages with offset failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message with offset 1, got %d", len(msgs))
	}
}

func TestTaskResults(t *testing.T) {
	store := newTestStore(t)

	tr := TaskResult{
		Instruction: "test task",
		Model:       "deepseek-v4-pro",
		Success:     true,
		Response:    "done",
		DurationMs:  100,
	}
	if err := store.SaveTaskResult(tr); err != nil {
		t.Fatalf("SaveTaskResult failed: %v", err)
	}

	tr2 := TaskResult{
		Instruction: "fail task",
		Model:       "claude",
		Success:     false,
		ErrorMsg:    "error occurred",
		DurationMs:  50,
	}
	store.SaveTaskResult(tr2)

	results, err := store.GetTaskResults(10)
	if err != nil {
		t.Fatalf("GetTaskResults failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Instruction != "fail task" || results[0].Success != false {
		t.Errorf("expected fail task first (DESC), got %+v", results[0])
	}
	if results[0].CreatedAt == "" {
		t.Errorf("missing created_at")
	}

	// Limit
	results, err = store.GetTaskResults(1)
	if err != nil {
		t.Fatalf("GetTaskResults limit failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with limit, got %d", len(results))
	}
}

func TestTaskStats(t *testing.T) {
	store := newTestStore(t)

	// Empty store
	stats, err := store.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats failed: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("expected 0 total, got %d", stats.Total)
	}

	store.SaveTaskResult(TaskResult{Instruction: "ok", Success: true, DurationMs: 100})
	store.SaveTaskResult(TaskResult{Instruction: "ok2", Success: true, DurationMs: 200})
	store.SaveTaskResult(TaskResult{Instruction: "fail", Success: false, DurationMs: 50})

	stats, err = store.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats failed: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("expected 3 total, got %d", stats.Total)
	}
	if stats.Successes != 2 {
		t.Errorf("expected 2 successes, got %d", stats.Successes)
	}
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
	if stats.SuccessRate != 66.66666666666666 {
		t.Errorf("expected ~66.67 success rate, got %f", stats.SuccessRate)
	}
	if stats.AvgDuration != 116 {
		t.Errorf("expected 116 avg duration, got %d", stats.AvgDuration)
	}
}

func TestEmbeddings(t *testing.T) {
	store := newTestStore(t)
	emb := []float64{0.1, 0.2, 0.3}

	if err := store.SaveEmbedding("test", "content here", emb); err != nil {
		t.Fatalf("SaveEmbedding failed: %v", err)
	}

	count, err := store.CountEmbeddings()
	if err != nil {
		t.Fatalf("CountEmbeddings failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 embedding, got %d", count)
	}

	// Save another
	store.SaveEmbedding("test2", "other content", []float64{0.4, 0.5, 0.6})

	recent, err := store.GetRecentEmbeddings(10)
	if err != nil {
		t.Fatalf("GetRecentEmbeddings failed: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent embeddings, got %d", len(recent))
	}
	if recent[0].Source != "test2" || recent[0].Content != "other content" {
		t.Errorf("expected test2 first (DESC), got %+v", recent[0])
	}
}

func TestSearchSimilar(t *testing.T) {
	store := newTestStore(t)

	emb1 := []float64{1.0, 0.0, 0.0}
	emb2 := []float64{0.0, 1.0, 0.0}
	emb3 := []float64{1.0, 0.5, 0.0}

	store.SaveEmbedding("src1", "cat", emb1)
	store.SaveEmbedding("src2", "dog", emb2)
	store.SaveEmbedding("src3", "bird", emb3)

	// Search with query close to emb1
	results, err := store.SearchSimilar([]float64{0.9, 0.1, 0.0}, 2)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	// First result should be most similar to [1.0, 0.0, 0.0]
	if results[0].Source != "src1" {
		t.Errorf("expected src1 first, got %s", results[0].Source)
	}

	// Empty embedding
	results, err = store.SearchSimilar([]float64{}, 2)
	if err != nil {
		t.Fatalf("SearchSimilar with empty embedding failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestEmbeddingResultsAreSorted(t *testing.T) {
	store := newTestStore(t)

	store.SaveEmbedding("far", "far", []float64{0.0, 0.0})
	store.SaveEmbedding("close", "close", []float64{0.9, 0.1})
	store.SaveEmbedding("exact", "exact", []float64{1.0, 0.0})

	results, err := store.SearchSimilar([]float64{1.0, 0.0}, 10)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	if !sort.SliceIsSorted(results, func(i, j int) bool {
		return results[i].Similarity >= results[j].Similarity
	}) {
		t.Errorf("results not sorted by similarity descending: %+v", results)
	}
}

func TestNewStoreError(t *testing.T) {
	_, err := NewStore("/non/existent/path/memory.db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestStoreNilDB(t *testing.T) {
	s := &Store{db: nil}
	if err := s.SaveMessage("user", "test", ""); err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBCountMessages(t *testing.T) {
	s := &Store{db: nil}
	_, err := s.CountMessages()
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBCountEmbeddings(t *testing.T) {
	s := &Store{db: nil}
	_, err := s.CountEmbeddings()
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBGetRecentEmbeddings(t *testing.T) {
	s := &Store{db: nil}
	_, err := s.GetRecentEmbeddings(10)
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestGetTaskResultsEmpty(t *testing.T) {
	store := newTestStore(t)
	results, err := store.GetTaskResults(10)
	if err != nil {
		t.Fatalf("GetTaskResults failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestCountMessagesEmpty(t *testing.T) {
	store := newTestStore(t)
	count, err := store.CountMessages()
	if err != nil {
		t.Fatalf("CountMessages failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountEmbeddings(t *testing.T) {
	store := newTestStore(t)
	count, err := store.CountEmbeddings()
	if err != nil {
		t.Fatalf("CountEmbeddings failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
	store.SaveEmbedding("test", "content", []float64{0.1, 0.2})
	count, _ = store.CountEmbeddings()
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestStoreNilDBSaveTaskResult(t *testing.T) {
	s := &Store{db: nil}
	err := s.SaveTaskResult(TaskResult{Instruction: "test", Success: true})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBSaveEmbedding(t *testing.T) {
	s := &Store{db: nil}
	err := s.SaveEmbedding("test", "content", []float64{0.1})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBGetTaskStats(t *testing.T) {
	s := &Store{db: nil}
	_, err := s.GetTaskStats()
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestStoreNilDBSearchSimilar(t *testing.T) {
	s := &Store{db: nil}
	_, err := s.SearchSimilar([]float64{0.1}, 5)
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestSaveEmbeddingJSONError(t *testing.T) {
	store := newTestStore(t)
	err := store.SaveEmbedding("test", "content", []float64{math.NaN()})
	if err == nil {
		t.Error("expected error for NaN embedding that cannot be JSON marshaled")
	}
}

func TestNewStoreSchemaError(t *testing.T) {
	// Create a directory at the db path so that SQLite cannot create the database file
	dirPath := "test_schema_err"
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)
	dbPath := dirPath + "/memory.db"
	// Create a directory with that name
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := NewStore(dbPath)
	if err == nil {
		t.Error("expected error when db path is a directory")
	}
}

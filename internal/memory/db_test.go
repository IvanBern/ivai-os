package memory

import (
	"os"
	"testing"
)

func TestStore(t *testing.T) {
	// 1. Setup temporary DB
	dbPath := "test_memory.db"
	defer os.Remove(dbPath)

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 2. Save Message
	err = store.SaveMessage("user", "hello ivai", "")
	if err != nil {
		t.Errorf("failed to save message: %v", err)
	}

	// 3. Retrieve Messages
	messages, err := store.GetRecentMessages(10)
	if err != nil {
		t.Errorf("failed to get messages: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "hello ivai" {
		t.Errorf("expected 'hello ivai', got %s", messages[0].Content)
	}
}

func TestNewStoreError(t *testing.T) {
	// Trying to create a DB in a non-existent directory
	_, err := NewStore("/non/existent/path/memory.db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/memory"
	"github.com/IvanBern/ivai-os/internal/sandbox"
)

func TestProcessTask(t *testing.T) {
	// 1. Mock LLM Gateway
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.OpenAIResponse{
			Choices: []struct {
				Message llm.Message `json:"message"`
			}{
				{Message: llm.Message{Role: "assistant", Content: "Task completed successfully"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("key", "", "")
	gateway.DeepSeekURL = server.URL

	// 2. Setup Memory
	dbPath := "test_main_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)

	// 3. Setup Sandbox
	wasmEngine := sandbox.NewWasmRuntime()

	// 4. Run processTask
	ctx := context.Background()
	processTask(ctx, "hello", gateway, store, wasmEngine)

	// 5. Verify results in memory
	messages, _ := store.GetRecentMessages(10)
	foundAssistant := false
	for _, msg := range messages {
		if msg.Role == "assistant" && msg.Content == "Task completed successfully" {
			foundAssistant = true
			break
		}
	}

	if !foundAssistant {
		t.Errorf("assistant response not found in memory")
	}
}

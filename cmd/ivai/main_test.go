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

	dbPath := "test_main_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	processTask(context.Background(), TaskInput{
		Instruction: "hello",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	})
}

func TestProcessTaskTools(t *testing.T) {
	dbPath := "test_tools_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	tools := []string{"read_file", "write_file", "execute_command", "http_request"}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var resp llm.OpenAIResponse
				if callCount == 0 {
					args := "{}"
					if toolName == "read_file" {
						args = `{"filepath":"f"}`
					}
					if toolName == "write_file" {
						args = `{"filepath":"f", "content":"c"}`
					}
					if toolName == "execute_command" {
						args = `{"command":"echo"}`
					}
					if toolName == "http_request" {
						args = `{"method":"GET", "url":"http://localhost"}`
					}

					resp = llm.OpenAIResponse{
						Choices: []struct {
							Message llm.Message `json:"message"`
						}{
							{
								Message: llm.Message{
									Role: "assistant",
									ToolCalls: []llm.ToolCall{
										{
											ID:   "c1",
											Type: "function",
											Function: llm.ToolCallFunction{
												Name:      toolName,
												Arguments: args,
											},
										},
									},
								},
							},
						},
					}
				} else {
					resp = llm.OpenAIResponse{
						Choices: []struct {
							Message llm.Message `json:"message"`
						}{
							{Message: llm.Message{Role: "assistant", Content: "done"}},
						},
					}
				}
				callCount++
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			gateway := llm.NewGateway("k", "", "")
			gateway.DeepSeekURL = server.URL
			processTask(context.Background(), TaskInput{
				Instruction: "run " + toolName,
				Gateway:     gateway,
				DBStore:     store,
				WasmEngine:  wasmEngine,
			})
		})
	}
}

func TestProcessTaskWasm(t *testing.T) {
	dbPath := "test_wasm_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp llm.OpenAIResponse
		if callCount == 0 {
			resp = llm.OpenAIResponse{
				Choices: []struct {
					Message llm.Message `json:"message"`
				}{
					{
						Message: llm.Message{
							Role: "assistant",
							ToolCalls: []llm.ToolCall{
								{
									ID:   "c1",
									Type: "function",
									Function: llm.ToolCallFunction{
										Name:      "execute_wasm",
										Arguments: `{"filepath":"nonexistent.wasm", "payload":"p", "timeout_ms":100}`,
									},
								},
							},
						},
					},
				},
			}
		} else {
			resp = llm.OpenAIResponse{
				Choices: []struct {
					Message llm.Message `json:"message"`
				}{
					{Message: llm.Message{Role: "assistant", Content: "done"}},
				},
			}
		}
		callCount++
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "", "")
	gateway.DeepSeekURL = server.URL
	processTask(context.Background(), TaskInput{
		Instruction: "run wasm",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	})
}

func TestProcessTaskRouting(t *testing.T) {
	dbPath := "test_routing_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.OpenAIResponse{
			Choices: []struct {
				Message llm.Message `json:"message"`
			}{
				{Message: llm.Message{Role: "assistant", Content: "ok"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "k", "k")
	gateway.DeepSeekURL = server.URL
	gateway.AnthropicURL = server.URL
	gateway.GeminiURL = server.URL

	tests := []string{"@claude", "@gemini", "@research", "@deepseek"}
	for _, m := range tests {
		processTask(context.Background(), TaskInput{
			Instruction: m + " hi",
			Gateway:     gateway,
			DBStore:     store,
			WasmEngine:  wasmEngine,
		})
	}
}

func TestProcessTaskWithHistory(t *testing.T) {
	dbPath := "test_hist_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	store.SaveMessage("user", "u", "")
	store.SaveMessage("assistant", "a", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.OpenAIResponse{
			Choices: []struct {
				Message llm.Message `json:"message"`
			}{
				{Message: llm.Message{Role: "assistant", Content: "ok"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "", "")
	gateway.DeepSeekURL = server.URL
	wasmEngine := sandbox.NewWasmRuntime()
	processTask(context.Background(), TaskInput{
		Instruction: "q",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	})
}

func TestProcessTaskError(t *testing.T) {
	dbPath := "test_err_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "", "")
	gateway.DeepSeekURL = server.URL
	processTask(context.Background(), TaskInput{
		Instruction: "hi",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	})
}

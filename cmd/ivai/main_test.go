package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

	result := processTask(context.Background(), TaskInput{
		Instruction: "hello",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	}, nil)
	if result != "Task completed successfully" {
		t.Errorf("expected 'Task completed successfully', got %q", result)
	}
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
			}, nil)
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
	}, nil)
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
		}, nil)
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
	}, nil)
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
	}, nil)
}

// --- SSE Streaming Tests ---

func TestProcessTaskWithProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.OpenAIResponse{
			Choices: []struct {
				Message llm.Message `json:"message"`
			}{
				{Message: llm.Message{Role: "assistant", Content: "Task completed with progress"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("key", "", "")
	gateway.DeepSeekURL = server.URL

	dbPath := "test_progress_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	progressChan := make(chan ProgressEvent, 20)

	go func() {
		result := processTask(context.Background(), TaskInput{
			Instruction: "hello",
			Gateway:     gateway,
			DBStore:     store,
			WasmEngine:  wasmEngine,
		}, progressChan)
		if result != "Task completed with progress" {
			t.Errorf("expected 'Task completed with progress', got %q", result)
		}
	}()

	events := collectEvents(progressChan)

	// Verify events: task_start
	if len(events) < 1 {
		t.Fatal("expected at least task_start event")
	}
	if events[0].Type != "task_start" {
		t.Errorf("expected task_start, got %s", events[0].Type)
	}
}

func TestProcessTaskWithProgressAndTools(t *testing.T) {
	dbPath := "test_progress_tools_memory.db"
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
										Name:      "execute_command",
										Arguments: `{"command":"echo hello"}`,
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
					{Message: llm.Message{Role: "assistant", Content: "done with tools"}},
				},
			}
		}
		callCount++
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "", "")
	gateway.DeepSeekURL = server.URL

	progressChan := make(chan ProgressEvent, 20)

	go func() {
		result := processTask(context.Background(), TaskInput{
			Instruction: "run command",
			Gateway:     gateway,
			DBStore:     store,
			WasmEngine:  wasmEngine,
		}, progressChan)
		if result != "done with tools" {
			t.Errorf("expected 'done with tools', got %q", result)
		}
	}()

	events := collectEvents(progressChan)

	if !hasEventType(events, "tool_call") {
		t.Error("expected tool_call event")
	}
	if !hasEventType(events, "tool_result") {
		t.Error("expected tool_result event")
	}
}

func hasEventType(events []ProgressEvent, typ string) bool {
	for _, e := range events {
		if e.Type == typ {
			return true
		}
	}
	return false
}

func TestProcessTaskErrorWithProgress(t *testing.T) {
	dbPath := "test_progress_err_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	gateway := llm.NewGateway("k", "", "")
	gateway.DeepSeekURL = server.URL

	progressChan := make(chan ProgressEvent, 20)

	result := processTask(context.Background(), TaskInput{
		Instruction: "fail",
		Gateway:     gateway,
		DBStore:     store,
		WasmEngine:  wasmEngine,
	}, progressChan)

	if !strings.Contains(result, "Error:") {
		t.Errorf("expected error result, got %q", result)
	}

	events := collectEvents(progressChan)

	hasError := false
	for _, e := range events {
		if e.Type == "task_error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected task_error event")
	}
}

// --- SSE HTTP Endpoint Tests ---

func TestHTTPSseEndpoint(t *testing.T) {
	// Mock LLM server
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.OpenAIResponse{
			Choices: []struct {
				Message llm.Message `json:"message"`
			}{
				{Message: llm.Message{Role: "assistant", Content: "SSE response"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer llmServer.Close()

	gateway := llm.NewGateway("key", "", "")
	gateway.DeepSeekURL = llmServer.URL

	dbPath := "test_sse_http_memory.db"
	defer os.Remove(dbPath)
	store, _ := memory.NewStore(dbPath)
	wasmEngine := sandbox.NewWasmRuntime()

	taskChan := make(chan taskWithResponder, 10)

	// Start event loop in background
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go runEventLoop(ctx, taskChan, nil, gateway, store, wasmEngine)

	// Create test HTTP server with our handler
	handler := http.NewServeMux()
	handler.HandleFunc("/api/task/stream", func(w http.ResponseWriter, r *http.Request) {
		handleTaskStreaming(w, r, taskChan)
	})

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	resp, err := http.Post(testServer.URL+"/api/task/stream", "application/json",
		strings.NewReader(`{"instruction":"stream test"}`))
	if err != nil {
		t.Fatalf("failed to POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", contentType)
	}

	// Read SSE events
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "event: task_start") {
		t.Errorf("expected task_start event, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "event: task_complete") {
		t.Errorf("expected task_complete event, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "SSE response") {
		t.Errorf("expected response in data, got: %s", bodyStr)
	}
}

// --- Helpers ---

func collectEvents(ch chan ProgressEvent) []ProgressEvent {
	var events []ProgressEvent
	for evt := range ch {
		events = append(events, evt)
	}
	return events
}

func TestRegressionAllToolDispatch(t *testing.T) {
	tools := buildTools()
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Function.Name] = true
	}
	expected := []string{"read_file", "write_file", "execute_command", "execute_wasm", "http_request", "github_pr", "code_health", "create_issue", "list_issues", "update_wiki", "swarm_clone", "swarm_deploy", "swarm_dispatch", "swarm_gather", "swarm_status", "swarm_spawn", "swarm_kill"}
	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("tool %q not found in buildTools()", name)
		}
	}
	if len(tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(tools))
	}
}

func TestRegressionBuildPayload(t *testing.T) {
	dbPath := "test_regression.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveMessage("user", "test message", "")

	gw := llm.NewGateway("test-key", "", "")
	payload := buildPayload(store, gw)
	if len(payload) < 2 {
		t.Errorf("expected at least 2 messages (system + user), got %d", len(payload))
	}
	if payload[0].Role != "system" {
		t.Errorf("first message should be system, got %s", payload[0].Role)
	}
}

func TestRegressionFeatureFlags(t *testing.T) {
	t.Setenv("IVAI_FEATURE_RAG", "false")
	if featureEnabled("rag") {
		t.Error("RAG should be disabled when IVAI_FEATURE_RAG=false")
	}
	t.Setenv("IVAI_FEATURE_RAG", "true")
	if !featureEnabled("rag") {
		t.Error("RAG should be enabled when IVAI_FEATURE_RAG=true")
	}
	if !featureEnabled("nonexistent") {
		t.Error("unknown features should default to enabled")
	}
}

func TestRegressionSSEFormat(t *testing.T) {
	evt := ProgressEvent{
		Type:    "task_start",
		Message: "Task started",
		Data:    map[string]string{"model": "test"},
	}
	data, _ := json.Marshal(evt)
	if !strings.Contains(string(data), "task_start") {
		t.Error("SSE event should contain type")
	}
	if !strings.Contains(string(data), "model") {
		t.Error("SSE event should contain data")
	}
}

func TestRegressionExtractModel(t *testing.T) {
	tests := []struct {
		input, wantModel, wantInst string
	}{
		{"hello", "deepseek-v4-pro", "hello"},
		{"@claude hi", "claude-3-5-sonnet-20241022", " hi"},
		{"@gemini test", "gemini-2.5-pro", " test"},
		{"@research deep", "deep-research-max-preview", " deep"},
	}
	for _, tc := range tests {
		model, inst := extractModel(tc.input)
		if model != tc.wantModel || inst != tc.wantInst {
			t.Errorf("extractModel(%q) = (%q, %q), want (%q, %q)", tc.input, model, inst, tc.wantModel, tc.wantInst)
		}
	}
}

func TestSwarmFunctionsCompile(t *testing.T) {
	// Verify all swarm functions exist and parse args
	_, _ = executeSwarmClone(`{"name":"test"}`)
	_, _ = executeSwarmDeploy(`{"name":"test"}`)
	_, _ = executeSwarmDispatch(`{"worker":"w","instruction":"hi"}`)
	_, _ = executeSwarmGather(`{"worker":"w"}`)
	_, _ = executeSwarmStatus(`{}`)
	_, _ = executeSwarmStatus(`{"name":"test"}`)
	_ = callVMBridge
}

func TestSwarmToolRegistryComplete(t *testing.T) {
	for _, name := range []string{"swarm_clone","swarm_deploy","swarm_dispatch","swarm_gather","swarm_status"} {
		if _, ok := toolRegistry[name]; !ok {
			t.Errorf("swarm tool %q not in toolRegistry", name)
		}
	}
}

// --- Swarm Bug-Fix Tests ---

func TestWorkerURL(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"192.168.1.5", "192.168.1.5:8080"},
		{"localhost", "localhost:8080"},
		{"localhost:8081", "localhost:8081"},
		{"192.168.1.5:9090", "192.168.1.5:9090"},
		{"worker.example.com", "worker.example.com:8080"},
		{"worker.example.com:443", "worker.example.com:443"},
	}
	for _, tc := range tests {
		got := workerURL(tc.input)
		if got != tc.want {
			t.Errorf("workerURL(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestLocalWorkerPortReadsEnv(t *testing.T) {
	os.MkdirAll("/tmp/ivai-testport", 0755)
	os.WriteFile("/tmp/ivai-testport/.env", []byte("IVAI_PORT=9092\n"), 0644)
	defer os.RemoveAll("/tmp/ivai-testport")

	port := localWorkerPort("testport")
	if port != "9092" {
		t.Errorf("localWorkerPort(testport) = %q, want 9092", port)
	}

	port = localWorkerPort("nonexistent")
	if port != "" {
		t.Errorf("localWorkerPort(nonexistent) = %q, want \"\"", port)
	}
}

func TestMergeWorkerListsProducesValidJSON(t *testing.T) {
	vmResult := `[{"name":"vm1","status":"ok"}]`
	localResult := `[{"name":"worker1","port":"8081"}]`
	merged := mergeWorkerLists(vmResult, localResult)

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(merged), &result); err != nil {
		t.Fatalf("mergeWorkerLists produced invalid JSON: %v", err)
	}
	if _, ok := result["vm_workers"]; !ok {
		t.Error("merged result missing vm_workers key")
	}
	if _, ok := result["local_workers"]; !ok {
		t.Error("merged result missing local_workers key")
	}
}

func TestSwarmDispatchUsesWorkerURL(t *testing.T) {
	// Verify executeSwarmDispatch sends request to the correct URL via workerURL
	var capturedPath, capturedMethod string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":"ok"}`))
	}))
	defer mockServer.Close()

	addr := mockServer.URL[strings.Index(mockServer.URL, "://")+3:]
	if strings.HasPrefix(addr, ":") { addr = "127.0.0.1" + addr }

	result, err := executeSwarmDispatch(`{"worker":"` + addr + `","instruction":"test"}`)
	if err != nil {
		t.Fatalf("executeSwarmDispatch failed: %v", err)
	}
	if result == "" {
		t.Error("executeSwarmDispatch returned empty result")
	}
	if capturedPath != "/api/task" {
		t.Errorf("expected path /api/task, got %q", capturedPath)
	}
	if capturedMethod != "POST" {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
}

func TestSwarmGatherUsesWorkerURL(t *testing.T) {
	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"response":"result1"}]`))
	}))
	defer mockServer.Close()

	addr := mockServer.URL[strings.Index(mockServer.URL, "://")+3:]
	if strings.HasPrefix(addr, ":") { addr = "127.0.0.1" + addr }

	result, err := executeSwarmGather(`{"worker":"` + addr + `"}`)
	if err != nil {
		t.Fatalf("executeSwarmGather failed: %v", err)
	}
	if result == "" {
		t.Error("executeSwarmGather returned empty result")
	}
	if capturedPath != "/api/task-results" {
		t.Errorf("expected path /api/task-results, got %q", capturedPath)
	}
}

func TestCheckLocalWorkerIntegration(t *testing.T) {
	mockWorker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"0.1.0","uptime_sec":42}`))
	}))
	defer mockWorker.Close()

	// Extract port from mock server URL
	url := mockWorker.URL
	port := url[strings.LastIndex(url, ":")+1:]

	workerName := "testcheckint"
	os.MkdirAll("/tmp/ivai-"+workerName, 0755)
	os.WriteFile("/tmp/ivai-"+workerName+"/.env", []byte("IVAI_PORT="+port+"\n"), 0644)
	defer os.RemoveAll("/tmp/ivai-" + workerName)

	status := checkLocalWorker(workerName)
	if !strings.Contains(status, "42") {
		t.Errorf("checkLocalWorker returned: %s (expected uptime_sec:42)", status)
	}

	// Nonexistent worker returns empty
	if got := checkLocalWorker("nonexistent99"); got != "" {
		t.Errorf("checkLocalWorker for nonexistent = %q, want \"\"", got)
	}
}

func TestListLocalWorkersFindsActiveWorkers(t *testing.T) {
	mockWorker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer mockWorker.Close()

	url := mockWorker.URL
	port := url[strings.LastIndex(url, ":")+1:]

	workerName := "testlist"
	os.MkdirAll("/tmp/ivai-"+workerName, 0755)
	os.WriteFile("/tmp/ivai-"+workerName+"/.env", []byte("IVAI_PORT="+port+"\n"), 0644)
	defer os.RemoveAll("/tmp/ivai-" + workerName)

	result := listLocalWorkers()
	if result == "" {
		t.Error("listLocalWorkers returned empty, expected at least one worker")
	}
	if !strings.Contains(result, workerName) {
		t.Errorf("listLocalWorkers result missing worker name %q: %s", workerName, result)
	}
	if !strings.Contains(result, `"type":"local"`) {
		t.Errorf("listLocalWorkers result missing type field: %s", result)
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/memory"
	"github.com/IvanBern/ivai-os/internal/sandbox"
)

func TestResolvePaths(t *testing.T) {
	t.Run("default darwin", func(t *testing.T) {
		env, db := resolvePaths()
		if env != ".env" || db != "memory.db" {
			t.Errorf("unexpected default paths: env=%s db=%s", env, db)
		}
	})
	t.Run("with IVAI_DATA_DIR", func(t *testing.T) {
		t.Setenv("IVAI_DATA_DIR", "/custom/dir")
		env, db := resolvePaths()
		if env != "/custom/dir/.env" || db != "/custom/dir/memory.db" {
			t.Errorf("unexpected custom paths: env=%s db=%s", env, db)
		}
	})
}

func TestResolvePort(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("IVAI_PORT", "")
		if p := resolvePort(); p != "8080" {
			t.Errorf("expected 8080, got %s", p)
		}
	})
	t.Run("custom", func(t *testing.T) {
		t.Setenv("IVAI_PORT", "9090")
		if p := resolvePort(); p != "9090" {
			t.Errorf("expected 9090, got %s", p)
		}
	})
}

func TestInitGateway(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "ds-key")
	t.Setenv("ANTHROPIC_API_KEY", "an-key")
	gw := initGateway()
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if gw.DeepSeekKey != "ds-key" || gw.AnthropicKey != "an-key" {
		t.Errorf("gateway keys not set correctly: %+v", gw)
	}
}

func TestInitMemory(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		store := initMemory("test_init_mem.db")
		if store == nil {
			t.Fatal("expected non-nil store")
		}
		os.Remove("test_init_mem.db")
	})
	t.Run("invalid path", func(t *testing.T) {
		store := initMemory("/nonexistent/test.db")
		if store != nil {
			t.Error("expected nil store for invalid path")
		}
	})
}

func TestInitWasm(t *testing.T) {
	engine := initWasm()
	if engine == nil {
		t.Error("expected non-nil wasm engine")
	}
}

func TestHandleGatewayMissing(t *testing.T) {
	t.Run("with progress channel", func(t *testing.T) {
		ch := make(chan ProgressEvent, 1)
		msg := handleGatewayMissing(ch)
		if !strings.HasPrefix(msg, "Error:") {
			t.Errorf("expected error message, got %s", msg)
		}
		_, ok := <-ch
		if ok {
			t.Error("expected channel to be closed")
		}
	})
	t.Run("without progress channel", func(t *testing.T) {
		msg := handleGatewayMissing(nil)
		if !strings.HasPrefix(msg, "Error:") {
			t.Errorf("expected error message, got %s", msg)
		}
	})
}

func TestNoLLMKeysConfigured(t *testing.T) {
	if !noLLMKeysConfigured("", "", "") {
		t.Error("expected true for all empty")
	}
	if noLLMKeysConfigured("key", "", "") {
		t.Error("expected false if any key present")
	}
}

func TestParseQueryInt(t *testing.T) {
	if v := parseQueryInt("10", 5, func(v int) bool { return v > 0 && v <= 20 }); v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
	if v := parseQueryInt("invalid", 5, func(v int) bool { return v > 0 }); v != 5 {
		t.Errorf("expected default 5, got %d", v)
	}
	if v := parseQueryInt("100", 5, func(v int) bool { return v > 0 && v <= 20 }); v != 5 {
		t.Errorf("expected default 5 for out-of-range, got %d", v)
	}
}

func TestHandleStatus(t *testing.T) {
	gateway := llm.NewGateway("ds-key", "an-key", "")
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	handleStatus(w, req, gateway)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["version"] != "dev" {
		t.Errorf("expected dev version, got %v", body["version"])
	}
	if body["active_model"] != "deepseek-v4-pro" {
		t.Errorf("expected deepseek-v4-pro, got %v", body["active_model"])
	}
}

func TestHandleTools(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/tools", nil)
	w := httptest.NewRecorder()
	handleTools(w, req)

	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	tools, ok := body["tools"].([]any)
	if !ok {
		t.Fatal("expected tools array")
	}
	if len(tools) != 17 {
		t.Errorf("expected 17 tools, got %d", len(tools))
	}
}

func TestHandleMemory_NilStore(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/memory", nil)
	w := httptest.NewRecorder()
	handleMemory(w, req, nil)
	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Memory store not available" {
		t.Errorf("expected memory error, got %v", body["error"])
	}
}

func TestHandleTaskResults_NilStore(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/task-results", nil)
	w := httptest.NewRecorder()
	handleTaskResults(w, req, nil)
	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Memory store not available" {
		t.Errorf("expected memory error, got %v", body["error"])
	}
}

func TestHandleSystem_NilStore(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/system", nil)
	w := httptest.NewRecorder()
	handleSystem(w, req, nil)
	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["system_prompt"]; !ok {
		t.Error("expected system_prompt field")
	}
}

func TestHandleEmbeddings_NilStore(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/embeddings", nil)
	w := httptest.NewRecorder()
	handleEmbeddings(w, req, nil)
	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Memory store not available" {
		t.Errorf("expected memory error, got %v", body["error"])
	}
}

func TestHandleMemoryWithStore(t *testing.T) {
	dbPath := "test_handle_mem.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveMessage("user", "hello", "")

	req := httptest.NewRequest("GET", "/api/memory?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	handleMemory(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["total"].(float64) != 1 {
		t.Errorf("expected 1 total, got %v", body["total"])
	}
}

func TestHandleTaskResultsWithStore(t *testing.T) {
	dbPath := "test_handle_tr.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveTaskResult(memory.TaskResult{Instruction: "test", Success: true, DurationMs: 50})

	req := httptest.NewRequest("GET", "/api/task-results?limit=10", nil)
	w := httptest.NewRecorder()
	handleTaskResults(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	results := body["results"].([]any)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHandleSystemWithStore(t *testing.T) {
	dbPath := "test_handle_sys.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/system", nil)
	w := httptest.NewRecorder()
	handleSystem(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if _, ok := body["embeddings_count"]; !ok {
		t.Error("expected embeddings_count")
	}
}

func TestHandleEmbeddingsWithStore(t *testing.T) {
	dbPath := "test_handle_emb.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveEmbedding("test", "hi", []float64{0.1, 0.2})

	req := httptest.NewRequest("GET", "/api/embeddings?limit=10", nil)
	w := httptest.NewRecorder()
	handleEmbeddings(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	results := body["embeddings"].([]any)
	if len(results) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(results))
	}
}

func TestSSEHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	setSSEHeaders(rec)
	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}
}

func TestWriteSSEEvent(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEEvent(w, w, "test_event", ProgressEvent{
		Type:    "test_event",
		Message: "hello",
	})
	body := w.Body.String()
	if !strings.Contains(body, "event: test_event") {
		t.Errorf("expected event in body, got: %s", body)
	}
	if !strings.Contains(body, "hello") {
		t.Errorf("expected message in body, got: %s", body)
	}
}

func TestFeatureEnabled(t *testing.T) {
	t.Setenv("IVAI_FEATURE_RAG", "false")
	if featureEnabled("rag") {
		t.Error("RAG should be disabled when false")
	}
	t.Setenv("IVAI_FEATURE_RAG", "true")
	if !featureEnabled("rag") {
		t.Error("RAG should be enabled when true")
	}
	if !featureEnabled("nonexistent") {
		t.Error("unknown features should default to enabled")
	}
}

func TestTruncate(t *testing.T) {
	if s := truncate("hello", 10); s != "hello" {
		t.Errorf("expected hello, got %s", s)
	}
	if s := truncate("hello world", 5); s != "hello..." {
		t.Errorf("expected hello..., got %s", s)
	}
}

func TestRecordTaskResultNilDB(t *testing.T) {
	state := &taskState{
		dbStore: nil,
		gateway: nil,
	}
	// Should not panic
	recordTaskResult(state, "result", 100)
}

func TestHandleTaskBlocking(t *testing.T) {
	taskChan := make(chan taskWithResponder, 1)
	body := `{"instruction":"test"}`
	req := httptest.NewRequest("POST", "/api/task", strings.NewReader(body))
	w := httptest.NewRecorder()

	go func() {
		task := <-taskChan
		task.responder <- "response"
	}()

	handleTaskBlocking(w, req, taskChan)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleTaskBlockingMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/task", nil)
	w := httptest.NewRecorder()
	handleTaskBlocking(w, req, nil)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Result().StatusCode)
	}
}

func TestHandleTaskBlockingBadJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/task", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	handleTaskBlocking(w, req, nil)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestShutdownServerNil(t *testing.T) {
	// Should not panic
	shutdownServer(nil)
}

func TestPrintPromptNoTTY(t *testing.T) {
	// Should not panic when not a terminal
	printPrompt()
}

func TestExecuteToolCallUnknown(t *testing.T) {
	tc := llm.ToolCall{
		ID:   "test",
		Type: "function",
		Function: llm.ToolCallFunction{
			Name:      "nonexistent_tool",
			Arguments: "{}",
		},
	}
	result := executeToolCall(context.Background(), tc, nil)
	if !strings.Contains(result, "Unknown tool") {
		t.Errorf("expected Unknown tool, got %s", result)
	}
}

func TestResultOrError(t *testing.T) {
	if s := resultOrError("ok", nil); s != "ok" {
		t.Errorf("expected ok, got %s", s)
	}
	if s := resultOrError("", fmt.Errorf("fail")); !strings.Contains(s, "Error:") {
		t.Errorf("expected error, got %s", s)
	}
}

func TestHandleTaskStreamingMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/task/stream", nil)
	w := httptest.NewRecorder()
	handleTaskStreaming(w, req, nil)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Result().StatusCode)
	}
}

func TestHandleTaskStreamingBadJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/task/stream", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	handleTaskStreaming(w, req, nil)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestToolCountConsistent(t *testing.T) {
	tools := buildTools()
	expected := []string{
		"read_file", "write_file", "execute_command", "execute_wasm", "http_request",
		"github_pr", "code_health", "create_issue", "list_issues", "update_wiki",
		"swarm_clone", "swarm_deploy", "swarm_dispatch", "swarm_gather", "swarm_status",
		"swarm_spawn", "swarm_kill",
	}
	if len(tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(tools))
	}
	names := make(map[string]bool)
	for _, t := range tools {
		names[t.Function.Name] = true
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestStreamProgressEventsTimeout(t *testing.T) {
	w := httptest.NewRecorder()
	progressChan := make(chan ProgressEvent)
	respChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately to trigger the ctx.Done() path
	streamProgressEvents(w, w, progressChan, respChan, ctx)
	// If we got here without deadlock, test passes
}

func TestTaskStateEmit(t *testing.T) {
	t.Run("nil progress channel", func(t *testing.T) {
		s := &taskState{progressChan: nil}
		s.emit(ProgressEvent{Type: "test"}) // should not panic
	})

	t.Run("with progress channel", func(t *testing.T) {
		ch := make(chan ProgressEvent, 1)
		s := &taskState{progressChan: ch}
		s.emit(ProgressEvent{Type: "test", Message: "hello"})
		evt := <-ch
		if evt.Type != "test" || evt.Message != "hello" {
			t.Errorf("unexpected event: %+v", evt)
		}
	})

	t.Run("full channel non-blocking", func(t *testing.T) {
		ch := make(chan ProgressEvent, 1)
		ch <- ProgressEvent{Type: "full"}
		s := &taskState{progressChan: ch}
		s.emit(ProgressEvent{Type: "dropped"}) // should not block
	})
}

func TestBuildToolsFns(t *testing.T) {
	core := buildCoreTools()
	if len(core) != 5 {
		t.Errorf("expected 5 core tools, got %d", len(core))
	}
	gh := buildGitHubTools()
	if len(gh) != 5 {
		t.Errorf("expected 5 github tools, got %d", len(gh))
	}
	swarm := buildSwarmTools()
	if len(swarm) != 7 {
		t.Errorf("expected 7 swarm tools, got %d", len(swarm))
	}
}

func TestProcessTaskWithNilGateway(t *testing.T) {
	result := processTask(context.Background(), TaskInput{Gateway: nil}, nil)
	if !strings.HasPrefix(result, "Error:") {
		t.Errorf("expected error, got %s", result)
	}
}

func TestExecuteGitHubPR(t *testing.T) {
	t.Run("with base", func(t *testing.T) {
		_, err := executeGitHubPR(`{"title":"t","body":"b","base":"develop","repo":"/tmp/repo"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("default base", func(t *testing.T) {
		_, err := executeGitHubPR(`{"title":"t","body":"b"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
}

func TestExecuteCodeHealthTool(t *testing.T) {
	_, err := executeCodeHealthTool(`{"repo":"./nonexistent"}`)
	if err != nil {
		t.Logf("expected non-fatal, got %v", err)
	}
}

func TestExecuteCreateIssue(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		_, err := executeCreateIssue(`{"title":"t","body":"b"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("with labels and assignee", func(t *testing.T) {
		_, err := executeCreateIssue(`{"title":"t","body":"b","labels":"bug","assignee":"user"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
}

func TestExecuteListIssues(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		_, err := executeListIssues(`{}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("with filters", func(t *testing.T) {
		_, err := executeListIssues(`{"state":"all","labels":"bug","limit":"5"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
}

func TestExecuteUpdateWiki(t *testing.T) {
	_, err := executeUpdateWiki(`{"page":"TestPage","content":"hello"}`)
	if err != nil {
		t.Logf("expected non-fatal, got %v", err)
	}
}

func TestExecuteSwarmSpawn(t *testing.T) {
	t.Run("with name and port", func(t *testing.T) {
		// Creates /tmp/ivai-test-worker (harmless) and tries to launch binary
		_, err := executeSwarmSpawn(`{"name":"test-worker","port":"9999"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("default port", func(t *testing.T) {
		_, err := executeSwarmSpawn(`{"name":"test-worker2"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
}

func TestExecuteSwarmKill(t *testing.T) {
	t.Run("by port", func(t *testing.T) {
		_, err := executeSwarmKill(`{"port":"9999"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("by name", func(t *testing.T) {
		_, err := executeSwarmKill(`{"name":"test-worker"}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
	})
	t.Run("no args", func(t *testing.T) {
		result, err := executeSwarmKill(`{}`)
		if err != nil {
			t.Logf("expected non-fatal, got %v", err)
		}
		if result != "No port or name specified" {
			t.Errorf("expected no-op message, got %s", result)
		}
	})
}

func TestShutdownServerWithServer(t *testing.T) {
	server := &http.Server{Addr: ":0"}
	// Should not panic — server isn't started so Shutdown returns an error
	shutdownServer(server)
}

func TestResolvePathsOtherPlatform(t *testing.T) {
	t.Setenv("IVAI_DATA_DIR", "")
	// The path depends on runtime.GOOS — just verify it returns something reasonable
	env, db := resolvePaths()
	if env == "" || db == "" {
		t.Errorf("expected non-empty paths, got env=%s db=%s", env, db)
	}
}

func TestInitGatewayMissingKeys(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	gw := initGateway()
	if gw == nil {
		t.Fatal("expected non-nil gateway even with no keys")
	}
	if gw.DeepSeekKey != "" || gw.AnthropicKey != "" || gw.GeminiKey != "" {
		t.Error("expected all keys empty")
	}
}

func TestInjectRAGContextNoHistory(t *testing.T) {
	payload := []llm.Message{{Role: "system", Content: "test"}}
	// With empty history, injectRAGContext should return payload unchanged
	result := injectRAGContext(payload, nil, nil, []memory.Message{})
	if len(result) != 1 {
		t.Errorf("expected unchanged payload, got length %d", len(result))
	}
}

func TestInjectRAGContextEmbedError(t *testing.T) {
	payload := []llm.Message{{Role: "system", Content: "test"}}
	history := []memory.Message{{Role: "user", Content: "hello"}}
	// Gateway with embed URL pointing to a server that returns error
	embedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer embedSrv.Close()
	gw := llm.NewGateway("test-key", "", "")
	gw.DeepSeekEmbedURL = embedSrv.URL
	result := injectRAGContext(payload, nil, gw, history)
	if len(result) != 1 {
		t.Errorf("expected unchanged payload on embed error, got length %d", len(result))
	}
}

func TestShowThinking(t *testing.T) {
	t.Run("with reasoning", func(t *testing.T) {
		// Should not panic
		showThinking("step by step")
	})
	t.Run("without reasoning", func(t *testing.T) {
		showThinking("")
	})
}

func TestHandleWasmFileError(t *testing.T) {
	wasmEngine := sandbox.NewWasmRuntime()
	result, err := handleWasm(context.Background(), `{"filepath":"/nonexistent/test.wasm","payload":"","timeout_ms":100}`, wasmEngine)
	if err != nil {
		t.Logf("handleWasm returned err: %v", err)
	}
	if !strings.Contains(result, "Error:") {
		t.Errorf("expected Error: for nonexistent file, got %s", result)
	}
}

func TestResolvePortCustom(t *testing.T) {
	t.Setenv("IVAI_PORT", "9090")
	if p := resolvePort(); p != "9090" {
		t.Errorf("expected 9090, got %s", p)
	}
}

func TestProcessTaskWithNilDBStore(t *testing.T) {
	dbPath := "test_nil_db_task.db"
	defer os.Remove(dbPath)

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

	gateway := llm.NewGateway("key", "", "")
	gateway.DeepSeekURL = server.URL
	wasmEngine := sandbox.NewWasmRuntime()

	result := processTask(context.Background(), TaskInput{
		Instruction: "test",
		Gateway:     gateway,
		DBStore:     nil,
		WasmEngine:  wasmEngine,
	}, nil)
	// Should still work without DB store
	if result != "ok" {
		t.Errorf("expected ok, got %s", result)
	}
}

func TestStreamProgressEventsFullFlow(t *testing.T) {
	w := httptest.NewRecorder()
	progressChan := make(chan ProgressEvent, 1)
	respChan := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	progressChan <- ProgressEvent{Type: "thinking", Message: "thinking..."}
	respChan <- "final response"
	close(progressChan)

	done := make(chan struct{})
	go func() {
		streamProgressEvents(w, w, progressChan, respChan, ctx)
		close(done)
	}()
	<-done

	body := w.Body.String()
	if !strings.Contains(body, "thinking") || !strings.Contains(body, "task_complete") {
		t.Errorf("expected thinking and task_complete events, got: %s", body)
	}
}

func TestHandleMemoryErrorPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/memory?limit=999", nil)
	w := httptest.NewRecorder()
	// With a nil store, should return error
	handleMemory(w, req, nil)
	resp := w.Result()
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Memory store not available" {
		t.Errorf("expected memory error, got %v", body["error"])
	}
}

func TestHandleTaskResultsLimitParsing(t *testing.T) {
	dbPath := "test_tr_limit.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveTaskResult(memory.TaskResult{Instruction: "test", Success: true, DurationMs: 50})

	req := httptest.NewRequest("GET", "/api/task-results?limit=5", nil)
	w := httptest.NewRecorder()
	handleTaskResults(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	t.Logf("body: %+v", body)
	results := body["results"].([]any)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHandleEmbeddingsLimitParsing(t *testing.T) {
	dbPath := "test_emb_limit.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveEmbedding("test", "hi", []float64{0.1})

	req := httptest.NewRequest("GET", "/api/embeddings?limit=3", nil)
	w := httptest.NewRecorder()
	handleEmbeddings(w, req, store)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	results := body["embeddings"].([]any)
	if len(results) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(results))
	}
}

func TestHandleMemoryEmptyStore(t *testing.T) {
	dbPath := "test_mem_empty.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/memory?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	handleMemory(w, req, store)
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["total"].(float64) != 0 {
		t.Errorf("expected 0 total, got %v", body["total"])
	}
}

func TestHandleTaskResultsEmptyStore(t *testing.T) {
	dbPath := "test_tr_empty.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/task-results?limit=5", nil)
	w := httptest.NewRecorder()
	handleTaskResults(w, req, store)
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	results := body["results"].([]any)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestHandleEmbeddingsEmptyStore(t *testing.T) {
	dbPath := "test_emb_empty.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/embeddings?limit=5", nil)
	w := httptest.NewRecorder()
	handleEmbeddings(w, req, store)
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	results := body["embeddings"].([]any)
	if len(results) != 0 {
		t.Errorf("expected 0 embeddings, got %d", len(results))
	}
}

func TestInjectRAGContextFullPath(t *testing.T) {
	// Set up a store with an embedding
	dbPath := "test_rag_full.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.SaveEmbedding("past", "previous context", []float64{0.9, 0.1, 0.0})

	// Mock embed server returning a valid embedding
	embedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.85, 0.15, 0.0}},
			},
		})
	}))
	defer embedSrv.Close()

	gw := llm.NewGateway("test-key", "", "")
	gw.DeepSeekEmbedURL = embedSrv.URL

	payload := []llm.Message{{Role: "system", Content: "test"}}
	history := []memory.Message{{Role: "user", Content: "hello world"}}

	result := injectRAGContext(payload, store, gw, history)
	if len(result) != 2 {
		t.Errorf("expected 2 messages (system + RAG), got %d", len(result))
	}
	if !strings.Contains(result[1].Content, "previous context") {
		t.Errorf("expected RAG context to contain 'previous context', got: %s", result[1].Content)
	}
}

func TestRecordTaskResultWithGateError(t *testing.T) {
	dbPath := "test_rec_tr.db"
	defer os.Remove(dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Embed will fail silently (gateway has no valid URL)
	gw := llm.NewGateway("", "", "")
	state := &taskState{
		instruction: "test",
		model:       "deepseek-v4-pro",
		dbStore:     store,
		gateway:     gw,
	}
	recordTaskResult(state, "ok result", 50)
	// Should not panic even though embed fails
}

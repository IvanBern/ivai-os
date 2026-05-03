package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/memory"
	"github.com/IvanBern/ivai-os/internal/sandbox"
	"github.com/IvanBern/ivai-os/internal/telemetry"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
)

// ProgressEvent is emitted by the reasoning loop for SSE streaming.
type ProgressEvent struct {
	Type    string `json:"type"`    // task_start, thinking, tool_call, tool_result, task_complete, task_error
	Message string `json:"message"` // Human-readable description
	Data    any    `json:"data,omitempty"`
}

type TaskRequest struct {
	Instruction string `json:"instruction"`
}

type taskWithResponder struct {
	instruction  string
	responder    chan string
	progressChan chan<- ProgressEvent // optional SSE progress channel
}

//go:embed SYSTEM_PROMPT.md
var systemPromptTemplate string

// Build-time variables injected via ldflags (see Makefile).
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var startTime = time.Now()

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Ivai OS starting up...", "version", Version, "commit", Commit, "built", BuildDate)

	tp, err := telemetry.InitTracer("ivai-os")
	if err != nil {
		slog.Warn("Tracer init failed, continuing without telemetry", "error", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				slog.Warn("Tracer shutdown error", "error", err)
			}
		}()
	}

	envPath, dbPath := resolvePaths()
	if err := godotenv.Load(envPath); err == nil {
		slog.Info("Configuration loaded successfully", "path", envPath)
	}

	gateway := initGateway()
	dbStore := initMemory(dbPath)
	wasmEngine := initWasm()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	taskChan := make(chan taskWithResponder, 10)
	port := resolvePort()
	server := startHTTPServer(port, taskChan, gateway, dbStore)
	startCLI(taskChan)

	slog.Info("Ivai OS is now running. Awaiting input via CLI or port " + port + ".")
	runEventLoop(ctx, taskChan, server, gateway, dbStore, wasmEngine)
}

func resolvePaths() (envPath, dbPath string) {
	if dir := os.Getenv("IVAI_DATA_DIR"); dir != "" {
		return dir + "/.env", dir + "/memory.db"
	}
	if runtime.GOOS == "darwin" {
		return ".env", "memory.db"
	}
	return "/etc/ivai/.env", "/etc/ivai/memory.db"
}

func resolvePort() string {
	port := os.Getenv("IVAI_PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func initGateway() *llm.Gateway {
	deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if noLLMKeysConfigured(deepSeekKey, anthropicKey, geminiKey) {
		slog.Warn("No LLM API keys (DeepSeek, Anthropic, or Gemini) are set. LLM execution will fail.")
	}
	return llm.NewGateway(deepSeekKey, anthropicKey, geminiKey)
}

func noLLMKeysConfigured(keys ...string) bool {
	for _, k := range keys {
		if k != "" {
			return false
		}
	}
	return true
}

func initMemory(dbPath string) *memory.Store {
	slog.Info("Mounting persistent memory subsystem...", "path", dbPath)
	store, err := memory.NewStore(dbPath)
	if err != nil {
		slog.Warn("Memory store init failed, continuing without persistence", "error", err)
		return nil
	}
	slog.Info("Memory database mounted successfully")
	return store
}

func initWasm() *sandbox.WasmRuntime {
	slog.Info("Initializing Wazero execution sandbox...")
	engine := sandbox.NewWasmRuntime()
	slog.Info("Execution sandbox ready with strict millisecond timeouts")
	return engine
}

func runEventLoop(ctx context.Context, taskChan <-chan taskWithResponder, server *http.Server, gateway *llm.Gateway, dbStore *memory.Store, wasmEngine *sandbox.WasmRuntime) {
	for {
		select {
		case t := <-taskChan:
			go func(task taskWithResponder) {
				response := processTask(ctx, TaskInput{
					Instruction: task.instruction,
					Gateway:     gateway,
					DBStore:     dbStore,
					WasmEngine:  wasmEngine,
				}, task.progressChan)
				if task.responder != nil {
					task.responder <- response
				}
			}(t)

		case <-ctx.Done():
			shutdownServer(server)
			return
		}
	}
}

func shutdownServer(server *http.Server) {
	slog.Info("Shutting down Ivai OS...")
	if server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "err", err)
		}
	}
	slog.Info("Ivai OS gracefully stopped.")
}

func startHTTPServer(port string, taskChan chan<- taskWithResponder, gateway *llm.Gateway, dbStore *memory.Store) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		handleTaskBlocking(w, r, taskChan)
	})
	mux.HandleFunc("/api/task/stream", func(w http.ResponseWriter, r *http.Request) {
		handleTaskStreaming(w, r, taskChan)
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		handleStatus(w, r, gateway)
	})
	mux.HandleFunc("/api/memory", func(w http.ResponseWriter, r *http.Request) {
		handleMemory(w, r, dbStore)
	})
	mux.HandleFunc("/api/tools", func(w http.ResponseWriter, r *http.Request) {
		handleTools(w, r)
	})
	mux.HandleFunc("/api/task-results", func(w http.ResponseWriter, r *http.Request) {
		handleTaskResults(w, r, dbStore)
	})
	mux.HandleFunc("/api/system", func(w http.ResponseWriter, r *http.Request) {
		handleSystem(w, r, dbStore)
	})
	mux.HandleFunc("/api/embeddings", func(w http.ResponseWriter, r *http.Request) {
		handleEmbeddings(w, r, dbStore)
	})

	// Serve embedded web dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(dashboardHTML))
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP Server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	return server
}

func handleTaskBlocking(w http.ResponseWriter, r *http.Request, taskChan chan<- taskWithResponder) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	respChan := make(chan string)
	taskChan <- taskWithResponder{
		instruction: req.Instruction,
		responder:   respChan,
	}

	select {
	case finalResponse := <-respChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": finalResponse})
	case <-time.After(120 * time.Second):
		http.Error(w, "Task processing timed out", http.StatusGatewayTimeout)
	case <-r.Context().Done():
		return
	}
}

func handleTaskStreaming(w http.ResponseWriter, r *http.Request, taskChan chan<- taskWithResponder) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	setSSEHeaders(w)

	progressChan := make(chan ProgressEvent, 20)
	respChan := make(chan string, 1)

	taskChan <- taskWithResponder{
		instruction:  req.Instruction,
		responder:    respChan,
		progressChan: progressChan,
	}

	streamProgressEvents(w, flusher, progressChan, respChan, r.Context())
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func streamProgressEvents(w http.ResponseWriter, flusher http.Flusher, progressChan <-chan ProgressEvent, respChan <-chan string, ctx context.Context) {
	for {
		select {
		case event, ok := <-progressChan:
			if !ok {
				finalResponse := <-respChan
				writeSSEEvent(w, flusher, "task_complete", ProgressEvent{
					Type: "task_complete", Message: "Task completed",
					Data: map[string]string{"response": finalResponse},
				})
				return
			}
			writeSSEEvent(w, flusher, event.Type, event)

		case <-time.After(120 * time.Second):
			writeSSEEvent(w, flusher, "task_error", ProgressEvent{
				Type: "task_error", Message: "Task timed out",
				Data: map[string]string{"error": "timeout after 120s"},
			})
			return

		case <-ctx.Done():
			return
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, event ProgressEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
	flusher.Flush()
}

func startCLI(taskChan chan<- taskWithResponder) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		scanner := bufio.NewScanner(os.Stdin)
		printPrompt()
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				taskChan <- taskWithResponder{instruction: input}
			}
			time.Sleep(50 * time.Millisecond)
			printPrompt()
		}
	}()
}

func printPrompt() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Print("Ivai > ")
	}
}


// --- Web Dashboard API Handlers ---

func handleStatus(w http.ResponseWriter, r *http.Request, gateway *llm.Gateway) {
	w.Header().Set("Content-Type", "application/json")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	activeModel := "deepseek-v4-pro"
	models := []map[string]string{
		{"id": "deepseek-v4-pro", "provider": "DeepSeek", "available": strconv.FormatBool(gateway.DeepSeekKey != "")},
		{"id": "claude-3-5-sonnet-20241022", "provider": "Anthropic", "available": strconv.FormatBool(gateway.AnthropicKey != "")},
		{"id": "gemini-2.5-pro", "provider": "Gemini", "available": strconv.FormatBool(gateway.GeminiKey != "")},
	}

	json.NewEncoder(w).Encode(map[string]any{
		"version":       Version,
		"commit":        Commit,
		"build_date":    BuildDate,
		"uptime_sec":    int(time.Since(startTime).Seconds()),
		"go_version":    runtime.Version(),
		"goroutines":    runtime.NumGoroutine(),
		"heap_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"num_cpu":       runtime.NumCPU(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"active_model":  activeModel,
		"models":        models,
	})
}

func handleMemory(w http.ResponseWriter, r *http.Request, dbStore *memory.Store) {
	w.Header().Set("Content-Type", "application/json")

	if dbStore == nil {
		json.NewEncoder(w).Encode(map[string]any{"error": "Memory store not available", "messages": []memory.DashboardMessage{}})
		return
	}

	limit := parseQueryInt(r.URL.Query().Get("limit"), 50, func(v int) bool { return v > 0 && v <= 200 })
	offset := parseQueryInt(r.URL.Query().Get("offset"), 0, func(v int) bool { return v >= 0 })

	total, _ := dbStore.CountMessages()
	messages, err := dbStore.GetAllMessages(limit, offset)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	if messages == nil {
		messages = []memory.DashboardMessage{}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"messages": messages,
	})
}

func handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	toolList := buildTools()
	toolsInfo := make([]map[string]any, 0, len(toolList))
	for _, t := range toolList {
		toolsInfo = append(toolsInfo, map[string]any{
			"name":        t.Function.Name,
			"description": t.Function.Description,
			"parameters":  t.Function.Parameters,
		})
	}
	json.NewEncoder(w).Encode(map[string]any{
		"tools": toolsInfo,
	})
}

func handleTaskResults(w http.ResponseWriter, r *http.Request, dbStore *memory.Store) {
	w.Header().Set("Content-Type", "application/json")
	if dbStore == nil {
		json.NewEncoder(w).Encode(map[string]any{"error": "Memory store not available", "results": []memory.TaskResult{}})
		return
	}
	limit := parseQueryInt(r.URL.Query().Get("limit"), 20, func(v int) bool { return v > 0 && v <= 200 })
	results, err := dbStore.GetTaskResults(limit)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	if results == nil {
		results = []memory.TaskResult{}
	}
	json.NewEncoder(w).Encode(map[string]any{
		"results": results,
	})
}

func parseQueryInt(s string, defaultVal int, validate func(int) bool) int {
	v, err := strconv.Atoi(s)
	if err != nil || !validate(v) {
		return defaultVal
	}
	return v
}

func handleSystem(w http.ResponseWriter, r *http.Request, dbStore *memory.Store) {
	w.Header().Set("Content-Type", "application/json")
	if dbStore == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"system_prompt":    systemPromptTemplate,
			"embeddings_count": 0,
			"messages_count":   0,
			"task_stats":       memory.TaskStats{},
		})
		return
	}
	embCount, _ := dbStore.CountEmbeddings()
	msgCount, _ := dbStore.CountMessages()
	stats, _ := dbStore.GetTaskStats()
	json.NewEncoder(w).Encode(map[string]any{
		"system_prompt":    systemPromptTemplate,
		"embeddings_count": embCount,
		"messages_count":   msgCount,
		"task_stats":       stats,
	})
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request, dbStore *memory.Store) {
	w.Header().Set("Content-Type", "application/json")
	if dbStore == nil {
		json.NewEncoder(w).Encode(map[string]any{"error": "Memory store not available", "embeddings": []memory.EmbeddingResult{}})
		return
	}
	limit := parseQueryInt(r.URL.Query().Get("limit"), 50, func(v int) bool { return v > 0 && v <= 200 })
	results, err := dbStore.GetRecentEmbeddings(limit)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	if results == nil {
		results = []memory.EmbeddingResult{}
	}
	json.NewEncoder(w).Encode(map[string]any{"embeddings": results})
}


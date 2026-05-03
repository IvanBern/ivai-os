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
	"github.com/IvanBern/ivai-os/internal/tools"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
		slog.Error("Failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Tracer shutdown error", "error", err)
		}
	}()

	envPath, dbPath := resolvePaths()
	if err := godotenv.Load(envPath); err == nil {
		slog.Info("Configuration loaded successfully", "path", envPath)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	taskChan := make(chan taskWithResponder, 10)

	gateway, dbStore, wasmEngine := initDependencies(dbPath)

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

func initDependencies(dbPath string) (*llm.Gateway, *memory.Store, *sandbox.WasmRuntime) {
	deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	noKeys := deepSeekKey == "" && anthropicKey == "" && geminiKey == ""
	if noKeys {
		slog.Warn("No LLM API keys (DeepSeek, Anthropic, or Gemini) are set. LLM execution will fail.")
	}

	gateway := llm.NewGateway(deepSeekKey, anthropicKey, geminiKey)

	slog.Info("Mounting persistent memory subsystem...", "path", dbPath)
	dbStore, err := memory.NewStore(dbPath)
	if err != nil {
		slog.Error("Failed to initialize memory database", "error", err)
		os.Exit(1)
	}
	slog.Info("Memory database mounted successfully")

	slog.Info("Initializing Wazero execution sandbox...")
	wasmEngine := sandbox.NewWasmRuntime()
	slog.Info("Execution sandbox ready with strict millisecond timeouts")

	return gateway, dbStore, wasmEngine
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

type TaskInput struct {
	Instruction string
	Gateway     *llm.Gateway
	DBStore     *memory.Store
	WasmEngine  *sandbox.WasmRuntime
}

type taskState struct {
	gateway      *llm.Gateway
	dbStore      *memory.Store
	wasmEngine   *sandbox.WasmRuntime
	tools        []llm.Tool
	model        string
	progressChan chan<- ProgressEvent
}

func (s *taskState) emit(evt ProgressEvent) {
	if s.progressChan == nil {
		return
	}
	select {
	case s.progressChan <- evt:
	default:
	}
}

func processTask(ctx context.Context, in TaskInput, progressChan chan<- ProgressEvent) string {
	model, instruction := extractModel(in.Instruction)
	slog.Info("Task routing", "model", model, "instruction", instruction)

	in.DBStore.SaveMessage("user", instruction, "")

	state := &taskState{
		gateway:      in.Gateway,
		dbStore:      in.DBStore,
		wasmEngine:   in.WasmEngine,
		tools:        buildTools(),
		model:        model,
		progressChan: progressChan,
	}

	state.emit(ProgressEvent{Type: "task_start", Message: "Task started", Data: map[string]string{"model": model, "instruction": instruction}})

	startTime := time.Now()
	payload := buildPayload(in.DBStore, in.Gateway)
	result := runReasoningLoop(ctx, payload, state)
	duration := time.Since(startTime).Milliseconds()

	// Track result for self-evolution
	success := !strings.HasPrefix(result, "Error: ")
	errMsg := ""
	if !success {
		errMsg = result
	}
	in.DBStore.SaveTaskResult(memory.TaskResult{
		Instruction: instruction,
		Model:       model,
		Success:     success,
		Response:    result,
		ErrorMsg:    errMsg,
		DurationMs:  duration,
	})

	// Auto-embed the instruction for future semantic recall (fire and forget)
	go func() {
		emb, err := in.Gateway.Embed(context.Background(), instruction)
		if err != nil {
			return
		}
		in.DBStore.SaveEmbedding("instruction", instruction, emb)
	}()

	// Close progress channel when done (if it exists) to signal completion to SSE handler.
	if progressChan != nil {
		close(progressChan)
	}

	return result
}

func extractModel(t string) (model, instruction string) {
	model = "deepseek-v4-pro"
	instruction = t

	lower := strings.ToLower(t)
	switch {
	case strings.Contains(lower, "@claude"):
		return "claude-3-5-sonnet-20241022", strings.Replace(t, "@claude", "", 1)
	case strings.Contains(lower, "@gemini"):
		return "gemini-2.5-pro", strings.Replace(t, "@gemini", "", 1)
	case strings.Contains(lower, "@deepseek"):
		return "deepseek-v4-pro", strings.Replace(t, "@deepseek", "", 1)
	case strings.Contains(lower, "@research"):
		return "deep-research-max-preview", strings.Replace(t, "@research", "", 1)
	default:
		return model, instruction
	}
}

func buildTools() []llm.Tool {
	define := func(name, desc string, props map[string]any, required []string) llm.Tool {
		return llm.Tool{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters: map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		}
	}

	strProp := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}

	return []llm.Tool{
		define("read_file", "Reads the contents of a file at the given path on the local filesystem.",
			map[string]any{"filepath": map[string]any{"type": "string"}},
			[]string{"filepath"}),

		define("write_file", "Writes text content to a file at the given path, overwriting it if it exists.",
			map[string]any{
				"filepath": map[string]any{"type": "string"},
				"content":  map[string]any{"type": "string"},
			},
			[]string{"filepath", "content"}),

		define("execute_command", "Executes a bash shell command on the host Debian system and returns the output.",
			map[string]any{"command": map[string]any{"type": "string"}},
			[]string{"command"}),

		define("execute_wasm", "Executes a compiled WebAssembly (.wasm) binary in a secure, isolated sandbox with strict timeouts. Passes data via stdin and returns stdout.",
			map[string]any{
				"filepath":   strProp("Absolute path to the .wasm file on disk"),
				"payload":    strProp("Data to send to the Wasm module via standard input (stdin)"),
				"timeout_ms": map[string]any{"type": "integer", "description": "Execution timeout in milliseconds (e.g., 1000)"},
			},
			[]string{"filepath", "payload", "timeout_ms"}),

		define("http_request", "Performs an HTTP request (GET, POST, etc.) and returns the response body.",
			map[string]any{
				"method":  strProp("HTTP method (e.g., GET, POST)"),
				"url":     strProp("Target URL"),
				"body":    strProp("Request body (optional)"),
				"headers": map[string]any{"type": "object", "description": "HTTP headers (optional)"},
			},
			[]string{"method", "url"}),

		define("github_pr", "Creates a GitHub Pull Request from a repo directory. Uses the gh CLI (must be authenticated). Provide title, body, repo path, and optionally a base branch.",
			map[string]any{
				"title": strProp("Pull request title"),
				"body":  strProp("Pull request description"),
				"base":  strProp("Target branch (default: main)"),
				"repo":  strProp("Path to the git repository (e.g., /tmp/ivai-sandbox)"),
			},
			[]string{"title", "body", "repo"}),

		define("code_health", "Runs CodeScene delta analysis on a git repository to check code health. Returns issues found or No issues found!.",
			map[string]any{
				"repo": strProp("Path to the git repository (e.g., /tmp/ivai-sandbox)"),
			},
			[]string{"repo"}),

		define("create_issue", "Creates a GitHub Issue. Uses gh CLI. Provide title, body, labels (comma-separated), and optional assignee.",
			map[string]any{
				"title":    strProp("Issue title"),
				"body":     strProp("Issue description"),
				"labels":   strProp("Comma-separated labels (e.g., bug,phase-13)"),
				"assignee": strProp("GitHub username to assign (optional)"),
			},
			[]string{"title", "body"}),

		define("list_issues", "Lists GitHub Issues with optional filters.",
			map[string]any{
				"state":  strProp("open, closed, or all (default: open)"),
				"labels": strProp("Comma-separated label filter (optional)"),
				"limit":  strProp("Max issues to return (default: 10)"),
			},
			[]string{}),
	}
}

func buildPayload(dbStore *memory.Store, gateway *llm.Gateway) []llm.Message {
	history, _ := dbStore.GetRecentMessages(10)
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	payload := []llm.Message{
		{Role: "system", Content: fmt.Sprintf(systemPromptTemplate, homeDir, cwd)},
	}

	// RAG: inject semantically similar past context
	payload = injectRAGContext(payload, dbStore, gateway, history)

	for _, msg := range history {
		payload = append(payload, llm.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		})
	}
	return payload
}

func injectRAGContext(payload []llm.Message, dbStore *memory.Store, gateway *llm.Gateway, history []memory.Message) []llm.Message {
	if len(history) == 0 {
		return payload
	}
	latestMsg := history[len(history)-1].Content
	emb, err := gateway.Embed(context.Background(), latestMsg)
	if err != nil {
		return payload
	}
	similar, err := dbStore.SearchSimilar(emb, 3)
	if err != nil || len(similar) == 0 {
		return payload
	}
	ragCtx := "## Relevant Past Context (from semantic memory)\n"
	for i, s := range similar {
		ragCtx += fmt.Sprintf("%d. [%.0f%% match] %s\n", i+1, s.Similarity*100, s.Content)
	}
	return append(payload, llm.Message{Role: "system", Content: ragCtx})
}

func runReasoningLoop(ctx context.Context, payload []llm.Message, s *taskState) string {
	tracer := otel.Tracer("ivai-os")
	for {
		ctx, span := tracer.Start(ctx, "reasoning-step",
			trace.WithAttributes(
				attribute.String("model", s.model),
				attribute.Int("messages", len(payload)),
			),
		)
		responseMsg, err := s.gateway.Chat(ctx, payload, s.tools, s.model)
		if err != nil {
			slog.Error("LLM Execution Failed", "error", err)
			s.emit(ProgressEvent{Type: "task_error", Message: "LLM error", Data: map[string]string{"error": err.Error()}})
			span.End()
			printPrompt()
			return "Error: " + err.Error()
		}

		if done, result := checkCompletion(responseMsg, s); done {
			span.End()
			return result
		}

		showThinking(responseMsg.ReasoningContent)
		s.emit(ProgressEvent{
			Type:    "thinking",
			Message: "Model is thinking",
			Data:    map[string]string{"reasoning": responseMsg.ReasoningContent, "content": responseMsg.Content},
		})
		payload = append(payload, responseMsg)
		payload = appendToolResults(ctx, payload, responseMsg.ToolCalls, s)
		span.End()
	}
}

func checkCompletion(msg llm.Message, s *taskState) (done bool, result string) {
	if len(msg.ToolCalls) > 0 {
		return false, ""
	}
	slog.Info("Task completed", "response_length", len(msg.Content))
	s.dbStore.SaveMessage("assistant", msg.Content, msg.ReasoningContent)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Printf("\n[Ivai] %s\n", msg.Content)
	}
	printPrompt()
	return true, msg.Content
}

func showThinking(reasoningContent string) {
	if reasoningContent != "" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			fmt.Printf("\n[Thinking] %s\n", reasoningContent)
		} else {
			slog.Info("Thinking", "reasoning", reasoningContent)
		}
	} else {
		slog.Info("Thinking...")
	}
}

func appendToolResults(ctx context.Context, payload []llm.Message, toolCalls []llm.ToolCall, s *taskState) []llm.Message {
	for _, tc := range toolCalls {
		slog.Info("Executing tool", "name", tc.Function.Name, "args", tc.Function.Arguments)
		s.emit(ProgressEvent{
			Type:    "tool_call",
			Message: fmt.Sprintf("Calling tool: %s", tc.Function.Name),
			Data:    map[string]any{"name": tc.Function.Name, "args": tc.Function.Arguments},
		})
		toolResult := executeToolCall(ctx, tc, s.wasmEngine)
		s.emit(ProgressEvent{
			Type:    "tool_result",
			Message: fmt.Sprintf("Tool result: %s", tc.Function.Name),
			Data:    map[string]any{"name": tc.Function.Name, "result": truncate(toolResult, 500)},
		})
		payload = append(payload, llm.Message{
			Role:       "tool",
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
	}
	return payload
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func executeGitHubPR(argsJSON string) (string, error) {
	var args struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Base  string `json:"base"`
		Repo  string `json:"repo"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	base := args.Base
	if base == "" {
		base = "main"
	}
	cmd := fmt.Sprintf("gh pr create --title %q --body %q --base %s", args.Title, args.Body, base)
	if args.Repo != "" {
		cmd = fmt.Sprintf("cd %s && %s", args.Repo, cmd)
	}
	return tools.ExecuteCommand(cmd)
}

func executeCodeHealth(repoPath string) (string, error) {
	body, _ := json.Marshal(map[string]string{"repo": repoPath})
	result, err := tools.HttpRequest("POST", "http://host.orb.internal:9876/", string(body), map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return "", err
	}
	var resp struct {
		OK     bool   `json:"ok"`
		Output string `json:"output"`
	}
	json.Unmarshal([]byte(result), &resp)
	return resp.Output, nil
}

func executeCodeHealthTool(argsJSON string) (string, error) {
	var args struct {
		Repo string `json:"repo"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	return executeCodeHealth(args.Repo)
}

func executeCreateIssue(argsJSON string) (string, error) {
	var args struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Labels   string `json:"labels"`
		Assignee string `json:"assignee"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	cmd := fmt.Sprintf("gh issue create --repo IvanBern/ivai-os --title %q --body %q", args.Title, args.Body)
	if args.Labels != "" {
		cmd += fmt.Sprintf(" --label %q", args.Labels)
	}
	if args.Assignee != "" {
		cmd += fmt.Sprintf(" --assignee %q", args.Assignee)
	}
	return tools.ExecuteCommand(cmd)
}

func executeListIssues(argsJSON string) (string, error) {
	var args struct {
		State  string `json:"state"`
		Labels string `json:"labels"`
		Limit  string `json:"limit"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	if args.State == "" {
		args.State = "open"
	}
	if args.Limit == "" {
		args.Limit = "10"
	}
	cmd := fmt.Sprintf("gh issue list --repo IvanBern/ivai-os --state %s --limit %s --json title,state,labels,assignees", args.State, args.Limit)
	if args.Labels != "" {
		cmd += fmt.Sprintf(" --label %q", args.Labels)
	}
	return tools.ExecuteCommand(cmd)
}

func resultOrError(result string, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return result
}

func executeToolCall(ctx context.Context, tc llm.ToolCall, wasmEngine *sandbox.WasmRuntime) string {
	tracer := otel.Tracer("ivai-os")
	ctx, span := tracer.Start(ctx, "tool."+tc.Function.Name,
		trace.WithAttributes(
			attribute.String("tool.name", tc.Function.Name),
			attribute.Int("tool.args_len", len(tc.Function.Arguments)),
		),
	)
	defer span.End()

	switch tc.Function.Name {
	case "read_file":
		var args struct {
			Filepath string `json:"filepath"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return resultOrError(tools.ReadFile(args.Filepath))

	case "write_file":
		var args struct {
			Filepath string `json:"filepath"`
			Content  string `json:"content"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return resultOrError("File written successfully.", tools.WriteFile(args.Filepath, args.Content))

	case "execute_command":
		var args struct {
			Command string `json:"command"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return resultOrError(tools.ExecuteCommand(args.Command))

	case "execute_wasm":
		var args struct {
			Filepath  string `json:"filepath"`
			Payload   string `json:"payload"`
			TimeoutMs int    `json:"timeout_ms"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		wasmBytes, err := os.ReadFile(args.Filepath)
		if err != nil {
			return fmt.Sprintf("Error reading Wasm file: %v", err)
		}
		return resultOrError(wasmEngine.Execute(ctx, wasmBytes, args.Payload, args.TimeoutMs))

	case "http_request":
		var args struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Body    string            `json:"body"`
			Headers map[string]string `json:"headers"`
		}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return resultOrError(tools.HttpRequest(args.Method, args.URL, args.Body, args.Headers))

	case "github_pr":
		output, err := executeGitHubPR(tc.Function.Arguments)
		return resultOrError(output, err)

	case "code_health":
		output, err := executeCodeHealthTool(tc.Function.Arguments)
		return resultOrError(output, err)

	case "create_issue":
		output, err := executeCreateIssue(tc.Function.Arguments)
		return resultOrError(output, err)

	case "list_issues":
		output, err := executeListIssues(tc.Function.Arguments)
		return resultOrError(output, err)

	default:
		return fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
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

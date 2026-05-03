package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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
	"go.opentelemetry.io/otel/trace"
)

type TaskRequest struct {
	Instruction string `json:"instruction"`
}

type taskWithResponder struct {
	instruction string
	responder   chan string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Ivai OS starting up...", "version", "0.1.0")

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

	port := resolvePort()
	server := startHTTPServer(port, taskChan)
	startCLI(taskChan)

	gateway, dbStore, wasmEngine := initDependencies(dbPath)

	slog.Info("Ivai OS is now running. Awaiting input via CLI or port 8080.")
	runEventLoop(ctx, taskChan, server, gateway, dbStore, wasmEngine)
}

func resolvePaths() (envPath, dbPath string) {
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
			})
				if task.responder != nil {
					task.responder <- response
				}
			}(t)

		case <-ctx.Done():
			slog.Info("Shutting down Ivai OS...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				slog.Error("HTTP server shutdown error", "err", err)
			}
			slog.Info("Ivai OS gracefully stopped.")
			return
		}
	}
}

func startHTTPServer(port string, taskChan chan<- taskWithResponder) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
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
		go func() {
			taskChan <- taskWithResponder{
				instruction: req.Instruction,
				responder:   respChan,
			}
		}()

		select {
		case finalResponse := <-respChan:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"response": finalResponse})
		case <-time.After(120 * time.Second):
			http.Error(w, "Task processing timed out", http.StatusGatewayTimeout)
		case <-r.Context().Done():
			return
		}
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
	gateway    *llm.Gateway
	dbStore    *memory.Store
	wasmEngine *sandbox.WasmRuntime
	tools      []llm.Tool
	model      string
}

func processTask(ctx context.Context, in TaskInput) string {
	model, instruction := extractModel(in.Instruction)
	slog.Info("Task routing", "model", model, "instruction", instruction)

	in.DBStore.SaveMessage("user", instruction, "")

	state := &taskState{
		gateway:    in.Gateway,
		dbStore:    in.DBStore,
		wasmEngine: in.WasmEngine,
		tools:      buildTools(),
		model:      model,
	}

	payload := buildPayload(in.DBStore, instruction)
	return runReasoningLoop(ctx, payload, state)
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
	define := func(name, desc string, props map[string]interface{}, required []string) llm.Tool {
		return llm.Tool{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        name,
				Description: desc,
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		}
	}

	strProp := func(desc string) map[string]interface{} {
		return map[string]interface{}{"type": "string", "description": desc}
	}

	return []llm.Tool{
		define("read_file", "Reads the contents of a file at the given path on the local filesystem.",
			map[string]interface{}{"filepath": map[string]interface{}{"type": "string"}},
			[]string{"filepath"}),

		define("write_file", "Writes text content to a file at the given path, overwriting it if it exists.",
			map[string]interface{}{
				"filepath": map[string]interface{}{"type": "string"},
				"content":  map[string]interface{}{"type": "string"},
			},
			[]string{"filepath", "content"}),

		define("execute_command", "Executes a bash shell command on the host Debian system and returns the output.",
			map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
			[]string{"command"}),

		define("execute_wasm", "Executes a compiled WebAssembly (.wasm) binary in a secure, isolated sandbox with strict timeouts. Passes data via stdin and returns stdout.",
			map[string]interface{}{
				"filepath":   strProp("Absolute path to the .wasm file on disk"),
				"payload":    strProp("Data to send to the Wasm module via standard input (stdin)"),
				"timeout_ms": map[string]interface{}{"type": "integer", "description": "Execution timeout in milliseconds (e.g., 1000)"},
			},
			[]string{"filepath", "payload", "timeout_ms"}),

		define("http_request", "Performs an HTTP request (GET, POST, etc.) and returns the response body.",
			map[string]interface{}{
				"method":  strProp("HTTP method (e.g., GET, POST)"),
				"url":     strProp("Target URL"),
				"body":    strProp("Request body (optional)"),
				"headers": map[string]interface{}{"type": "object", "description": "HTTP headers (optional)"},
			},
			[]string{"method", "url"}),
	}
}

func buildPayload(dbStore *memory.Store, instruction string) []llm.Message {
	history, _ := dbStore.GetRecentMessages(10)
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	systemPrompt := fmt.Sprintf(
		"You are Ivai, an advanced AI Operating System. Your home directory is %s. You are currently running in %s. Use your tools to interact with the filesystem. You have 'git' installed.",
		homeDir, cwd,
	)

	payload := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}
	for _, msg := range history {
		payload = append(payload, llm.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		})
	}
	return payload
}

func runReasoningLoop(ctx context.Context, payload []llm.Message, s *taskState) string {
	tracer := otel.Tracer("ivai-os")
	for {
		ctx, span := tracer.Start(ctx, "reasoning-step",
			trace.WithAttributes(),
		)
		responseMsg, err := s.gateway.Chat(ctx, payload, s.tools, s.model)
		if err != nil {
			slog.Error("LLM Execution Failed", "error", err)
			span.End()
			printPrompt()
			return "Error: " + err.Error()
		}

		if done, result := checkCompletion(responseMsg, s.dbStore); done {
			span.End()
			return result
		}

		showThinking(responseMsg.ReasoningContent)
		payload = append(payload, responseMsg)
		payload = appendToolResults(ctx, payload, responseMsg.ToolCalls, s.wasmEngine)
		span.End()
	}
}

func checkCompletion(msg llm.Message, dbStore *memory.Store) (done bool, result string) {
	if len(msg.ToolCalls) > 0 {
		return false, ""
	}
	slog.Info("Task completed", "response_length", len(msg.Content))
	dbStore.SaveMessage("assistant", msg.Content, msg.ReasoningContent)
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

func appendToolResults(ctx context.Context, payload []llm.Message, toolCalls []llm.ToolCall, wasmEngine *sandbox.WasmRuntime) []llm.Message {
	for _, tc := range toolCalls {
		slog.Info("Executing tool", "name", tc.Function.Name, "args", tc.Function.Arguments)
		toolResult := executeToolCall(ctx, tc, wasmEngine)
		payload = append(payload, llm.Message{
			Role:       "tool",
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
	}
	return payload
}

func resultOrError(result string, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return result
}

func executeToolCall(ctx context.Context, tc llm.ToolCall, wasmEngine *sandbox.WasmRuntime) string {
	tracer := otel.Tracer("ivai-os")
	ctx, span := tracer.Start(ctx, "tool."+tc.Function.Name)
	defer span.End()

	switch tc.Function.Name {
	case "read_file":
		var args struct{ Filepath string `json:"filepath"` }
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
		var args struct{ Command string `json:"command"` }
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

	default:
		return fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
	}
}

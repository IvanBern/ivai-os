package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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
	"github.com/IvanBern/ivai-os/internal/tools"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
)

// Task payload structure for the HTTP server
type TaskRequest struct {
	Instruction string `json:"instruction"`
}

func main() {
	// 1. Initialize Logger & Config
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Ivai OS starting up...", "version", "0.1.0")

	// Determine paths based on OS
	envPath := "/etc/ivai/.env"
	dbPath := "/etc/ivai/memory.db"
	if runtime.GOOS == "darwin" {
		envPath = ".env"
		dbPath = "memory.db"
	}

	if err := godotenv.Load(envPath); err == nil {
		slog.Info("Configuration loaded successfully", "path", envPath)
	}

	// 2. Setup Context and Channels
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	type taskWithResponder struct {
		instruction string
		responder   chan string
	}

	// This channel is Ivai's "ear". Both CLI and HTTP will send tasks here.
	taskChan := make(chan taskWithResponder, 10)

	// 3. Start the HTTP Server (Background Goroutine)
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

		// Channel to receive the final response
		respChan := make(chan string)
		
		// Send task to the central channel with a responder
		go func() {
			taskChan <- taskWithResponder{
				instruction: req.Instruction,
				responder:   respChan,
			}
		}()

		// Wait for the reasoning loop to finish
		select {
		case finalResponse := <-respChan:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"response": finalResponse})
		case <-time.After(120 * time.Second): // 2 minute timeout for complex reasoning
			http.Error(w, "Task processing timed out", http.StatusGatewayTimeout)
		case <-r.Context().Done():
			return
		}
	})

	port := os.Getenv("IVAI_PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			slog.Error("HTTP server failed to bind", "port", port, "err", err)
			return
		}
		slog.Info("HTTP Server listening", "port", port)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	// 4. Start the CLI REPL (Background Goroutine)
	go func() {
		// Small delay so the prompt appears after initialization logs
		time.Sleep(100 * time.Millisecond)
		scanner := bufio.NewScanner(os.Stdin)
		printPrompt()
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				taskChan <- taskWithResponder{instruction: input} // No responder for REPL, prints to stdout
			}
			// Wait a moment for logs to print before re-prompting
			time.Sleep(50 * time.Millisecond)
			printPrompt()
		}
	}()

	// Initialize the LLM Gateway
	deepSeekKey := os.Getenv("DEEPSEEK_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if deepSeekKey == "" && anthropicKey == "" && geminiKey == "" {
		slog.Warn("No LLM API keys (DeepSeek, Anthropic, or Gemini) are set. LLM execution will fail.")
	}
	gateway := llm.NewGateway(deepSeekKey, anthropicKey, geminiKey)

	// Initialize the Persistent Memory Subsystem
	slog.Info("Mounting persistent memory subsystem...", "path", dbPath)
	dbStore, err := memory.NewStore(dbPath)
	if err != nil {
		slog.Error("Failed to initialize memory database", "error", err)
		os.Exit(1)
	}
	slog.Info("Memory database mounted successfully")

	// Initialize the Wasm Execution Sandbox
	slog.Info("Initializing Wazero execution sandbox...")
	wasmEngine := sandbox.NewWasmRuntime()
	slog.Info("Execution sandbox ready with strict millisecond timeouts")

	slog.Info("Ivai OS is now running. Awaiting input via CLI or port 8080.")

	// 5. The Main OS Event Loop
	for {
		select {
		case t := <-taskChan:
			// Process tasks asynchronously
			go func(task taskWithResponder) {
				response := processTask(ctx, task.instruction, gateway, dbStore, wasmEngine)
				if task.responder != nil {
					task.responder <- response
				}
			}(t)

		case <-ctx.Done():
			slog.Info("Shutting down Ivai OS...")

			// Gracefully shutdown the HTTP server
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

func printPrompt() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Print("Ivai > ")
	}
}

func processTask(ctx context.Context, t string, gateway *llm.Gateway, dbStore *memory.Store, wasmEngine *sandbox.WasmRuntime) string {
	// 0. Determine which model to use (default to deepseek-v4-pro)
	model := "deepseek-v4-pro"
	if strings.Contains(strings.ToLower(t), "@claude") {
		model = "claude-3-5-sonnet-20241022"
		t = strings.Replace(t, "@claude", "", 1)
	} else if strings.Contains(strings.ToLower(t), "@gemini") {
		model = "gemini-2.5-pro"
		t = strings.Replace(t, "@gemini", "", 1)
	} else if strings.Contains(strings.ToLower(t), "@deepseek") {
		model = "deepseek-v4-pro"
		t = strings.Replace(t, "@deepseek", "", 1)
	} else if strings.Contains(strings.ToLower(t), "@research") {
		model = "deep-research-max-preview"
		t = strings.Replace(t, "@research", "", 1)
	}

	slog.Info("Task routing", "model", model, "instruction", t)

	// 1. Define the tools available to Ivai
	availableTools := []llm.Tool{
		{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        "read_file",
				Description: "Reads the contents of a file at the given path on the local filesystem.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filepath": map[string]interface{}{"type": "string"},
					},
					"required": []string{"filepath"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        "write_file",
				Description: "Writes text content to a file at the given path, overwriting it if it exists.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filepath": map[string]interface{}{"type": "string"},
						"content":  map[string]interface{}{"type": "string"},
					},
					"required": []string{"filepath", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        "execute_command",
				Description: "Executes a bash shell command on the host Debian system and returns the output.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{"type": "string"},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        "execute_wasm",
				Description: "Executes a compiled WebAssembly (.wasm) binary in a secure, isolated sandbox with strict timeouts. Passes data via stdin and returns stdout.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filepath":   map[string]interface{}{"type": "string", "description": "Absolute path to the .wasm file on disk"},
						"payload":    map[string]interface{}{"type": "string", "description": "Data to send to the Wasm module via standard input (stdin)"},
						"timeout_ms": map[string]interface{}{"type": "integer", "description": "Execution timeout in milliseconds (e.g., 1000)"},
					},
					"required": []string{"filepath", "payload", "timeout_ms"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        "http_request",
				Description: "Performs an HTTP request (GET, POST, etc.) and returns the response body.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"method":  map[string]interface{}{"type": "string", "description": "HTTP method (e.g., GET, POST)"},
						"url":     map[string]interface{}{"type": "string", "description": "Target URL"},
						"body":    map[string]interface{}{"type": "string", "description": "Request body (optional)"},
						"headers": map[string]interface{}{"type": "object", "description": "HTTP headers (optional)"},
					},
					"required": []string{"method", "url"},
				},
			},
		},
	}

	// 2. Save user prompt to memory
	dbStore.SaveMessage("user", t)
	history, _ := dbStore.GetRecentMessages(10)

	// 3. Construct the payload
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	systemPrompt := fmt.Sprintf("You are Ivai, an advanced AI Operating System. Your home directory is %s. You are currently running in %s. Use your tools to interact with the filesystem. You have 'git' installed.", homeDir, cwd)

	var payload []llm.Message
	payload = append(payload, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	for _, msg := range history {
		payload = append(payload, llm.Message{Role: msg.Role, Content: msg.Content})
	}

	// 4. Send to DeepSeek and loop until it stops requesting tools
	for {
		responseMsg, err := gateway.Chat(ctx, payload, availableTools, model) 
		if err != nil {
			slog.Error("LLM Execution Failed", "error", err)
			printPrompt()
			return "Error: " + err.Error()
		}
		
		// If there are no tool calls, it's a final text response. We are done!
		if len(responseMsg.ToolCalls) == 0 {
			slog.Info("Task completed", "response_length", len(responseMsg.Content))
			dbStore.SaveMessage("assistant", responseMsg.Content)
			if isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf("\n[Ivai] %s\n", responseMsg.Content)
			}
			printPrompt()
			return responseMsg.Content
		}

		// Otherwise, the LLM wants to execute tools.
		if responseMsg.ReasoningContent != "" {
			if isatty.IsTerminal(os.Stdout.Fd()) {
				fmt.Printf("\n[Thinking] %s\n", responseMsg.ReasoningContent)
			} else {
				slog.Info("Thinking", "reasoning", responseMsg.ReasoningContent)
			}
		} else {
			slog.Info("Thinking...")
		}
		
		// Append the LLM's tool request to history
		payload = append(payload, responseMsg)

		// Execute all requested tools
		for _, toolCall := range responseMsg.ToolCalls {
			slog.Info("Executing tool", "name", toolCall.Function.Name)
			
			var toolResult string

			if toolCall.Function.Name == "read_file" {
				var args struct{ Filepath string `json:"filepath"` }
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				content, err := tools.ReadFile(args.Filepath)
				if err != nil { toolResult = fmt.Sprintf("Error: %v", err) } else { toolResult = content }
				
			} else if toolCall.Function.Name == "write_file" {
				var args struct {
					Filepath string `json:"filepath"`
					Content  string `json:"content"`
				}
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				err := tools.WriteFile(args.Filepath, args.Content)
				if err != nil { toolResult = fmt.Sprintf("Error: %v", err) } else { toolResult = "File written successfully." }
				
			} else if toolCall.Function.Name == "execute_command" {
				var args struct{ Command string `json:"command"` }
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				output, err := tools.ExecuteCommand(args.Command)
				if err != nil { toolResult = fmt.Sprintf("Error: %v", err) } else { toolResult = output }
				
			} else if toolCall.Function.Name == "execute_wasm" {
				var args struct {
					Filepath  string `json:"filepath"`
					Payload   string `json:"payload"`
					TimeoutMs int    `json:"timeout_ms"`
				}
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				
				wasmBytes, err := os.ReadFile(args.Filepath)
				if err != nil {
					toolResult = fmt.Sprintf("Error reading Wasm file: %v", err)
				} else {
					output, err := wasmEngine.Execute(ctx, wasmBytes, args.Payload, args.TimeoutMs)
					if err != nil { toolResult = fmt.Sprintf("Sandbox error: %v", err) } else { toolResult = output }
				}
			} else if toolCall.Function.Name == "http_request" {
				var args struct {
					Method  string            `json:"method"`
					URL     string            `json:"url"`
					Body    string            `json:"body"`
					Headers map[string]string `json:"headers"`
				}
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				output, err := tools.HttpRequest(args.Method, args.URL, args.Body, args.Headers)
				if err != nil { toolResult = fmt.Sprintf("HTTP error: %v", err) } else { toolResult = output }
			}

			// Append the tool result to the payload so the LLM can read it in the next loop iteration
			payload = append(payload, llm.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})
		}
	}
}

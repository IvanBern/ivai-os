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
	"strings"
	"syscall"
	"time"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/memory"
	"github.com/IvanBern/ivai-os/internal/sandbox"
	"github.com/IvanBern/ivai-os/internal/tools"
	"github.com/joho/godotenv"
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

	if err := godotenv.Load("/etc/ivai/.env"); err == nil {
		slog.Info("Configuration loaded successfully from /etc/ivai/.env")
	}

	// 2. Setup Context and Channels
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// This channel is Ivai's "ear". Both CLI and HTTP will send tasks here.
	taskChan := make(chan string, 10)

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

		// Send task to the central channel
		taskChan <- req.Instruction

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "task accepted"})
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP Server listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	// 4. Start the CLI REPL (Background Goroutine)
	go func() {
		// Small delay so the prompt appears after initialization logs
		time.Sleep(100 * time.Millisecond)
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("\nIvai > ")
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				taskChan <- input // Send task to the central channel
			}
			// Wait a moment for logs to print before re-prompting
			time.Sleep(50 * time.Millisecond)
			fmt.Print("Ivai > ")
		}
	}()

	// Initialize the LLM Gateway
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		slog.Warn("DEEPSEEK_API_KEY is not set. LLM execution will fail.")
	}
	gateway := llm.NewGateway(apiKey)

	// Initialize the Persistent Memory Subsystem
	slog.Info("Mounting persistent memory subsystem...")
	dbStore, err := memory.NewStore("/etc/ivai/memory.db")
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
		case task := <-taskChan:
			// Process tasks asynchronously
			go func(t string) {
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
				}

				// 2. Save user prompt to memory
				dbStore.SaveMessage("user", t)
				history, _ := dbStore.GetRecentMessages(10)

				// 3. Construct the payload
				homeDir, _ := os.UserHomeDir()
				systemPrompt := fmt.Sprintf("You are Ivai, an advanced AI Operating System. Your persistent workspace is %s. You have a continuous memory and access to the local filesystem. Use your workspace for projects and data storage.", homeDir)

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
					responseMsg, err := gateway.Chat(context.Background(), payload, availableTools, "deepseek-chat") 
					if err != nil {
						slog.Error("LLM Execution Failed", "error", err)
						fmt.Printf("\n[Ivai Error] %v\nIvai > ", err)
						return
					}
					
					// If there are no tool calls, it's a final text response. We are done!
					if len(responseMsg.ToolCalls) == 0 {
						slog.Info("Task completed", "response_length", len(responseMsg.Content))
						dbStore.SaveMessage("assistant", responseMsg.Content)
						fmt.Printf("\n[Ivai] %s\nIvai > ", responseMsg.Content)
						break // Exit the reasoning loop
					}

					// Otherwise, the LLM wants to execute tools.
					fmt.Printf("\n[Ivai System] Thinking...\n")
					
					// Append the LLM's tool request to history
					payload = append(payload, responseMsg)

					// Execute all requested tools
					for _, toolCall := range responseMsg.ToolCalls {
						slog.Info("Executing tool", "name", toolCall.Function.Name)
						fmt.Printf("[Ivai Tool] Running: %s\n", toolCall.Function.Name)
						
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
								output, err := wasmEngine.Execute(context.Background(), wasmBytes, args.Payload, args.TimeoutMs)
								if err != nil { toolResult = fmt.Sprintf("Sandbox error: %v", err) } else { toolResult = output }
							}
						}

						// Append the tool result to the payload so the LLM can read it in the next loop iteration
						payload = append(payload, llm.Message{
							Role:       "tool",
							Content:    toolResult,
							ToolCallID: toolCall.ID,
						})
					}
					// The loop continues, sending the updated payload back to DeepSeek...
				}

			}(task)

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

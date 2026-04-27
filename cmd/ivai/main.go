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
	_ = wasmEngine // Placeholder for future use when we implement plugin execution
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
				}

				// 2. Save user prompt to memory
				dbStore.SaveMessage("user", t)
				history, _ := dbStore.GetRecentMessages(10)

				// 3. Construct the payload
				var payload []llm.Message
				payload = append(payload, llm.Message{
					Role:    "system",
					Content: "You are Ivai, an advanced AI Operating System. You have a continuous memory and access to the local filesystem.",
				})
				for _, msg := range history {
					payload = append(payload, llm.Message{Role: msg.Role, Content: msg.Content})
				}

				// 4. Send to DeepSeek (Now passing availableTools!)
				responseMsg, err := gateway.Chat(context.Background(), payload, availableTools, "deepseek-chat")
				if err != nil {
					slog.Error("LLM Execution Failed", "error", err)
					fmt.Printf("\n[Ivai Error] %v\nIvai > ", err)
					return
				}

				// 5. Handle Tool Executions
				if len(responseMsg.ToolCalls) > 0 {
					// The LLM wants to take an action!
					fmt.Printf("\n[Ivai System] Thinking...\n")

					// Append the assistant's tool call request to the message history so the API knows what we are responding to
					payload = append(payload, responseMsg)

					for _, toolCall := range responseMsg.ToolCalls {
						slog.Info("Executing tool", "name", toolCall.Function.Name)
						fmt.Printf("[Ivai Tool] Running: %s\n", toolCall.Function.Name)

						var toolResult string

						// Route the tool call to the correct Go function
						if toolCall.Function.Name == "read_file" {
							var args struct {
								Filepath string `json:"filepath"`
							}
							json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

							content, err := tools.ReadFile(args.Filepath)
							if err != nil {
								toolResult = fmt.Sprintf("Error reading file: %v", err)
							} else {
								toolResult = content
							}
						} else if toolCall.Function.Name == "write_file" {
							var args struct {
								Filepath string `json:"filepath"`
								Content  string `json:"content"`
							}
							json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

							err := tools.WriteFile(args.Filepath, args.Content)
							if err != nil {
								toolResult = fmt.Sprintf("Error writing file: %v", err)
							} else {
								toolResult = "File written successfully."
							}
						} else if toolCall.Function.Name == "execute_command" {
							var args struct {
								Command string `json:"command"`
							}
							json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

							output, err := tools.ExecuteCommand(args.Command)
							if err != nil {
								toolResult = fmt.Sprintf("Command execution failed: %v", err)
							} else {
								toolResult = output
							}
						}

						// Append the result of the tool execution as a "tool" role message
						payload = append(payload, llm.Message{
							Role:       "tool",
							Content:    toolResult,
							ToolCallID: toolCall.ID,
						})
					}

					// 6. Send the tool results back to the LLM to get the final conversational answer
					finalResponseMsg, err := gateway.Chat(context.Background(), payload, availableTools, "deepseek-chat")
					if err != nil {
						slog.Error("LLM Final Response Failed", "error", err)
						return
					}

					// Overwrite our responseMsg with the final answer
					responseMsg = finalResponseMsg
				}

				// 7. Save the final textual response to memory and print it
				dbStore.SaveMessage("assistant", responseMsg.Content)
				fmt.Printf("\n[Ivai] %s\nIvai > ", responseMsg.Content)

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

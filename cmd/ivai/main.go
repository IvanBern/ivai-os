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

	slog.Info("Ivai OS is now running. Awaiting input via CLI or port 8080.")

	// 5. The Main OS Event Loop
	for {
		select {
		case task := <-taskChan:
			slog.Info("New task received", "task", task)
			// TODO: Pass this task to the DeepSeek Gateway for processing!

		case <-ctx.Done():
			slog.Info("Shutting down Ivai OS...")

			// Gracefully shutdown the HTTP server
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Shutdown(shutdownCtx)

			slog.Info("Ivai OS gracefully stopped.")
			return
		}
	}
}

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// Ivai OS Entry Point
func main() {
	// Initialize structured logging using slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Ivai OS starting up...", "version", "0.1.0")

	// Attempt to load the secure configuration file
	if err := godotenv.Load("/etc/ivai/.env"); err != nil {
		slog.Warn("No /etc/ivai/.env file found, falling back to system environment variables")
	} else {
		slog.Info("Configuration loaded successfully from /etc/ivai/.env")
	}

	// Verify the key is loaded (we only log the first few characters for security)
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if len(apiKey) > 5 {
		slog.Info("DeepSeek API Key detected", "key_prefix", apiKey[:5]+"...")
	} else {
		slog.Error("DeepSeek API Key is missing or invalid!")
	}

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Initializing core subsystems...")

	// Main OS Loop
	slog.Info("Ivai OS is now running. Press Ctrl+C to shut down.")

	select {
	case <-ctx.Done():
		slog.Info("Shutting down Ivai OS...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slog.Info("Ivai OS gracefully stopped.", "timeout", shutdownCtx)
	}
}

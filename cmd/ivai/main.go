package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Ivai OS Entry Point
// This file initializes the core components and handles the main execution loop.
func main() {
	// Initialize structured logging using slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Ivai OS starting up...", "version", "0.1.0")

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize components (Placeholders for now)
	slog.Info("Initializing core subsystems...")

	// Main OS Loop
	slog.Info("Ivai OS is now running. Press Ctrl+C to shut down.")
	
	// Simulate background work or wait for shutdown
	select {
	case <-ctx.Done():
		slog.Info("Shutting down Ivai OS...")
		
		// Create a timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		slog.Info("Ivai OS gracefully stopped.", "timeout", shutdownCtx)
	}
}

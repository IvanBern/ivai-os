# Ivai OS

I have scaffolded the 'Ivai' OS project structure with robust Go boilerplate.

## Project Overview

- **`cmd/ivai/main.go`**: Entry point using `slog` for structured logging and `signal.NotifyContext` for graceful shutdowns.
- **`internal/llm/gateway.go`**: LLM Gateway interface and client for DeepSeek, including exponential backoff retry logic.
- **`internal/sandbox/wazero.go`**: Secure WASM runtime using `wazero` with strict context timeouts for plugin execution.
- **`internal/memory/db.go`**: In-memory SQLite store using the pure-Go `modernc.org/sqlite` driver, with an automated bootstrap for the tasks table.
- **`internal/telemetry/otel.go`**: OpenTelemetry tracing initialization boilerplate.
- **`Makefile`**: Standardized commands for build, run, and tidy.

## Getting Started

All dependencies have been resolved via `go mod tidy`. You can now start the OS by running:

```bash
make run
```

## Transfer and Run on OrbStack

If you generated this on your macOS file system, the beauty of Go is that you can just compile it for your Debian VM directly from your Mac. We have fully automated this pipeline in the `Makefile`.

### Step 1: Create a Dedicated Service User (One-Time Setup)

Inside your OrbStack VM, create a restricted system user to run the OS securely:

```bash
sudo adduser --system --group ivai
```

### Step 2: Deploy and Run

From your Mac terminal (in the project directory), run the "one-click" deployment command:

```bash
make dev
```

Here is what happens automatically:
1. Your Mac cross-compiles the new ARM64 binary (`ivai-os-linux`).
2. It pushes the binary to OrbStack via SCP.
3. It moves the binary to `/usr/local/bin/`, sets the `ivai` user ownership, and makes it executable.
4. It instantly starts the OS and streams the JSON logs right back to your Mac terminal.

You should see logs indicating a successful startup:

```json
{"time":"2026-04-27T21:59:42.49773692+04:00","level":"INFO","msg":"Ivai OS starting up...","version":"0.1.0"}
{"time":"2026-04-27T21:59:42.498021755+04:00","level":"INFO","msg":"Initializing core subsystems..."}
{"time":"2026-04-27T21:59:42.498032672+04:00","level":"INFO","msg":"Ivai OS is now running. Press Ctrl+C to shut down."}
```
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

If you generated this on your macOS file system, the beauty of Go is that you can just compile it for your Debian VM directly from your Mac.

### Step 1: Build the Linux Binary

To build the Linux binary right there on your Mac:

```bash
GOOS=linux GOARCH=arm64 go build -o ivai-os-linux cmd/ivai/main.go
```

### Step 2: Create a Dedicated Service User

Inside your OrbStack VM, create a restricted system user to run the OS securely:

```bash
sudo adduser --system --group ivai
```

### Step 3: Install the Binary

Move the compiled `ivai-os-linux` binary into your VM's system binaries path and set the correct ownership and permissions:

```bash
sudo mv ivai-os-linux /usr/local/bin/ivai-os
sudo chown ivai:ivai /usr/local/bin/ivai-os
sudo chmod +x /usr/local/bin/ivai-os
```

### Step 4: Run the Core Engine

You can now start the AI Operating System securely under its restricted user account.

```bash
sudo -u ivai /usr/local/bin/ivai-os
```

Because it was cross-compiled with `GOOS=linux` and `GOARCH=arm64`, and uses pure-Go libraries for its database and Wasm sandbox, it will start up instantly with zero external dependencies, completely isolated from your Mac host. You should see logs indicating a successful startup:

```json
{"time":"2026-04-27T21:59:42.49773692+04:00","level":"INFO","msg":"Ivai OS starting up...","version":"0.1.0"}
{"time":"2026-04-27T21:59:42.498021755+04:00","level":"INFO","msg":"Initializing core subsystems..."}
{"time":"2026-04-27T21:59:42.498032672+04:00","level":"INFO","msg":"Ivai OS is now running. Press Ctrl+C to shut down."}
```
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

### Step 2: Configure Environment Variables

Create a secure directory for the OS configuration and add your `DEEPSEEK_API_KEY`:

```bash
sudo mkdir -p /etc/ivai
echo 'DEEPSEEK_API_KEY="your-api-key-here"' | sudo tee /etc/ivai/.env
sudo chown -R ivai:ivai /etc/ivai
sudo chmod 600 /etc/ivai/.env
```

### Step 3: Deploy and Run

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

## Interacting with Ivai OS

Ivai OS supports a dual-interface architecture, running concurrently:
1. **Interactive CLI**: Type commands directly into the terminal prompt.
2. **HTTP API**: Send JSON payloads to a background listener on port 8080.

### Using the CLI

When running `make dev`, after the initial logs, you will see an interactive prompt:

```plaintext
Ivai > Hello, who are you?
```

Type your instruction and press Enter. The task will be sent to the central processing channel.

### Using the HTTP API

You can also send tasks to the OS via HTTP POST requests from another terminal. For example, if you SSH into the VM:

```bash
ssh ivai-os-linux@orb
curl -X POST http://localhost:8080/api/task \
  -H "Content-Type: application/json" \
  -d '{"instruction": "Write a python script to calculate fibonacci"}'
```

The API will return `{"status": "task accepted"}` and the core engine will log the new task.
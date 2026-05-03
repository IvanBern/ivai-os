# Ivai OS: The Autonomous AI Operating System

Ivai OS is a fully functional, autonomous AI Operating System built in Go. It combines LLM reasoning with low-level system access, persistent memory, and a secure WebAssembly sandbox.

## 🚀 Final Achievements

Ivai OS has successfully moved from a scaffold to a complete, production-ready autonomous agent system. Key highlights include:

- **Autonomous Reasoning Loop**: Implemented a multi-step "Chain of Thought" execution loop that allows the AI to autonomously sequence multiple tools (e.g., `write_file` -> `compile` -> `execute_wasm`) to solve complex tasks.
- **Secure Wasm Micro-VM**: Integrated the **Wazero** runtime to provide a high-performance, secure, and isolated execution environment for untrusted code with WASI support.
- **Persistent Continuous Memory**: A SQLite-backed memory subsystem that allows the AI to retain context across sessions, enabling long-term task management and personality consistency.
- **Dual-Interface Control**: Seamlessly handles tasks via a high-performance **CLI REPL** and a concurrent **HTTP JSON API**.
- **Agentic Toolset**: Equipped with a standardized API for File System manipulation, Shell execution, HTTP networking, and WebAssembly instantiation.

## 🏗 Architecture Overview

Ivai OS follows a modular kernel-and-subsystem design. The `cmd/ivai/` source is split across five files by responsibility:

| File | Responsibility |
|---|---|
| `main.go` | Minimal startup, HTTP handlers, CLI REPL, event loop |
| `process.go` | Task processing, reasoning loop, RAG payload builder |
| `tools.go` | Tool definitions (17 tools), dispatcher, result formatting |
| `github_tools.go` | GitHub PR/issues/wiki and CodeScene executors |
| `swarm.go` | VM worker lifecycle (clone/deploy/dispatch/kill) |

For a deep dive into the internal specifications, see **[ARCHITECTURE.md](ARCHITECTURE.md)**.

- **The Kernel**: A Go-based event loop orchestrating reasoning and tool routing.
- **Cognitive Engine**: Multi-model (DeepSeek-V4-Pro, Claude 3.5 Sonnet, Gemini 2.5 Pro & Deep Research) powered tool calling with autonomous multi-step reasoning. Swap models on the fly using `@claude`, `@gemini`, `@deepseek`, or `@research`.
- **Execution Sandbox**: Secure WebAssembly (Wazero) micro-VM for untrusted code.
- **Memory Subsystem**: Persistent SQLite storage for long-term context.
- **Mac Client**: Remote control via the `ivaictl` CLI.

## ⚙️ Getting Started

### 1. VM Setup (OrbStack / Debian)
Inside your target VM, create a restricted system user and configuration:

```bash
sudo adduser --system --group ivai
sudo mkdir -p /etc/ivai
echo 'DEEPSEEK_API_KEY="your-api-key-here"' | sudo tee /etc/ivai/.env
# Optionally add Anthropic and Gemini keys:
# echo 'ANTHROPIC_API_KEY="your-key"' | sudo tee -a /etc/ivai/.env
# echo 'GEMINI_API_KEY="your-key"' | sudo tee -a /etc/ivai/.env
sudo chown -R ivai:ivai /etc/ivai
sudo chmod 600 /etc/ivai/.env
```

### 2. Deployment
From your host machine (macOS/Linux):

```bash
# Full deploy to VM
make dev

# Or run locally on macOS
make run-local
```

## 🧠 Core Capabilities Demo

Ivai OS can autonomously handle complex, multi-step engineering tasks. For example:

**User Prompt:**
> "Write a tiny Go program that takes a name via stdin and prints 'Hello [name] from WebAssembly!'. Save it to /tmp/hello.go, compile it to /tmp/hello.wasm, and execute it in your sandbox with the payload 'Ivan'."

**Ivai OS Execution Trace:**
1.  **Thinking**: Decides it needs to write the source code.
2.  **Tool (`write_file`)**: Writes `hello.go` to `/tmp`.
3.  **Thinking**: Decides it needs to compile the code.
4.  **Tool (`execute_command`)**: Runs `GOOS=wasip1 GOARCH=wasm go build`.
5.  **Thinking**: Decides to run the resulting binary.
6.  **Tool (`execute_wasm`)**: Executes the `.wasm` file in the Wazero sandbox.
7.  **Final Response**: "Hello Ivan from WebAssembly!"

## 🛠 Interaction

### CLI REPL
The interactive prompt is available immediately after running `make dev`.
```bash
Ivai > What files are in /tmp?
```

### HTTP API
Send tasks programmatically to the background listener:

**Blocking (wait for full response):**
```bash
curl -X POST http://localhost:8080/api/task \
  -H "Content-Type: application/json" \
  -d '{"instruction": "List all active processes"}'
```

**Streaming SSE (real-time progress):**
```bash
curl -N -X POST http://localhost:8080/api/task/stream \
  -H "Content-Type: application/json" \
  -d '{"instruction": "@claude find all Go files and count lines"}'
```

### Mac Client (`ivaictl`)

```bash
# Blocking mode
./ivaictl "@deepseek what is the current time?"

# Streaming mode — see live progress as Ivai thinks and calls tools
./ivaictl --stream "@claude refactor the auth module"

# Pipe instructions
echo "check disk usage" | ./ivaictl --stream
```

**Streaming output example:**
```
[model] claude-3-5-sonnet-20241022
[instruction] find all Go files and count lines
[thinking] I need to find Go files first, then count lines
[tool] execute_command → find . -name '*.go' -type f
[tool result] execute_command → ./main.go\n./gateway.go\n./db.go
[thinking] Now I'll run wc -l on each file
[tool] execute_command → wc -l ./main.go ./gateway.go ./db.go
[tool result] execute_command → 463 ./main.go / 466 ./gateway.go / 97 ./db.go / 1026 total

[complete] Found 3 Go files with 1,026 total lines of code.
```

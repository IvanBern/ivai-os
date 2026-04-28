# Ivai OS v0.1.0: Technical Architecture Specification

Ivai OS has transitioned from a simple Go script to a daemonized, agentic operating system kernel. This document specifies the current internal architecture and subsystems.

## 1. Core Architecture (The Kernel)

- **Runtime**: Pure Go (linux/arm64) binary running inside a minimal Debian OrbStack VM.
- **Daemonization**: Managed by systemd (`ivai.service`). Runs automatically on boot, self-heals on crashes, and pipes structured JSON logs to the system journal via `journald`.
- **Privilege Isolation**: Executes under a dedicated, restricted Linux system user (`ivai`) with zero password login or sudo privileges.
- **Configuration**: Secrets loaded via `godotenv` from a locked-down file (`/etc/ivai/.env` with 600 permissions).

## 2. Cognitive Engine (The Brain)

- **LLM Gateway**: Multi-model support for DeepSeek-V4-Pro and Anthropic (Claude 3.5) utilizing a unified Tool Calling standard. Supports seamless model swapping via instruction-level hints (e.g., `@claude`).
- **Reasoning Loop**: A recursive Go `for` loop that intercepts tool requests, executes local Go functions, feeds the results back to the LLM (DeepSeek or Anthropic), and continues chaining actions autonomously until a final textual answer is reached.

## 3. Memory Subsystem (The Context)

- **Storage Engine**: Embedded pure-Go SQLite (`modernc.org/sqlite`), eliminating CGO dependencies.
- **Data Structure**: A continuous `messages` table storing the conversation and tool-execution history at `/etc/ivai/memory.db`.
- **Context Window**: Automatically fetches the last 10 interactions to maintain a rolling short-term memory across daemon restarts.

## 4. Execution Subsystems (The Hands)

- **File I/O (`read_file`, `write_file`)**: Grants the agent ability to read host files and write raw code/configurations to disk.
- **System Shell (`execute_command`)**: Utilizes `os/exec` wrapped in a `bash -c` subshell. Allows the agent to run native Linux utilities, compile code, and explore its host environment.
- **Wasm Sandbox (`execute_wasm`)**: An embedded WebAssembly micro-VM using **Wazero**. Enables zero-trust execution of compiled `.wasm` binaries (WASI preview 1 standard) with strict millisecond timeouts and sandboxed standard I/O.
- **Network Tooling (`http_request`)**: Performs HTTP requests (GET, POST, etc.) using Go's `net/http` client, allowing interaction with external APIs.
- **Version Control (`git`)**: Pre-installed and configured `git` utility available via `execute_command` for autonomous repository management.

## 5. Interfaces (The Ears)

- **HTTP Server**: A background goroutine listening on port 8080, accepting JSON payloads at `/api/task`.
- **Mac Client (`ivaictl`)**: A native macOS Go binary that communicates with the kernel via the HTTP API, supporting command-line arguments and piped input.
- **CLI REPL**: Standard input scanner for interactive terminal debugging (integrated into the kernel event loop).

---

## 🛡️ Security Posture

- **Non-Root Execution**: The kernel runs as a restricted user.
- **Wasm Isolation**: Untrusted code is executed in a memory-safe, capability-based sandbox.
- **Graceful Shutdown**: Handles `SIGTERM` and `SIGINT` to close the database and stop the HTTP server cleanly.

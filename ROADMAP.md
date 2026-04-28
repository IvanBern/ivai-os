# The Ivai OS Roadmap

## ✅ Phase 1: Core Kernel & Foundation
- [x] Scaffold project structure with robust Go boilerplate.
- [x] Implement LLM Gateway for DeepSeek integration.
- [x] Setup structured logging and graceful shutdown signals.

## ✅ Phase 2: System Capabilities
- [x] **The Memory Subsystem**: Embedded SQLite database for persistent conversation history and task context.
- [x] **The Execution Sandbox**: Secure Wazero WebAssembly runtime for isolated plugin execution.
- [x] **The Agentic Tool Protocol**: Multi-step reasoning loop with support for `read_file`, `write_file`, `execute_command`, and `execute_wasm`.

## 🚀 Phase 3: Residency & Advanced Interaction
- [x] **The Daemon (Systemd)**: Implement a systemd service unit to make Ivai OS a permanent resident of the Debian VM. It will start automatically on boot and restart on failure.
- [x] **The Mac Client**: Develop a native macOS CLI (or web interface) that communicates with Ivai via the HTTP port 8080. This removes the need for SSH to assign tasks.
- [x] **Persistent Workspace**: Define a standard "Ivai Home" directory for the agent to manage its own projects and persistent data.

## 🛠 Phase 4: Advanced Tooling & Autonomy
- [ ] **Git Integration**: Teach Ivai to use `git` so it can clone repositories, manage branches, and commit code autonomously.
- [ ] **GitHub API Access**: Allow Ivai to read PRs, issues, and repository metadata to analyze and contribute to external codebases.
- [ ] **Network Tooling**: Add tools for safe HTTP requests and network diagnostics.
- [ ] **Sub-Agent Spawning**: Enable Ivai to spawn child processes or sandboxed "sub-agents" for parallel task execution.

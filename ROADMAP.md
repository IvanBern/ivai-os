# The Ivai OS Roadmap

## ✅ Phase 1: Core Kernel & Foundation
- [x] Scaffold project structure with robust Go boilerplate.
- [x] Implement LLM Gateway for DeepSeek integration.
- [x] Setup structured logging and graceful shutdown signals.

## ✅ Phase 2: System Capabilities
- [x] **The Memory Subsystem**: Embedded SQLite database for persistent conversation history.
- [x] **The Execution Sandbox**: Secure Wazero WebAssembly runtime for isolated execution.
- [x] **The Agentic Tool Protocol**: Multi-step reasoning loop with `read_file`, `write_file`, `execute_command`, and `execute_wasm`.

## ✅ Phase 3: Residency & Mac Integration
- [x] **The Daemon (Systemd)**: Background residency with auto-restart and journaled logging.
- [x] **The Mac Client (ivaictl)**: Native macOS CLI for remote task assignment via HTTP.
- [x] **Persistent Workspace**: Standardized `/home/ivai` directory for agent projects.

## ✅ Phase 4: Advanced Tooling
- [x] **Git Integration**: Version control for autonomous code management.
- [x] **Network Tooling**: Native `http_request` tool for external API interaction.

---

## 🛡️ Phase 5: Production Hardening (Next)

### 1. API Security (The HTTP Layer)
- [ ] **Authentication Middleware**: Implement API Key validation for the HTTP router.
- [ ] **mTLS**: Enforce mutual TLS for cryptographically secure Mac-to-VM communication.
- [ ] **Interactive Response**: Update the Mac Client and Kernel to "wait" for task completion and stream the agent's response back to the terminal in real-time.

### 2. Advanced Sandboxing (The OS Layer)
- [ ] **Directory Whitelisting**: Restrict File I/O tools to specific workspace boundaries.
- [ ] **cgroups (Control Groups)**: Enforce strict CPU and RAM limits via systemd to prevent resource exhaustion.
- [ ] **Egress Filtering**: Use `iptables` to block unauthorized outbound network requests.

## 🧠 Phase 6: Long-Term Intelligence
- [ ] **Vector Database**: Integrate an embedded vector store (e.g., Chroma/Milvus) for semantic memory.
- [ ] **RAG Pipeline**: Implement Retrieval-Augmented Generation to search past solutions before querying the LLM.

## 🤖 Phase 7: Self-Evolution & Parallelism
- [ ] **GitHub API Access**: Provide Ivai with a token to interact with repositories (clone, commit, open Pull Requests).
- [ ] **Sub-Agent Spawning**: Enable Ivai to spawn child processes or sandboxed "sub-agents" for parallel task execution.
- [ ] **Self-Modification Loop**: Enable Ivai to branch its own source code, implement features, and submit PRs for human review.

## 📊 Phase 8: Observability
- [ ] **OpenTelemetry (OTel)**: Instrument the daemon with spans for visual waterfall charts of reasoning and tool execution.

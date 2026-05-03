# The Ivai OS Roadmap

## ✅ Phase 1: Core Kernel & Foundation
- [x] Scaffold project structure with robust Go boilerplate.
- [x] Implement LLM Gateway for DeepSeek, Anthropic, and Gemini integration.
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
- [ ] **Interactive Response Streaming**: Implement Server-Sent Events (SSE) or WebSockets in the Go server to stream `[Ivai System] Thinking...` logs in real-time to the Mac Client without HTTP timeouts.

### 2. Advanced Sandboxing (The OS Layer)
- [ ] **Directory Whitelisting**: Restrict File I/O tools to specific workspace boundaries via strict filepath cleaning.
- [ ] **cgroups (Control Groups)**: Enforce strict CPU and RAM limits via systemd to prevent runaway agent scripts.
- [ ] **Egress Filtering**: Use `iptables` to block unauthorized outbound network requests.

## 🧠 Phase 6: Long-Term Intelligence
- [ ] **Embedded Vector Database**: Integrate a pure-Go vector search extension (`sqlite-vec` or `chromem-go`) to keep the OS a single, dependency-free binary.
- [ ] **RAG Pipeline**: Implement Retrieval-Augmented Generation to automatically search past solved problems, code snippets, or architecture docs before querying the LLM.
- [ ] **Dynamic Model Discovery**: Implement tools for Ivai to query provider APIs (`/v1/models`) to discover available brains and cache them in its persistent memory, allowing the system to self-update its routing logic.

## 🤖 Phase 7: Self-Evolution & Parallelism
- [ ] **Model Context Protocol (MCP)**: Establish a standardized Tool Registry so spawned sub-agents can dynamically share and discover tools without duplicating code.
- [ ] **GitHub API Access**: Provide Ivai with a token to interact with repositories (clone, commit, open Pull Requests).
- [ ] **Sub-Agent Spawning**: Enable Ivai to spawn child processes or sandboxed "sub-agents" for parallel task execution.
- [ ] **Self-Modification Loop**: Enable Ivai to branch its own source code, implement features, write unit tests, compile, and submit PRs for human review.

## 📊 Phase 8: Observability
- [x] **OpenTelemetry Tracer**: Initialized and wired into the reasoning loop and tool execution (foundation for exporters).
- [ ] **OTel Exporters**: Add Jaeger, OTLP, or stdout exporters for visual waterfall charts.
- [ ] **Span Enrichment**: Add attributes for model name, tool arguments, and timing data.

---

## 🚀 The Next Leap: Proactive & Distributed Systems

## ⏱️ Phase 9: Proactive Autonomy (Event-Driven OS)
- [ ] **Agentic Cron Jobs**: Provide a `register_cron` tool so Ivai can schedule its own background tasks (e.g., daily log reviews, morning summaries).
- [ ] **Event Watchers (`inotify`)**: Allow Ivai to watch specific directories (e.g., `/home/ivai/data`) to automatically wake up, analyze incoming files, and write reports unprompted.
- [ ] **Webhook Receivers**: Expose an endpoint for external services (GitHub, Stripe, PagerDuty) to send webhooks directly into Ivai's event loop for automated triage.

## 🚦 Phase 10: Human-in-the-Loop (HITL) Governance
- [ ] **Asynchronous Interrupts**: Create a `request_human_approval` tool. For destructive or sensitive actions, Ivai pauses its execution state and sends a push notification/CLI prompt to the Mac, waiting for a cryptographic "YES".
- [ ] **Token & Budget Quotas**: Implement daily API spend and token limits in the kernel. If Ivai gets stuck in a loop, it automatically suspends itself to prevent budget overrun.

## 🌐 Phase 11: The Distributed Swarm (Multi-Node)
- [ ] **gRPC Fleet Communication**: Connect multiple Ivai OS instances (Raspberry Pi, AWS EC2, Mac) via a secure gRPC network.
- [ ] **Task Delegation**: Allow a local Ivai node to autonomously RPC sub-tasks to remote nodes (e.g., executing AWS-specific scripts) and aggregate the results.

## 👁️ Phase 12: Multimodal Sensory Input
- [ ] **Computer Use / Vision**: Integrate tools like Playwright or Puppeteer alongside a multimodal LLM endpoint. Enable Ivai to spin up headless browsers, take screenshots, "look" at the DOM, and interact with visual web apps lacking APIs.

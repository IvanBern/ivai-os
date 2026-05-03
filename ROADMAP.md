# The Ivai OS Roadmap

> **How to read this:** Each phase table links to a detailed requirements spec in `docs/specs/`. Cross-cutting concerns (memory model, agent architecture) live in `docs/designs/`. The "as-built" reality lives in `ARCHITECTURE.md`. Aspirational long-term goals live in `docs/VISION.md`.
>
> **Status legend:** 🟢 Done &nbsp; 🟡 In Progress &nbsp; 🔴 Not Started &nbsp; ⚫ Blocked

---

## ✅ Phase 1: Core Kernel & Foundation

**Spec:** Built-in (foundational — no separate spec)
**Depends on:** —
**Blocks:** Phase 2, Phase 3

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 1.1 | Scaffold project structure with Go boilerplate | 🟢 | `go build ./...` |
| 1.2 | LLM Gateway (DeepSeek, Anthropic, Gemini) | 🟢 | `TestLLMGateway` |
| 1.3 | Structured logging + graceful shutdown | 🟢 | `TestGracefulShutdown` |

**Architecture impacted:** §1 Core, §2 Cognitive Engine

---

## ✅ Phase 2: System Capabilities

**Spec:** Built-in
**Depends on:** Phase 1
**Blocks:** Phase 3, Phase 4, Phase 6, Phase 8

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 2.1 | Memory Subsystem: Embedded SQLite | 🟢 | `TestMemoryDB` |
| 2.2 | Execution Sandbox: Wazero WebAssembly | 🟢 | `TestWasmSandbox` |
| 2.3 | Agentic Tool Protocol: multi-step reasoning loop | 🟢 | `TestReasoningLoop` |

**Architecture impacted:** §2 Cognitive Engine, §3 Memory, §4 Execution

---

## ✅ Phase 3: Residency & Mac Integration

**Spec:** Built-in
**Depends on:** Phase 1, Phase 2
**Blocks:** Phase 5

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 3.1 | Daemon (systemd) with auto-restart | 🟢 | `systemctl status ivai` |
| 3.2 | Mac Client (`ivaictl`) via HTTP API | 🟢 | `ivaictl "hello"` |
| 3.3 | Persistent Workspace (`/home/ivai`) | 🟢 | `ls /home/ivai` |

**Architecture impacted:** §1 Core, §5 Interfaces

---

## ✅ Phase 4: Advanced Tooling

**Spec:** Built-in
**Depends on:** Phase 2
**Blocks:** Phase 5, Phase 7

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 4.1 | Git Integration | 🟢 | `TestGitTool` |
| 4.2 | Network Tooling (`http_request`) | 🟢 | `TestHTTPRequest` |

**Architecture impacted:** §4 Execution

---

## 🛡️ Phase 5: Production Hardening (In Progress — 17%)

**Spec:** [docs/specs/phase-5-security.md](docs/specs/phase-5-security.md)
**Depends on:** Phase 3 ✅, Phase 4 ✅
**Blocks:** Phase 9 (auth for webhooks), Phase 10 (secure HITL channel), Phase 11 (mTLS foundation)

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 5.1 | API Key Authentication Middleware | 🔴 | `TestAuthMiddleware` |
| 5.2 | mTLS Enforcement | 🔴 | `TestMTLSHandshake` |
| 5.3 | SSE Streaming | 🟢 (80ccba7) | `TestSSEStream` |
| 5.4 | Directory Whitelisting | 🔴 | `TestPathTraversalBlocked` |
| 5.5 | cgroups Resource Limits | 🔴 | `TestMemoryCapEnforced` |
| 5.6 | Egress Filtering | 🔴 | `TestOutboundBlocked` |
| 5.7 | Web Admin Dashboard (status, SSE console, memory, tools) | 🟢 (bc5ab9d) | `TestDashboardEndpoint` |
| 5.8 | Self-Knowledge System Prompt (`//go:embed SYSTEM_PROMPT.md`) | 🟢 (bc5ab9d) | `TestSystemPromptEmbed` |

**Architecture impacted:** §1 Core, §4 Execution, §5 Interfaces

---

## 🧠 Phase 6: Long-Term Intelligence

**Spec:** (to create: `docs/specs/phase-6-intelligence.md`)
**Depends on:** Phase 2 ✅
**Blocks:** Phase 7 (RAG needed for self-evolution), Phase 9, Phase 12

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 6.1 | Task Result Tracking (`task_results` table: success, error, duration, tokens) | 🔴 | `TestTaskResults` |
| 6.2 | Embedded Vector Database (`sqlite-vec` or `chromem-go`) | 🔴 | `TestVectorSearch` |
| 6.3 | RAG Pipeline (auto-search past context before LLM) | 🔴 | `TestRAGInjection` |
| 6.4 | Dynamic Model Discovery (`/v1/models` cache) | 🔴 | `TestModelDiscovery` |

**Design docs referenced:** [MEMORY-MODEL.md](docs/designs/MEMORY-MODEL.md) (data model for Episodic + Semantic tables)
**Architecture impacted:** §3 Memory, §2 Cognitive Engine
**VISION link:** [Horizon A — Cognitive Synthesis](docs/VISION.md#horizon-a-cognitive-synthesis-beyond-information-retrieval)

---

## 🤖 Phase 7: Self-Evolution & Parallelism

**Spec:** (to create: `docs/specs/phase-7-self-evolution.md`)
**Depends on:** Phase 4 ✅, Phase 5.8 ✅ (self-knowledge prompt in place), Phase 6.1 (task tracking for feedback loop)
**Blocks:** Phase 10, Phase 11, Phase 12

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 7.0 | Self-Knowledge Foundation (anatomy, sandbox PR workflow in SYSTEM_PROMPT.md) | 🟢 (bc5ab9d) | `TestSelfKnowledge` |
| 7.1 | MCP Tool Registry (cross-agent tool discovery) | 🔴 | `TestToolRegistry` |
| 7.2 | GitHub API Access (clone, commit, PR) | 🟡 | `TestGitHubAPI` |
| 7.3 | Sub-Agent Spawning (child processes for parallel tasks) | 🔴 | `TestSubAgentSpawn` |
| 7.4 | Self-Modification Loop (branch → implement → test → PR) | 🔴 | `TestSelfModification` |

**Design docs referenced:** [AGENTIC-ARCHITECTURE.md](docs/designs/AGENTIC-ARCHITECTURE.md) (sub-agent provisioning, MCP builder skill, memory integration)
**Architecture impacted:** §2 Cognitive Engine, §4 Execution
**VISION link:** [Horizon B — Proactive Skill Acquisition](docs/VISION.md#horizon-b-proactive-skill-acquisition)

---

## 📊 Phase 8: Observability

**Spec:** (to create: `docs/specs/phase-8-observability.md`)
**Depends on:** Phase 2 ✅
**Blocks:** Phase 13 (meta-cognition, VISION Horizon I)

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 8.1 | OpenTelemetry Tracer (initialized) | 🟢 | `TestOTelInit` |
| 8.2 | OTel Exporters (Jaeger, OTLP, stdout) | 🔴 | `TestJaegerExport` |
| 8.3 | Span Enrichment (model, tool args, timing) | 🔴 | `TestSpanAttributes` |
| 8.4 | Makefile `stop` target (kill local Ivai process) | 🟢 (3accdf6) | `make stop` |

**Architecture impacted:** §6 Observability

---

## ⏱️ Phase 9: Proactive Autonomy (Event-Driven OS)

**Spec:** (to create: `docs/specs/phase-9-autonomy.md`)
**Depends on:** Phase 5, Phase 6
**Blocks:** Phase 10

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 9.1 | Agentic Cron Jobs (`register_cron` tool) | 🔴 | `TestCronRegistration` |
| 9.2 | Event Watchers (`inotify` directory monitoring) | 🔴 | `TestInotifyWake` |
| 9.3 | Webhook Receivers (GitHub, Stripe, PagerDuty) | 🔴 | `TestWebhookEndpoint` |

**Design docs referenced:** [MEMORY-MODEL.md](docs/designs/MEMORY-MODEL.md) (Episodic → Semantic/Procedural consolidation cron)
**Architecture impacted:** §5 Interfaces, §1 Core
**VISION link:** [Horizon E — Abstract Problem Solving](docs/VISION.md#horizon-e-abstract-problem-solving)

---

## 🚦 Phase 10: Human-in-the-Loop (HITL) Governance

**Spec:** (to create: `docs/specs/phase-10-hitl.md`)
**Depends on:** Phase 5, Phase 7
**Blocks:** Phase 11

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 10.1 | Asynchronous Interrupts (`request_human_approval`) | 🔴 | `TestApprovalFlow` |
| 10.2 | Token & Budget Quotas (daily spend/iteration caps) | 🔴 | `TestBudgetEnforcement` |

**Design docs referenced:** [MEMORY-MODEL.md](docs/designs/MEMORY-MODEL.md) (Procedural memory for approval workflows)
**Architecture impacted:** §2 Cognitive Engine, §5 Interfaces
**VISION link:** [Horizon G — Social Delegation](docs/VISION.md#horizon-g-social-delegation-external-representation)

---

## 🌐 Phase 11: The Distributed Swarm (Multi-Node)

**Spec:** (to create: `docs/specs/phase-11-swarm.md`)
**Depends on:** Phase 5, Phase 7, Phase 10
**Blocks:** —

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 11.1 | gRPC Fleet Communication (secure node mesh) | 🔴 | `TestGRPCMesh` |
| 11.2 | Task Delegation (cross-node RPC + result aggregation) | 🔴 | `TestCrossNodeDelegation` |

**Architecture impacted:** §5 Interfaces

---

## 👁️ Phase 12: Multimodal Sensory Input

**Spec:** (to create: `docs/specs/phase-12-multimodal.md`)
**Depends on:** Phase 6, Phase 7
**Blocks:** —

| # | Item | Status | Verifier |
|---|------|--------|----------|
| 12.1 | Computer Use / Vision (headless browser + screenshot + DOM) | 🔴 | `TestBrowserVision` |

**Architecture impacted:** §4 Execution
**VISION link:** [Horizon D — Multi-Modal Creative Collaboration](docs/VISION.md#horizon-d-multi-modal-creative-collaboration)

---

## 📋 Spec Backlog

These phases need detailed requirements specs created before implementation begins:

- [ ] `docs/specs/phase-6-intelligence.md`
- [ ] `docs/specs/phase-7-self-evolution.md`
- [ ] `docs/specs/phase-8-observability.md`
- [ ] `docs/specs/phase-9-autonomy.md`
- [ ] `docs/specs/phase-10-hitl.md`
- [ ] `docs/specs/phase-11-swarm.md`
- [ ] `docs/specs/phase-12-multimodal.md`

---

## 📐 Design Docs

Cross-cutting architectural patterns that span multiple phases:

| Document | Covers | Referenced By |
|----------|--------|---------------|
| [MEMORY-MODEL.md](docs/designs/MEMORY-MODEL.md) | Memory taxonomy, lifecycle, TTL rules | Phases 6, 9, 10 |
| [AGENTIC-ARCHITECTURE.md](docs/designs/AGENTIC-ARCHITECTURE.md) | Sub-agent spawning, MCP builders, delegation | Phase 7 |

---

## 🔭 Vision

Long-term aspirational horizons live in [docs/VISION.md](docs/VISION.md). They are not committed to the roadmap but inform architectural decisions today.

---

## 🔧 Phase 13: SDLC Excellence (In Progress)

**Spec:** (to create: `docs/specs/phase-13-sdlc.md`)
**Depends on:** Phase 7 ✅ (self-evolution foundation), Phase 8 ✅ (observability)
**Goal:** Full automated development lifecycle — Ivai manages its own work from issue to deploy.

| # | Item | Status | Design | Verifier |
|---|------|--------|--------|----------|
| 13.1 | GitHub Issues API Tool (`create_issue`, `list_issues`, `label_issue`) | 🔴 | Ivai creates structured issues from roadmap items, self-assigns, labels by phase | `TestIssueCreation` |
| 13.2 | Issue Templates (`.github/ISSUE_TEMPLATE/`) | 🔴 | bug.md, feature.md with acceptance criteria, phase linkage | Template renders on new issue |
| 13.3 | Dashboard CI Badge | 🔴 | Web UI shows current pipeline status (green/red) via GitHub API | `TestCIBadge` |
| 13.4 | ADR Directory (`docs/adr/`) | 🔴 | Architecture Decision Records: why sqlite-vec over chromem-go, why wazero, etc. Template + first 3 ADRs | `docs/adr/0001-*.md` exists |
| 13.5 | Rollback (`make rollback`) | 🔴 | Keeps previous binary as `/usr/local/bin/ivai-os.prev`, rollback restores it + restarts | `make rollback` restores prev version |
| 13.6 | Feature Flags | 🔴 | Env-var based: `IVAI_FEATURE_*=true/false`. New capabilities ship disabled, enabled after staging validation | `TestFeatureFlag` |
| 13.7 | GitHub Wiki API Tool | 🔴 | Ivai writes capability docs to wiki after each merged PR | `TestWikiUpdate` |
| 13.8 | Signed Commits (`git commit -S`) | 🔴 | GPG signing configured for Ivai's GitHub identity, verified by branch protection | `TestSignedCommit` |
| 13.9 | Regression Test Suite | 🔴 | Tests for core reasoning loop, all 7 tool dispatches, buildPayload, SSE streaming | `TestRegressionSuite` |
| 13.10 | Mermaid Diagrams in Specs | 🔴 | Architecture diagrams in docs/specs/ using mermaid.js blocks | `TestMermaidRender` |

**Design docs referenced:** This phase (solution designs in section below)
**Architecture impacted:** §2 Cognitive Engine, §5 Interfaces, §7 Web Dashboard
**VISION link:** [Horizon I — Meta-Cognition](docs/VISION.md#horizon-i-meta-cognition--self-debugging)

### Design Solutions

#### 13.1 GitHub Issues API Tool

Ivai gains `create_issue`, `list_issues`, `label_issue` tools via `gh issue` CLI.
```
create_issue(title, body, labels[], assignee?)
list_issues(state, labels, limit)
label_issue(number, labels[])
```
Creates issues from ROADMAP.md items. Labels map to phases: `phase-6`, `phase-13`, etc.
Self-assigns to `ivaiber`.

#### 13.2 Issue Templates

Three templates with YAML frontmatter for GitHub's form builder:
- `bug.md`: Steps to reproduce, expected vs actual, phase
- `feature.md`: Problem, acceptance criteria, phase, design doc ref
- `spec.md`: For spec documents — links to ROADMAP item, design decisions

#### 13.3 Dashboard CI Badge

New `Pipeline` tab in Web UI fetches GitHub Actions status via this repo's API:
```
GET https://api.github.com/repos/IvanBern/ivai-os/actions/runs?branch=main&per_page=1
```
Shows: last run status (✅/❌), timestamp, link to run. Auto-refreshes.

#### 13.4 ADR Directory

Template: `docs/adr/0001-template.md`
First 3 ADRs: sqlite-vec choice, wazero over docker, embedded Go over microservices.

#### 13.5 Rollback

`make rollback`:
1. Stops ivai service
2. `mv /usr/local/bin/ivai-os.prev /usr/local/bin/ivai-os`
3. Starts ivai service
4. Keeps current binary as new `.prev` for next rollback

#### 13.6 Feature Flags

Minimal implementation — env var prefix `IVAI_FEATURE_`:
```go
func featureEnabled(name string) bool {
    return os.Getenv("IVAI_FEATURE_"+strings.ToUpper(name)) == "true"
}
```
First flag: `IVAI_FEATURE_RAG=true` — controls RAG context injection.

#### 13.7 GitHub Wiki API Tool

`update_wiki(page, content)` — writes/updates a wiki page via:
```
gh api repos/IvanBern/ivai-os/pages -f title="$page" -f content="$content"
```
Auto-runs after each merged PR. Documents capabilities, tools, architecture changes.

#### 13.8 Signed Commits

Ivai's VM identity gets a GPG key. Git config: `commit.gpgsign=true`.
Branch protection requires signed commits.

#### 13.9 Regression Test Suite

`cmd/ivai/main_test.go` additions:
- `TestAllToolDispatch` — all 7 tools execute without panic
- `TestReasoningLoopCompletes` — simple task → SSE events → final response
- `TestBuildPayloadRAG` — RAG context injected when embeddings exist
- `TestSSEStreamFormat` — valid event: data: format

#### 13.10 Mermaid Diagrams

Add to `docs/specs/` and `ARCHITECTURE.md`:
- System context diagram
- Data flow: CLI/Web → HTTP → taskChan → reasoning loop → LLM → tools
- State machine: Ivai task lifecycle (queued → thinking → tool_call → tool_result → complete/error)

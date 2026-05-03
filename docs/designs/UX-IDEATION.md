# Ivai UX Ideation Plan

**Status:** Research complete. Ready for roadmap integration.

## Research Summary

### 1. Charmbracelet Bubble Tea — TUI Framework
- **What:** Elm Architecture for Go terminals. Model/Update/View pattern.
- **Ecosystem:** Bubbles (text inputs, viewports, spinners, tables, paginators), Lip Gloss (terminal styling, colors, layouts), Huh? (interactive forms and prompts), Glamour (markdown rendering), Harmonica (animations)
- **Adoption:** 18,000+ apps including Azure, AWS, NVIDIA, CockroachDB
- **Relevance:** Ivai could have a rich TUI admin interface alongside the web dashboard. Terminal-native, works over SSH, no browser needed.

### 2. Warp Terminal + Oz Agent
- **Warp:** Modern terminal with IDE-like editing, AI integration, GPU-accelerated rendering
- **Oz:** Cloud orchestration platform for parallel AI coding agents. Programmable, auditable, steerable.
- **Key Oz features:**
  - Spin up unlimited parallel agents in the cloud
  - Agents are programmable with custom instructions
  - Full audit trail of agent actions
  - Steerable — can interrupt, redirect, approve mid-execution
  - Works in terminal but scales to cloud
- **Relevance:** Ivai's Phase 7.3 (sub-agent spawning) maps directly to Oz's multi-agent orchestration. Oz's audit trail → Ivai's task_results. Oz's steerability → Ivai's HITL (Phase 10).

### 3. Charm Mods (now sunset → Crush)
- **What:** AI for command line, built for pipelines. Could ingest command output, format as markdown/JSON.
- **Features:** Saved conversations, custom roles, MCP server support, streaming, multi-model
- **Relevance:** Already in Crush. Ivai could adopt conversation history browsing from the TUI.

### 4. Other Notable Interfaces
- **gh-dash** (Bubble Tea): GitHub PR/issue dashboard in terminal — exact pattern Ivai could use for its own task management
- **Lazygit:** Git TUI — shows how complex state can be managed in a terminal
- **k9s:** Kubernetes TUI — resource monitoring pattern applicable to Ivai's system monitoring

---

## Ideation Items

### 🖥️ T1: Ivai TUI (`ivai-tui`) — Terminal Admin Interface

**Inspiration:** Bubble Tea + gh-dash + k9s

A Bubble Tea-based TUI that serves as the primary admin interface:

```
┌─────────────────────────────────────────────────────────┐
│ Ivai OS v0.2.0                          Up 3h 15m        │
├──────────┬──────────┬──────────┬──────────┬──────────────┤
│ Dashboard│ Tasks    │ Memory   │ Pipeline │ Logs         │
├──────────┴──────────┴──────────┴──────────┴──────────────┤
│                                                          │
│  ╔══════════════╗  ╔══════════════╗  ╔══════════════════╗│
│  ║ Uptime       ║  ║ Tasks Today  ║  ║ Active Model    ║│
│  ║              ║  ║              ║  ║                 ║│
│  ║   3h 15m     ║  ║  12 ok  3 err║  ║ deepseek-v4-pro ║│
│  ╚══════════════╝  ╚══════════════╝  ╚══════════════════╝│
│                                                          │
│  Live Task Stream (SSE):                                 │
│  ┌──────────────────────────────────────────────────────┐│
│  │ [start]  Model: deepseek-v4-pro                      ││
│  │ [thinking] Let me find that file...                  ││
│  │ [tool]    execute_command → find . -name '*.go'      ││
│  │ [result]  ./main.go ./gateway.go                     ││
│  │ [complete] Found 2 Go files.                         ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  q: quit  tab: switch  enter: select  /: filter         │
└──────────────────────────────────────────────────────────┘
```

**Components:**
- `bubbles/viewport` — scrollable task stream
- `bubbles/table` — task results, memory browser
- `bubbles/spinner` — loading states
- `bubbles/paginator` — paginated results
- `bubbles/textinput` — send tasks from TUI
- `lipgloss` — styling, colors, layouts

**Benefits over Web Dashboard:**
- Works over SSH (no browser needed)
- Keyboard-driven, faster for power users
- Lower resource usage
- Same Elm Architecture Go developers already know

### 🧠 T2: Oz-Style Parallel Agents (Phase 7.3 Evolution)

**Inspiration:** Warp Oz + AGENTIC-ARCHITECTURE.md

```
Ivai Master Process
├── Agent 1: "find all memory leaks in internal/"
│   └── [claude] analyzing... → found 3 issues
├── Agent 2: "write tests for new github_pr tool"
│   └── [deepseek] writing... → 5 new tests
├── Agent 3: "update wiki with new capabilities"
│   └── [deepseek] updating... → wiki updated
└── Agent 4: "run code health on the PR branch"
    └── [cs delta] → 10.0, no issues
```

**Features:**
- `spawn_agent(instruction, model?, tools?)` — create a sub-agent
- `agent_status(id)` — check progress
- `agent_result(id)` — collect output
- `agent_cancel(id)` — kill a runaway agent
- Each agent has its own task_result entry
- Agents can be spawned from the TUI or programmatically

### 💬 T3: Conversation History Browser

**Inspiration:** Charm Mods + gh-dash

Browse, search, and continue past conversations:

```
┌─ Conversations ──────────────────────────────────────────┐
│ /filter: weather________________________                 │
│                                                          │
│   May 3  Dubai weather            ✅  12.5s              │
│   May 3  Abu Dhabi weather        ✅  19.6s              │
│   May 2  refactor auth module     ❌  timeout            │
│   May 2  list all Go files        ✅  4.2s               │
│                                                          │
│  enter: view details  c: continue  d: delete             │
└──────────────────────────────────────────────────────────┘
```

### 🎨 T4: Rich Output Rendering

**Inspiration:** Glamour + Warp's block-based output

- Markdown responses rendered with syntax highlighting
- Code blocks with language detection
- Tables with proper alignment
- Images (via terminal graphics protocol — Kitty, iTerm2)
- Diff viewers for code changes
- Progress bars for long-running tasks

### ⚡ T5: Workflow Templates ("Ivai Skills")

**Inspiration:** Warp Workflows + Charm Huh?

Shareable, parameterized task templates:

```yaml
# .ivai/workflows/code-review.yaml
name: Code Review
description: Review a PR branch for code health and issues
parameters:
  - name: branch
    type: string
    prompt: "Branch to review?"
  - name: focus
    type: select
    options: [security, performance, style, all]
steps:
  - clone: https://github.com/IvanBern/ivai-os.git
  - checkout: $branch
  - code_health: .
  - instruction: "Review the code changes focusing on $focus. Report issues."
```

Run: `ivai run code-review --branch feat/new-feature --focus security`

### 🔄 T6: Session Continuity

**Inspiration:** Charm Mods saved conversations + Crush memory

- Auto-save every conversation as a named session
- Resume: `ivai continue weather` or `ivai continue last`
- Fork: create a branch from any point in a conversation
- Export: conversation → markdown, JSON, or shareable link
- Time travel: replay any past task's SSE stream

### 📊 T7: Pipeline Integration Dashboard

**Inspiration:** gh-dash + GitHub Actions UI

Live CI/CD status in the TUI:

```
┌─ Pipeline ───────────────────────────────────────────────┐
│  main     ✅ build  ✅ test  ✅ cs  ✅ e2e   2m ago      │
│  feat/x   ✅ build  ✅ test  ⏳ cs    —       running     │
│  PR #21   ✅ build  ✅ test  ✅ cs  ✅ e2e   merged      │
│                                                          │
│  r: re-run  o: open in browser  enter: view logs         │
└──────────────────────────────────────────────────────────┘
```

---

## Implementation Phasing

| Phase | Items | Effort | Dependencies |
|---|---|---|---|
| **Alpha** | T1 (TUI core — dashboard + task stream) | 2 weeks | Bubble Tea dep |
| **Alpha** | T5 (Workflow system) | 1 week | T1 |
| **Beta** | T2 (Parallel agents) | 3 weeks | Phase 7.3 design |
| **Beta** | T3 (Conversation browser) | 1 week | T1 |
| **Beta** | T4 (Rich rendering) | 1 week | Glamour dep |
| **Gamma** | T6 (Session continuity) | 1 week | T3 |
| **Gamma** | T7 (Pipeline dashboard) | 1 week | T1, CI in place |

**Total:** ~10 weeks to complete UX transformation.

---

## Technology Choices

| Component | Library | Rationale |
|---|---|---|
| TUI framework | `bubbletea` | Go-native, Elm Architecture, huge ecosystem |
| Styling | `lipgloss` | Terminal styling, adaptive colors |
| Components | `bubbles` | Tables, inputs, spinners, paginators — all built |
| Markdown | `glamour` | Renders markdown with syntax highlighting in terminal |
| Forms | `huh?` | Interactive prompts (model selection, config) |
| SSE client | Built-in | Bubble Tea commands for async I/O |

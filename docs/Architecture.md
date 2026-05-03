# Architecture

## System Overview

ivAI follows a hub-and-spoke architecture where `ivai-core` (the Memory Hub) serves as the central nervous system. All interfaces — CLI, Telegram bot, Gemini CLI — connect through `@ivai/sdk` to the hub.

```
ivai-cli (TypeScript, terminal) ──┐
ivai-bot (Node.js, Telegram) ────┤
Gemini CLI ───────────────────────┤
                                   ↓
         @ivai/sdk (centralized client)
                                   ↓
    ivai-core-daemon (Express REST API, port 4200)
         │                          │
    4-Tier Memory Hub          ivai-workspace
    ┌─────────────────┐        ┌──────────────────┐
    │ Episodic (events)│        │ agents/ (14 .md) │
    │ Semantic (facts) │        │ tools/           │
    │ Procedural (SOPs)│        │ rules/           │
    │ Working (session)│        │ hoops/           │
    └─────────────────┘        │ mcp/             │
         SQLite-vec + FTS5      │ skills/          │
                                │ ivai-core/       │
                                │ spawn_agent.sh   │
                                │ model_router.sh  │
                                │ test_runner.sh   │
                                └──────────────────┘
```

## Component Deep-Dive

### ivai-cli — Terminal Agent

The primary user interface. A TypeScript application (~46K lines) that provides:

- **Autonomous agent loop** with adaptive turn budgets, stagnation detection, and checkpoint/resume
- **Slash command system** (`/model`, `/agents`, `/compact`, `/resume`, etc.)
- **Tool execution** with styled badges, syntax highlighting, inline output previews, per-call timing
- **Browser automation** via Playwright MCP server
- **Model routing** integration via `model_router.sh`
- **Sub-agent spawning** via `spawn_agent.sh`

Key files:
- `ivai-cli/index.ts` — Main entry point and agent loop
- `ivai-cli/system-prompt.ts` — Dynamic system prompt builder
- `ivai-cli/browser-tools.ts` — Browser automation wrappers
- `ivai-cli/mcp-client.ts` — MCP (Model Context Protocol) client
- `ivai-cli/tests.ts` — CLI test suite

### ivai-bot — Telegram Brain

A Node.js bot using Telegraf that provides remote access:

- **Voice message processing** via Gemini multimodal STT
- **Inline keyboard** for confirm/deny actions
- **PIN-gated execution** for destructive commands
- **Live dashboard** pinned message with system stats
- **Autonomous triage** — error stream monitoring, Gemini analysis, "Fix" button alerts

Key files:
- `ivai-bot/src/brain.js` — Core bot logic
- `ivai-bot/src/security.js` — PIN gate and command blacklist
- `ivai-bot/src/api.js` — Memory Hub API client
- `ivai-bot/src/utils.js` — Formatting and helpers

### ivai-core — Memory Hub

An Express.js REST API backed by SQLite-vec and FTS5. The central data store.

See [Memory Hub](./Memory-Hub) for full documentation.

Key files:
- `ivai-core/daemon.js` — Express server entry point
- `ivai-core/db.js` — Database schema and connection
- `ivai-core/embed.js` — Embedding generation (FastEmbed / API)
- `ivai-core/consolidate.js` — Episodic → Semantic consolidation worker
- `ivai-core/search.js` — Vector + full-text hybrid search
- `ivai-core/schemas.js` — Zod validation schemas
- `ivai-core/queue.js` — Async task queue for embeddings

### ivai-sdk — Shared Library

Centralized TypeScript library consumed by all interfaces:

```
@ivai/sdk
├── src/
│   ├── index.ts          — Public API surface
│   ├── config/index.ts   — Configuration loading
│   ├── memory/client.ts  — Memory Hub HTTP client
│   └── model/gemini.ts   — Gemini model client
```

### ivai-security — Security Framework

Python library implementing 6-layer security hardening:

| Layer | Component | Purpose |
|---|---|---|
| 1 | `input_sanitizer.py` | Input validation and escaping |
| 2 | `url_guard.py` | URL allowlist/denylist enforcement |
| 3 | `tool_guardrails.py` | Per-tool safety constraints |
| 4 | `human_loop.py` | PIN-gated destructive action approval |
| 5 | `audit_logger.py` | Tamper-evident audit trail |
| 6 | `system_prompt.py` | Hardened system prompt injection defense |

- **124 tests, 97% coverage**
- Tests: `ivai-security/tests/` (pytest)

### ivai-workspace — Operations Center

The jailed workspace containing all operational assets:

```
ivai-workspace/
├── agents/                  — Sub-agent definitions (Markdown + YAML frontmatter)
│   ├── explore.md
│   ├── plan.md
│   ├── general-purpose.md
│   ├── monitor.md
│   ├── security-auditor.md
│   ├── deploy.md
│   ├── code-reviewer.md
│   ├── cognitive-synthesizer.md
│   ├── meta-cognition-auditor.md
│   ├── agent-factory.md
│   ├── skill-creator.md
│   ├── image-analyst.md
│   └── image-generator.md
├── tools/                   — Executable tool scripts
│   ├── playwright.sh        — Browser automation
│   ├── image_vision.sh      — Image analysis
│   ├── image_generate.sh    — Image generation
│   ├── cs_health.sh         — CodeScene delta analysis
│   └── cs_precommit_hook.sh — Pre-commit health check
├── rules/                   — Dynamic rules (loaded at runtime)
│   └── code-health.json     — CodeHealth 10.0 thresholds
├── hoops/                   — Safety guard configurations
│   └── secrets-guard.json   — Secrets detection patterns
├── mcp/                     — MCP server configs
│   └── playwright.json      — Playwright browser MCP
├── skills/                  — Skill definitions
│   └── code-review.json     — Code review skill config
├── spawn_agent.sh           — Sub-agent spawner
├── agent_runner.sh          — Sub-agent runtime
├── model_router.sh          — Task-to-model router
├── test_runner.sh           — 99-test regression suite
└── bootstrap.sh             — Environment bootstrap
```

## Data Flow

### Task Execution Flow

```
User Input (CLI / Telegram)
    │
    ▼
Model Router (model_router.sh)
    │  Selects: gemini-3-flash | gemini-3-pro | deepseek-v4-flash | deepseek-janus-pro
    ▼
ivai-cli System Prompt Builder
    │  Injects: rules, skills, agent definitions, memory context
    ▼
LLM API Call (Gemini / DeepSeek)
    │
    ▼
Tool Call Resolution
    │  ├── Shell commands → SafeExecute (jailed workspace)
    │  ├── Browser actions → Playwright MCP
    │  ├── Sub-agent tasks → spawn_agent.sh → agent_runner.sh
    │  └── Memory operations → ivai-core REST API
    │
    ▼
Response & Memory Logging
    │  Episodic: tool calls, errors, decisions
    │  Semantic: facts, insights, patterns
```

### Memory Write Path

```
Any interface
    │
    ▼
@ivai/sdk memory client
    │
    ▼
POST /memory/episodic   → Raw event with role, content, interface
POST /memory/semantic   → Fact with category, tags, embedding
POST /memory/procedural → Workflow/sequence with steps
POST /memory/working    → Session context (temporary)
    │
    ▼
ivai-core queue (async)
    │  └── Generate embeddings (FastEmbed BAAI/bge-small-en-v1.5)
    ▼
SQLite-vec (vectors) + FTS5 (full-text)
```

### Memory Read Path

```
Interface requests context
    │
    ▼
GET /context?query=<current task>
    │
    ▼
Hybrid Search (vector similarity + FTS5 keyword match)
    │  1. Generate embedding for query
    │  2. Cosine similarity scan (vec0 table)
    │  3. FTS5 keyword boost
    │  4. Merge + deduplicate + rank
    │
    ▼
Return top-N relevant memories (all 4 tiers)
```

## Runtime Dependencies

```
ivai-cli
  ├── @google/generative-ai (Gemini SDK)
  ├── @anthropic-ai/sdk (Claude - future)
  ├── @ivai/sdk (local)
  └── playwright, cheerio (browser tools)

ivai-core
  ├── express (HTTP server)
  ├── better-sqlite3 (SQLite)
  ├── sqlite-vec (vector extension)
  ├── fastembed (local embeddings)
  └── zod (validation)

ivai-bot
  ├── telegraf (Telegram framework)
  ├── node-fetch
  └── @ivai/sdk (local)

ivai-security
  ├── pytest, pytest-cov
  └── (no runtime deps — pure Python stdlib)
```

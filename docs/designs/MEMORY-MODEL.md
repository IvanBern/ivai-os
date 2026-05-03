# Memory Model: Taxonomy & Lifecycle

**Status:** Design (pre-implementation)
**Roadmap phase:** Phase 6 (Long-Term Intelligence) — RAG pipeline data model
**References:** MemGPT/Letta, Mem0, LangMem research

## 1. Memory Types

ivAI's memory is organized into four distinct types, each with its own storage strategy, TTL, and access pattern.

| Type | What it stores | TTL | Storage | Examples |
|---|---|---|---|---|
| **Episodic** | Time-stamped raw events | 30 days → consolidate or expire | SQLite `episodic_memory` table | "Ivan asked to fix daemon restart 2026-04-22" |
| **Semantic** | Consolidated facts, preferences, routing rules | Long-lived, rewritten on conflict | SQLite `semantic_memory` table + vector index | "Ivan prefers terse reports", "PIN approval required for destructive cmds" |
| **Procedural** | Command workflows, approval sequences, tool patterns | Permanent (versioned) | SQLite `procedural_memory` table | "PIN-gate flow: propose → confirm → PIN hash → execute" |
| **Working** | Current session context | Session-scoped (never persisted) | In-memory Go slice (existing `messages` fetch) | Active message thread, pending tool calls, intermediate reasoning |

## 2. Lifecycle Rules

### Episodic Memory

```
[Raw event] → INSERT into episodic_memory
                    ↓
            [30 days] → Consolidation job runs (Phase 9 cron)
                    ↓
            ┌──────────────────┐
            │ Extract patterns  │ → Promote to Semantic
            │ Detect procedures │ → Promote to Procedural
            │ Delete noise      │ → Expire
            └──────────────────┘
```

- **Insert:** Every tool call, user message, sub-agent spawn, and task result.
- **Consolidation:** Nightly cron (Phase 9) scans episodic entries older than 30 days. Repeated patterns become semantic facts; repeated sequences become procedural workflows. Everything else is pruned.
- **RAG access:** The Phase 6 RAG pipeline queries episodic memory for relevant past events. TTL-filtered: only entries ≤ 30 days unless promoted.

### Semantic Memory

- **Insert:** Only via consolidation from Episodic, or explicit user declaration ("remember: I prefer X").
- **Conflict resolution:** If a new fact contradicts an existing one, the newer fact wins. The old fact is archived with an `overridden_at` timestamp.
- **Preference change vectors:** When a preference flips (e.g., "terse" → "verbose"), this delta is itself recorded as an episodic event for Horizon F (Goal Drift Alignment).
- **RAG access:** Always included in RAG context alongside episodic results.

### Procedural Memory

- **Insert:** Consolidation from Episodic (repeated action sequences) or explicit user teaching ("when I say X, do Y").
- **Versioning:** Procedures are immutable once created. Updates produce a new version; the old version is marked `deprecated`.
- **Access:** Loaded at reasoning-loop start for the current task context. Not included in RAG (procedures are deterministic, not similarity-search).

### Working Memory

- **Scope:** Never persisted to disk. Exists only for the duration of a single task execution.
- **Contents:** The message thread (last N interactions from SQLite `messages` table), current tool call chain, intermediate LLM reasoning.
- **Boundary:** On task completion or crash, working memory is discarded. What matters for persistence must be explicitly written to Episodic.

## 3. Current State (v0.1.0)

- Only the `messages` table exists (SQLite at `/etc/ivai/memory.db`).
- Last 10 interactions loaded as rolling short-term memory — closest analog to Working Memory.
- No Episodic, Semantic, or Procedural tables exist yet.
- This model is the **target architecture** for Phase 6 and Phase 9.

## 4. Migration Path

1. **Phase 6:** Add `episodic_memory` and `semantic_memory` tables. Begin logging tool calls and user messages to Episodic. Build vector index on Semantic facts for RAG.
2. **Phase 6 (RAG):** Query both Episodic and Semantic at LLM call time. Episodic filtered by TTL.
3. **Phase 9:** Add `procedural_memory` table. Implement the consolidation cron job that promotes Episodic → Semantic/Procedural.
4. **Phase 10 (HITL):** Procedural memory stores approval workflows (PIN-gate). Semantic stores user authorization preferences.

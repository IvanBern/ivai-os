# Memory Manager — External World → Crush Memory → Ivai Context

## Problem

Ivai's context is frozen at startup. The system prompt, crush memory files, and recent conversation are loaded once. But the world moves — PRs merge, CI runs, workers spawn and die, code drifts. Ivai needs to stay aware without the operator manually recapping.

## Architecture

```
EXTERNAL WORLD (GitHub, git, swarm, system)
         │
         ├── webhooks (real-time)
         ├── polling (every 5 min)
         └── action-time writes
         │
         ▼
MEMORY-MANAGER AGENT
  Detection → Extraction → Filtering → Formatting
  State diff (.state.json) → Delta → Crush memory files
         │
         ▼
CRUSH MEMORY (.crush/memory/*.md)
  Tagged, journal-style, dated files
         │
         ▼
IVAI CONTEXT
  System prompt + crush memory + conversation
  Reloaded on startup, SIGUSR1, or mtime change
```

## Three Detection Paths

### 1. Webhooks (Real-time)
GitHub pushes events to a lightweight HTTP receiver:
- pull_request: opened, closed, merged, synchronize
- push: new commits on main/staging
- check_run: CI completed (especially failures)

### 2. Scheduled Polling (Every 5 min)
Systemd timer triggers the memory-manager agent:
- git fetch + git log: new commits since last seen
- gh pr list: open PRs relevant to Ivai
- gh run list: recent CI runs
- swarm_status: worker health
- du -sh memory.db: memory usage
- df -h: disk space

### 3. Action-time Writes
Ivai writes its own crush memory after substantive actions:
- PR created → memory with PR number, branch, changes
- Bug found → memory with symptoms, hypothesis
- Documentation written → memory with coverage map

## Importance Filter

- Ivai's own PRs → always write
- CI failures → always write
- Bugs / critical issues → always write
- Changes to cmd/ivai/** or internal/** → always write
- Dependabot/renovate → skip
- PRs stale >7 days → skip

## Deduplication

Content hashing prevents re-reporting the same event within 24h.

## Memory File Format

```markdown
{one-line summary}

## source
- {type}: {detail}

## what changed
- {change 1}

## implications for ivai
- {what this means}

#tag1 #tag2
```

## Context Injection

Three ways Ivai loads crush memory:
1. Startup: read all .crush/memory/*.md, sort by date
2. SIGUSR1 signal: re-read memory directory
3. Auto-refresh: after each response, stat directory for mtime changes

## Implementation Path

1. ✅ Design document (this file)
2. ✅ Agent task YAML spec (agents/tasks/memory-manager.yaml)
3. ✅ Systemd timer + service (systemd/ivai-memory-sync.*)
4. ✅ Initial crush memory baseline (11 files)
5. ⬜ Build ivai-memory-sync binary (Go, uses gh CLI + git)
6. ⬜ Build webhook receiver endpoint
7. ⬜ Install systemd units, enable timer
8. ⬜ Configure GitHub webhook in repo settings
9. ⬜ Add SIGUSR1 handler in Ivai main loop
10. ⬜ End-to-end test

## What This Unlocks

- Ivai knows about merged PRs without being told
- Ivai detects CI failures and investigates proactively
- Ivai tracks its own memory usage, triggers consolidation
- Ivai knows when workers die and can respawn them
- Operator doesn't need to recap — Ivai stays current
- Crush memory is the single source of truth for situational awareness

# Distributed Swarm — Gap Analysis & Architecture

**Status:** Self-reflection complete. Feeds into Phase 7 (Self-Evolution) and Phase 11 (Distributed Swarm).

## Current State

| Capability | Status | Gap |
|---|---|---|
| Single Ivai instance | ✅ ivai-os-linux VM | Need N instances |
| Task execution | ✅ HTTP API + SSE | Need fan-out dispatch |
| Code changes | ✅ Clone → modify → test → PR | Need autonomous task selection |
| PR review | ✅ Ivan approves | Need quality gates for auto-approval |
| Memory | ✅ RAG + embeddings | Per-instance, not shared |
| Code health | ✅ cs delta | Need per-PR enforcement |
| CI/CD | ✅ GitHub Actions | Need swarm-level pipeline |

## What's Missing

### 1. VM Lifecycle Management

Ivai (and Crush) need to manage VMs as resources:

```
Missing tools:
  clone_vm(source, name)     → orbctl clone ivai-os-linux ivai-worker-1
  deploy_ivai(vm_name)       → scp binary + start service
  destroy_vm(vm_name)        → orbctl stop + delete
  list_workers()             → orb list | grep running
  health_check(vm_name)      → GET /api/status
```

### 2. Task Distribution & Aggregation

```
Missing tools:
  swarm_dispatch(tasks[])    → POST to N instances in parallel
  gather_results(job_id)     → collect from all workers
  swarm_status()             → which worker is doing what
```

### 3. Autonomous Roadmap Execution

Ivai should pick the next undone roadmap item and execute it:

```
Missing capability:
  roadmap_next()             → scan ROADMAP.md, find first 🔴 item
  decompose_item(item)       → break into parallel subtasks
  self_assign(item)          → create issue, assign to ivaiber
  implement(item)            → full cycle: design → code → test → PR
```

### 4. Self-Directed Learning

Ivai studies new tools, APIs, and patterns from examples:

```
Missing tools:
  study_repo(url)            → clone, analyze patterns, extract learnings
  study_api(openapi_spec)    → generate tool definition from OpenAPI spec
  learn_from_example(task)   → observe human solving pattern, replicate
```

### 5. Shared Memory Across Swarm

Currently each Ivai has its own memory.db. Swarm needs shared context:

```
Missing infrastructure:
  shared_memory_db           → SQLite over NFS or distributed store
  swarm_embeddings           → cross-instance RAG
  consensus_on_learnings     → what did all workers learn?
```

---

## Architecture: Autonomous Swarm

```
┌─────────────────────────────────────────────────────────────┐
│                      Crush (Orchestrator)                    │
│  ┌─────────┐  ┌──────────┐  ┌───────────┐  ┌────────────┐  │
│  │ Roadmap │  │ Decompose│  │ Dispatch  │  │ Aggregate  │  │
│  │ Scanner │→ │   Task   │→ │  to Swarm │→ │  Results   │  │
│  └─────────┘  └──────────┘  └───────────┘  └────────────┘  │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
    ┌─────────┐  ┌─────────┐  ┌─────────┐
    │ Ivai #1 │  │ Ivai #2 │  │ Ivai #3 │
    │ (prod)  │  │(worker) │  │(worker) │
    └─────────┘  └─────────┘  └─────────┘
         │             │             │
         └─────────────┼─────────────┘
                       ▼
              ┌─────────────────┐
              │  Shared Memory   │
              │  (SQLite + RAG)  │
              └─────────────────┘
```

## Implementation Plan

### Phase A: Swarm Foundation (2 weeks)

```
New Ivai tools:
  T1: clone_worker_vm(name)      — orbctl clone ivai-os-linux $name
  T2: deploy_to_worker(vm, bin)  — deploy Ivai binary to worker VM  
  T3: dispatch_task(vm, task)    — POST /api/task to worker
  T4: gather_results(vm)         — GET /api/task-results from worker
  T5: worker_status(vm)          — GET /api/status from worker
```

### Phase B: Autonomous Execution (2 weeks)

```
New Ivai capabilities:
  T6: roadmap_scanner()          — parse ROADMAP.md, find next 🔴 item
  T7: task_decomposer(item)      — split into independent subtasks
  T8: autonomous_cycle(item)     — design → implement → test → PR
  T9: quality_gate(pr)           — verify CI passed + cs 10.0 before merge
```

### Phase C: Learning & Evolution (2 weeks)

```
New Ivai tools:
  T10: study_github_repo(url)    — clone, analyze structure, extract patterns
  T11: study_api_docs(url)       — read OpenAPI/GraphQL spec, suggest tools
  T12: learn_from_failure(id)    — analyze task_result, suggest fix
  T13: propose_improvement()     — scan failures, propose roadmap items
```

### Phase D: Full Autonomy (2 weeks)

```
Integration:
  T14: swarm_dashboard            — web UI showing all workers, tasks, health
  T15: auto_merge_gate            — if CI green + cs 10.0 + coverage ↑ → auto-merge
  T16: shared_memory              — cross-instance RAG via shared SQLite or API
```

## What Crush Can Do Today

Even without new tools, Crush can manually orchestrate:

```bash
# Fan out 3 tasks to 3 VMs (or same VM with different instructions)
for task in "task1" "task2" "task3"; do
  orb -m ivai-os-linux curl -s -X POST http://localhost:8080/api/task \
    -H 'Content-Type: application/json' \
    -d "{\"instruction\":\"$task\"}" &
done
wait

# Gather results
orb -m ivai-os-linux curl -s http://localhost:8080/api/task-results
```

## Immediate Next Step

Build `clone_worker_vm` tool — Ivai can then provision its own workers:

```go
// New tool: clone_worker_vm
// Ivai: "I need 3 workers for parallel roadmap execution"
// → clones ivai-os-linux → ivai-worker-1, ivai-worker-2, ivai-worker-3
// → deploys latest binary to each
// → fans out tasks
// → aggregates results
// → destroys workers when done
```

This is the single most impactful next step — it lets Ivai scale itself horizontally on demand.

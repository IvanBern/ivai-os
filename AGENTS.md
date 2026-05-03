# AGENTS.md — Ivai OS

Instructions for AI agents (Crush, Claude, Ivai) working on this repository.

## Development Rules

1. **Run `cs delta --verbose`** after every file edit. Fix all issues to 10.0. If delta totals drop below 8.0, stop and refactor before committing more code.
2. **Run `go test -cover ./...`** before every commit. Maintain **85%+ overall** test coverage, **90%+** for new packages and files.
3. **Commit messages** follow conventional format: `type(scope): description`.
4. **PR descriptions** must include: Summary, Changes, Test Results, Code Health, Coverage, Checklist. Use `--body-file`, never `--body`.
5. **Never clear memory.db** during deploy. Use `make reset` intentionally.
6. **Branch protection:** All changes require PRs. No direct pushes to main.

## Code Quality Standards (CodeScene)

Target **Code Health ≥ 8.0** for all files. The following smells degrade AI performance and human maintainability — fix them proactively:

| Smell | Rule | Fix |
|---|---|---|
| **Brain Class** | Single file >300 lines or too many responsibilities | Split by SRP into smaller components |
| **Brain Method** | Function centers too much behavior | Extract Method for specific steps |
| **Complex Method** | Cyclomatic complexity > 10 | Simplify with guard clauses |
| **Deep Nested Complexity** | `if`/loop nested 3+ levels | Flatten with early returns |
| **Bumpy Road** | Multiple distinct logic chunks in one function | Extract each "bump" into a private method |
| **Low Cohesion (LCOM4)** | Methods in a type don't share data | Extract unrelated methods into dedicated types |
| **DRY Violations** | Duplicated logic | Extract shared helper |
| **Primitive Obsession** | Raw strings/ints for domain concepts | Introduce value objects |

**AI-Ready Code:** Healthy code (score ≥ 8) makes AI assistants 2x more effective and 60% less likely to introduce defects. Treat unhealthy files as "Danger Zones" — refactor before any AI-assisted edit.

## Test Coverage Requirements

| Metric | Target |
|---|---|
| **Overall** | **≥ 85%** |
| **New packages/files** | **≥ 90%** |
| **Core logic packages** (`internal/`) | **≥ 85%** |
| **Entry points** (`main`, `startCLI`, `startHTTPServer`) | exempted (blocking/server code) |
| **API-dependent code** (Embed, code_health HTTP) | exempted (requires live service) |

Rules:
- Every new function/method must have at least one test exercising its primary path.
- All error branches in public APIs must be tested (nil-db guards, JSON marshal errors, connection errors).
- Use table-driven tests for repetitive cases, subtests for discrete scenarios.
- Run `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` to verify per-function coverage.

## PR Description Template

```markdown
## Summary

## Changes
-

## Test Results
\`\`\`
ok  github.com/IvanBern/ivai-os/...
\`\`\`

## Code Health
\`\`\`
<!-- Paste: cs delta --verbose main <branch> 2>&1 | grep -v "DEBUG|New version|Use .cs" -->
\`\`\`

## Coverage
\`\`\`
go tool cover -func=coverage.out | grep total
total: (statements) XX.X%
\`\`\`

## Checklist
- [ ] Tests pass
- [ ] Coverage ≥ 85%
- [ ] Code health ≥ 8.0 (cs delta --verbose)
- [ ] No Brain/Complex methods introduced
- [ ] No secrets committed
```

## How Ivai Works

Ivai runs on `ivai-os-linux` VM (OrbStack) at port 8080. It has:
- **15 tools**: read/write/execute/http/wasm + github_pr + code_health + create/list issues + update_wiki + swarm_clone/deploy/dispatch/gather/status
- **Web dashboard**: 6 tabs (Dashboard, Console, Results, Memory, Tools, System)
- **SSE streaming**: real-time task progress
- **RAG memory**: semantic search across past tasks
- **CI/CD**: GitHub Actions (build, test, coverage, gitleaks, E2E)

## Key Files

| File | Purpose |
|---|---|
| `cmd/ivai/main.go` | Entry point, types, core loop |
| `cmd/ivai/tool_registry.go` | Tool dispatch map |
| `cmd/ivai/dashboard.go` | Web UI (embedded HTML) |
| `cmd/ivai/SYSTEM_PROMPT.md` | Ivai's self-knowledge (embedded at build) |
| `internal/llm/gateway.go` | Multi-model LLM gateway |
| `internal/memory/db.go` | SQLite with embeddings and task tracking |
| `internal/tools/` | Tool implementations (fs, shell, network) |
| `docs/designs/` | Architecture, UX ideation, swarm docs |
| `docs/CodeScene Comprehensive Guide*.md` | Code Health reference for AI-quality research |
| `ROADMAP.md` | Phased development plan |
| `.github/workflows/` | CI, E2E, PR check |

## How to Add a Tool

1. Add handler in `cmd/ivai/tool_registry.go` (map entry)
2. Add LLM definition in `buildTools()` in `cmd/ivai/main.go`
3. Add execution function in `cmd/ivai/main.go`
4. Update `TestRegressionAllToolDispatch` expected count
5. Add tests covering primary path and at least one error case

## Deploy

```bash
make build           # cross-compile for Linux ARM64
make service         # scp + restart on VM
make deploy-staging  # deploy to ai-server VM
make rollback        # restore previous binary
make reset           # wipe Ivai's memory
make test-web        # Playwright E2E dashboard test
make tag             # create dated git tag
```

## VM Management

```bash
orb list                          # list all VMs
orb -m ivai-os-linux <command>    # run on Ivai VM
orb -m ai-server <command>        # run on ai-server VM
```

# AGENTS.md — Ivai OS

Instructions for AI agents (Crush, Claude, Ivai) working on this repository.

## Development Rules

1. **Run `cs delta`** after every file edit. Fix issues to 10.0.
2. **Run `go test ./...`** before every commit.
3. **Commit messages** follow conventional format: `type(scope): description`.
4. **PR descriptions** must include: Summary, Changes, Test Results, Code Health, Checklist. Use `--body-file`, never `--body`.
5. **Never clear memory.db** during deploy. Use `make reset` intentionally.
6. **Branch protection:** All changes require PRs. No direct pushes to main.

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
<!-- Paste: cs delta main <branch> 2>&1 | grep -v "DEBUG|New version|Use .cs" -->
\`\`\`

## Checklist
- [ ] Tests pass
- [ ] Code health checked
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
| `ROADMAP.md` | Phased development plan |
| `.github/workflows/` | CI, E2E, PR check |

## How to Add a Tool

1. Add handler in `cmd/ivai/tool_registry.go` (map entry)
2. Add LLM definition in `buildTools()` in `cmd/ivai/main.go`
3. Add execution function in `cmd/ivai/main.go`
4. Update `TestRegressionAllToolDispatch` expected count

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

# Ivai OS — Development Guide

Based on Effective Go, Go best practices, and the Ivai OS architecture.

## Formatting

- **Use `gofmt`** on save. No discussions about formatting — the machine decides.
- **Tabs** for indentation, not spaces.
- **No line length limit** — wrap long lines with an extra tab indent.
- **No parentheses** on `if`, `for`, `switch`.

```go
// Good
if err != nil {
    return err
}

// Bad
if (err != nil) {
    return err
}
```

## Naming

| Context | Convention | Example |
|---|---|---|
| Packages | lowercase, single word, no underscores | `memory`, `sandbox` |
| Exported types | MixedCaps | `TaskResult`, `ProgressEvent` |
| Unexported types | mixedCaps | `taskState`, `providerCall` |
| Getters | Omit `Get` prefix | `Owner()` not `GetOwner()` |
| Interfaces | `-er` suffix for single-method | `io.Reader`, `http.Handler` |
| Constants | MixedCaps, not ALL_CAPS | `maxRetries` not `MAX_RETRIES` |

## Package Structure

Each package gets its own directory. File names match responsibility:

```
cmd/ivai/          — main application
  main.go          — entry point + types
  server.go        — HTTP server setup
  handlers.go      — API handlers
  engine.go        — reasoning loop
  tool_defs.go     — LLM tool definitions
  tool_handlers.go — tool execution
  config.go        — paths, ports, flags
internal/
  llm/             — LLM gateway (DeepSeek, Anthropic, Gemini)
  memory/          — SQLite storage + embeddings
  tools/           — tool implementations (fs, shell, network)
  sandbox/         — Wazero WASM runtime
  telemetry/       — OpenTelemetry
```

## Error Handling

**Always check errors.** Never use `_` to discard an error.

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("something failed: %w", err)
}

// Bad — discards error
result, _ := doSomething()
```

**Wrap errors** with context using `fmt.Errorf("...: %w", err)`.

**Early returns** — handle errors first, keep the happy path unindented:

```go
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
// happy path continues here
```

## Concurrency

- **Goroutines are cheap** but not free. Don't spawn thousands unnecessarily.
- **Use channels** for communication between goroutines.
- **Use `sync.WaitGroup`** for waiting on multiple goroutines.
- **Always pass context** for cancellation and timeouts.
- **Never share memory** without synchronization.

```go
// Good — fire-and-forget with context
go func() {
    emb, err := gateway.Embed(context.Background(), instruction)
    if err != nil {
        return
    }
    dbStore.SaveEmbedding("instruction", instruction, emb)
}()
```

## Interfaces

- **Small, focused** — 1-2 methods per interface.
- **Accept interfaces, return structs.**
- **Don't export interfaces** that only have one implementation.
- **Use `io.Writer` / `io.Reader`** for generic I/O.

```go
// Good — small interface
type ToolExecutor interface {
    Execute(ctx context.Context, args string) (string, error)
}

// Bad — bloated interface
type Everything interface {
    Read(), Write(), Execute(), Clone(), Deploy(), Status()
}
```

## Testing

- **Table-driven tests** for multiple cases.
- **Test file in same package** suffixed `_test.go`.
- **Coverage target:** 30%+ (growing to 80%).
- **Regression tests** for core flows (reasoning loop, tool dispatch, SSE).
- **E2E tests** with Playwright for dashboard.

```go
func TestExtractModel(t *testing.T) {
    tests := []struct{ input, wantModel, wantInst string }{
        {"hello", "deepseek-v4-pro", "hello"},
        {"@claude hi", "claude-3-5-sonnet-20241022", " hi"},
    }
    for _, tc := range tests {
        model, inst := extractModel(tc.input)
        if model != tc.wantModel || inst != tc.wantInst {
            t.Errorf("extractModel(%q) = (%q, %q)", tc.input, model, inst)
        }
    }
}
```

## Memory Management

- **Go garbage-collected** — no manual `free()`.
- **Pre-allocate slices** when size is known: `make([]int, 0, 100)`.
- **Reuse buffers** instead of allocating new ones.
- **Profile with `pprof`** for memory leaks.
- **Zero values are useful** — `sync.Mutex` is ready to use, `bytes.Buffer` is empty.

## Tool Registry Pattern

New tools are added in `tool_registry.go` as map entries, not switch cases:

```go
var toolRegistry = map[string]toolHandler{
    "my_tool": func(ctx context.Context, args string, _ *sandbox.WasmRuntime) (string, error) {
        return executeMyTool(args)
    },
}
```

Then add the LLM definition in `buildTools()` in `tool_defs.go`.

## Code Health

- **Run `cs delta`** before every commit.
- **Aim for 10.0** — fix issues immediately.
- **Acceptable degradation:** tool registry growth (Complex Method) is structural.
- **Refactor when** a file exceeds 300 lines or a function exceeds 50 lines.

## PR Workflow

1. Branch: `feat/description` or `fix/description`
2. `go test ./...` — must pass
3. `cs delta main <branch>` — paste real output in PR body
4. PR body must have: Summary, Changes, Test Results, Code Health, Checklist
5. Use `--body-file`, never `--body`
6. Ivai creates PRs, operator approves and merges

## Commit Messages

Conventional commits: `<type>(<scope>): <description>`

| Type | When |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation |
| `refactor` | Code change without feature/fix |
| `test` | Adding tests |
| `chore` | Build, CI, dependencies |
| `perf` | Performance improvement |

## Project-Specific Patterns

### SSE Streaming

```go
state.emit(ProgressEvent{Type: "tool_call", Message: "...", Data: map[string]any{...}})
```

### Tool Execution

All tools follow: parse JSON args → execute → return `(string, error)`.
The registry dispatches via `toolRegistry[tc.Function.Name]`.

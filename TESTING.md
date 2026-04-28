# Ivai OS Testing Strategy

Ivai OS utilizes a layered testing architecture to ensure the stability of the kernel, the accuracy of the multi-model gateway, and the reliability of autonomous tool execution.

## 🏛 Test Architecture

### 1. Subsystem Unit Tests (Deterministic)
- **Scope**: `internal/tools`, `internal/sandbox`, `internal/memory`.
- **Focus**: Verify the "hands" and "memory" of the OS.
- **Coverage Target**: >90%
- **Techniques**: In-memory SQLite, No-Op Wasm modules, HTTP mocking.

### 2. Gateway Translation Tests (Protocol Integrity)
- **Scope**: `internal/llm/gateway.go`.
- **Focus**: Verify that internal `Message` and `Tool` structures correctly map to DeepSeek, Anthropic, and Gemini JSON schemas.
- **Coverage Target**: 100%
- **Techniques**: Table-driven tests comparing generated JSON against provider specifications.

### 3. Reasoning Loop Evals (Cognitive Reliability)
- **Scope**: `cmd/ivai/main.go` logic.
- **Focus**: Ensure the kernel's `for` loop correctly handles tool-call sequences and state management.
- **Techniques**: Mock LLM providers that return specific tool-call sequences.

### 4. Integration & E2E Tests (System Integrity)
- **Scope**: `ivaictl` -> `HTTP API` -> `Kernel`.
- **Focus**: Verify end-to-end task execution on the target VM environment.

---

## 📊 Reporting & Automation

### Standard Outputs
- **JUnit XML**: For CI/CD integration and trend analysis.
- **LCOV**: For standard coverage tracking.
- **HTML**: For human-readable coverage deep dives.

### Commands
```bash
make test          # Run all tests and generate console summary
make test-reports  # Run tests and generate JUnit, LCOV, and HTML reports
```

## 🛠 Toolchain
- **`testing`**: Standard Go testing package.
- **`httptest`**: For mocking external API responses.
- **`gotestsum`**: (Optional) For JUnit XML generation.
- **`go tool cover`**: For coverage analysis.

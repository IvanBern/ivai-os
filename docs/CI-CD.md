# CI/CD Pipeline

All changes to ivAI go through an automated CI/CD pipeline defined in `.github/workflows/ci.yml`.

## Pipeline Overview

```
Git Push / PR
    │
    ▼
┌─────────────────────────────────┐
│ Job 1: security-tests (Python)  │
│  ├── Install ivai-security      │
│  └── pytest tests/ -v --cov     │
└─────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────┐
│ Job 2: build-and-test           │
│  ├── Setup Node.js 24.x         │
│  ├── Build ivai-sdk             │
│  ├── Install all modules        │
│  ├── Run ivai-core tests        │
│  ├── Start ivai-core daemon     │
│  ├── Run ivai-cli tests         │
│  └── Run ivai-workspace tests   │
└─────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────┐
│ Quality Gates                   │
│  ├── All tests pass             │
│  ├── CodeScene health check     │
│  │   (pre-merge delta analysis) │
│  └── Coverage thresholds met    │
└─────────────────────────────────┘
    │
    ▼
Merge to main (human-gated)
```

## CI Configuration

**File:** `.github/workflows/ci.yml`

### Trigger Conditions

```yaml
on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]
```

### Job: security-tests

```yaml
security-tests:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-python@v5
      with:
        python-version: '3.12'
    - name: Install
      run: cd ivai-security && pip install -e '.[dev]'
    - name: Test
      run: cd ivai-security && pytest tests/ -v --cov=ivai_security
```

### Job: build-and-test

```yaml
build-and-test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
      with:
        node-version: 24.x
    
    # Build SDK first (dependency for all other modules)
    - name: Build ivai-sdk
      run: cd ivai-sdk && npm ci && npx tsc
    
    # Install dependencies
    - name: Install modules
      run: |
        cd ivai-core && npm ci
        cd ivai-cli && npm ci
        cd ivai-bot && npm ci
    
    # Core tests
    - name: Run ivai-core tests
      run: cd ivai-core && npm test
    
    # Start daemon for integration tests
    - name: Start ivai-core daemon
      env:
        IVAI_PORT: 7771
        IVAI_DB_PATH: /home/runner/.ivai_memory/core.sqlite
      run: |
        mkdir -p /home/runner/.ivai_memory
        cd ivai-core
        nohup node daemon.js > daemon.log 2>&1 &
        sleep 5
        curl http://127.0.0.1:7771/health || (cat daemon.log && exit 1)
    
    # CLI tests
    - name: Run ivai-cli tests
      env:
        GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
        IVAI_PORT: 7771
      run: cd ivai-cli && npx tsx tests.ts
    
    # Workspace regression suite (99 tests)
    - name: Run ivai-workspace tests
      run: cd ivai-workspace && bash test_runner.sh
```

## Required Secrets

| Secret | Purpose |
|---|---|
| `GEMINI_API_KEY` | Used by CLI tests that call Gemini API |
| `DEEPSEEK_API_KEY` | Used for model routing tests (optional) |

Secrets are configured in GitHub repository settings → Secrets and variables → Actions.

## Test Suite Breakdown

### ivai-security (124 tests, Python)

```
tests/
├── test_input_sanitizer.py    (28 tests)
├── test_url_guard.py           (19 tests)
├── test_tool_guardrails.py     (24 tests)
├── test_human_loop.py          (18 tests)
├── test_audit_logger.py        (16 tests)
├── test_system_prompt.py       (12 tests)
└── test_integration.py          (7 tests)
```

### ivai-core (Node.js)

Tests cover: CRUD operations, search accuracy, vector similarity, consolidation, and migration.

### ivai-cli (TypeScript)

Tests cover: slash commands, tool execution, model routing, sub-agent spawning.

### ivai-workspace (99 tests, Bash)

```
Section  1: Infrastructure      (5 tests)
Section  2: CLI Commands        (8 tests)
Section  3: Agent Definitions   (14 tests)
Section  4: Spawn Workflow      (12 tests)
Section  5: Task Output         (10 tests)
Section  6: Agent Runner        (15 tests)
Section  7: Tool Scopes         (14 tests)
Section  8: Edge Cases          (10 tests)
Section  9: Body Validation     (6 tests)
Section 10: Cleanup             (5 tests)
```

## CodeScene Health Integration

CodeScene delta analysis runs as a pre-merge quality gate:

```bash
# Full delta analysis
./ivai-workspace/tools/cs_health.sh

# Pre-commit check (fast)
./ivai-workspace/tools/cs_precommit_hook.sh
```

The analysis checks for:
- Code health regression (complexity, duplication, coupling)
- Hotspot creation (frequently changed, low-health files)
- Knowledge distribution risks

## Local CI Simulation

Run the full pipeline locally before pushing:

```bash
# Security tests
cd ivai-security && pytest tests/ -v --cov=ivai_security

# Build SDK
cd ivai-sdk && npm ci && npx tsc

# Core tests
cd ivai-core && npm test

# Start daemon
IVAI_PORT=7771 IVAI_DB_PATH=/tmp/ivai_test.sqlite node ivai-core/daemon.js &
sleep 3
curl http://127.0.0.1:7771/health

# CLI tests
cd ivai-cli && GEMINI_API_KEY=$GEMINI_API_KEY IVAI_PORT=7771 npx tsx tests.ts

# Workspace tests
cd ivai-workspace && bash test_runner.sh
```

## PR Status Checks

A PR must pass these checks before merge:

| Check | Required | Automated |
|---|---|---|
| `security-tests` | ✅ | ✅ |
| `build-and-test` | ✅ | ✅ |
| CodeScene health | ✅ | ✅ |
| Human review | ✅ | ❌ |
| GPG signature | ✅ | ❌ |

## Deployment

Deployment is human-gated:

| Environment | Who Can Deploy | How |
|---|---|---|
| **Staging** | ivAI (autonomous) | `swarm_deploy staging` |
| **Production** | Human only | Manual merge + deploy |

ivAI can deploy to staging autonomously after all checks pass. Production deployment requires explicit human approval and execution.

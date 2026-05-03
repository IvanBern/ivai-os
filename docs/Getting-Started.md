# Getting Started

This guide walks through setting up a development environment for ivAI.

## Prerequisites

| Tool | Purpose | Install |
|---|---|---|
| Node.js 24.x | Runtime for CLI, bot, core, SDK | `fnm install 24 && fnm use 24` |
| Python 3.12+ | Security framework | `uv python install 3.12` |
| Git | Version control | System package manager |
| GitHub CLI | PR management | `apt install gh` / `brew install gh` |

## Clone and Setup

```bash
git clone https://github.com/ivaiber/ivai-core.git
cd ivai-core
```

## Module Setup

### 1. ivai-sdk (shared library — build first)

```bash
cd ivai-sdk
npm ci
npx tsc
cd ..
```

### 2. ivai-core (memory hub)

```bash
cd ivai-core
npm ci
# Create data directory
mkdir -p ~/.ivai_memory
# Start daemon (default port 4200)
IVAI_PORT=4200 IVAI_DB_PATH=~/.ivai_memory/core.sqlite node daemon.js
```

Verify: `curl http://127.0.0.1:4200/health` → should return `{"status":"ok"}`

### 3. ivai-cli (terminal agent)

```bash
cd ivai-cli
npm ci
npm link  # makes 'ivai' available globally
```

Set API keys:

```bash
export GEMINI_API_KEY="your-gemini-key"
export DEEPSEEK_API_KEY="your-deepseek-key"
export IVAI_PORT=4200  # memory hub
```

Run: `ivai "your task here"`

### 4. ivai-bot (Telegram brain)

```bash
cd ivai-bot
npm ci
# Create bot token
export TELEGRAM_BOT_TOKEN="your-bot-token"
export IVAI_PORT=4200
node ivai-telegram-brain.js
```

### 5. ivai-security (test only)

```bash
cd ivai-security
pip install -e '.[dev]'
pytest tests/ -v --cov=ivai_security
```

## Environment Variables

| Variable | Required By | Description |
|---|---|---|
| `GEMINI_API_KEY` | CLI, bot, core | Gemini API key |
| `DEEPSEEK_API_KEY` | CLI, core | DeepSeek API key |
| `TELEGRAM_BOT_TOKEN` | Bot | Telegram bot token |
| `IVAI_PORT` | Core, CLI, bot | Memory hub port (default: 4200) |
| `IVAI_DB_PATH` | Core | SQLite database path |
| `IVAI_MODEL` | CLI | Override default model |

Secrets can also be stored in `~/.ivai_secrets/` files:
- `~/.ivai_secrets/gemini_env`
- `~/.ivai_secrets/deepseek_env`

## Running Tests

### Full regression suite

```bash
cd ivai-workspace
chmod +x test_runner.sh spawn_agent.sh agent_runner.sh ivai-agent
./test_runner.sh
```

This runs 99 tests across 10 sections: infrastructure, CLI commands, agent definitions, spawn workflow, task output, agent runner, tool scopes, edge cases, body content validation, and cleanup.

### Module-specific tests

```bash
# ivai-core
cd ivai-core && npm test

# ivai-cli (requires GEMINI_API_KEY)
cd ivai-cli && npx tsx tests.ts

# ivai-security (124 tests)
cd ivai-security && pytest tests/ -v --cov=ivai_security
```

## Code Quality

### Linting

```bash
# TypeScript/JavaScript (ESLint Flat Config)
npx eslint .

# Python
uvx ruff check .

# Go
golangci-lint run
```

### CodeScene Health

```bash
cd ivai-workspace
./tools/cs_health.sh          # Full delta analysis
./tools/cs_precommit_hook.sh  # Pre-commit check
```

## Project Structure

```
ivai-core/                    ← Root repo
├── ivai-sdk/                 ← Shared TypeScript library
│   └── src/
│       ├── index.ts
│       ├── config/index.ts
│       ├── memory/client.ts
│       └── model/gemini.ts
├── ivai-core/                ← Memory hub daemon
│   ├── daemon.js             ← Express server
│   ├── db.js                 ← SQLite-vec schema
│   ├── embed.js              ← Embedding generation
│   ├── search.js             ← Hybrid search
│   ├── consolidate.js        ← Memory consolidation
│   ├── schemas.js            ← Zod validation
│   └── routes/
│       ├── memory.js
│       ├── search.js
│       └── system.js
├── ivai-cli/                 ← Terminal agent
│   ├── index.ts              ← CLI entry + agent loop
│   ├── system-prompt.ts      ← Prompt builder
│   ├── browser-tools.ts      ← Browser automation
│   ├── mcp-client.ts         ← MCP client
│   └── tests.ts              ← CLI tests
├── ivai-bot/                 ← Telegram bot
│   └── src/
│       ├── brain.js
│       ├── security.js
│       ├── api.js
│       └── utils.js
├── ivai-security/            ← Security framework
│   ├── ivai_security/
│   │   ├── input_sanitizer.py
│   │   ├── url_guard.py
│   │   ├── tool_guardrails.py
│   │   ├── human_loop.py
│   │   ├── audit_logger.py
│   │   └── system_prompt.py
│   └── tests/
│       ├── test_input_sanitizer.py
│       ├── test_url_guard.py
│       ├── test_tool_guardrails.py
│       ├── test_human_loop.py
│       ├── test_audit_logger.py
│       ├── test_system_prompt.py
│       └── test_integration.py
├── ivai-workspace/           ← Operational assets
│   ├── agents/               ← 13 sub-agent definitions
│   ├── tools/                ← Executable tool scripts
│   ├── rules/                ← Dynamic rules
│   ├── hoops/                ← Safety guards
│   ├── mcp/                  ← MCP server configs
│   ├── skills/               ← Skill definitions
│   ├── spawn_agent.sh        ← Sub-agent spawner
│   ├── agent_runner.sh       ← Sub-agent runtime
│   ├── model_router.sh       ← Task-to-model router
│   └── test_runner.sh        ← Regression suite
├── .github/workflows/        ← CI pipeline
│   └── ci.yml
├── AGENTS.md                 ← AI agent directory
├── GEMINI.md                 ← Workspace overview
└── README.md                 ← Project readme
```

## Development Workflow

1. **Create branch:** `git checkout -b feature/my-feature`
2. **Implement:** Follow CodeHealth 10.0 thresholds
3. **Lint:** Run language-appropriate linter
4. **Test:** Run module tests + regression suite
5. **Self-review:** Review diff for secrets, correctness
6. **Commit:** Signed commits with GPG
7. **Push:** `git push origin feature/my-feature`
8. **Create PR:** `gh pr create --title "feat(scope): description"`
9. **Monitor CI:** Wait for green checkmark
10. **Merge:** After review (human-gated)

## Common Issues

### "IVAI_PORT connection refused"

Ensure ivai-core daemon is running:
```bash
cd ivai-core && node daemon.js &
```

### "GEMINI_API_KEY is not set"

Export your key or create `~/.ivai_secrets/gemini_env`:
```bash
echo 'GEMINI_API_KEY=your-key' > ~/.ivai_secrets/gemini_env
```

### "Module not found: @ivai/sdk"

Build the SDK first:
```bash
cd ivai-sdk && npm ci && npx tsc
```

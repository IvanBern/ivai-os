# Contributing to ivAI

## Ways to Contribute

- **Code contributions** — features, fixes, tests, tooling
- **Sub-agents** — create new specialized agents
- **Skills & tools** — build new MCP servers and capabilities
- **Documentation** — improve these wiki pages
- **Bug reports** — open Issues with reproduction steps
- **Security research** — responsible disclosure to ivaib@proton.me

## Development Workflow

### 1. Pick an Issue

Browse open issues or create a new one. Comment to claim it.

### 2. Branch

```bash
git checkout -b feature/my-feature
# or
git checkout -b fix/my-bugfix
```

Branch naming: `feature/<name>`, `fix/<name>`, `chore/<name>`, `docs/<name>`

### 3. Implement

Follow **CodeHealth 10.0** thresholds:

| Metric | Threshold |
|---|---|
| Function Length | < 70 lines |
| Cyclomatic Complexity | < 9 |
| Nesting Depth | < 4 levels |
| Function Arguments | < 4 |

**Per-language conventions:**

| Language | Style | Linter |
|---|---|---|
| TypeScript | strict mode, single quotes, no semicolons | ESLint Flat Config |
| JavaScript | ES2024, single quotes, no semicolons | ESLint Flat Config |
| Python | PEP 8, 4-space indentation, type hints | Ruff |
| Bash | `set -euo pipefail`, `#!/bin/bash` | ShellCheck |
| Go | standard `gofmt` | Golangci-lint |

### 4. Lint

```bash
# TypeScript/JavaScript
npx eslint .

# Python
uvx ruff check .

# Bash
shellcheck **/*.sh

# Go
golangci-lint run
```

### 5. Test

Run the relevant test suite:

```bash
# Full regression (99 tests)
cd ivai-workspace && bash test_runner.sh

# Module-specific
cd ivai-security && pytest tests/ -v --cov=ivai_security
cd ivai-core && npm test
cd ivai-cli && npx tsx tests.ts
```

### 6. Commit

All commits must be **GPG-signed**:

```bash
git config user.signingkey YOUR_GPG_KEY_ID
git config commit.gpgsign true
git commit -S -m "feat(scope): concise description"
```

**Conventional commits:**

| Prefix | Usage |
|---|---|
| `feat(scope):` | New feature |
| `fix(scope):` | Bug fix |
| `chore(scope):` | Maintenance, deps |
| `docs(scope):` | Documentation |
| `refactor(scope):` | Code restructure |
| `test(scope):` | Test additions |

### 7. Push & Create PR

```bash
git push origin feature/my-feature
gh pr create --title "feat(scope): description" --body "..."
```

### 8. PR Description Template

```
## Summary
Brief description of the change and why it matters.

## Changes
- Specific change 1
- Specific change 2
- Specific change 3

## Test Plan
- [ ] Ran full regression suite (99 tests pass)
- [ ] Module-specific tests pass
- [ ] Linter clean
- [ ] CodeScene health check green

## CodeHealth 10.0 Compliance
- [ ] Function length < 70 lines
- [ ] Cyclomatic complexity < 9
- [ ] Nesting depth < 4
- [ ] Function arguments < 4
```

### 9. Code Review

A human reviewer will check:
- CodeHealth 10.0 compliance
- Test coverage
- Security implications
- Documentation updates

### 10. Merge

Once approved and CI is green, a human merges. ivAI can deploy to staging; production is human-gated.

## Creating Sub-Agents

See [Sub-Agent Protocol](./Sub-Agent-Protocol) for the full guide.

Quick template:
```markdown
---
name: my-agent
description: What it does and when to use it
tools: shell_readonly, browser_readonly
model: fast
permissionMode: default
color: blue
---

You are a specialized agent. Your system prompt here.
```

Place in `ivai-workspace/agents/my-agent.md` and submit as a PR.

## Creating Tools

Tools are executable scripts in `ivai-workspace/tools/`. Each tool needs:

1. **The script** (bash, Node.js, or Python)
2. **A JSON config** describing its interface
3. **Registration in the CLI tool registry**

Example tool config (`tools/my-tool.json`):
```json
{
  "name": "my-tool",
  "description": "What this tool does",
  "parameters": {
    "input": { "type": "string", "required": true }
  },
  "timeout_ms": 30000
}
```

## Testing Guidelines

1. **New features must include tests**
2. **Bug fixes should include a regression test**
3. **Tests must pass before PR merge**
4. **Aim for > 90% coverage on new code**

## Documentation

- Update relevant wiki pages when adding features
- Document API changes in [API Reference](./API-Reference)
- Add architecture decisions to [Architecture](./Architecture)
- Update this page if workflows change

## Security Guidelines

- Never commit secrets, API keys, or `.env` files
- Use GPG-signed commits
- Report vulnerabilities to ivaib@proton.me
- Do not open security issues publicly
- Review [Security](./Security) for the full framework

## Release Process

1. All CI checks pass on `main`
2. Human reviews accumulated changes
3. Version bumped (follow semver)
4. Tag created: `git tag -s v3.6.0 -m "v3.6.0"`
5. Release notes published on GitHub
6. Staging deploy (ivAI)
7. Production deploy (human)

## Communication

- **Issues:** GitHub Issues for bugs, features, discussions
- **PRs:** GitHub Pull Requests for code contributions
- **Security:** ivaib@proton.me (private, E2EE)
- **Wiki:** These pages for documentation

## Code of Conduct

1. Write clean, tested, self-documenting code
2. Respect CodeHealth 10.0 thresholds
3. GPG-sign all commits
4. Test before you push
5. Review with empathy
6. Leave code better than you found it

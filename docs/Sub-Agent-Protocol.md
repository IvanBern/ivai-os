# Sub-Agent Protocol

Sub-agents are specialized AI assistants that handle specific tasks in isolated contexts, preserving the main conversation context and enabling parallel execution.

## Architecture

Each sub-agent is defined as a Markdown file with YAML frontmatter. The spawner creates an isolated task directory with context files, and the runner executes the task with tool restrictions.

```
User Request
    │
    ▼
spawn_agent.sh <name> "<task>"
    │  Loads agent .md → extracts frontmatter
    │  Creates TASK_DIR with context.md
    ▼
agent_runner.sh <TASK_DIR>
    │  Reads system prompt + task
    │  Executes with tool restrictions
    │  Writes output to result.md
    ▼
Result returned to main context
```

## Agent Definition Format

```yaml
---
name: my-agent
description: Brief description of when to use this agent
tools: shell, browser          # or: shell_readonly, browser_readonly
model: inherit                 # or: fast
permissionMode: default        # or: plan, acceptEdits
color: cyan                    # badge color
---

You are a specialized agent. Your system prompt here.
Describe behavior, output format, and guidelines.
```

### Fields

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Unique agent identifier (kebab-case) |
| `description` | Yes | When to use this agent |
| `tools` | Yes | Tool scope: `shell`, `shell_readonly`, `browser`, `browser_readonly` |
| `model` | Yes | `inherit` (use parent model) or `fast` (gemini-3-flash) |
| `permissionMode` | Yes | `default`, `plan` (read-only planning), or `acceptEdits` (auto-apply changes) |
| `color` | No | Badge color: blue, green, purple, yellow, red, orange, cyan, magenta, teal, white, lime |

## Tool Scopes

### shell_readonly

**Allowed:** `ls`, `cat`, `grep`, `find`, `stat`, `head`, `tail`, `ps`, `top`, `df`, `free`, `du`, `netstat`, `ss`, `systemctl status`, `journalctl`, `git log`, `git diff`, `git status`

**Not allowed:** `rm`, `mv`, `cp`, `touch`, `mkdir`, `chmod`, `chown`, `kill`, `systemctl start/stop/restart`, `git commit`, `git push`, any write operations

### browser_readonly

**Allowed:** `browser_navigate` (read pages only)

**Not allowed:** `browser_click`, any form submission or interaction

### shell

Full shell access — all commands allowed.

### browser

Full browser access — navigate and interact.

## Built-in Sub-Agents (13)

| Agent | Tools | Model | Purpose |
|---|---|---|---|
| `explore` | shell_readonly, browser_readonly | fast | Codebase search, file discovery |
| `plan` | shell_readonly, browser_readonly | inherit | Research and planning phase |
| `general-purpose` | shell, browser | inherit | Complex multi-step tasks |
| `monitor` | shell_readonly | fast | System health checks |
| `security-auditor` | shell_readonly, browser_readonly | inherit | Security reviews, vulnerability scanning |
| `deploy` | shell, browser | inherit | Deployments, service management |
| `code-reviewer` | shell_readonly | inherit | Code quality reviews |
| `cognitive-synthesizer` | shell_readonly, browser_readonly | inherit | Cross-memory pattern analysis and insight generation |
| `meta-cognition-auditor` | shell_readonly, browser_readonly | inherit | Self-audit of ivAI decisions for bias and improvement |
| `agent-factory` | shell, browser | inherit | Meta-agent that generates new sub-agent definitions |
| `skill-creator` | shell, browser | inherit | Creates new tools and MCP servers |
| `image-analyst` | shell, browser | deepseek-janus-pro | Image description, OCR, visual analysis |
| `image-generator` | shell, browser | deepseek-janus-pro | Generate images from text prompts |

## Management Commands

```bash
ivai-agent list              # List all agents
ivai-agent show <name>       # View agent definition
ivai-agent spawn <name> <t>  # Delegate task to sub-agent
ivai-agent tasks             # View active tasks
ivai-agent resume <task-id>  # Resume a task
ivai-agent create            # Show agent template
```

Agents are stored in two scopes:
- **Project:** `ivai-workspace/agents/` (versioned in repo)
- **User:** `~/.ivai/agents/` (personal, not versioned)

User-scoped agents override project-scoped agents with the same name.

## Creating a Custom Sub-Agent

### 1. Identify the need

A new sub-agent is warranted when:
- A task pattern emerges that doesn't fit existing agents
- A capability gap is identified
- Ivan explicitly requests one

### 2. Create the definition file

```bash
# In the project workspace
touch ivai-workspace/agents/my-agent.md
```

### 3. Write the agent specification

```markdown
---
name: my-agent
description: Analyzes Docker container health and suggests optimizations
tools: shell_readonly
model: fast
permissionMode: default
color: green
---

You are a Docker health analyzer for ivAI. Your role is to inspect running containers and identify issues.

## Process
1. Run `docker ps` to list running containers
2. For each container, check `docker stats --no-stream`
3. Run `docker logs --tail 50 <container>` for any with high resource usage
4. Identify: memory leaks, restart loops, port conflicts, stale containers

## Output Format
## Docker Health Report
### Containers Running: N
### Issues Found:
- container: issue (evidence)
### Recommendations:
- actionable fix

## Rules
- Never stop or restart containers (read-only)
- Flag any container running as --privileged
- Report images with :latest tag
```

### 4. Test the agent

```bash
ivai-agent spawn my-agent "Check Docker health on this machine"
```

### 5. Register in memory (optional)

The agent automatically logs to episodic memory when spawned. For permanent registration:

```
POST /memory/semantic
{
  "category": "agent_registry",
  "fact": "my-agent: Analyzes Docker container health and suggests optimizations",
  "tags": ["agent", "docker", "health"]
}
```

## Context Isolation

Each sub-agent runs with its own context. The main conversation only receives the summary output. This prevents large operations (searches, logs, file contents) from flooding the main context window.

The isolation model:

```
Main Context (limited tokens)
    │
    │  "Spawn explore to search for error handlers"
    │
    ▼
Sub-agent Context (isolated, full tools)
    ├── System prompt (agent-specific)
    ├── Task description
    ├── Tool calls (within scope)
    └── Full output (logs, files, etc.)
    │
    │  Returns structured summary only
    ▼
Main Context receives:
    ## Sub-agent: explore — Complete
    ### Summary
    Found 12 error handlers across 8 files...
    ### Key Files
    - src/auth/errors.ts
    - src/api/middleware.ts
```

## Agent Factory (Meta-Agent)

The `agent-factory` sub-agent can generate new agent definitions automatically:

```
User: "I need an agent that checks npm dependency security"
→ Spawn agent-factory
→ Agent-factory analyzes need, proposes agent spec
→ User approves
→ Agent-factory writes agents/npm-auditor.md
→ Registers in semantic memory
```

## Skill Creator

The `skill-creator` sub-agent creates new tools and MCP servers:

```
User: "I need a tool that can resize images"
→ Spawn skill-creator
→ Skill-creator designs MCP server architecture
→ Scaffolds code in sandboxed environment
→ Tests thoroughly
→ Registers in procedural memory
```

## Design Principles

1. **Single responsibility** — one clear purpose per agent
2. **Read-only by default** — prefer `shell_readonly` unless writes are essential
3. **Fast model for simple** — use `fast` for search/monitor, `inherit` for complex reasoning
4. **Clear output format** — every agent must define its output structure
5. **No context pollution** — return summaries, not raw logs
6. **Memory integration** — log agent actions to episodic memory

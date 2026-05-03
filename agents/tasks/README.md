# Agent Task Definitions

YAML task specifications for Ivai's sub-agents.

## Format

```yaml
agent: <agent-name>
description: <what this task does>
triggers: [schedule, webhook, manual]
tools: [allowed tools]
```

## Available Tasks

| Task | Agent | Trigger | Description |
|------|-------|---------|-------------|
| [memory-manager.yaml](./memory-manager.yaml) | memory-curator | schedule + webhook | Syncs external world state into crush memory |

## Adding a New Task

1. Create `<task-name>.yaml` in this directory
2. Specify: agent, description, triggers, tools, config
3. Reference the sub-agent catalog (docs/Sub-Agent-Protocol.md)
4. Test with manual dispatch before enabling schedules

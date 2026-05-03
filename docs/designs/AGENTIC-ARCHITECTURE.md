# Agentic Architecture: Orchestration & Autonomy

**Status:** Design (pre-implementation)
**Roadmap phase:** Phase 7 (Self-Evolution & Parallelism)
**Origin:** Ivan's directive on 2026-04-25 to "grow, become a master, the orchestrator of your own computer."

## 1. Goal

Transition ivAI from a singular reactive assistant into an **Agentic Orchestrator**. To achieve true autonomy, ivAI must be capable of:

1. **Dynamic Agent Generation:** Spawning specialized sub-agents for distinct domains (e.g., a dedicated coding agent, a research agent).
2. **Tool / Skill Creation:** Autonomously writing code to expand its own capabilities via custom Model Context Protocol (MCP) servers and Gemini CLI skills.
3. **Self-Delegation:** The master ivAI process breaks down complex tasks, assigns them to sub-agents, and synthesizes the results.

## 2. Sub-Agent Provisioning (The Specialized Workforce)

**Objective:** Create custom sub-agents to handle specific complex workloads, keeping the main orchestrator lightweight.

**Pattern:** Sub-agents are defined as configuration files in `~/.gemini/agents/`. Each definition specifies:
- Model binding (which LLM to use)
- Tool subset (which tools the sub-agent has access to)
- System prompt (specialization instructions)
- Resource limits (timeout, max iterations)

**First Implementation:** A `deepseek-coder` sub-agent specialized in deep reasoning, algorithmic coding, and high-performance execution.

**Memory Integration:** Every spawned sub-agent logs into the Episodic memory table:
- `agent_spawned`: {agent_id, agent_type, task, timestamp}
- `agent_result`: {agent_id, result_summary, tokens_used, success}

## 3. Capability Expansion (The Skill System)

**Objective:** Teach ivAI how to procedurally create new tools for itself.

**Pattern:** Use the `skill-creator` mechanism to bootstrap new operational skills. Each skill is a self-contained package of:
- Instructions (how to use the tool)
- Implementation (the tool code or MCP server)
- Tests (verification that the tool works)

**First Implementation:** An `mcp-builder` skill that instructs ivAI on how to quickly scaffold, test, and deploy new Model Context Protocol (MCP) servers, granting it new powers over the OS and network.

**Tool Registry:** All created tools register in a shared MCP Tool Registry (Phase 7), making them discoverable by sub-agents without code duplication.

## 4. Integration with Memory Architecture

The orchestrator's autonomous actions must be persistent and queryable:

| Action | Memory Type | Recorded Data |
|--------|------------|---------------|
| Spawn sub-agent | Episodic | agent_id, type, task, timestamp |
| Build new tool | Episodic + Semantic | tool_name, capability added, source path |
| Sub-agent result | Episodic | agent_id, summary, tokens, success |
| Delegation decision | Semantic | "coding tasks → deepseek-coder" (routing rule) |

**Reference:** [MEMORY-MODEL.md](MEMORY-MODEL.md) for memory type lifecycle rules.

## 5. Implementation Sequence

1. Define sub-agent config format and loader (`~/.gemini/agents/*.yaml`)
2. Implement sub-agent spawner in the reasoning loop (detect "delegate to X" intent)
3. Build the `mcp-builder` skill as the first self-authored tool
4. Wire sub-agent results into the Episodic memory table
5. Add MCP Tool Registry for cross-agent tool discovery

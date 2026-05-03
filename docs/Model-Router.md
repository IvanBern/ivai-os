# Model Router

The Model Router intelligently selects the optimal LLM per task type. This prevents vendor lock-in, optimizes cost, and matches model strengths to task requirements.

## Routing Table

| Task Type | Keyword Triggers | Model | Rationale |
|---|---|---|---|
| Fast triage | `quick`, `fast`, `simple`, `check`, `status`, `health`, `monitor` | `gemini-3-flash-preview` | Low latency, cheap |
| Deep reasoning | `analyze`, `reason`, `complex`, `research`, `plan`, `strategy`, `design` | `gemini-3-pro-preview` | Strong reasoning, large context |
| Code generation | `code`, `implement`, `build`, `program`, `script`, `debug`, `fix`, `refactor` | `deepseek-v4-flash` | Optimized for code |
| Complex code | `architecture`, `system design`, `deep debug` | `deepseek-v4-pro` | Maximum code reasoning |
| Image analysis | `image`, `vision`, `OCR`, `describe`, `screenshot`, `photo` | `deepseek-janus-pro` | Multimodal vision |
| Image generation | `generate`, `draw`, `create image`, `illustrate` | `deepseek-janus-pro` / Imagen 3 | Image synthesis |
| Default | (anything else) | `gemini-3-flash-preview` | Safe, fast default |

## Implementation

The router is a bash script (`ivai-workspace/model_router.sh`) that accepts a task string and outputs the model name:

```bash
#!/bin/bash
# model_router.sh — Routes task to optimal model

TASK="$1"

if echo "$TASK" | grep -qiE 'image|vision|ocr|describe this|screenshot|photo|generate|draw|create.*image|illustrate'; then
    echo "deepseek-janus-pro"
elif echo "$TASK" | grep -qiE 'code|implement|build|program|script|debug|fix|refactor'; then
    echo "deepseek-v4-flash"
elif echo "$TASK" | grep -qiE 'analyze|reason|complex|research|plan|strategy|design'; then
    echo "gemini-3-pro-preview"
elif echo "$TASK" | grep -qiE 'architecture|system design|deep debug'; then
    echo "deepseek-v4-pro"
else
    echo "gemini-3-flash-preview"
fi
```

### CLI Integration

The CLI calls the router before each task:

```bash
MODEL=$(./model_router.sh "$USER_TASK")
ivai --model "$MODEL" "$USER_TASK"
```

Users can override routing with `--model` flag or `/model` slash command:

```bash
ivai --model deepseek-v4-pro "quick status check"  # force deep reasoning
```

```
/model deepseek-v4-pro    ← switch mid-session
/model list               ← see all available
```

## Available Models

### Gemini (Google)

| Model | Context | Strengths | Use |
|---|---|---|---|
| `gemini-3-pro-preview` | 1M tokens | Deep reasoning, planning | Complex analysis |
| `gemini-3-flash-preview` | 1M tokens | Speed, cost-efficiency | Quick tasks, default |
| `gemini-2.5-pro` | 1M tokens | Stable reasoning | Production workloads |
| `gemini-2.5-flash` | 1M tokens | Fast, reliable | Monitoring, health |
| `gemini-2.0-flash` | 1M tokens | Legacy stable | Fallback |
| `deep-research-max-preview-04-2026` | 1M tokens | Multi-step research | Deep investigations |

### DeepSeek

| Model | Context | Strengths | Use |
|---|---|---|---|
| `deepseek-v4-pro` | 128K tokens | Maximum code reasoning | Architecture, system design |
| `deepseek-v4-flash` | 128K tokens | Fast code generation | Implementation, debugging |
| `deepseek-janus-pro` | Multimodal | Vision + generation | Images, OCR, screenshots |

## Cost Optimization Strategy

The router implements a tiered cost strategy:

```
Tier 1 (Free/Cheapest)  → gemini-3-flash-preview
Tier 2 (Mid)             → gemini-3-pro-preview, deepseek-v4-flash
Tier 3 (Premium)         → deepseek-v4-pro, deepseek-janus-pro
```

Default routing stays in Tier 1 unless the task explicitly requires higher-tier reasoning.

## Session Stats

After each task, the CLI reports:

```
Session Summary
├── Model: gemini-3-flash-preview
├── Duration: 45.2s
├── Turns: 7 (budget: 22)
├── Tool calls: 12
├── Tokens: 8,234 prompt + 1,456 completion
├── Cached: 3,120 tokens
└── Est. cost: $0.0032
```

## Adding a New Model

1. Add model to `ivai-workspace/model_router.sh` routing table
2. Add model key to `~/.ivai_secrets/deepseek_env` or equivalent
3. Update `ivai-cli/system-prompt.ts` available models list
4. Update `~/.ivai_memory/available_models.json`
5. Test routing:
   ```bash
   ./model_router.sh "generate an image of a cat"
   # Should output: deepseek-janus-pro
   ```

## Model Selection Heuristics

The router considers these factors in order:

1. **Keyword match** — highest priority, explicit triggers
2. **Task complexity** — estimated from input length and structure
3. **Session history** — if previous turns used a premium model, continues with it
4. **Cost budget** — not yet enforced, but tracked per session
5. **User override** — `--model` flag or `/model` command always wins

## Future: Dynamic Routing (Phase 10)

Phase 10 of the maturation roadmap introduces true model agnosticism:

- **Performance tracking** — per-model latency, success rate, token efficiency
- **A/B routing** — split traffic between models, compare outcomes
- **Automatic fallback** — if a model errors, retry with next-best
- **Cost-aware routing** — respect daily/monthly budget limits
- **Task embedding routing** — embed the task, find the most similar historical successful task, use that model

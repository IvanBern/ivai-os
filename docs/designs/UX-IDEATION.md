# Ivai UX Ideation Plan

**Status:** Research complete. Ready for roadmap integration.

## Research Summary

### 1. Charmbracelet Bubble Tea — TUI Framework
- **What:** Elm Architecture for Go terminals. Model/Update/View pattern.
- **Ecosystem:** Bubbles (text inputs, viewports, spinners, tables, paginators), Lip Gloss (terminal styling, colors, layouts), Huh? (interactive forms and prompts), Glamour (markdown rendering), Harmonica (animations)
- **Adoption:** 18,000+ apps including Azure, AWS, NVIDIA, CockroachDB
- **Relevance:** Ivai could have a rich TUI admin interface alongside the web dashboard. Terminal-native, works over SSH, no browser needed.

### 2. Warp Terminal + Oz Agent
- **Warp:** Modern terminal with IDE-like editing, AI integration, GPU-accelerated rendering
- **Oz:** Cloud orchestration platform for parallel AI coding agents. Programmable, auditable, steerable.
- **Key Oz features:**
  - Spin up unlimited parallel agents in the cloud
  - Agents are programmable with custom instructions
  - Full audit trail of agent actions
  - Steerable — can interrupt, redirect, approve mid-execution
  - Works in terminal but scales to cloud
- **Relevance:** Ivai's Phase 7.3 (sub-agent spawning) maps directly to Oz's multi-agent orchestration. Oz's audit trail → Ivai's task_results. Oz's steerability → Ivai's HITL (Phase 10).

### 3. Charm Mods (now sunset → Crush)
- **What:** AI for command line, built for pipelines. Could ingest command output, format as markdown/JSON.
- **Features:** Saved conversations, custom roles, MCP server support, streaming, multi-model
- **Relevance:** Already in Crush. Ivai could adopt conversation history browsing from the TUI.

### 4. Other Notable Interfaces
- **gh-dash** (Bubble Tea): GitHub PR/issue dashboard in terminal — exact pattern Ivai could use for its own task management
- **Lazygit:** Git TUI — shows how complex state can be managed in a terminal
- **k9s:** Kubernetes TUI — resource monitoring pattern applicable to Ivai's system monitoring

---

## Ideation Items

### 🖥️ T1: Ivai TUI (`ivai-tui`) — Terminal Admin Interface

**Inspiration:** Bubble Tea + gh-dash + k9s

A Bubble Tea-based TUI that serves as the primary admin interface:

```
┌─────────────────────────────────────────────────────────┐
│ Ivai OS v0.2.0                          Up 3h 15m        │
├──────────┬──────────┬──────────┬──────────┬──────────────┤
│ Dashboard│ Tasks    │ Memory   │ Pipeline │ Logs         │
├──────────┴──────────┴──────────┴──────────┴──────────────┤
│                                                          │
│  ╔══════════════╗  ╔══════════════╗  ╔══════════════════╗│
│  ║ Uptime       ║  ║ Tasks Today  ║  ║ Active Model    ║│
│  ║              ║  ║              ║  ║                 ║│
│  ║   3h 15m     ║  ║  12 ok  3 err║  ║ deepseek-v4-pro ║│
│  ╚══════════════╝  ╚══════════════╝  ╚══════════════════╝│
│                                                          │
│  Live Task Stream (SSE):                                 │
│  ┌──────────────────────────────────────────────────────┐│
│  │ [start]  Model: deepseek-v4-pro                      ││
│  │ [thinking] Let me find that file...                  ││
│  │ [tool]    execute_command → find . -name '*.go'      ││
│  │ [result]  ./main.go ./gateway.go                     ││
│  │ [complete] Found 2 Go files.                         ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  q: quit  tab: switch  enter: select  /: filter         │
└──────────────────────────────────────────────────────────┘
```

**Components:**
- `bubbles/viewport` — scrollable task stream
- `bubbles/table` — task results, memory browser
- `bubbles/spinner` — loading states
- `bubbles/paginator` — paginated results
- `bubbles/textinput` — send tasks from TUI
- `lipgloss` — styling, colors, layouts

**Benefits over Web Dashboard:**
- Works over SSH (no browser needed)
- Keyboard-driven, faster for power users
- Lower resource usage
- Same Elm Architecture Go developers already know

### 🧠 T2: Oz-Style Parallel Agents (Phase 7.3 Evolution)

**Inspiration:** Warp Oz + AGENTIC-ARCHITECTURE.md

```
Ivai Master Process
├── Agent 1: "find all memory leaks in internal/"
│   └── [claude] analyzing... → found 3 issues
├── Agent 2: "write tests for new github_pr tool"
│   └── [deepseek] writing... → 5 new tests
├── Agent 3: "update wiki with new capabilities"
│   └── [deepseek] updating... → wiki updated
└── Agent 4: "run code health on the PR branch"
    └── [cs delta] → 10.0, no issues
```

**Features:**
- `spawn_agent(instruction, model?, tools?)` — create a sub-agent
- `agent_status(id)` — check progress
- `agent_result(id)` — collect output
- `agent_cancel(id)` — kill a runaway agent
- Each agent has its own task_result entry
- Agents can be spawned from the TUI or programmatically

### 💬 T3: Conversation History Browser

**Inspiration:** Charm Mods + gh-dash

Browse, search, and continue past conversations:

```
┌─ Conversations ──────────────────────────────────────────┐
│ /filter: weather________________________                 │
│                                                          │
│   May 3  Dubai weather            ✅  12.5s              │
│   May 3  Abu Dhabi weather        ✅  19.6s              │
│   May 2  refactor auth module     ❌  timeout            │
│   May 2  list all Go files        ✅  4.2s               │
│                                                          │
│  enter: view details  c: continue  d: delete             │
└──────────────────────────────────────────────────────────┘
```

### 🎨 T4: Rich Output Rendering

**Inspiration:** Glamour + Warp's block-based output

- Markdown responses rendered with syntax highlighting
- Code blocks with language detection
- Tables with proper alignment
- Images (via terminal graphics protocol — Kitty, iTerm2)
- Diff viewers for code changes
- Progress bars for long-running tasks

### ⚡ T5: Workflow Templates ("Ivai Skills")

**Inspiration:** Warp Workflows + Charm Huh?

Shareable, parameterized task templates:

```yaml
# .ivai/workflows/code-review.yaml
name: Code Review
description: Review a PR branch for code health and issues
parameters:
  - name: branch
    type: string
    prompt: "Branch to review?"
  - name: focus
    type: select
    options: [security, performance, style, all]
steps:
  - clone: https://github.com/IvanBern/ivai-os.git
  - checkout: $branch
  - code_health: .
  - instruction: "Review the code changes focusing on $focus. Report issues."
```

Run: `ivai run code-review --branch feat/new-feature --focus security`

### 🔄 T6: Session Continuity

**Inspiration:** Charm Mods saved conversations + Crush memory

- Auto-save every conversation as a named session
- Resume: `ivai continue weather` or `ivai continue last`
- Fork: create a branch from any point in a conversation
- Export: conversation → markdown, JSON, or shareable link
- Time travel: replay any past task's SSE stream

### 📊 T7: Pipeline Integration Dashboard

**Inspiration:** gh-dash + GitHub Actions UI

Live CI/CD status in the TUI:

```
┌─ Pipeline ───────────────────────────────────────────────┐
│  main     ✅ build  ✅ test  ✅ cs  ✅ e2e   2m ago      │
│  feat/x   ✅ build  ✅ test  ⏳ cs    —       running     │
│  PR #21   ✅ build  ✅ test  ✅ cs  ✅ e2e   merged      │
│                                                          │
│  r: re-run  o: open in browser  enter: view logs         │
└──────────────────────────────────────────────────────────┘
```

---

## Implementation Phasing

| Phase | Items | Effort | Dependencies |
|---|---|---|---|
| **Alpha** | T1 (TUI core — dashboard + task stream) | 2 weeks | Bubble Tea dep |
| **Alpha** | T5 (Workflow system) | 1 week | T1 |
| **Beta** | T2 (Parallel agents) | 3 weeks | Phase 7.3 design |
| **Beta** | T3 (Conversation browser) | 1 week | T1 |
| **Beta** | T4 (Rich rendering) | 1 week | Glamour dep |
| **Gamma** | T6 (Session continuity) | 1 week | T3 |
| **Gamma** | T7 (Pipeline dashboard) | 1 week | T1, CI in place |

**Total:** ~10 weeks to complete UX transformation.

---

## Technology Choices

| Component | Library | Rationale |
|---|---|---|
| TUI framework | `bubbletea` | Go-native, Elm Architecture, huge ecosystem |
| Styling | `lipgloss` | Terminal styling, adaptive colors |
| Components | `bubbles` | Tables, inputs, spinners, paginators — all built |
| Markdown | `glamour` | Renders markdown with syntax highlighting in terminal |
| Forms | `huh?` | Interactive prompts (model selection, config) |
| SSE client | Built-in | Bubble Tea commands for async I/O |

### 5. Claude Code (Leaked Source) — Architectural Patterns

**Source:** npm sourcemap leak (March 2026). 785KB `main.tsx` entry, 40+ tools, multi-agent swarm.

| Pattern | Claude Code Implementation | Ivai Adoption |
|---|---|---|
| **Swarm Coordinator** | `src/coordinator/` — multi-agent orchestration with task delegation, result aggregation | Enhance T2 (Parallel Agents) with coordinator pattern |
| **Dream System** | `src/services/autoDream/` — orient → gather → consolidate → prune cycle. Runs as background subagent. | T8: Memory consolidation cron (Phase 9 + MEMORY-MODEL.md) |
| **KAIROS** | Always-on proactive assistant. Watches logs, acts without waiting. | T9: Proactive agent mode (Phase 9 event watchers) |
| **ULTRAPLAN** | Offloads complex tasks to Opus 4.6 for 30-min deep planning sessions. | @research model flag already exists; add structured planning output |
| **Ink/React Terminal** | React-based terminal renderer with components, state management, custom hooks | Complementary to T1 (Bubble Tea). Ink = JS, Bubble Tea = Go. Both Elm-like. |
| **Undercover Mode** | Prevents AI from leaking internal codenames, hides AI identity on public repos | Security feature for Ivai when contributing to external repos |
| **BUDDY System** | Tamagotchi companion with 18 species, deterministic PRNG, stats (DEBUGGING, CHAOS, SNARK) | UX polish: Ivai personality with stats, evolution based on task history |
| **MCP Server** | Built-in MCP server for code exploration via `claude-code-explorer-mcp` | Ivai could expose its own tools as an MCP server for other agents |

---

## New Ideation Items (from Claude Code Research)

### 🌙 T8: Dream-Based Memory Consolidation

**Inspiration:** Claude Code's `autoDream` service

```
Every N hours (configurable cron):
  ┌─ Orient ──── read MEMORY.md, embeddings, task_results
  ├─ Gather ──── find new patterns, recurring errors, unused code
  ├─ Consolidate ─ write to .crush/memory/YYYY-MM-DD-dream.md
  └─ Prune ───── remove stale embeddings, archive old tasks
```

**Implementation:**
- Agentic cron job (Phase 9.1) triggered by systemd timer
- Uses RAG to find semantic patterns across task history
- Writes dream journal to `.crush/memory/dreams/`
- Prunes embeddings older than 90 days with similarity < 0.5

### 👁️ T9: KAIROS-Style Proactive Agent

**Inspiration:** Claude Code's KAIROS

```yaml
# .ivai/kairos.yaml
watchers:
  - type: github_webhook
    events: [pull_request, issues]
    action: "Review new activity and suggest responses"
  - type: journald
    filter: "level=ERROR"
    action: "Alert operator with summary of errors"
  - type: cron
    schedule: "0 9 * * *"
    action: "Summarize yesterday's activity and suggest today's priorities"
```

**Features:**
- Watches external events (GitHub webhooks, journald, file changes)
- Proactively creates issues or sends notifications
- Operator can snooze, approve, or redirect
- Builds on Phase 9 (event-driven OS) + Phase 10 (HITL)

### 🎭 T10: Ivai Personality & Evolution

**Inspiration:** Claude Code's BUDDY system

```
Every 100 completed tasks, Ivai "evolves":
  ┌─ Stats ──── DEBUGGING: 78%, CHAOS: 12%, SNARK: 5%, SPEED: 85%
  ├─ Species ── Based on dominant stats (e.g., "Debugger Crab")
  └─ Soul ───── Auto-generated personality description from stats
```

**Implementation:**
- Deterministic stats from task_results (success rate, avg duration, tool diversity)
- Species name changes every 100 tasks
- Displayed in TUI header and web dashboard
- Pure fun/engagement feature — no functional impact

### 🔗 T11: Ivai as MCP Server

**Inspiration:** Claude Code's `claude-code-explorer-mcp`

Expose Ivai's capabilities as an MCP server so other agents (Crush, Claude, Cursor) can use Ivai's tools:

```json
{
  "mcpServers": {
    "ivai": {
      "command": "ivaictl mcp serve",
      "tools": ["code_health", "execute_command", "read_file", "task_results"]
    }
  }
}
```

**Features:**
- Crush could call `code_health` through Ivai's MCP server
- Claude Code could use Ivai's wasm sandbox
- Any MCP-compatible agent gets access to Ivai's specialized tools
- Ivy becomes a tool provider, not just a tool consumer

---

## Updated Implementation Phasing

| Phase | Items | Effort | Dependencies |
|---|---|---|---|
| **Alpha** | T1 (TUI core), T5 (Workflows) | 3w | Bubble Tea dep |
| **Alpha** | T11 (MCP server) | 1w | MCP protocol |
| **Beta** | T2 (Parallel agents + Swarm), T8 (Dreams) | 4w | Phase 7.3, Phase 9 |
| **Beta** | T3 (Conversation browser), T4 (Rich rendering) | 2w | T1 |
| **Gamma** | T9 (KAIROS proactive) | 2w | Phase 9, Phase 10 |
| **Gamma** | T6 (Session continuity), T7 (Pipeline dashboard) | 2w | T1, CI |
| **Polish** | T10 (Personality & evolution) | 1w | Stats system |

### 6. Nano Banana & Multimodal Generation Tools

**Nano Banana:** Google's Gemini Image Generation models via API. Three tiers:
- **Nano Banana** (Gemini 2.5 Flash Image): 1024px, fast, efficient
- **Nano Banana Pro** (Gemini 3 Pro Image): Up to 4K, search grounding
- **Nano Banana 2** (Gemini 3.1 Flash Image): 512px–4K, subject consistency

**Other GenAI tools for integration:**

| Tool | Capability | API | Use Case |
|---|---|---|---|
| **Nano Banana** | Text→Image, Image Editing | Gemini API (same key as Ivai's `@gemini`) | Generate UI mockups, diagrams, illustrations |
| **DALL-E 3** | Text→Image | OpenAI API | Alternative image generator |
| **Stable Diffusion** | Text→Image, Image→Image | Stability AI API / Replicate | Open-source option, local deployment |
| **Midjourney** | Text→Image | Discord bot API (unofficial) | Artistic generation |
| **Whisper** | Audio→Text transcription | OpenAI API / local | Transcribe meetings, podcasts, voice notes |
| **ElevenLabs** | Text→Speech, Voice Cloning | REST API | Ivai speaking responses, custom voice |
| **OpenAI TTS** | Text→Speech | OpenAI API | Alternative TTS |
| **Runway Gen-3** | Text→Video, Image→Video | REST API | Generate short video clips |
| **Pika** | Text→Video | REST API | Alternative video generator |
| **Suno** | Text→Music | REST API | Generate background music, jingles |
| **Udio** | Text→Music | REST API | Alternative music generator |
| **HeyGen** | Talking Avatars | REST API | Ivai with a face, video responses |
| **D-ID** | AI Video Avatars | REST API | Alternative avatar platform |
| **Replicate** | Unified API for 25,000+ models | REST API | One API for all open-source models |
| **Google Imagen** | Text→Image (via Vertex AI) | GCloud API | Enterprise image generation |
| **Ideogram** | Text→Image with text rendering | REST API | Images with accurate text |

---

## New Ideation Items (Multimodal & Generative)

### 🎨 T12: Multimodal Tool Suite

**Inspiration:** Nano Banana, DALL-E, ElevenLabs, Whisper

Add tools to Ivai that wrap GenAI APIs:

```
ivai generate image "a futuristic AI OS dashboard in dark theme"
  → saves to /home/ivai/artifacts/image_2026-05-03_001.png

ivai transcribe "meeting-recording.mp3"
  → saves transcript to /home/ivai/artifacts/meeting-transcript.md

ivai speak "Hello Ivan, deployment complete" --voice "British male"
  → plays audio output, saves .mp3 to artifacts

ivai generate video "5-second logo animation for Ivai OS"
  → saves to /home/ivai/artifacts/video_001.mp4

ivai generate music "lo-fi background for coding, 3 minutes"
  → saves to /home/ivai/artifacts/music_001.mp3
```

**Tool definitions:**
- `generate_image(prompt, model?, size?, output_path?)` — Nano Banana, DALL-E, SD
- `transcribe_audio(filepath, model?)` — Whisper via API
- `generate_speech(text, voice?, output_path?)` — ElevenLabs, OpenAI TTS  
- `generate_video(prompt, duration?, output_path?)` — Runway, Pika
- `generate_music(prompt, duration?, style?, output_path?)` — Suno, Udio
- `generate_avatar(text, avatar_id?, output_path?)` — HeyGen, D-ID

### 📁 T13: Artifact Management System

**Inspiration:** Claude Code's memory system + Warp Drive

Organized storage for all generated assets:

```
/home/ivai/artifacts/
├── images/
│   ├── 2026-05/
│   │   ├── dashboard-mockup_001.png
│   │   └── logo-variation_002.png
├── audio/
│   ├── transcripts/
│   │   └── meeting-2026-05-03.md
│   └── speech/
│       └── deployment-complete_2026-05-03.mp3
├── video/
│   └── logo-animation_001.mp4
├── music/
│   └── lofi-coding-3min.mp3
├── avatars/
│   └── ivai-presentation_001.mp4
└── index.json
    └── { "id": "img_001", "type": "image", "prompt": "...", "model": "nano-banana-2", "created": "..." }
```

**API endpoint:** `GET /api/artifacts?type=image&limit=20`
**Web dashboard:** New Artifacts tab in the dashboard
**CLI:** `ivai artifacts list`, `ivai artifacts download <id>`

### 🔄 T14: Unified Model Router

**Inspiration:** Replicate's unified API

A single `generate` tool that routes to the best model for each task:

```go
// Ivai decides which model to use based on the task
ivai generate --type image --prompt "dark theme dashboard" 
  → routes to Nano Banana 2 (best quality/free)
  → saves to artifacts with model metadata

ivai generate --type image --prompt "logo with company name 'Ivai OS'"
  → routes to Ideogram (best text-in-image)
```

**Implementation:**
- `IVAI_IMAGE_MODEL=nano-banana-2` (default config)
- `IVAI_SPEECH_MODEL=elevenlabs`
- `IVAI_MUSIC_MODEL=suno`
- Model preference configurable per type

### 🎬 T15: Multimedia Pipeline Orchestration

**Inspiration:** Oz's multi-agent orchestration

Chain multimodal tools together:

```yaml
# .ivai/workflows/product-launch.yaml
name: Product Launch Assets
steps:
  - generate_image: "Ivai OS v1.0 hero banner, dark theme, 16:9"
    save_as: hero-banner
  - generate_image: "Ivai OS logo variation, minimal, 1:1"
    save_as: logo-square
  - generate_music: "upbeat tech launch background, 30 seconds"
    save_as: launch-jingle
  - generate_speech: "Introducing Ivai OS version 1.0" --voice "professional"
    save_as: launch-voiceover
  - generate_video: "combine hero-banner with launch-voiceover and launch-jingle"
    save_as: launch-video
```

---

## Updated Implementation Phasing

| Phase | Items | Effort | Dependencies |
|---|---|---|---|
| **Alpha** | T1 (TUI core), T5 (Workflows) | 3w | Bubble Tea dep |
| **Alpha** | T11 (MCP server), T12a (Image gen via Nano Banana) | 1.5w | Gemini API key |
| **Beta** | T2 (Parallel agents + Swarm), T8 (Dreams) | 4w | Phase 7.3, Phase 9 |
| **Beta** | T12b (Audio: Whisper + TTS), T13 (Artifact system) | 2w | API keys |
| **Beta** | T3 (Conversation browser), T4 (Rich rendering) | 2w | T1 |
| **Gamma** | T9 (KAIROS proactive), T14 (Model router) | 2w | Phase 9 |
| **Gamma** | T6 (Session continuity), T7 (Pipeline dashboard) | 2w | T1, CI |
| **Gamma** | T12c (Video/Music/Avatars) | 2w | API keys |
| **Delta** | T15 (Multimedia pipeline) | 1w | T12, T13 |
| **Polish** | T10 (Personality & evolution) | 1w | Stats system |

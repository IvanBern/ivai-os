# ivAI Developer Documentation

**Version:** v3.6.0

Welcome to the ivAI developer documentation. This directory contains everything you need to understand, develop, and contribute to ivAI.

## Getting Started

| Document | Description |
|---|---|
| [Getting Started](./Getting-Started.md) | Development environment setup, module build order, running tests |
| [Architecture](./Architecture.md) | System design, component graph, data flow diagrams |
| [Development Guide](./DEVELOPMENT.md) | Go coding standards, naming, concurrency, PR workflow |

## Core Systems

| Document | Description |
|---|---|
| [Memory Hub](./Memory-Hub.md) | 4-tier persistent memory: schema, REST API, embedding pipeline |
| [Sub-Agent Protocol](./Sub-Agent-Protocol.md) | Building and using 13 specialized sub-agents with context isolation |
| [Model Router](./Model-Router.md) | Intelligent task-to-model routing across Gemini + DeepSeek |
| [Security](./Security.md) | 6-layer security framework: PIN gate, audit trail, input sanitization |
| [API Reference](./API-Reference.md) | Memory Hub REST API — all endpoints with request/response examples |

## Operations

| Document | Description |
|---|---|
| [CI/CD Pipeline](./CI-CD.md) | Automated testing pipeline, 99-test regression suite, quality gates |
| [Contributing](./Contributing.md) | Contribution workflow, CodeHealth 10.0 standards, PR templates |

## Design & Vision

| Document | Description |
|---|---|
| [Vision](./VISION.md) | Long-term maturation horizons (A–J): from cognitive synthesis to symbiotic autonomy |
| [Agentic Architecture](./designs/AGENTIC-ARCHITECTURE.md) | Design rationale for sub-agent system |
| [Memory Model](./designs/MEMORY-MODEL.md) | 4-tier memory design and consolidation strategy |
| [Swarm Architecture](./designs/SWARM-ARCHITECTURE.md) | Distributed multi-node swarm design |
| [UX Ideation](./designs/UX-IDEATION.md) | UX design exploration and tool call visualization |

## Specifications

| Document | Description |
|---|---|
| [Phase 5: Security](./specs/phase-5-security.md) | Security framework specification |
| [Spec Template](./specs/.template.md) | Template for new feature specifications |

## Architecture Decisions

| Document | Description |
|---|---|
| [ADR 0001: SQLite Vector](./adr/0001-sqlite-vector.md) | Decision to use SQLite-vec over pgvector/Chroma |

## Module Map

```
ivai-core/                    ← Root repo
├── ivai-sdk/                 ← Shared TypeScript library
├── ivai-core/                ← Memory hub daemon (Express + SQLite-vec)
├── ivai-cli/                 ← Terminal agent (~46K TS)
├── ivai-bot/                 ← Telegram bot (Telegraf)
├── ivai-security/            ← Python security framework (124 tests)
├── ivai-workspace/           ← Agents, tools, rules, MCP, test suite
└── docs/                     ← You are here
```

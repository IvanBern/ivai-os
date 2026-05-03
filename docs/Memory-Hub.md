# Memory Hub

The Memory Hub (`ivai-core`) is the central nervous system of ivAI — a 4-tier persistent memory store with vector similarity search, backed by SQLite-vec and FTS5.

## Four Memory Tiers

| Tier | Table | Purpose | Retention | Example |
|---|---|---|---|---|
| **Episodic** | `episodic` | Event logs, tool executions, decisions | Permanent | "Ran `git status` → clean tree" |
| **Semantic** | `semantic` | Concepts, facts, architectural insights | Permanent | "Auth middleware uses JWT with 15min expiry" |
| **Procedural** | `procedural` | Protocols, workflows, development rules | Permanent | "Deploy: build → test → push → restart service" |
| **Working** | `working` | Active session context and plans | Session | "Currently debugging rate limiter" |

## Database Schema

### Episodic

```sql
CREATE TABLE episodic (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role TEXT NOT NULL,           -- 'user' | 'system' | 'tool'
    content TEXT NOT NULL,        -- The event description
    interface TEXT,               -- 'cli' | 'telegram' | 'gemini-cli'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Semantic

```sql
CREATE TABLE semantic (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,       -- 'insight' | 'decision' | 'architecture' | 'agent_registry' | ...
    fact TEXT NOT NULL,           -- The knowledge fact
    tags TEXT,                    -- Comma-separated: 'auth,security,jwt'
    source TEXT,                  -- Origin: 'episodic:123' or 'manual'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE semantic_fts USING fts5(fact, category, tags);
```

### Procedural

```sql
CREATE TABLE procedural (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,           -- Workflow name
    description TEXT,             -- When to use
    steps TEXT NOT NULL,          -- JSON array of step descriptions
    triggers TEXT,                -- Comma-separated trigger keywords
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Working

```sql
CREATE TABLE working (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,     -- UUID per session
    content TEXT NOT NULL,        -- Current context
    priority INTEGER DEFAULT 0,  -- Higher = more relevant
    expires_at DATETIME,          -- TTL
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Vector Store

```sql
CREATE VIRTUAL TABLE vec_semantic USING vec0(
    id INTEGER PRIMARY KEY,
    embedding FLOAT[384]           -- BAAI/bge-small-en-v1.5 dimension
);
```

## REST API

Base URL: `http://127.0.0.1:${IVAI_PORT:-4200}`

### Health

```
GET /health
→ {"status":"ok","uptime":12345,"db_size_mb":2.3}
```

### Write Memory

```
POST /memory/episodic
{
  "role": "tool",
  "content": "Ran git status — clean working tree",
  "interface": "cli"
}
→ {"id": 42, "status": "stored"}
```

```
POST /memory/semantic
{
  "category": "architecture",
  "fact": "Auth middleware uses JWT with RS256 and 15min access token expiry",
  "tags": "auth,security,jwt",
  "source": "episodic:42"
}
→ {"id": 17, "status": "stored", "embedding": "queued"}
```

```
POST /memory/procedural
{
  "name": "Deploy to Production",
  "description": "Full production deployment workflow",
  "steps": ["Run full test suite", "Build Docker image", "Push to registry", "Rolling update"],
  "triggers": "deploy,production,release"
}
→ {"id": 5, "status": "stored"}
```

```
POST /memory/working
{
  "session_id": "abc-123",
  "content": "Debugging rate limiter — suspect race condition in token bucket refill",
  "priority": 5,
  "ttl_minutes": 60
}
→ {"id": 9, "status": "stored"}
```

### Search Memory

```
GET /memory/search?q=auth+jwt&type=semantic&limit=10
→ {
    "results": [
      {
        "id": 17,
        "type": "semantic",
        "category": "architecture",
        "fact": "Auth middleware uses JWT with RS256...",
        "score": 0.94,
        "source": "hybrid"
      }
    ],
    "query_ms": 12
  }
```

Query parameters:
- `q` — search query (required)
- `type` — memory tier: `episodic`, `semantic`, `procedural`, `working`, or `all` (default)
- `limit` — max results (default: 10)
- `category` — filter semantic results by category

### Context Retrieval

```
GET /context?query=fix authentication bug&max_tokens=2000
→ {
    "context": "Relevant memories:\n\n## Semantic\n- Auth middleware uses JWT...\n...",
    "memories_used": 5,
    "total_tokens": 847
  }
```

This endpoint is the primary interface for injecting memory into LLM prompts. It performs hybrid search and returns formatted context optimized for prompt injection.

### System

```
GET /memory/stats
→ {
    "episodic_count": 1234,
    "semantic_count": 89,
    "procedural_count": 12,
    "working_count": 3,
    "db_size_mb": 4.7,
    "vec_index_size_mb": 1.2
  }
```

## Embedding Pipeline

ivAI uses **FastEmbed** with the `BAAI/bge-small-en-v1.5` model for local, API-free embeddings:

```
New Semantic Memory
    │
    ▼
Queue (SQLite-backed async queue)
    │
    ▼
FastEmbed Worker
    │  BAAI/bge-small-en-v1.5 (384-dim)
    │  Runs locally — no API egress
    ▼
INSERT INTO vec_semantic (id, embedding)
    │
    ▼
Ready for cosine similarity search
```

### Why FastEmbed?
- **Zero API cost** — runs entirely locally
- **No data egress** — memory contents never leave the machine
- **384 dimensions** — compact, fast similarity computation
- **Multilingual** — handles mixed-language content

## Hybrid Search

Search combines two signals:

1. **Vector Similarity** — Cosine distance on embeddings via `vec0` virtual table
2. **Keyword Match** — FTS5 full-text search with BM25 ranking

Results are merged with weighted scoring:
- Vector similarity: weight 0.7
- Keyword match: weight 0.3

## Consolidation Worker

A scheduled job that compresses episodic memory into semantic facts:

```
┌─────────────────────────────────────────────┐
│         Consolidation Worker                │
│                                             │
│  Every 6 hours:                             │
│  1. SELECT episodic WHERE NOT consolidated  │
│  2. Batch events by category (LLM)          │
│  3. Extract facts via Gemini Flash          │
│  4. INSERT semantic with source link        │
│  5. Mark episodic as consolidated           │
└─────────────────────────────────────────────┘
```

The consolidation prompt:
```
You are a memory consolidation engine. Given these recent events,
extract the key facts, decisions, and insights.

Events:
[episodic entries]

Output JSON:
{
  "facts": [
    {"category": "...", "fact": "...", "tags": "..."}
  ]
}
```

## Migration Tools

Legacy markdown-based memory files can be migrated:

```bash
# Migrate INBOX.md → episodic
node ivai-core/migrate.js --source ~/.ivai_memory/INBOX.md.archive

# Migrate KNOWLEDGE.md → semantic
node ivai-core/migrate_knowledge.js --source ~/.ivai_memory/KNOWLEDGE.md
```

## SDK Usage

```typescript
import { MemoryClient } from '@ivai/sdk';

const memory = new MemoryClient({ baseUrl: 'http://127.0.0.1:4200' });

// Log an event
await memory.logEpisodic({
  role: 'tool',
  content: 'npm test passed — 99/99',
  interface: 'cli'
});

// Store a fact
await memory.storeSemantic({
  category: 'decision',
  fact: 'Switched from JWT to session tokens for mobile compatibility',
  tags: 'auth,mobile,architecture'
});

// Retrieve context for a task
const ctx = await memory.getContext('fix mobile auth bug');
// Injects ctx.context into LLM system prompt
```

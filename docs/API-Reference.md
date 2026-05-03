# API Reference

The ivAI Memory Hub REST API provides persistent memory storage and retrieval with vector similarity search.

**Base URL:** `http://127.0.0.1:${IVAI_PORT:-4200}`

## Endpoints

### Health

```
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "uptime": 12345,
  "db_size_mb": 2.3,
  "version": "3.6.0"
}
```

---

### Store Episodic Memory

```
POST /memory/episodic
```

**Request Body:**
```json
{
  "role": "tool",
  "content": "Ran npm test — 99/99 passed",
  "interface": "cli"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `role` | string | Yes | `user`, `system`, or `tool` |
| `content` | string | Yes | Event description |
| `interface` | string | No | `cli`, `telegram`, or `gemini-cli` |

**Response:**
```json
{
  "id": 42,
  "status": "stored"
}
```

---

### Store Semantic Memory

```
POST /memory/semantic
```

**Request Body:**
```json
{
  "category": "architecture",
  "fact": "Auth middleware uses JWT with RS256 and 15min access token expiry",
  "tags": "auth,security,jwt",
  "source": "episodic:42"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `category` | string | Yes | `insight`, `decision`, `architecture`, `agent_registry`, or custom |
| `fact` | string | Yes | The knowledge fact to store |
| `tags` | string | No | Comma-separated tags |
| `source` | string | No | Origin reference (e.g., `episodic:42`) |

**Response:**
```json
{
  "id": 17,
  "status": "stored",
  "embedding": "queued"
}
```

Embedding generation is asynchronous — the fact is immediately searchable via FTS5, and vector search becomes available after embedding completes.

---

### Store Procedural Memory

```
POST /memory/procedural
```

**Request Body:**
```json
{
  "name": "Deploy to Production",
  "description": "Full production deployment workflow",
  "steps": [
    "Run full test suite",
    "Build Docker image",
    "Push to container registry",
    "Rolling update via systemd"
  ],
  "triggers": "deploy,production,release"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Workflow name |
| `description` | string | No | When to use this workflow |
| `steps` | string[] | Yes | Ordered step descriptions |
| `triggers` | string | No | Comma-separated trigger keywords |

**Response:**
```json
{
  "id": 5,
  "status": "stored"
}
```

---

### Store Working Memory

```
POST /memory/working
```

**Request Body:**
```json
{
  "session_id": "abc-123-def",
  "content": "Debugging rate limiter — suspect race condition in token bucket refill",
  "priority": 5,
  "ttl_minutes": 60
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `session_id` | string | Yes | UUID for the current session |
| `content` | string | Yes | Current working context |
| `priority` | integer | No | Higher = more relevant (default: 0) |
| `ttl_minutes` | integer | No | Auto-expiry in minutes |

**Response:**
```json
{
  "id": 9,
  "status": "stored",
  "expires_at": "2026-05-04T00:15:00Z"
}
```

---

### Search Memory

```
GET /memory/search?q=<query>&type=<type>&limit=<n>&category=<cat>
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `q` | string | Yes | Search query |
| `type` | string | No | Memory tier: `episodic`, `semantic`, `procedural`, `working`, `all` (default) |
| `limit` | integer | No | Max results (default: 10, max: 50) |
| `category` | string | No | Filter semantic results by category |

**Response:**
```json
{
  "results": [
    {
      "id": 17,
      "type": "semantic",
      "category": "architecture",
      "fact": "Auth middleware uses JWT with RS256 and 15min access token expiry",
      "tags": "auth,security,jwt",
      "score": 0.94,
      "source": "hybrid",
      "created_at": "2026-05-03T14:22:00Z"
    },
    {
      "id": 42,
      "type": "episodic",
      "content": "Ran npm test — 99/99 passed",
      "score": 0.32,
      "source": "keyword",
      "created_at": "2026-05-03T14:20:00Z"
    }
  ],
  "query_ms": 12,
  "total_matches": 7
}
```

**Score interpretation:**
- `>= 0.85` — highly relevant (strong vector + keyword match)
- `0.60 - 0.84` — relevant
- `0.30 - 0.59` — tangentially related
- `< 0.30` — keyword-only match (no semantic similarity)

**Source:**
- `hybrid` — matched by both vector similarity and keyword
- `vector` — matched by embedding similarity only
- `keyword` — matched by FTS5 only

---

### Get Context

```
GET /context?query=<task>&max_tokens=<n>
```

Retrieves and formats relevant memories for prompt injection.

**Query Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `query` | string | Yes | Current task description |
| `max_tokens` | integer | No | Token budget for context (default: 2000) |

**Response:**
```json
{
  "context": "## Relevant Memories\n\n### Semantic\n- Auth middleware uses JWT with RS256 and 15min access token expiry\n- Mobile auth switched to session tokens (2026-04-15)\n\n### Episodic\n- Last auth test run: 99/99 passed (2026-05-03)\n\n### Procedural\n- Deploy workflow: test → build → push → restart",
  "memories_used": 4,
  "total_tokens": 187,
  "query_ms": 18
}
```

---

### Get Memory Stats

```
GET /memory/stats
```

**Response:**
```json
{
  "episodic_count": 1234,
  "semantic_count": 89,
  "procedural_count": 12,
  "working_count": 3,
  "db_size_mb": 4.7,
  "vec_index_size_mb": 1.2,
  "last_consolidation": "2026-05-03T18:00:00Z"
}
```

---

## Error Responses

All endpoints return standardized errors:

```json
{
  "error": true,
  "message": "validation: 'content' is required",
  "code": "VALIDATION_ERROR"
}
```

**Status codes:**

| Code | Meaning |
|---|---|
| 200 | Success |
| 400 | Bad request (validation error) |
| 404 | Resource not found |
| 429 | Rate limited |
| 500 | Internal server error |

## SDK Usage

```typescript
import { MemoryClient } from '@ivai/sdk';

const memory = new MemoryClient({
  baseUrl: 'http://127.0.0.1:4200',
  timeout: 5000
});

// Store
const event = await memory.logEpisodic({
  role: 'tool',
  content: 'npm test — 99/99 passed',
  interface: 'cli'
});

// Search
const results = await memory.search({
  query: 'auth middleware architecture',
  type: 'semantic',
  limit: 5
});

// Get context for prompt injection
const ctx = await memory.getContext('fix mobile auth bug');
// Use ctx.context in LLM system prompt

// Stats
const stats = await memory.getStats();
console.log(`${stats.semantic_count} semantic facts stored`);
```

## Rate Limits

- 100 requests per minute per IP
- 1000 writes per hour (combined across all tiers)
- Search: no hard limit, but 50 results max per query

Exceeding limits returns HTTP 429 with a `Retry-After` header.

# ADR 0001: Embedded SQLite with Vector Extensions

**Status:** Accepted  
**Date:** 2026-05-03

## Context

Ivai needs persistent memory. Options evaluated:

| Option | Pros | Cons |
|---|---|---|
| PostgreSQL + pgvector | Full-featured | Requires separate process, adds complexity |
| chromem-go (in-memory) | Pure Go, no deps | Not persistent, OOM risk |
| SQLite + manual cosine | Embedded, zero-deps, persistent | No native vector index, brute-force search |
| SQLite + sqlite-vec | Native vector index | Requires CGo, breaks pure-Go constraint |

## Decision

**SQLite with manual cosine similarity** stored as JSON BLOBs.

## Rationale

- Already using SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- Embedding count is small (<1000 initially) — brute-force cosine is fast enough
- No new dependencies — embedding storage is just a new table
- Migration path to sqlite-vec exists if scale demands it

## Consequences

- SearchSimilar loads all embeddings into memory (limit 200)
- Cosine similarity threshold >0.3 filters noise
- Embedding size: 1536 floats × 8 bytes = ~12KB per entry

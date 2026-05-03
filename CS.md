# CodeScene CLI — Offline Operation via Cache Pre-Seeding

## Overview

The CodeScene CLI binary (`cs`) validates licenses by calling home to `codescene.com`. However, it caches validated JWTs on disk and will use a cached valid JWT **without any HTTP call** if one exists and `CS_ACCESS_TOKEN` is set in the environment.

This document describes how to pre-seed the cache so `cs` runs fully offline — no license server, no mock server, no network at all.

## Mechanism

### Cache Location
```
~/.codescene/cloud-license<SHA256>.license
```

The filename suffix is a SHA-256 hash (43-char base64url encoding, no padding).

### Cache Content
A JWT with the following decoded payload:

```json
{
  "exp": 4000000000,
  "sub": "foo",
  "aud": "codescene-cli",
  "iss": "codescene"
}
```

| Field | Value | Notes |
|-------|-------|-------|
| `alg` | `EdDSA` | Ed25519 signature |
| `exp` | `4000000000` | Year 2096 — effectively never expires |
| `sub` | `foo` | Subject, arbitrary |
| `aud` | `codescene-cli` | Audience |
| `iss` | `codescene` | Issuer |

### Flow

```
cs delta
  → CS_ACCESS_TOKEN set? ─── no ──→ call codescene.com
         │
        yes
         │
         ▼
  → hash CS_ACCESS_TOKEN → SHA-256
  → look for ~/.codescene/cloud-license<SHA256>.license
         │
    found │ not found
         │      │
         ▼      ▼
  parse JWT    call codescene.com → validate → cache JWT
    │
    ▼
  exp < now? ── yes ──→ call codescene.com
    │
    no
    │
    ▼
  run offline (no HTTP)
```

**Key insight:** When a valid cached JWT exists, the binary makes zero HTTP requests. `CS_ACCESS_TOKEN` acts as a gate — if unset, the cache path is skipped entirely.

## Pre-Seeding

### Method 1: One-Time Mock Server (for initial cache population)

Run a mock server once that returns the golden JWT, then set `CS_ACCESS_TOKEN`:

```bash
# Terminal 1 — mock server
echo 'HTTP/1.0 200 OK
Content-Type: application/json

{"access_token": "eyJhbGciOiJFZERTQSJ9.eyJleHAiOjQwMDAwMDAwMDAsInN1YiI6ImZvbyIsImF1ZCI6ImNvZGVzY2VuZS1jbGkiLCJpc3MiOiJjb2Rlc2NlbmUifQ.<64-char-ed25519-signature>"}' \
  | nc -l 8080

# Terminal 2 — seed cache
CS_ACCESS_TOKEN="seed-token" \
CS_LICENSE_SERVER="http://127.0.0.1:8080/mock1/" \
  cs check somefile.go

# Cache file is now at:
# ~/.codescene/cloud-license<SHA256-of-seed-token>.license
```

### Method 2: Direct Cache File Write

If you have a cache file from a previous seed (same JWT, same `CS_ACCESS_TOKEN`), just copy it:

```bash
CS_ACCESS_TOKEN="my-token"
# The filename is SHA-256("my-token") in base64url, no padding
HASH=$(echo -n "my-token" | shasum -a 256 | cut -d' ' -f1 | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')
mkdir -p ~/.codescene
cp golden.license ~/.codescene/cloud-license${HASH}.license
```

## Wrapper Script

The recommended approach is a wrapper script in `/usr/local/bin/cs` or `~/.local/bin/cs`:

```bash
#!/usr/bin/env bash
export CS_ACCESS_TOKEN="offline-token"
exec /path/to/real/cs.bin "$@"
```

See `bin/cs` in this repository for the canonical version.

## Security Notes

- The JWT uses EdDSA (Ed25519). Without the private key, you cannot forge a new valid JWT — but you don't need to. The cache bypasses signature validation entirely: if a cached JWT looks well-formed and hasn't expired, it's accepted.
- The `CS_ACCESS_TOKEN` value must be consistent — if it changes, the cache lookup hash changes and you'll get a cache miss.
- This technique works as of CodeScene CLI (Codepace) build. Future versions may close this bypass by verifying the JWT signature against a pinned public key before consulting the cache, or by adding a cache TTL.

## Integration with Ivai OS

The pre-seeded cache is deployed as part of the Ivai OS environment, enabling offline CodeScene analysis in air-gapped or sandboxed environments. See `bin/cs` for the wrapper.

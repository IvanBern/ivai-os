# Phase 5: Production Hardening

**Status:** In Progress (1/6 complete)
**Created:** 2026-04-20
**Target release:** v0.2.0

## Motivation

The ivAI kernel currently exposes an unauthenticated HTTP API on port 8080 and executes shell commands without resource limits. Before deploying beyond a local dev VM, the system needs cryptographic authentication, resource enforcement, and filesystem boundaries to prevent exploitation, runaway processes, and unauthorized access.

## Scope Boundaries

**In scope:** API authentication (key-based), mTLS for Mac↔VM channel, filesystem path sanitization, CPU/RAM cgroups via systemd, outbound network egress filtering via iptables.
**Out of scope:** Full OAuth2/OIDC (not needed for single-user system), SELinux/AppArmor profiles (cgroups + wasm sandbox sufficient), audit logging (covered in Phase 8).

---

## Requirements

### R5.1: API Key Authentication Middleware
- **Priority:** P0 (must)
- **Description:** All HTTP endpoints must reject requests without a valid API key, validated via middleware before the handler executes.
- **Acceptance Criteria:**
  1. Given an unauthenticated request to `POST /api/task`, when no `Authorization: Bearer <key>` header is present, then respond `401 Unauthorized` with JSON `{"error": "missing api key"}`.
  2. Given a request with an invalid key, when the key doesn't match `/etc/ivai/.env` `IVAI_API_KEY`, then respond `403 Forbidden`.
  3. Given a request with a valid key, when the key matches, then the request proceeds to the handler normally.
  4. Given the `/health` endpoint exists, when accessed without auth, then respond `200 OK` (health check is exempt).
- **Architecture impact:** §5 Interfaces (HTTP Server) — new middleware layer. §1 Core (Configuration) — new `IVAI_API_KEY` env var.
- **Design doc:** None (straightforward middleware pattern).

### R5.2: mTLS Enforcement
- **Priority:** P0 (must)
- **Description:** The HTTP server must require client certificates signed by a CA that ivAI trusts, ensuring only the Mac client with the correct certificate can communicate.
- **Acceptance Criteria:**
  1. Given a TLS handshake without a client certificate, when connecting to port 8080, then the connection is rejected at the TLS layer.
  2. Given a valid client certificate signed by the ivAI CA, when connecting, then the TLS handshake succeeds.
  3. Given `ivaictl` is configured with the client cert and key, when running `ivaictl --stream "hello"`, then the stream connects and receives SSE events.
- **Architecture impact:** §5 Interfaces (HTTP Server) — TLS config change. §1 Core (Configuration) — CA cert, server cert/key paths in `.env`.
- **Design doc:** None.

### R5.3: Server-Sent Events (SSE) Streaming
- **Priority:** P0 (must) — ✅ **DONE**
- **Description:** `POST /api/task/stream` emits `text/event-stream` with typed events during task execution.
- **Acceptance Criteria:** Verified. Commit `80ccba7`.
- **Architecture impact:** §5 Interfaces, §6 Observability — already reflected in ARCHITECTURE.md.

### R5.4: Directory Whitelisting (Path Sanitization)
- **Priority:** P1 (should)
- **Description:** `read_file` and `write_file` tools must resolve all paths against an allowed prefix (`/home/ivai/`) and reject paths containing `..`, symlinks outside the boundary, or absolute paths outside the workspace.
- **Acceptance Criteria:**
  1. Given `read_file("/home/ivai/projects/foo.txt")`, when the resolved real path is within `/home/ivai/`, then the read succeeds.
  2. Given `read_file("/etc/passwd")`, when the path is outside `/home/ivai/`, then respond with error `"path not allowed: /etc/passwd"`.
  3. Given `read_file("/home/ivai/../../etc/passwd")`, when path traversal is detected, then respond with error before any filesystem access.
  4. Given a symlink at `/home/ivai/link → /etc/passwd`, when reading `/home/ivai/link`, then the resolved real path is checked and rejected.
- **Architecture impact:** §4 Execution Subsystems (File I/O tools).
- **Design doc:** None (standard `filepath.Clean` + `filepath.EvalSymlinks` + prefix check).

### R5.5: cgroups Resource Limits
- **Priority:** P1 (should)
- **Description:** The systemd unit must enforce CPU and RAM limits so a runaway agent script cannot exhaust host resources.
- **Acceptance Criteria:**
  1. Given `ivai.service` with `MemoryMax=512M`, when the ivAI process exceeds 512MB, then the kernel OOM-kills it and systemd restarts it.
  2. Given `ivai.service` with `CPUQuota=200%`, when ivAI spawns compute-heavy subprocesses, then total CPU usage is capped at 2 cores.
  3. Given the service restarts after OOM, when ivAI starts, then it recovers its last consistent state from the memory database.
- **Architecture impact:** §1 Core (Daemonization) — systemd unit file update.
- **Design doc:** None (systemd-native feature).

### R5.6: Egress Filtering
- **Priority:** P2 (nice)
- **Description:** The `http_request` tool must only be allowed to reach explicitly whitelisted domains. All other outbound network requests from the VM are blocked via iptables.
- **Acceptance Criteria:**
  1. Given `http_request("https://api.anthropic.com/v1/messages", ...)`, when `api.anthropic.com` is in the whitelist, then the request proceeds.
  2. Given `http_request("https://evil.example.com/exfil", ...)`, when the domain is not whitelisted, then the request is blocked with error `"egress denied: evil.example.com"`.
  3. Given `execute_command("curl http://evil.example.com")`, when the domain is not whitelisted, then the shell command also fails (iptables-level block).
- **Architecture impact:** §4 Execution Subsystems (`http_request` tool), §1 Core (iptables rules in VM provisioning).
- **Design doc:** None.

---

## Dependencies

- **Upstream:** Phase 3 ✅ (HTTP server must exist), Phase 4 ✅ (`http_request` tool must exist for R5.6)
- **Downstream:** Phase 9 (webhook receivers need auth), Phase 10 (HITL needs secure channel), Phase 11 (gRPC fleet needs mTLS foundation)

---

## Test Plan

- **Unit tests:** Middleware tests for R5.1 (valid, invalid, missing key). Path sanitization tests for R5.4 (traversal, symlink, boundary). Egress whitelist tests for R5.6.
- **Integration tests:** `ivaictl` end-to-end with mTLS certs (R5.2). SSE stream with auth header (R5.1 + R5.3).
- **Manual verification:** Deploy to OrbStack VM, verify cgroups limits via `systemd-cgtop` (R5.5). Verify iptables blocks via `curl` from VM shell (R5.6).

## Open Questions

- [ ] R5.2: Should we use self-signed CA or integrate with a PKI (step-ca, etc.)? Self-signed is simpler for a single-user system but harder to rotate.
- [ ] R5.4: Should the workspace be configurable per-task, or always `/home/ivai/`? Per-task adds complexity; start with fixed workspace.
- [ ] R5.6: Should the whitelist be static (env var) or dynamic (tool-callable `allow_domain`)? Static is simpler and more secure; dynamic can come later with HITL approval.

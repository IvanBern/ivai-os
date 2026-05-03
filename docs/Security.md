# Security Framework

ivAI implements a 6-layer security hardening framework with 124 tests and 97% code coverage. The `ivai-security` Python library enforces defense-in-depth across all interfaces.

## Security Architecture

```
User Input
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 1: Input Sanitizer                │
│ Strips ANSI escapes, null bytes, CRLF   │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 2: URL Guard                      │
│ Allowlist/denylist enforcement          │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 3: Tool Guardrails                │
│ Per-tool safety constraints             │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 4: Human Loop (PIN Gate)          │
│ Destructive action approval             │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 5: Audit Logger                   │
│ Tamper-evident audit trail              │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Layer 6: System Prompt Hardening        │
│ Injection defense at LLM boundary       │
└─────────────────────────────────────────┘
    │
    ▼
Safe Output
```

## Layer 1: Input Sanitizer

**File:** `ivai-security/ivai_security/input_sanitizer.py`

Sanitizes all user input before it reaches any processing layer:

- Strips ANSI escape sequences (terminal injection prevention)
- Removes null bytes
- Normalizes line endings (CRLF → LF)
- Truncates at max safe length
- Detects prompt injection patterns

```python
def sanitize(user_input: str, max_length: int = 100000) -> str:
    # Strip ANSI escape sequences
    sanitized = re.sub(r'\x1b\[[0-9;]*[a-zA-Z]', '', user_input)
    # Remove null bytes
    sanitized = sanitized.replace('\x00', '')
    # Normalize line endings
    sanitized = sanitized.replace('\r\n', '\n').replace('\r', '\n')
    # Truncate
    if len(sanitized) > max_length:
        sanitized = sanitized[:max_length]
    return sanitized
```

## Layer 2: URL Guard

**File:** `ivai-security/ivai_security/url_guard.py`

Enforces allowlist/denylist for all outbound HTTP requests from tools:

- **Allowlist:** trusted domains that ivAI can access
- **Denylist:** blocked domains (localhost, internal IPs, SSRF targets)
- **IP conversion check:** resolves domain to IP, blocks private/multicast ranges

```python
BLOCKED_PREFIXES = [
    "127.", "10.", "172.16.", "172.17.", "172.18.",
    "172.19.", "172.20.", "172.21.", "172.22.", "172.23.",
    "172.24.", "172.25.", "172.26.", "172.27.", "172.28.",
    "172.29.", "172.30.", "172.31.", "192.168.", "0.", "169.254."
]
```

## Layer 3: Tool Guardrails

**File:** `ivai-security/ivai_security/tool_guardrails.py`

Per-tool safety constraints that validate tool calls before execution:

| Tool | Constraint |
|---|---|
| `execute_command` | Command denylist: `sudo`, `rm -rf /`, `mkfs`, `dd`, `:(){ :|:& };:` |
| `file_write` | Path must be within allowed directories |
| `file_read` | Cannot read `.env`, `*_secret*`, `*.pem`, `id_*` |
| `http_request` | Must pass URL Guard (Layer 2) |

## Layer 4: Human Loop (PIN Gate)

**File:** `ivai-security/ivai_security/human_loop.py`

Implements a secure approval gate for destructive actions:

- **PIN verification:** SHA256 hash comparison against `~/.ivai_secrets/pin_hash`
- **Action categorization:** each tool call classified as `safe`, `sensitive`, or `destructive`
- **Time-limited approvals:** PIN approval valid for 5 minutes
- **Telegram integration:** PIN prompt sent to Telegram for remote approval

```
Action: systemctl restart nginx
Category: destructive
→ PIN required
→ Telegram: "ivAI wants to restart nginx. Approve? [Yes] [No]"
→ User enters PIN
→ SHA256(PIN) == stored_hash → Approved
→ Action executes
→ Approval expires in 5 minutes
```

## Layer 5: Audit Logger

**File:** `ivai-security/ivai_security/audit_logger.py`

Tamper-evident audit trail of all sensitive operations:

```json
{
  "timestamp": "2026-05-03T22:15:00Z",
  "action": "systemctl_restart",
  "target": "nginx.service",
  "interface": "telegram",
  "result": "success",
  "hash": "sha256:abc123...",
  "previous_hash": "sha256:def456..."
}
```

- **Hash chain:** each entry includes the hash of the previous entry (blockchain-style integrity)
- **Redaction:** API keys and PII are automatically stripped before logging
- **Immutable append:** audit log is append-only

## Layer 6: System Prompt Hardening

**File:** `ivai-security/ivai_security/system_prompt.py`

Defends against prompt injection attacks at the LLM boundary:

- **Instruction hierarchy:** system instructions take precedence over user input
- **Delimiter injection:** user input is wrapped in XML-style delimiters to prevent boundary confusion
- **Input scrubbing:** removes known injection patterns (`ignore previous instructions`, `system:`, etc.)
- **Context isolation:** user data and system instructions never share the same delimiter scope

```python
def wrap_user_input(user_input: str) -> str:
    """Wrap user input in delimiters to separate from system instructions."""
    return f"<user_input>\n{user_input}\n</user_input>"
```

## SafeExecute

The `safe_execute` function enforces shell command safety:

- **Blacklist scanning:** blocks `sudo`, `rm -rf /`, fork bombs, etc.
- **Jailed workspace:** all shell operations confined to `~/ivai-workspace`
- **Timeout enforcement:** commands killed after 30 seconds
- **Output redaction:** API keys and PII stripped from stdout/stderr

```python
BLACKLIST = [
    "sudo ", "rm -rf /", "mkfs.", "dd if=",
    ":(){ :|:& };:", "chmod 777 /", "> /dev/sda"
]
```

## Telegram Security

The Telegram bot adds additional security layers:

- **User whitelist:** only authorized Telegram user IDs can interact
- **Inline keyboard:** destructive commands require explicit button confirmation
- **PIN gate:** sensitive operations require PIN even from authorized users
- **Command debouncing:** rapid repeated commands are throttled

## Dependency Auditing

- **npm audit** runs in CI for all Node.js modules
- **pip-audit** runs for Python dependencies
- Known safe packages with flagged CVEs are documented with override justifications

## Test Coverage

```
Module                    Tests   Coverage
─────────────────────────────────────────
input_sanitizer.py          28      98%
url_guard.py                19      96%
tool_guardrails.py          24      97%
human_loop.py               18      97%
audit_logger.py             16      98%
system_prompt.py            12      97%
integration                  7      95%
─────────────────────────────────────────
Total                      124      97%
```

Run: `cd ivai-security && pytest tests/ -v --cov=ivai_security`

## Adding a Security Rule

1. Identify the threat vector
2. Add detection/blocking logic in the appropriate layer
3. Add test case(s) in `ivai-security/tests/`
4. Ensure all existing tests still pass
5. Document the rule in this wiki page
6. Update the threat model (if significant)

## Security Model Principles

1. **Defense in depth** — no single layer is trusted to catch everything
2. **Fail closed** — if a security check errors, deny the action
3. **Least privilege** — sub-agents use read-only tools by default
4. **Auditability** — every sensitive action is logged with hash-chain integrity
5. **Human sovereignty** — the PIN gate ensures the human is always the ultimate authority

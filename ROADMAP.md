# The Ivai OS Roadmap

## Phase 1.

## Phase 2.
1. The Memory Subsystem (internal/memory/db.go) - We embed the pure-Go SQLite database so Ivai can store conversation history, remember past tasks, and maintain context across its event loop.

2. The Execution Sandbox (internal/sandbox/wazero.go) - We integrate the wazero WebAssembly runtime so Ivai can safely execute compiled code plugins directly inside the core OS engine.
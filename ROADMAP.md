# The Ivai OS Roadmap

## Phase 1.

## Phase 2.

### 1. The Memory Subsystem (internal/memory/db.go) 
We embed the pure-Go SQLite database so Ivai can store conversation history, remember past tasks, and maintain context across its event loop.

### 2. The Execution Sandbox (internal/sandbox/wazero.go) 
We integrate the wazero WebAssembly runtime so Ivai can safely execute compiled code plugins directly inside the core OS engine.

## Phase 3.

### 1. The Agentic Tool Protocol (Function Calling)
 DeepSeek V4 fully supports the OpenAI Tool Calling specification. Instead of just sending the conversation history, we send a JSON schema describing the tools Ivai has available (e.g., execute_wasm, write_file). When DeepSeek decides it needs to use a tool, it returns a special JSON payload instead of a text message. Our Go engine catches that payload, runs the local Go function, and feeds the result back to DeepSeek.

 ### Tools

 1. File I/O (read_file / write_file): Allows Ivai to read its own Go source code and write new scripts.

2. System Shell (execute_command): Allows Ivai to run commands like go build or make dev inside the Debian VM.

3. The Sandbox (execute_wasm): Wires up the Wazero engine we just built so Ivai can execute compiled .wasm plugins safely.
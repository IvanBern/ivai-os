package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmRuntime manages the sandboxed execution of code plugins
type WasmRuntime struct{}

// NewWasmRuntime initializes the sandbox manager
func NewWasmRuntime() *WasmRuntime {
	return &WasmRuntime{}
}

// Execute runs a compiled WebAssembly binary with strict timeouts and standard I/O.
// - wasmBytes: The compiled .wasm file loaded into memory.
// - payload: The input string passed to the plugin via standard input (stdin).
// - timeoutMs: Maximum execution time in milliseconds.
func (w *WasmRuntime) Execute(ctx context.Context, wasmBytes []byte, payload string, timeoutMs int) (string, error) {
	// 1. Enforce a strict execution timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// 2. Initialize the Wazero runtime
	r := wazero.NewRuntime(timeoutCtx)
	// r.Close automatically cleans up all resources and memory used by this runtime
	defer r.Close(timeoutCtx)

	// 3. Instantiate the WASI subsystem (required for stdin/stdout/stderr)
	wasi_snapshot_preview1.MustInstantiate(timeoutCtx, r)

	// 4. Setup sandboxed I/O buffers
	var outBuf, errBuf bytes.Buffer
	config := wazero.NewModuleConfig().
		WithStdin(bytes.NewReader([]byte(payload))). // Inject the payload as stdin
		WithStdout(&outBuf).                         // Capture stdout as the result
		WithStderr(&errBuf)                          // Capture stderr for debugging

	// 5. Compile and execute the Wasm module
	// InstantiateWithConfig automatically calls the `_start` function for WASI modules
	_, err := r.InstantiateWithConfig(timeoutCtx, wasmBytes, config)

	if err != nil {
		// Check if the sandbox was killed due to our timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("sandbox killed: execution timed out after %d ms", timeoutMs)
		}
		// Otherwise, it was a crash or logic error inside the plugin
		return "", fmt.Errorf("plugin execution failed: %v\nstderr: %s", err, errBuf.String())
	}

	// 6. Return the captured standard output
	return outBuf.String(), nil
}

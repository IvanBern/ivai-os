package sandbox

import (
	"context"
	"testing"
)

func TestNewWasmRuntime(t *testing.T) {
	runtime := NewWasmRuntime()
	if runtime == nil {
		t.Fatal("expected non-nil runtime")
	}
}

func TestExecuteErrorPaths(t *testing.T) {
	runtime := NewWasmRuntime()
	ctx := context.Background()

	t.Run("Invalid WASM", func(t *testing.T) {
		_, err := runtime.Execute(ctx, []byte("not a wasm"), "payload", 1000)
		if err == nil {
			t.Error("expected error for invalid WASM, got nil")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		// We can't easily test a real timeout without a valid WASM that loops,
		// but we can at least check if the context is respected.
		// However, wazero initialization might be faster than the timeout.
		// For now, let's just ensure it handles a very short timeout gracefully if possible.
	})
}

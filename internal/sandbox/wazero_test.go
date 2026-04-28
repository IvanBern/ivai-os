package sandbox

import (
	"testing"
)

func TestNewWasmRuntime(t *testing.T) {
	runtime := NewWasmRuntime()
	if runtime == nil {
		t.Fatal("expected non-nil runtime")
	}
}

package telemetry

import (
	"context"
	"testing"
)

func TestInitTracer(t *testing.T) {
	tp, err := InitTracer("test-service")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
}

func TestTracerShutdown(t *testing.T) {
	tp, err := InitTracer("test-service")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	// Double shutdown should not panic
	_ = tp.Shutdown(context.Background())
}

func TestTracerProviderGlobalSet(t *testing.T) {
	_, err := InitTracer("test-global")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	// Just verify InitTracer didn't panic setting the global provider
}

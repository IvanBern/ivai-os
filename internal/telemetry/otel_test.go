package telemetry

import (
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

package tools

import (
	"strings"
	"testing"
)

func TestExecuteCommand(t *testing.T) {
	output, err := ExecuteCommand("echo 'hello ivai'")
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if !strings.Contains(output, "hello ivai") {
		t.Errorf("expected hello ivai in output, got %s", output)
	}
}

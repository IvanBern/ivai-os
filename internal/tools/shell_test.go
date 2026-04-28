package tools

import (
	"strings"
	"testing"
)

func TestExecuteCommand(t *testing.T) {
	output, err := ExecuteCommand("echo 'hello world'")
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if strings.TrimSpace(output) != "hello world" {
		t.Errorf("expected 'hello world', got %s", output)
	}
}

func TestExecuteCommandError(t *testing.T) {
	output, err := ExecuteCommand("nonexistentcommand_ivai")
	// The function returns err == nil but puts error in the output string
	if err != nil {
		t.Fatalf("ExecuteCommand should return nil error, got %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("expected output to contain 'Error:', got %s", output)
	}
}

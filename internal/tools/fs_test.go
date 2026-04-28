package tools

import (
	"os"
	"testing"
)

func TestFileOps(t *testing.T) {
	path := "test_file.txt"
	content := "hello world"
	defer os.Remove(path)

	// 1. Write
	err := WriteFile(path, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// 2. Read
	readContent, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if readContent != content {
		t.Errorf("expected %s, got %s", content, readContent)
	}
}

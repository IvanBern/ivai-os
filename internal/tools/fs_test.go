package tools

import (
	"os"
	"testing"
)

func TestFileOps(t *testing.T) {
	path := "test_file.txt"
	content := "hello world"

	err := WriteFile(path, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	defer os.Remove(path)

	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if got != content {
		t.Errorf("expected %s, got %s", content, got)
	}
}

func TestReadFileError(t *testing.T) {
	_, err := ReadFile("non_existent_file_ivai.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

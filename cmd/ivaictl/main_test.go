package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// 1. Mock API Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	os.Setenv("IVAI_API_URL", server.URL)
	defer os.Unsetenv("IVAI_API_URL")

	t.Run("Command line args", func(t *testing.T) {
		err := run([]string{"ivaictl", "hello"}, nil)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("Stdin input", func(t *testing.T) {
		stdin := strings.NewReader("hello from stdin")
		err := run([]string{"ivaictl"}, stdin)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("No instruction", func(t *testing.T) {
		err := run([]string{"ivaictl"}, strings.NewReader(""))
		if err == nil {
			t.Error("expected error for empty instruction, got nil")
		}
	})
}

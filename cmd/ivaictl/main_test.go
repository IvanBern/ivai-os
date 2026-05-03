package main

import (
	"fmt"
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

func TestRunStreaming(t *testing.T) {
	// Mock SSE server that emits progress events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/task/stream" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "event: task_start\ndata: {\"type\":\"task_start\",\"message\":\"Task started\",\"data\":{\"model\":\"deepseek-v4-pro\",\"instruction\":\"test\"}}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "event: thinking\ndata: {\"type\":\"thinking\",\"message\":\"Model is thinking\",\"data\":{\"reasoning\":\"Let me think...\"}}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "event: tool_call\ndata: {\"type\":\"tool_call\",\"message\":\"Calling tool: read_file\",\"data\":{\"name\":\"read_file\",\"args\":\"{\\\"filepath\\\":\\\"/tmp/test\\\"}\"}}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "event: tool_result\ndata: {\"type\":\"tool_result\",\"message\":\"Tool result: read_file\",\"data\":{\"name\":\"read_file\",\"result\":\"file contents here\"}}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "event: task_complete\ndata: {\"type\":\"task_complete\",\"message\":\"Task completed\",\"data\":{\"response\":\"All done!\"}}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	os.Setenv("IVAI_API_URL", server.URL+"/api/task")
	defer os.Unsetenv("IVAI_API_URL")

	t.Run("Streaming flag", func(t *testing.T) {
		err := run([]string{"ivaictl", "--stream", "test"}, nil)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("Streaming stdin", func(t *testing.T) {
		stdin := strings.NewReader("test from stdin")
		err := run([]string{"ivaictl", "--stream"}, stdin)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestBuildStreamURL(t *testing.T) {
	os.Setenv("IVAI_PORT", "9090")
	defer os.Unsetenv("IVAI_PORT")

	url := buildStreamURL()
	if !strings.Contains(url, "/api/task/stream") {
		t.Errorf("expected stream URL to contain /api/task/stream, got %s", url)
	}
	if !strings.Contains(url, "9090") {
		t.Errorf("expected stream URL to contain port 9090, got %s", url)
	}
}

func TestReadSSEStream(t *testing.T) {
	sseData := `event: task_start
data: {"type":"task_start","message":"Task started","data":{"model":"deepseek","instruction":"hi"}}

event: thinking
data: {"type":"thinking","message":"Model is thinking","data":{"reasoning":"test reasoning"}}

event: task_complete
data: {"type":"task_complete","message":"Task completed","data":{"response":"done"}}

`

	err := readSSEStream(strings.NewReader(sseData))
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

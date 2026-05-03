package main

import (
	"bufio"
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

func TestBuildAPIDefaultPort(t *testing.T) {
	os.Unsetenv("IVAI_API_URL")
	os.Unsetenv("IVAI_PORT")
	url := buildAPIURL()
	if url != "http://localhost:8080/api/task" {
		t.Errorf("expected default URL, got %s", url)
	}
}

func TestHandleResponse(t *testing.T) {
	t.Run("status OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"response":"hello"}`))
		}))
		defer server.Close()
		resp, _ := http.Get(server.URL)
		err := handleResponse(resp)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
	t.Run("status Accepted", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()
		resp, _ := http.Get(server.URL)
		err := handleResponse(resp)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
	t.Run("status error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()
		resp, _ := http.Get(server.URL)
		err := handleResponse(resp)
		if err == nil {
			t.Error("expected error for bad status")
		}
	})
}

func TestPrintTaskError(t *testing.T) {
	evt := progressEvent{Type: "task_error", Data: map[string]any{"error": "something failed"}}
	printTaskError(evt)
}

func TestPrintTaskErrorNilData(t *testing.T) {
	printTaskError(progressEvent{})
}

func TestPrintThinkingVariations(t *testing.T) {
	t.Run("with reasoning and content", func(t *testing.T) {
		printThinking(progressEvent{Data: map[string]any{"reasoning": "r", "content": "c"}})
	})
	t.Run("nil data", func(t *testing.T) {
		printThinking(progressEvent{})
	})
	t.Run("empty fields", func(t *testing.T) {
		printThinking(progressEvent{Data: map[string]any{"reasoning": "", "content": ""}})
	})
}

func TestPrintToolCallNilData(t *testing.T) {
	printToolCall(progressEvent{})
}

func TestPrintToolResultNilData(t *testing.T) {
	printToolResult(progressEvent{})
}

func TestPrintTaskCompleteNilData(t *testing.T) {
	printTaskComplete(progressEvent{})
}

func TestPrintSSEEventDefault(t *testing.T) {
	printSSEEvent("unknown_type", `{"type":"unknown"}`)
}

func TestSendTaskError(t *testing.T) {
	os.Setenv("IVAI_API_URL", "http://127.0.0.1:1")
	defer os.Unsetenv("IVAI_API_URL")
	err := sendTask("test")
	if err == nil {
		t.Error("expected error when no server running")
	}
}

func TestStreamTaskError(t *testing.T) {
	os.Setenv("IVAI_API_URL", "http://127.0.0.1:1")
	defer os.Unsetenv("IVAI_API_URL")
	err := streamTask("test")
	if err == nil {
		t.Error("expected error when no server running")
	}
}

func TestReadSSEScannerError(t *testing.T) {
	// A very long line (over default scanner buffer) should cause scanner error
	longPrefix := strings.Repeat("a", bufio.MaxScanTokenSize)
	sseData := "event: task_start\ndata: " + longPrefix + "\n\n"
	err := readSSEStream(strings.NewReader(sseData))
	if err == nil {
		t.Error("expected scanner error for oversized line")
	}
}

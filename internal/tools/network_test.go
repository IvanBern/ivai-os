package tools

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpRequest(t *testing.T) {
	// 1. Setup Mock Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Test") != "ivai" {
			t.Errorf("missing header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// 2. Call tool
	headers := map[string]string{"X-Test": "ivai"}
	resp, err := HttpRequest("POST", server.URL, `{"cmd": "test"}`, headers)

	// 3. Assertions
	if err != nil {
		t.Fatalf("HttpRequest failed: %v", err)
	}

	expected := `{"status": "ok"}`
	if resp != expected {
		t.Errorf("expected %s, got %s", expected, resp)
	}
}

func TestHttpRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal"}`))
	}))
	defer server.Close()

	resp, err := HttpRequest("GET", server.URL, "", nil)

	if err == nil {
		t.Fatal("expected error for 500 status")
	}

	expectedBody := `{"error": "internal"}`
	if resp != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, resp)
	}
}

func TestHttpRequestConnectionRefused(t *testing.T) {
	_, err := HttpRequest("GET", "http://127.0.0.1:1", "", nil)
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

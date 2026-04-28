package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayGeminiTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GeminiRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Assertions on Gemini mapping
		if len(req.Contents) == 0 || req.Contents[0].Role != "user" {
			t.Errorf("incorrect Gemini role mapping")
		}

		resp := GeminiResponse{
			Candidates: []struct {
				Content GeminiContent `json:"content"`
			}{
				{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGateway("", "", "key")
	g.GeminiURL = server.URL

	resp, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "gemini-1.5-pro")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "hi" {
		t.Errorf("expected hi, got %s", resp.Content)
	}
}

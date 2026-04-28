package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayDeepSeekTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OpenAIRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "deepseek-v4-pro" {
			t.Errorf("expected deepseek-v4-pro, got %s", req.Model)
		}
		if len(req.Messages) == 0 || req.Messages[0].Content != "hello" {
			t.Errorf("incorrect message mapping")
		}

		resp := OpenAIResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{
				{Message: Message{Role: "assistant", Content: "hi"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGateway("key", "", "")
	g.DeepSeekURL = server.URL

	resp, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "deepseek-v4-pro")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "hi" {
		t.Errorf("expected hi, got %s", resp.Content)
	}
}

func TestGatewayAnthropicTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AnthropicRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("expected claude, got %s", req.Model)
		}
		if len(req.Messages) == 0 || req.Messages[0].Content[0].Text != "hello" {
			t.Errorf("incorrect Anthropic message mapping")
		}

		resp := AnthropicResponse{
			Content: []AnthropicContent{{Type: "text", Text: "hi"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGateway("", "key", "")
	g.AnthropicURL = server.URL

	resp, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "hi" {
		t.Errorf("expected hi, got %s", resp.Content)
	}
}

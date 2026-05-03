package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayDeepSeekTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OpenAIRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "deepseek-v4-pro" {
			t.Errorf("expected deepseek-v4-pro, got %s", req.Model)
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

		resp := AnthropicResponse{
			Content: []AnthropicContent{{Type: "text", Text: "hi"}},
		}

		// If it's a tool call request
		for _, m := range req.Messages {
			for _, c := range m.Content {
				if strings.Contains(c.Text, "use tool") {
					resp.Content = []AnthropicContent{
						{
							Type:  "tool_use",
							ID:    "call_1",
							Name:  "read_file",
							Input: map[string]any{"filepath": "test.txt"},
						},
					}
				}
			}
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGateway("", "key", "")
	g.AnthropicURL = server.URL

	t.Run("Text only", func(t *testing.T) {
		resp, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}}, nil, "claude-3-5-sonnet-20241022")
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		if resp.Content != "hi" {
			t.Errorf("expected hi, got %s", resp.Content)
		}
	})

	t.Run("Complex history", func(t *testing.T) {
		messages := []Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "thinking", ToolCalls: []ToolCall{{ID: "1", Type: "function", Function: ToolCallFunction{Name: "t", Arguments: "{}"}}}},
			{Role: "tool", Content: "res", ToolCallID: "1"},
			{Role: "user", Content: "use tool"},
		}
		_, err := g.Chat(context.Background(), messages, []Tool{{Type: "function", Function: FunctionDefinition{Name: "read_file"}}}, "claude-3-5-sonnet-20241022")
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
	})
}

func TestGatewayGeminiTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GeminiRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := GeminiResponse{
			Candidates: []struct {
				Content GeminiContent `json:"content"`
			}{
				{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}}},
			},
		}

		// Mock tool call response - check all parts for "use tool"
		shouldToolCall := false
		for _, c := range req.Contents {
			for _, p := range c.Parts {
				if strings.Contains(p.Text, "use tool") {
					shouldToolCall = true
				}
			}
		}

		if shouldToolCall {
			resp.Candidates[0].Content.Parts = []GeminiPart{
				{
					FunctionCall: &GeminiFunctionCall{
						Name: "read_file",
						Args: map[string]any{"filepath": "test.txt"},
					},
				},
			}
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGateway("", "", "key")
	g.GeminiURL = server.URL

	t.Run("Tool call and history", func(t *testing.T) {
		messages := []Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "thinking", ToolCalls: []ToolCall{{ID: "1", Type: "function", Function: ToolCallFunction{Name: "read_file", Arguments: `{"filepath":"a"}`}}}},
			{Role: "tool", Content: "res", Name: "read_file", ToolCallID: "1"},
			{Role: "user", Content: "use tool"},
		}
		resp, err := g.Chat(context.Background(), messages, []Tool{{Type: "function", Function: FunctionDefinition{Name: "read_file"}}}, "gemini-1.5-pro")
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}
		if len(resp.ToolCalls) == 0 {
			t.Fatal("expected tool calls")
		}
	})
}

func TestGatewayErrorPaths(t *testing.T) {
	g := NewGateway("key", "key", "key")

	tests := []struct {
		name    string
		setup   func() *httptest.Server
		call    func(*Gateway) error
		wantErr string
	}{
		{
			name: "DeepSeek API Error",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("internal error"))
				}))
			},
			call: func(g *Gateway) error {
				_, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "deepseek-chat")
				return err
			},
			wantErr: "DeepSeek API error 500",
		},
		{
			name: "Anthropic API Error",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("bad request"))
				}))
			},
			call: func(g *Gateway) error {
				_, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "claude-3-5-sonnet")
				return err
			},
			wantErr: "Anthropic API error 400",
		},
		{
			name: "Gemini API Error",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("forbidden"))
				}))
			},
			call: func(g *Gateway) error {
				_, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "gemini-1.5-pro")
				return err
			},
			wantErr: "Gemini API error 403",
		},
		{
			name: "DeepSeek Empty Response",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"choices": []}`))
				}))
			},
			call: func(g *Gateway) error {
				_, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "deepseek-chat")
				return err
			},
			wantErr: "empty choices",
		},
		{
			name: "Gemini Empty Candidates",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"candidates": []}`))
				}))
			},
			call: func(g *Gateway) error {
				_, err := g.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "gemini-1.5-pro")
				return err
			},
			wantErr: "Gemini empty candidates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setup()
			defer server.Close()

			// Override URLs
			g.DeepSeekURL = server.URL
			g.AnthropicURL = server.URL
			g.GeminiURL = server.URL

			err := tt.call(g)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

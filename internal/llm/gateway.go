package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// --- Tool Calling Schemas ---

// FunctionDefinition describes the structure of a tool Ivai can use
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Tool wraps the function definition
type Tool struct {
	Type     string             `json:"type"` // Usually "function"
	Function FunctionDefinition `json:"function"`
}

// ToolCallFunction holds the arguments returned by the LLM
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // This is returned as a JSON string
}

// ToolCall represents an individual tool execution request from the LLM
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// --- Extended Message Schema ---

// Message now supports tool calling fields. We use omitempty so standard
// chat messages don't break the API by sending null tool fields.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`         // Used when returning tool results
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Populated when LLM wants to use a tool
	ToolCallID string     `json:"tool_call_id,omitempty"` // Used to link the result back to the call
}

// --- HTTP Payload Schemas ---

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"` // The list of available tools
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// --- Gateway Implementation ---

type Gateway struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewGateway(apiKey string) *Gateway {
	return &Gateway{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Chat replaces GenerateText. It accepts an array of Tools and returns a full Message
// struct instead of just a string, allowing us to inspect if the LLM called a tool.
func (g *Gateway) Chat(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.APIKey)

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return Message{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return Message{}, fmt.Errorf("API returned empty choices")
	}

	return chatResp.Choices[0].Message, nil
}

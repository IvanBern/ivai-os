package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// --- Tool Calling Schemas (Shared) ---

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Tool struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// --- Extended Message Schema (Ivai Internal) ---

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// --- OpenAI / DeepSeek API Schemas ---

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// --- Anthropic API Schemas ---

type AnthropicContent struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`        // tool_use id
	Name      string                 `json:"name,omitempty"`      // tool_use name
	Input     map[string]interface{} `json:"input,omitempty"`     // tool_use input
	ToolUseID string                 `json:"tool_use_id,omitempty"` // tool_result tool_use_id
	Content   string                 `json:"content,omitempty"`    // tool_result content
}

type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
}

// --- Gateway Implementation ---

type Gateway struct {
	DeepSeekKey  string
	AnthropicKey string
	HTTPClient   *http.Client
}

func NewGateway(deepSeekKey, anthropicKey string) *Gateway {
	return &Gateway{
		DeepSeekKey:  deepSeekKey,
		AnthropicKey: anthropicKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (g *Gateway) Chat(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	if strings.HasPrefix(model, "claude-") {
		return g.chatAnthropic(ctx, messages, tools, model)
	}
	return g.chatDeepSeek(ctx, messages, tools, model)
}

func (g *Gateway) chatDeepSeek(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	reqBody := OpenAIRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.DeepSeekKey)

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("DeepSeek API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return Message{}, err
	}

	if len(openAIResp.Choices) == 0 {
		return Message{}, fmt.Errorf("empty choices")
	}

	return openAIResp.Choices[0].Message, nil
}

func (g *Gateway) chatAnthropic(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	var systemPrompt string
	var anthropicMessages []AnthropicMessage

	// Translate Ivai messages to Anthropic format
	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}

		if m.Role == "tool" {
			// Find previous user message to append tool result, or create new user message
			// Anthropic requires tool_result to follow tool_use in a user message
			if len(anthropicMessages) > 0 && anthropicMessages[len(anthropicMessages)-1].Role == "user" {
				anthropicMessages[len(anthropicMessages)-1].Content = append(anthropicMessages[len(anthropicMessages)-1].Content, AnthropicContent{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				})
			} else {
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role: "user",
					Content: []AnthropicContent{
						{
							Type:      "tool_result",
							ToolUseID: m.ToolCallID,
							Content:   m.Content,
						},
					},
				})
			}
			continue
		}

		am := AnthropicMessage{Role: m.Role}
		if len(m.ToolCalls) > 0 {
			// Assistant tool call
			if m.Content != "" {
				am.Content = append(am.Content, AnthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				am.Content = append(am.Content, AnthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
		} else {
			am.Content = []AnthropicContent{{Type: "text", Text: m.Content}}
		}
		anthropicMessages = append(anthropicMessages, am)
	}

	anthropicTools := make([]AnthropicTool, len(tools))
	for i, t := range tools {
		anthropicTools[i] = AnthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		}
	}

	reqBody := AnthropicRequest{
		Model:     model,
		Messages:  anthropicMessages,
		System:    systemPrompt,
		MaxTokens: 4096,
		Tools:     anthropicTools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return Message{}, err
	}

	// Translate Anthropic response back to Ivai message
	resMsg := Message{Role: "assistant"}
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			resMsg.Content += c.Text
		} else if c.Type == "tool_use" {
			args, _ := json.Marshal(c.Input)
			resMsg.ToolCalls = append(resMsg.ToolCalls, ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      c.Name,
					Arguments: string(args),
				},
			})
		}
	}

	return resMsg, nil
}

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
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Name             string     `json:"name,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
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

// --- Gemini API Schemas ---

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiTool struct {
	FunctionDeclarations []FunctionDefinition `json:"function_declarations"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	Tools    []GeminiTool    `json:"tools,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content GeminiContent `json:"content"`
	} `json:"candidates"`
}

// --- Gateway Implementation ---

type Gateway struct {
	DeepSeekKey  string
	AnthropicKey string
	GeminiKey    string
	HTTPClient   *http.Client

	// Base URLs for testing
	DeepSeekURL    string
	AnthropicURL   string
	GeminiURL      string
}

func NewGateway(deepSeekKey, anthropicKey, geminiKey string) *Gateway {
	return &Gateway{
		DeepSeekKey:  deepSeekKey,
		AnthropicKey: anthropicKey,
		GeminiKey:    geminiKey,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		DeepSeekURL:  "https://api.deepseek.com/chat/completions",
		AnthropicURL: "https://api.anthropic.com/v1/messages",
		GeminiURL:    "https://generativelanguage.googleapis.com/v1beta/models",
	}
}

func (g *Gateway) Chat(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	if strings.HasPrefix(model, "claude-") {
		return g.chatAnthropic(ctx, messages, tools, model)
	}
	if strings.HasPrefix(model, "gemini-") {
		return g.chatGemini(ctx, messages, tools, model)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.DeepSeekURL, bytes.NewBuffer(jsonData))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.AnthropicURL, bytes.NewBuffer(jsonData))
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

func (g *Gateway) chatGemini(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	var geminiContents []GeminiContent
	var systemInstructions string

	// Gemini separates system instructions from contents
	for _, m := range messages {
		if m.Role == "system" {
			systemInstructions = m.Content
			continue
		}

		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		part := GeminiPart{}
		if m.Role == "tool" {
			role = "function"
			part.FunctionResponse = &GeminiFunctionResponse{
				Name: m.Name,
				Response: map[string]interface{}{
					"content": m.Content,
				},
			}
		} else if len(m.ToolCalls) > 0 {
			// Model requesting tool use
			role = "model"
			for _, tc := range m.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				geminiContents = append(geminiContents, GeminiContent{
					Role: role,
					Parts: []GeminiPart{{
						FunctionCall: &GeminiFunctionCall{
							Name: tc.Function.Name,
							Args: args,
						},
					}},
				})
			}
			continue
		} else {
			part.Text = m.Content
		}

		geminiContents = append(geminiContents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{part},
		})
	}

	geminiTools := []GeminiTool{}
	if len(tools) > 0 {
		declarations := make([]FunctionDefinition, len(tools))
		for i, t := range tools {
			declarations[i] = t.Function
		}
		geminiTools = append(geminiTools, GeminiTool{FunctionDeclarations: declarations})
	}

	reqBody := GeminiRequest{
		Contents: geminiContents,
		Tools:    geminiTools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.GeminiURL, model, g.GeminiKey)
	// If system instructions exist, they are often added differently or as a separate field in v1beta
	// but for simplicity we often append them to the first user message or use the dedicated field if supported.
	// Gemini 1.5 supports system_instruction field.
	type AdvancedGeminiRequest struct {
		GeminiRequest
		SystemInstruction *GeminiContent `json:"system_instruction,omitempty"`
	}

	advReq := AdvancedGeminiRequest{GeminiRequest: reqBody}
	if systemInstructions != "" {
		advReq.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemInstructions}},
		}
	}
	jsonData, _ = json.Marshal(advReq)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return Message{}, err
	}

	if len(geminiResp.Candidates) == 0 {
		return Message{}, fmt.Errorf("Gemini empty candidates")
	}

	resMsg := Message{Role: "assistant"}
	modelContent := geminiResp.Candidates[0].Content
	for _, p := range modelContent.Parts {
		if p.Text != "" {
			resMsg.Content += p.Text
		}
		if p.FunctionCall != nil {
			args, _ := json.Marshal(p.FunctionCall.Args)
			resMsg.ToolCalls = append(resMsg.ToolCalls, ToolCall{
				ID:   p.FunctionCall.Name, // Gemini doesn't always provide a call ID in the same way, we use Name
				Type: "function",
				Function: ToolCallFunction{
					Name:      p.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}
	}

	return resMsg, nil
}

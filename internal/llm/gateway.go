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
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
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
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`          // tool_use id
	Name      string         `json:"name,omitempty"`        // tool_use name
	Input     map[string]any `json:"input,omitempty"`       // tool_use input
	ToolUseID string         `json:"tool_use_id,omitempty"` // tool_result tool_use_id
	Content   string         `json:"content,omitempty"`     // tool_result content
}

type AnthropicMessage struct {
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
}

type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
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
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GeminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
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
	DeepSeekURL  string
	AnthropicURL string
	GeminiURL    string
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

type providerCall struct {
	url      string
	body     any
	headers  map[string]string
	provider string
	target   any
}

// doProviderRequest marshals reqBody, POSTs to url with headers, checks status, and decodes into target.
func (g *Gateway) doProviderRequest(ctx context.Context, c providerCall) error {
	jsonData, err := json.Marshal(c.body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s API error %d: %s", c.provider, resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(c.target)
}

func (g *Gateway) chatDeepSeek(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	reqBody := OpenAIRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}

	var openAIResp OpenAIResponse
	if err := g.doProviderRequest(ctx, providerCall{
		url:      g.DeepSeekURL,
		body:     reqBody,
		headers:  map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + g.DeepSeekKey},
		provider: "DeepSeek",
		target:   &openAIResp,
	}); err != nil {
		return Message{}, err
	}

	if len(openAIResp.Choices) == 0 {
		return Message{}, fmt.Errorf("empty choices")
	}

	return openAIResp.Choices[0].Message, nil
}

// --- Anthropic translation ---

func translateToAnthropic(messages []Message) (systemPrompt string, result []AnthropicMessage) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			systemPrompt = m.Content
		case "tool":
			appendToolResultAnthropic(&result, m)
		default:
			result = append(result, anthropicFromMessage(m))
		}
	}
	return
}

func appendToolResultAnthropic(result *[]AnthropicMessage, m Message) {
	content := AnthropicContent{Type: "tool_result", ToolUseID: m.ToolCallID, Content: m.Content}
	msgs := *result
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" {
		msgs[len(msgs)-1].Content = append(msgs[len(msgs)-1].Content, content)
	} else {
		*result = append(msgs, AnthropicMessage{Role: "user", Content: []AnthropicContent{content}})
	}
}

func anthropicFromMessage(m Message) AnthropicMessage {
	am := AnthropicMessage{Role: m.Role}
	if len(m.ToolCalls) == 0 {
		am.Content = []AnthropicContent{{Type: "text", Text: m.Content}}
		return am
	}
	if m.Content != "" {
		am.Content = append(am.Content, AnthropicContent{Type: "text", Text: m.Content})
	}
	for _, tc := range m.ToolCalls {
		var input map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &input)
		am.Content = append(am.Content, AnthropicContent{
			Type: "tool_use", ID: tc.ID, Name: tc.Function.Name, Input: input,
		})
	}
	return am
}

func translateToolsToAnthropic(tools []Tool) []AnthropicTool {
	out := make([]AnthropicTool, len(tools))
	for i, t := range tools {
		out[i] = AnthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		}
	}
	return out
}

func parseAnthropicResponse(resp AnthropicResponse) Message {
	resMsg := Message{Role: "assistant"}
	for _, c := range resp.Content {
		switch c.Type {
		case "text":
			resMsg.Content += c.Text
		case "tool_use":
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
	return resMsg
}

func (g *Gateway) chatAnthropic(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	systemPrompt, anthropicMessages := translateToAnthropic(messages)

	reqBody := AnthropicRequest{
		Model:     model,
		Messages:  anthropicMessages,
		System:    systemPrompt,
		MaxTokens: 4096,
		Tools:     translateToolsToAnthropic(tools),
	}

	var anthropicResp AnthropicResponse
	if err := g.doProviderRequest(ctx, providerCall{
		url:      g.AnthropicURL,
		body:     reqBody,
		headers:  map[string]string{"Content-Type": "application/json", "x-api-key": g.AnthropicKey, "anthropic-version": "2023-06-01"},
		provider: "Anthropic",
		target:   &anthropicResp,
	}); err != nil {
		return Message{}, err
	}

	return parseAnthropicResponse(anthropicResp), nil
}

// --- Gemini translation ---

func translateToGemini(messages []Message) (systemInstructions string, result []GeminiContent) {
	for _, m := range messages {
		switch {
		case m.Role == "system":
			systemInstructions = m.Content
		case m.Role == "tool":
			result = append(result, geminiFromToolResult(m))
		case len(m.ToolCalls) > 0:
			result = append(result, geminiFromToolCalls(m)...)
		default:
			result = append(result, geminiFromText(m))
		}
	}
	return
}

func geminiFromToolResult(m Message) GeminiContent {
	return GeminiContent{
		Role: "function",
		Parts: []GeminiPart{{
			FunctionResponse: &GeminiFunctionResponse{
				Name:     m.Name,
				Response: map[string]any{"content": m.Content},
			},
		}},
	}
}

func geminiFromToolCalls(m Message) []GeminiContent {
	var out []GeminiContent
	for _, tc := range m.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		out = append(out, GeminiContent{
			Role: "model",
			Parts: []GeminiPart{{
				FunctionCall: &GeminiFunctionCall{Name: tc.Function.Name, Args: args},
			}},
		})
	}
	return out
}

func geminiFromText(m Message) GeminiContent {
	role := m.Role
	if role == "assistant" {
		role = "model"
	}
	return GeminiContent{
		Role:  role,
		Parts: []GeminiPart{{Text: m.Content}},
	}
}

func translateToolsToGemini(tools []Tool) []GeminiTool {
	if len(tools) == 0 {
		return nil
	}
	declarations := make([]FunctionDefinition, len(tools))
	for i, t := range tools {
		declarations[i] = t.Function
	}
	return []GeminiTool{{FunctionDeclarations: declarations}}
}

func parseGeminiResponse(resp GeminiResponse) Message {
	resMsg := Message{Role: "assistant"}
	modelContent := resp.Candidates[0].Content
	for _, p := range modelContent.Parts {
		if p.Text != "" {
			resMsg.Content += p.Text
		}
		if p.FunctionCall != nil {
			args, _ := json.Marshal(p.FunctionCall.Args)
			resMsg.ToolCalls = append(resMsg.ToolCalls, ToolCall{
				ID:   p.FunctionCall.Name,
				Type: "function",
				Function: ToolCallFunction{
					Name:      p.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}
	}
	return resMsg
}

func (g *Gateway) chatGemini(ctx context.Context, messages []Message, tools []Tool, model string) (Message, error) {
	systemInstructions, geminiContents := translateToGemini(messages)

	reqBody := GeminiRequest{
		Contents: geminiContents,
		Tools:    translateToolsToGemini(tools),
	}

	type advancedGeminiRequest struct {
		GeminiRequest
		SystemInstruction *GeminiContent `json:"system_instruction,omitempty"`
	}

	advReq := advancedGeminiRequest{GeminiRequest: reqBody}
	if systemInstructions != "" {
		advReq.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemInstructions}},
		}
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.GeminiURL, model, g.GeminiKey)

	var geminiResp GeminiResponse
	if err := g.doProviderRequest(ctx, providerCall{
		url:      url,
		body:     advReq,
		headers:  map[string]string{"Content-Type": "application/json"},
		provider: "Gemini",
		target:   &geminiResp,
	}); err != nil {
		return Message{}, err
	}

	if len(geminiResp.Candidates) == 0 {
		return Message{}, fmt.Errorf("Gemini empty candidates")
	}

	return parseGeminiResponse(geminiResp), nil
}

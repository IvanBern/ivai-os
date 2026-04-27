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

// API Payloads
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Gateway manages LLM execution
type Gateway struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewGateway creates a new instance with a custom HTTP client and timeout
func NewGateway(apiKey string) *Gateway {
	return &Gateway{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second, // Agents need longer timeouts for heavy reasoning
		},
	}
}

// GenerateText sends the prompt to the DeepSeek API
func (g *Gateway) GenerateText(ctx context.Context, prompt string, model string) (string, error) {
	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are Ivai, an advanced AI Operating System. Be concise and precise."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.APIKey)

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned empty choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

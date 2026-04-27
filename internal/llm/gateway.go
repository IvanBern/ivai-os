package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LLMClient defines the interface for interacting with various language models.
type LLMClient interface {
	GenerateText(ctx context.Context, prompt string, model string) (string, error)
}

// Gateway manages LLM requests with built-in resilience.
type Gateway struct {
	httpClient *http.Client
	apiKey     string
}

// NewGateway creates a new instance of the LLM Gateway.
func NewGateway(apiKey string) *Gateway {
	return &Gateway{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// GenerateText sends a completion request to the DeepSeek API.
// Note: This is a boilerplate implementation with retry logic placeholders.
func (g *Gateway) GenerateText(ctx context.Context, prompt string, model string) (string, error) {
	url := "https://api.deepseek.com/chat/completions"
	
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiKey))

	// Implementation of basic retry logic placeholder
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = g.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if i < 2 {
			time.Sleep(time.Duration(1<<i) * time.Second) // Exponential backoff
		}
	}

	if err != nil {
		return "", fmt.Errorf("request failed after retries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %s", resp.Status)
	}

	// Parsing would go here
	return "Mock response from DeepSeek", nil
}

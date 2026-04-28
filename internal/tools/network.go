package tools

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HttpRequest performs a basic HTTP request and returns the response body as a string.
func HttpRequest(method, url, body string, headers map[string]string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return string(respBody), fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return string(respBody), nil
}

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type TaskRequest struct {
	Instruction string `json:"instruction"`
}

type progressEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func main() {
	if err := run(os.Args, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader) error {
	streamMode := false
	instructionArgs := args[1:]

	// Parse --stream flag
	var filtered []string
	for _, a := range instructionArgs {
		if a == "--stream" {
			streamMode = true
		} else {
			filtered = append(filtered, a)
		}
	}

	instruction, err := readInstruction(filtered, stdin)
	if err != nil {
		return err
	}
	if instruction == "" {
		return fmt.Errorf("no instruction provided")
	}

	if streamMode {
		return streamTask(instruction)
	}
	return sendTask(instruction)
}

func readInstruction(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	fmt.Println("Enter instruction for Ivai OS (Ctrl+D to finish):")
	body, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("error reading stdin: %v", err)
	}
	return strings.TrimSpace(string(body)), nil
}

func buildAPIURL() string {
	if url := os.Getenv("IVAI_API_URL"); url != "" {
		return url
	}
	port := os.Getenv("IVAI_PORT")
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("http://localhost:%s/api/task", port)
}

func buildStreamURL() string {
	base := buildAPIURL()
	return strings.TrimSuffix(base, "/api/task") + "/api/task/stream"
}

func sendTask(instruction string) error {
	reqBody := TaskRequest{Instruction: instruction}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	apiURL := buildAPIURL()

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error communicating with Ivai OS: %v\nHint: Ensure Ivai OS is running and port 8080 is accessible", err)
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

func handleResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		var result struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error decoding response: %v", err)
		}
		fmt.Printf("\n[Ivai] %s\n", result.Response)
		return nil
	case http.StatusAccepted:
		fmt.Println("✅ Task accepted by Ivai OS (processing in background).")
		return nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ Failed to send task. Status: %d, Response: %s", resp.StatusCode, string(body))
	}
}

// --- SSE Streaming ---

func streamTask(instruction string) error {
	reqBody := TaskRequest{Instruction: instruction}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	streamURL := buildStreamURL()
	resp, err := http.Post(streamURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error connecting to Ivai OS stream: %v\nHint: Ensure Ivai OS is running and port 8080 is accessible", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ Stream failed. Status: %d, Response: %s", resp.StatusCode, string(body))
	}

	return readSSEStream(resp.Body)
}

func readSSEStream(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	var eventType string
	var dataBuffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if dataBuffer.Len() > 0 {
				printSSEEvent(eventType, dataBuffer.String())
				eventType = ""
				dataBuffer.Reset()
			}
			continue
		}

		if after, ok := strings.CutPrefix(line, "event: "); ok {
			eventType = after
		} else if after, ok := strings.CutPrefix(line, "data: "); ok {
			data := after
			dataBuffer.WriteString(data)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading SSE stream: %v", err)
	}
	return nil
}

func printSSEEvent(eventType string, data string) {
	var evt progressEvent
	json.Unmarshal([]byte(data), &evt)

	switch eventType {
	case "task_start":
		printTaskStart(evt)
	case "thinking":
		printThinking(evt)
	case "tool_call":
		printToolCall(evt)
	case "tool_result":
		printToolResult(evt)
	case "task_complete":
		printTaskComplete(evt)
	case "task_error":
		printTaskError(evt)
	default:
		fmt.Printf("[%s] %s\n", eventType, data)
	}
}

func eventData(evt progressEvent) map[string]any {
	d, _ := evt.Data.(map[string]any)
	return d
}

func printTaskStart(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	fmt.Printf("\n[model] %v\n", d["model"])
	fmt.Printf("[instruction] %v\n", d["instruction"])
}

func printThinking(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	if r, _ := d["reasoning"].(string); r != "" {
		fmt.Printf("[thinking] %s\n", r)
	}
	if c, _ := d["content"].(string); c != "" {
		fmt.Printf("[thinking] %s\n", c)
	}
}

func printToolCall(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	name, _ := d["name"].(string)
	args, _ := d["args"].(string)
	fmt.Printf("[tool] %s → %s\n", name, args)
}

func printToolResult(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	name, _ := d["name"].(string)
	result, _ := d["result"].(string)
	fmt.Printf("[tool result] %s → %s\n", name, result)
}

func printTaskComplete(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	fmt.Printf("\n[complete] %v\n", d["response"])
}

func printTaskError(evt progressEvent) {
	d := eventData(evt)
	if d == nil {
		return
	}
	fmt.Printf("\n[error] %v\n", d["error"])
}

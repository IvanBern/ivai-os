package main

import (
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

func main() {
	if err := run(os.Args, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader) error {
	instruction, err := readInstruction(args, stdin)
	if err != nil {
		return err
	}
	if instruction == "" {
		return fmt.Errorf("no instruction provided")
	}
	return sendTask(instruction)
}

func readInstruction(args []string, stdin io.Reader) (string, error) {
	if len(args) > 1 {
		return strings.Join(args[1:], " "), nil
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

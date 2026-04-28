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
	var instruction string

	if len(args) > 1 {
		instruction = strings.Join(args[1:], " ")
	} else {
		// Read from stdin
		fmt.Println("Enter instruction for Ivai OS (Ctrl+D to finish):")
		body, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("error reading stdin: %v", err)
		}
		instruction = strings.TrimSpace(string(body))
	}

	if instruction == "" {
		return fmt.Errorf("no instruction provided")
	}

	reqBody := TaskRequest{Instruction: instruction}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	// We use localhost:8080 as the default, which works with OrbStack port mapping
	apiURL := os.Getenv("IVAI_API_URL")
	if apiURL == "" {
		port := os.Getenv("IVAI_PORT")
		if port == "" {
			port = "8080"
		}
		apiURL = fmt.Sprintf("http://localhost:%s/api/task", port)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error communicating with Ivai OS: %v\nHint: Ensure Ivai OS is running and port 8080 is accessible", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error decoding response: %v", err)
		}
		fmt.Printf("\n[Ivai] %s\n", result.Response)
		return nil
	} else if resp.StatusCode == http.StatusAccepted {
		fmt.Println("✅ Task accepted by Ivai OS (processing in background).")
		return nil
	} else {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ Failed to send task. Status: %d, Response: %s", resp.StatusCode, string(body))
	}
}

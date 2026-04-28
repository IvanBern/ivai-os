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
	var instruction string

	if len(os.Args) > 1 {
		instruction = strings.Join(os.Args[1:], " ")
	} else {
		// Read from stdin
		fmt.Println("Enter instruction for Ivai OS (Ctrl+D to finish):")
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		instruction = strings.TrimSpace(string(body))
	}

	if instruction == "" {
		fmt.Println("No instruction provided.")
		os.Exit(1)
	}

	reqBody := TaskRequest{Instruction: instruction}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	// We use localhost:8080 as the default, which works with OrbStack port mapping
	resp, err := http.Post("http://localhost:8080/api/task", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error communicating with Ivai OS: %v\n", err)
		fmt.Println("Hint: Ensure Ivai OS is running and port 8080 is accessible.")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		fmt.Println("✅ Task accepted by Ivai OS.")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Failed to send task. Status: %d, Response: %s\n", resp.StatusCode, string(body))
	}
}

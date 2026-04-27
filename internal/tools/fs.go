package tools

import (
	"os"
)

// ReadFile reads the content of a file from the disk
func ReadFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a file, creating it if it doesn't exist
func WriteFile(filepath, content string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

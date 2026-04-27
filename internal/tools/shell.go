package tools

import (
	"os/exec"
)

// ExecuteCommand runs a bash shell command and returns the output or the error message
func ExecuteCommand(command string) (string, error) {
	// We wrap the command in bash -c so Ivai can use pipes (|) and standard bash features
	cmd := exec.Command("bash", "-c", command)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Return the error alongside the output so the LLM knows *why* it failed
		return string(out) + "\nError: " + err.Error(), nil
	}

	return string(out), nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/IvanBern/ivai-os/internal/sandbox"
	"github.com/IvanBern/ivai-os/internal/tools"
)

type toolHandler func(ctx context.Context, args string, wasmEngine *sandbox.WasmRuntime) (string, error)

var toolRegistry = map[string]toolHandler{
	"read_file":       handleReadFile,
	"write_file":      handleWriteFile,
	"execute_command": handleExecCommand,
	"http_request":    handleHTTPReq,
	"execute_wasm":    handleWasm,
	"github_pr":       func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeGitHubPR(a) },
	"code_health":     func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeCodeHealthTool(a) },
	"create_issue":    func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeCreateIssue(a) },
	"list_issues":     func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeListIssues(a) },
	"update_wiki":     func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeUpdateWiki(a) },
	"swarm_clone":     func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmClone(a) },
	"swarm_deploy":    func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmDeploy(a) },
	"swarm_dispatch":  func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmDispatch(a) },
	"swarm_gather":    func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmGather(a) },
	"swarm_status":    func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmStatus(a) },
	"swarm_spawn":     func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmSpawn(a) },
	"swarm_kill":      func(_ context.Context, a string, _ *sandbox.WasmRuntime) (string, error) { return executeSwarmKill(a) },
}

func handleReadFile(_ context.Context, args string, _ *sandbox.WasmRuntime) (string, error) {
	var a struct{ Filepath string `json:"filepath"` }
	json.Unmarshal([]byte(args), &a)
	return tools.ReadFile(a.Filepath)
}

func handleWriteFile(_ context.Context, args string, _ *sandbox.WasmRuntime) (string, error) {
	var a struct {
		Filepath string `json:"filepath"`
		Content  string `json:"content"`
	}
	json.Unmarshal([]byte(args), &a)
	return "ok", tools.WriteFile(a.Filepath, a.Content)
}

func handleExecCommand(_ context.Context, args string, _ *sandbox.WasmRuntime) (string, error) {
	var a struct{ Command string `json:"command"` }
	json.Unmarshal([]byte(args), &a)
	return tools.ExecuteCommand(a.Command)
}

func handleHTTPReq(_ context.Context, args string, _ *sandbox.WasmRuntime) (string, error) {
	var a struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Body    string            `json:"body"`
		Headers map[string]string `json:"headers"`
	}
	json.Unmarshal([]byte(args), &a)
	return tools.HttpRequest(a.Method, a.URL, a.Body, a.Headers)
}

func handleWasm(ctx context.Context, args string, wasmEngine *sandbox.WasmRuntime) (string, error) {
	var a struct {
		Filepath  string `json:"filepath"`
		Payload   string `json:"payload"`
		TimeoutMs int    `json:"timeout_ms"`
	}
	json.Unmarshal([]byte(args), &a)
	b, err := os.ReadFile(a.Filepath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return wasmEngine.Execute(ctx, b, a.Payload, a.TimeoutMs)
}

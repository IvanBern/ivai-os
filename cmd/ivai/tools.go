package main

import (
	"context"
	"fmt"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/sandbox"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func defineTool(name, desc string, props map[string]any, required []string) llm.Tool {
	return llm.Tool{
		Type: "function",
		Function: llm.FunctionDefinition{
			Name:        name,
			Description: desc,
			Parameters: map[string]any{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		},
	}
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func buildTools() []llm.Tool {
	tools := buildCoreTools()
	tools = append(tools, buildGitHubTools()...)
	tools = append(tools, buildSwarmTools()...)
	return tools
}

func buildCoreTools() []llm.Tool {
	return []llm.Tool{
		defineTool("read_file", "Reads the contents of a file at the given path on the local filesystem.",
			map[string]any{"filepath": map[string]any{"type": "string"}},
			[]string{"filepath"}),

		defineTool("write_file", "Writes text content to a file at the given path, overwriting it if it exists.",
			map[string]any{
				"filepath": map[string]any{"type": "string"},
				"content":  map[string]any{"type": "string"},
			},
			[]string{"filepath", "content"}),

		defineTool("execute_command", "Executes a bash shell command on the host Debian system and returns the output.",
			map[string]any{"command": map[string]any{"type": "string"}},
			[]string{"command"}),

		defineTool("execute_wasm", "Executes a compiled WebAssembly (.wasm) binary in a secure, isolated sandbox with strict timeouts. Passes data via stdin and returns stdout.",
			map[string]any{
				"filepath":   strProp("Absolute path to the .wasm file on disk"),
				"payload":    strProp("Data to send to the Wasm module via standard input (stdin)"),
				"timeout_ms": map[string]any{"type": "integer", "description": "Execution timeout in milliseconds (e.g., 1000)"},
			},
			[]string{"filepath", "payload", "timeout_ms"}),

		defineTool("http_request", "Performs an HTTP request (GET, POST, etc.) and returns the response body.",
			map[string]any{
				"method":  strProp("HTTP method (e.g., GET, POST)"),
				"url":     strProp("Target URL"),
				"body":    strProp("Request body (optional)"),
				"headers": map[string]any{"type": "object", "description": "HTTP headers (optional)"},
			},
			[]string{"method", "url"}),
	}
}

func buildGitHubTools() []llm.Tool {
	return []llm.Tool{
		defineTool("github_pr", "Creates a GitHub Pull Request from a repo directory. Uses the gh CLI (must be authenticated). Provide title, body, repo path, and optionally a base branch.",
			map[string]any{
				"title": strProp("Pull request title"),
				"body":  strProp("Pull request description"),
				"base":  strProp("Target branch (default: main)"),
				"repo":  strProp("Path to the git repository (e.g., /tmp/ivai-sandbox)"),
			},
			[]string{"title", "body", "repo"}),

		defineTool("code_health", "Runs CodeScene delta analysis on a git repository to check code health. Returns issues found or No issues found!.",
			map[string]any{
				"repo": strProp("Path to the git repository (e.g., /tmp/ivai-sandbox)"),
			},
			[]string{"repo"}),

		defineTool("create_issue", "Creates a GitHub Issue. Uses gh CLI. Provide title, body, labels (comma-separated), and optional assignee.",
			map[string]any{
				"title":    strProp("Issue title"),
				"body":     strProp("Issue description"),
				"labels":   strProp("Comma-separated labels (e.g., bug,phase-13)"),
				"assignee": strProp("GitHub username to assign (optional)"),
			},
			[]string{"title", "body"}),

		defineTool("list_issues", "Lists GitHub Issues with optional filters.",
			map[string]any{
				"state":  strProp("open, closed, or all (default: open)"),
				"labels": strProp("Comma-separated label filter (optional)"),
				"limit":  strProp("Max issues to return (default: 10)"),
			},
			[]string{}),

		defineTool("update_wiki", "Updates a GitHub Wiki page by cloning the wiki repo, writing a markdown file, committing, and pushing. Provide page title and markdown content.",
			map[string]any{
				"page":    strProp("Wiki page title (creates page.md)"),
				"content": strProp("Markdown content for the page"),
			},
			[]string{"page", "content"}),
	}
}

func buildSwarmTools() []llm.Tool {
	return []llm.Tool{
		defineTool("swarm_clone", "Clones the ivai-os-linux VM to create a new worker VM. Provide a name for the new worker.",
			map[string]any{
				"name": strProp("Name for the new worker VM (e.g., ivai-worker-1)"),
			},
			[]string{"name"}),

		defineTool("swarm_deploy", "Deploys the latest Ivai binary to a worker VM and starts the service.",
			map[string]any{
				"name": strProp("Worker VM name"),
			},
			[]string{"name"}),

		defineTool("swarm_dispatch", "Sends a task to a worker VM for execution.",
			map[string]any{
				"worker":      strProp("Worker VM hostname or IP (e.g., 192.168.139.x)"),
				"instruction": strProp("Task instruction to execute"),
			},
			[]string{"worker", "instruction"}),

		defineTool("swarm_gather", "Collects task results from a worker VM.",
			map[string]any{
				"worker": strProp("Worker VM hostname or IP"),
			},
			[]string{"worker"}),

		defineTool("swarm_status", "Checks status of a worker VM or lists all VMs if no name provided.",
			map[string]any{
				"name": strProp("Optional VM name. Leave empty to list all VMs."),
			},
			[]string{}),

		defineTool("swarm_spawn", "Spawns a local Ivai worker process on the same host. Much faster than VM workers. Provide name and optional port.",
			map[string]any{
				"name": strProp("Worker name (data dir will be /tmp/ivai-<name>)"),
				"port": strProp("Port for the worker (default: 8081)"),
			},
			[]string{"name"}),

		defineTool("swarm_kill", "Kills a local Ivai worker process by port or name.",
			map[string]any{
				"port": strProp("Port of the worker to kill"),
				"name": strProp("Name of the worker to kill"),
			},
			[]string{}),
	}
}

func executeToolCall(ctx context.Context, tc llm.ToolCall, wasmEngine *sandbox.WasmRuntime) string {
	tracer := otel.Tracer("ivai-os")
	ctx, span := tracer.Start(ctx, "tool."+tc.Function.Name,
		trace.WithAttributes(
			attribute.String("tool.name", tc.Function.Name),
			attribute.Int("tool.args_len", len(tc.Function.Arguments)),
		),
	)
	defer span.End()
	if h, ok := toolRegistry[tc.Function.Name]; ok {
		return resultOrError(h(ctx, tc.Function.Arguments, wasmEngine))
	}
	return fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
}

func resultOrError(result string, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return result
}

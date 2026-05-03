package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/IvanBern/ivai-os/internal/llm"
	"github.com/IvanBern/ivai-os/internal/memory"
	"github.com/IvanBern/ivai-os/internal/sandbox"
	"github.com/mattn/go-isatty"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TaskInput struct {
	Instruction string
	Gateway     *llm.Gateway
	DBStore     *memory.Store
	WasmEngine  *sandbox.WasmRuntime
}

type taskState struct {
	instruction  string
	model        string
	gateway      *llm.Gateway
	dbStore      *memory.Store
	wasmEngine   *sandbox.WasmRuntime
	tools        []llm.Tool
	progressChan chan<- ProgressEvent
}

func (s *taskState) emit(evt ProgressEvent) {
	if s.progressChan == nil {
		return
	}
	select {
	case s.progressChan <- evt:
	default:
	}
}

func processTask(ctx context.Context, in TaskInput, progressChan chan<- ProgressEvent) string {
	if in.Gateway == nil {
		return handleGatewayMissing(progressChan)
	}

	model, instruction := extractModel(in.Instruction)
	slog.Info("Task routing", "model", model, "instruction", instruction)

	if in.DBStore != nil {
		in.DBStore.SaveMessage("user", instruction, "")
	}

	state := &taskState{
		instruction:  instruction,
		model:        model,
		gateway:      in.Gateway,
		dbStore:      in.DBStore,
		wasmEngine:   in.WasmEngine,
		tools:        buildTools(),
		progressChan: progressChan,
	}

	state.emit(ProgressEvent{Type: "task_start", Message: "Task started", Data: map[string]string{"model": state.model, "instruction": state.instruction}})

	startTime := time.Now()
	payload := buildPayload(in.DBStore, in.Gateway)
	result := runReasoningLoop(ctx, payload, state)
	duration := time.Since(startTime).Milliseconds()

	recordTaskResult(state, result, duration)

	if progressChan != nil {
		close(progressChan)
	}

	return result
}

func handleGatewayMissing(progressChan chan<- ProgressEvent) string {
	errMsg := "Error: LLM gateway is not configured"
	slog.Error(errMsg)
	if progressChan != nil {
		close(progressChan)
	}
	return errMsg
}

func recordTaskResult(s *taskState, result string, durationMs int64) {
	if s.dbStore == nil {
		return
	}
	success := !strings.HasPrefix(result, "Error: ")
	errMsg := ""
	if !success {
		errMsg = result
	}
	s.dbStore.SaveTaskResult(memory.TaskResult{
		Instruction: s.instruction,
		Model:       s.model,
		Success:     success,
		Response:    result,
		ErrorMsg:    errMsg,
		DurationMs:  durationMs,
	})

	go func() {
		emb, embedErr := s.gateway.Embed(context.Background(), s.instruction)
		if embedErr != nil {
			return
		}
		s.dbStore.SaveEmbedding("instruction", s.instruction, emb)
	}()
}

func extractModel(t string) (model, instruction string) {
	model = "deepseek-v4-pro"
	instruction = t

	lower := strings.ToLower(t)
	switch {
	case strings.Contains(lower, "@claude"):
		return "claude-3-5-sonnet-20241022", strings.Replace(t, "@claude", "", 1)
	case strings.Contains(lower, "@gemini"):
		return "gemini-2.5-pro", strings.Replace(t, "@gemini", "", 1)
	case strings.Contains(lower, "@deepseek"):
		return "deepseek-v4-pro", strings.Replace(t, "@deepseek", "", 1)
	case strings.Contains(lower, "@research"):
		return "deep-research-max-preview", strings.Replace(t, "@research", "", 1)
	default:
		return model, instruction
	}
}

func buildPayload(dbStore *memory.Store, gateway *llm.Gateway) []llm.Message {
	var history []memory.Message
	if dbStore != nil {
		history, _ = dbStore.GetRecentMessages(10)
	}
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	payload := []llm.Message{
		{Role: "system", Content: fmt.Sprintf(systemPromptTemplate, homeDir, cwd)},
	}

	if dbStore != nil {
		payload = injectRAGContext(payload, dbStore, gateway, history)
	}

	for _, msg := range history {
		payload = append(payload, llm.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		})
	}
	return payload
}

func injectRAGContext(payload []llm.Message, dbStore *memory.Store, gateway *llm.Gateway, history []memory.Message) []llm.Message {
	if len(history) == 0 {
		return payload
	}
	latestMsg := history[len(history)-1].Content
	emb, err := gateway.Embed(context.Background(), latestMsg)
	if err != nil {
		return payload
	}
	similar, err := dbStore.SearchSimilar(emb, 3)
	if err != nil || len(similar) == 0 {
		return payload
	}
	ragCtx := "## Relevant Past Context (from semantic memory)\n"
	for i, s := range similar {
		ragCtx += fmt.Sprintf("%d. [%.0f%% match] %s\n", i+1, s.Similarity*100, s.Content)
	}
	return append(payload, llm.Message{Role: "system", Content: ragCtx})
}

func runReasoningLoop(ctx context.Context, payload []llm.Message, s *taskState) string {
	tracer := otel.Tracer("ivai-os")
	for {
		ctx, span := tracer.Start(ctx, "reasoning-step",
			trace.WithAttributes(
				attribute.String("model", s.model),
				attribute.Int("messages", len(payload)),
			),
		)
		responseMsg, err := s.gateway.Chat(ctx, payload, s.tools, s.model)
		if err != nil {
			slog.Error("LLM Execution Failed", "error", err)
			s.emit(ProgressEvent{Type: "task_error", Message: "LLM error", Data: map[string]string{"error": err.Error()}})
			span.End()
			printPrompt()
			return "Error: " + err.Error()
		}

		if done, result := checkCompletion(responseMsg, s); done {
			span.End()
			return result
		}

		showThinking(responseMsg.ReasoningContent)
		s.emit(ProgressEvent{
			Type:    "thinking",
			Message: "Model is thinking",
			Data:    map[string]string{"reasoning": responseMsg.ReasoningContent, "content": responseMsg.Content},
		})
		payload = append(payload, responseMsg)
		payload = appendToolResults(ctx, payload, responseMsg.ToolCalls, s)
		span.End()
	}
}

func checkCompletion(msg llm.Message, s *taskState) (done bool, result string) {
	if len(msg.ToolCalls) > 0 {
		return false, ""
	}
	slog.Info("Task completed", "response_length", len(msg.Content))
	if s.dbStore != nil {
		s.dbStore.SaveMessage("assistant", msg.Content, msg.ReasoningContent)
	}
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Printf("\n[Ivai] %s\n", msg.Content)
	}
	printPrompt()
	return true, msg.Content
}

func showThinking(reasoningContent string) {
	if reasoningContent != "" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			fmt.Printf("\n[Thinking] %s\n", reasoningContent)
		} else {
			slog.Info("Thinking", "reasoning", reasoningContent)
		}
	} else {
		slog.Info("Thinking...")
	}
}

func appendToolResults(ctx context.Context, payload []llm.Message, toolCalls []llm.ToolCall, s *taskState) []llm.Message {
	for _, tc := range toolCalls {
		slog.Info("Executing tool", "name", tc.Function.Name, "args", tc.Function.Arguments)
		s.emit(ProgressEvent{
			Type:    "tool_call",
			Message: fmt.Sprintf("Calling tool: %s", tc.Function.Name),
			Data:    map[string]any{"name": tc.Function.Name, "args": tc.Function.Arguments},
		})
		toolResult := executeToolCall(ctx, tc, s.wasmEngine)
		s.emit(ProgressEvent{
			Type:    "tool_result",
			Message: fmt.Sprintf("Tool result: %s", tc.Function.Name),
			Data:    map[string]any{"name": tc.Function.Name, "result": truncate(toolResult, 500)},
		})
		payload = append(payload, llm.Message{
			Role:       "tool",
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
	}
	return payload
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

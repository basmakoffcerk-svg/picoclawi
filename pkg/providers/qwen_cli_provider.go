package providers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// QwenCliProvider implements LLMProvider using the qwen CLI as a subprocess.
//
// OAuth flow is handled by the qwen CLI itself:
// run `qwen` once and complete browser auth, then credentials are cached locally.
type QwenCliProvider struct {
	command   string
	workspace string
}

// NewQwenCliProvider creates a new Qwen CLI provider.
func NewQwenCliProvider(workspace string) *QwenCliProvider {
	return &QwenCliProvider{
		command:   "qwen",
		workspace: workspace,
	}
}

// Chat executes qwen in non-interactive prompt mode.
func (p *QwenCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if p.command == "" {
		return nil, fmt.Errorf("qwen command not configured")
	}

	prompt := p.buildPrompt(messages, tools)

	args := []string{"-p", prompt}
	useModelFlag := model != "" && model != "qwen-cli"
	if useModelFlag {
		args = append(args, "--model", model)
	}

	stdout, stderr, err := p.run(ctx, args, prompt)
	if err != nil && useModelFlag && isUnknownModelFlag(stderr) {
		// Backward compatibility: older qwen CLI builds may not support --model.
		stdout, stderr, err = p.run(ctx, []string{"-p", prompt}, prompt)
	}
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		stderr = strings.TrimSpace(stderr)
		if stderr != "" {
			return nil, fmt.Errorf("qwen cli error: %s", stderr)
		}
		return nil, fmt.Errorf("qwen cli error: %w", err)
	}

	content := strings.TrimSpace(stdout)
	if content == "" {
		stderr = strings.TrimSpace(stderr)
		if stderr != "" {
			return nil, fmt.Errorf("qwen cli returned empty response: %s", stderr)
		}
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	toolCalls := extractToolCallsFromText(content)
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = stripToolCallsFromText(content)
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}, nil
}

func (p *QwenCliProvider) run(ctx context.Context, args []string, _ string) (string, string, error) {
	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// GetDefaultModel returns the default model identifier.
func (p *QwenCliProvider) GetDefaultModel() string {
	return "qwen-cli"
}

func (p *QwenCliProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	systemPrompt := p.buildSystemPrompt(messages, tools)
	conversationPrompt := p.messagesToPrompt(messages)

	if systemPrompt == "" {
		return conversationPrompt
	}

	return strings.TrimSpace(systemPrompt + "\n\n" + conversationPrompt)
}

func (p *QwenCliProvider) buildSystemPrompt(messages []Message, tools []ToolDefinition) string {
	var parts []string

	for _, msg := range messages {
		if msg.Role == "system" {
			parts = append(parts, msg.Content)
		}
	}

	if len(tools) > 0 {
		parts = append(parts, buildCLIToolsPrompt(tools))
	}

	return strings.Join(parts, "\n\n")
}

func (p *QwenCliProvider) messagesToPrompt(messages []Message) string {
	var parts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// handled in buildSystemPrompt
		case "user":
			parts = append(parts, "User: "+msg.Content)
		case "assistant":
			parts = append(parts, "Assistant: "+msg.Content)
		case "tool":
			parts = append(parts, fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
		}
	}

	if len(parts) == 1 && strings.HasPrefix(parts[0], "User: ") {
		return strings.TrimPrefix(parts[0], "User: ")
	}

	return strings.Join(parts, "\n")
}

func isUnknownModelFlag(stderr string) bool {
	s := strings.ToLower(stderr)
	if !strings.Contains(s, "model") {
		return false
	}
	return strings.Contains(s, "unknown option") ||
		strings.Contains(s, "unknown flag") ||
		strings.Contains(s, "unrecognized option") ||
		strings.Contains(s, "unknown argument")
}

package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

var _ LLMProvider = (*QwenCliProvider)(nil)

func createMockQwenCLI(t *testing.T, scriptBody string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "qwen")
	content := "#!/bin/sh\nset -eu\n" + scriptBody + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestNewQwenCliProvider(t *testing.T) {
	p := NewQwenCliProvider("/tmp/workspace")
	if p == nil {
		t.Fatal("NewQwenCliProvider returned nil")
	}
	if p.command != "qwen" {
		t.Fatalf("command = %q, want qwen", p.command)
	}
	if p.workspace != "/tmp/workspace" {
		t.Fatalf("workspace = %q, want /tmp/workspace", p.workspace)
	}
}

func TestQwenCliProvider_ChatSuccess(t *testing.T) {
	script := createMockQwenCLI(t, `echo "Hello from Qwen CLI"`)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello from Qwen CLI" {
		t.Fatalf("Content = %q, want %q", resp.Content, "Hello from Qwen CLI")
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want stop", resp.FinishReason)
	}
}

func TestQwenCliProvider_ChatToolCalls(t *testing.T) {
	script := createMockQwenCLI(t, `cat <<'EOF'
Checking weather.
{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Minsk\"}"}}]}
EOF`)

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "weather"}}, nil, "", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want get_weather", resp.ToolCalls[0].Name)
	}
	if got := resp.ToolCalls[0].Arguments["city"]; got != "Minsk" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want Minsk", got)
	}
}

func TestQwenCliProvider_ModelFlagFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock CLI scripts not supported on Windows")
	}

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.log")
	script := createMockQwenCLI(t, fmt.Sprintf(`echo "$@" >> '%s'
if printf '%%s' "$@" | grep -q -- '--model'; then
  echo "unknown option --model" >&2
  exit 2
fi
echo "ok without model flag"`, argsFile))

	p := NewQwenCliProvider(t.TempDir())
	p.command = script

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, "qwen3-coder-plus", nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "ok without model flag" {
		t.Fatalf("Content = %q, want %q", resp.Content, "ok without model flag")
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected 2 invocations, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "--model qwen3-coder-plus") {
		t.Fatalf("first invocation should include model flag, got: %q", lines[0])
	}
	if strings.Contains(lines[1], "--model") {
		t.Fatalf("second invocation should not include model flag, got: %q", lines[1])
	}
}

func TestCreateProviderFromConfig_QwenCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "qwen-oauth-cli",
		Model:     "qwen-cli/qwen3-coder-plus",
		Workspace: "/tmp/ws",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error: %v", err)
	}
	if _, ok := provider.(*QwenCliProvider); !ok {
		t.Fatalf("provider type = %T, want *QwenCliProvider", provider)
	}
	if modelID != "qwen3-coder-plus" {
		t.Fatalf("modelID = %q, want qwen3-coder-plus", modelID)
	}
}

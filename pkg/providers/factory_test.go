package providers

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestCreateProviderReturnsHTTPProviderForOpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openrouter"
	modelCfg := &config.ModelConfig{
		ModelName: "test-openrouter",
		Model:     "openrouter/auto",
		APIBase:   "https://openrouter.ai/api/v1",
	}
	modelCfg.SetAPIKey("sk-or-test")
	cfg.ModelList = []*config.ModelConfig{modelCfg}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}

func TestCreateProviderReturnsCodexCliProviderForCodexCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-codex"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-codex",
			Model:     "codex-cli/codex-model",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexCliProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeCliProviderForClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-cli"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-claude-cli",
			Model:     "claude-cli/claude-sonnet",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProviderReturnsQwenCliProviderForQwenCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-qwen-cli"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-qwen-cli",
			Model:     "qwen-cli/qwen3-coder-plus",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*QwenCliProvider); !ok {
		t.Fatalf("provider type = %T, want *QwenCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeProviderForAnthropicOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "anthropic" {
			t.Fatalf("provider = %q, want anthropic", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "anthropic-token",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-oauth"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-claude-oauth",
			Model:      "anthropic/claude-sonnet-4.6",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeProvider", provider)
	}
	// TODO: Test custom APIBase when createClaudeAuthProvider supports it
}

func TestCreateProviderReturnsCodexProviderForOpenAIOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "openai-codex" && provider != "openai" {
			t.Fatalf("provider = %q, want openai-codex or openai", provider)
		}
		if provider == "openai-codex" {
			return &auth.AuthCredential{
				AccessToken: "openai-codex-token",
				AccountID:   "acc-123",
			}, nil
		}
		return nil, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openai-oauth"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-openai-oauth",
			Model:      "openai/gpt-5.4",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexProvider", provider)
	}
}

func TestCreateProviderReturnsCodexProviderForOpenAICodexProtocol(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "openai-codex" {
			t.Fatalf("provider = %q, want openai-codex", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "openai-codex-token",
			AccountID:   "acc-123",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openai-codex"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-openai-codex",
			Model:      "openai-codex/gpt-5.4",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexProvider", provider)
	}
}

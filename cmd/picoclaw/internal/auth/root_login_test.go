package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewRootLoginCommand(t *testing.T) {
	cmd := NewRootLoginCommand()
	require.NotNil(t, cmd)

	assert.Equal(t, "login", cmd.Use)
	assert.Equal(t, "Interactive provider login and LLM setup", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Flags().Lookup("provider"))
	assert.NotNil(t, cmd.Flags().Lookup("model"))
	assert.NotNil(t, cmd.Flags().Lookup("device-code"))
	assert.NotNil(t, cmd.Flags().Lookup("setup-token"))
}

func TestProviderModelCandidates(t *testing.T) {
	cfg := &config.Config{
		ModelList: config.SecureModelList{
			{ModelName: "gpt-5.4", Model: "openai/gpt-5.4", Enabled: true},
			{ModelName: "gpt-5.4-codex", Model: "openai-codex/gpt-5.4", Enabled: true},
			{ModelName: "qwen-oauth-cli", Model: "qwen-cli/qwen3-coder-plus", Enabled: true},
			{ModelName: "claude-sonnet-4.6", Model: "anthropic/claude-sonnet-4.6", Enabled: false},
		},
	}

	assert.Equal(t, []string{"gpt-5.4-codex"}, providerModelCandidates(cfg, "openai-codex"))
	assert.Equal(t, []string{"qwen-oauth-cli"}, providerModelCandidates(cfg, "qwen-cli"))
	assert.Equal(t, []string{"gpt-5.4"}, providerModelCandidates(cfg, "openai"))
	assert.Empty(t, providerModelCandidates(cfg, "anthropic"))
}

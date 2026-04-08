package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

type providerOption struct {
	Key   string
	Label string
}

var rootLoginProviders = []providerOption{
	{Key: "openai-codex", Label: "OpenAI Codex (OAuth)"},
	{Key: "qwen-cli", Label: "Qwen CLI (OAuth via qwen)"},
	{Key: "openai", Label: "OpenAI (OAuth)"},
	{Key: "anthropic", Label: "Anthropic"},
	{Key: "google-antigravity", Label: "Google Antigravity"},
}

// NewRootLoginCommand creates "picoclaw login" shortcut with interactive provider/model setup.
func NewRootLoginCommand() *cobra.Command {
	var (
		provider      string
		modelName     string
		useDeviceCode bool
		useOauth      bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Interactive provider login and LLM setup",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			selectedProvider := strings.TrimSpace(provider)
			if selectedProvider == "" {
				p, err := promptProviderSelection()
				if err != nil {
					return err
				}
				selectedProvider = p
			}

			if err := authLoginCmd(selectedProvider, useDeviceCode, useOauth); err != nil {
				return err
			}

			cfg, err := internal.LoadConfig()
			if err != nil {
				return nil
			}

			candidates := providerModelCandidates(cfg, selectedProvider)
			if len(candidates) == 0 {
				return nil
			}

			selectedModel := strings.TrimSpace(modelName)
			if selectedModel == "" {
				if len(candidates) == 1 {
					selectedModel = candidates[0]
				} else {
					selectedModel, err = promptModelSelection(candidates)
					if err != nil {
						return err
					}
				}
			}

			_, err = setDefaultModelByName(cfg, selectedModel)
			return err
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider key (optional; if empty, opens menu)")
	cmd.Flags().StringVar(&modelName, "model", "", "Default model_name to set after login")
	cmd.Flags().BoolVar(&useDeviceCode, "device-code", false, "Use device code flow when supported")
	cmd.Flags().BoolVar(
		&useOauth,
		"setup-token",
		false,
		"Use setup-token flow for Anthropic (from claude setup-token)",
	)

	return cmd
}

func promptProviderSelection() (string, error) {
	fmt.Println("Select provider:")
	for i, p := range rootLoginProviders {
		fmt.Printf("  %d) %s\n", i+1, p.Label)
	}
	return chooseFromMenu("Choose provider [1]: ", rootLoginProviders, 0)
}

func promptModelSelection(models []string) (string, error) {
	if len(models) == 1 {
		return models[0], nil
	}

	fmt.Println("\nSelect default LLM:")
	options := make([]providerOption, 0, len(models))
	for i, m := range models {
		fmt.Printf("  %d) %s\n", i+1, m)
		options = append(options, providerOption{Key: m, Label: m})
	}
	return chooseFromMenu("Choose model [1]: ", options, 0)
}

func chooseFromMenu(prompt string, options []providerOption, defaultIdx int) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(prompt)
		choice := ""
		if scanner.Scan() {
			choice = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}

		if choice == "" {
			return options[defaultIdx].Key, nil
		}

		for i := range options {
			if choice == fmt.Sprintf("%d", i+1) {
				return options[i].Key, nil
			}
		}
		fmt.Printf("Invalid choice: %s\n", choice)
	}
}

func providerModelCandidates(cfg *config.Config, provider string) []string {
	if cfg == nil {
		return nil
	}

	prefixes := providerModelPrefixes(provider)
	seen := make(map[string]struct{})
	models := make([]string, 0, 4)

	for _, m := range cfg.ModelList {
		if m == nil || !m.Enabled || strings.TrimSpace(m.ModelName) == "" {
			continue
		}
		if !hasAnyPrefix(strings.ToLower(m.Model), prefixes) {
			continue
		}
		if _, ok := seen[m.ModelName]; ok {
			continue
		}
		seen[m.ModelName] = struct{}{}
		models = append(models, m.ModelName)
	}

	return models
}

func providerModelPrefixes(provider string) []string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai":
		return []string{"openai/"}
	case "openai-codex":
		return []string{"openai-codex/"}
	case "anthropic":
		return []string{"anthropic/"}
	case "google-antigravity", "antigravity":
		return []string{"google-antigravity/", "antigravity/"}
	case "qwen-cli", "qwen":
		return []string{"qwen-cli/"}
	default:
		return nil
	}
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(value, p) {
			return true
		}
	}
	return false
}

func setDefaultModelByName(cfg *config.Config, modelName string) (bool, error) {
	if strings.TrimSpace(modelName) == "" {
		return false, nil
	}

	found := false
	for _, model := range cfg.ModelList {
		if model != nil && model.Enabled && model.ModelName == modelName {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("cannot found model '%s' in config", modelName)
	}

	oldModel := cfg.Agents.Defaults.ModelName
	if oldModel == modelName {
		return false, nil
	}
	cfg.Agents.Defaults.ModelName = modelName
	if err := config.SaveConfig(internal.GetConfigPath(), cfg); err != nil {
		return false, fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("Default model changed from '%s' to '%s'\n", oldModel, modelName)
	return true, nil
}

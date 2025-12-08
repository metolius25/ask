package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider info for setup
var providerInfo = []struct {
	name string
	desc string
	url  string
}{
	{"gemini", "Google Gemini (free tier available)", "https://makersuite.google.com/app/apikey"},
	{"claude", "Anthropic Claude", "https://console.anthropic.com/"},
	{"chatgpt", "OpenAI ChatGPT", "https://platform.openai.com/api-keys"},
	{"deepseek", "DeepSeek (cost-effective)", "https://platform.deepseek.com/"},
	{"mistral", "Mistral AI", "https://console.mistral.ai/"},
	{"qwen", "Alibaba Qwen", "https://dashscope.console.aliyun.com/apiKey"},
}

// runInteractiveSetup guides first-time users through configuration
func runInteractiveSetup() {
	fmt.Println()
	fmt.Println("  Welcome to Ask! Let's set up your API keys.")
	fmt.Println("  Press Enter to skip any provider you don't want to configure.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Load existing config if any
	config := &Config{
		Providers: make(map[string]ProviderConfig),
	}

	// Try to load existing config
	if existing, err := LoadConfigSafe(); err == nil && existing != nil {
		config = existing
	}

	configuredCount := 0
	var firstProvider string

	for _, p := range providerInfo {
		existing := ""
		if pc, ok := config.Providers[p.name]; ok && pc.APIKey != "" && !isPlaceholderKey(pc.APIKey) {
			existing = " [configured ✓]"
		}

		fmt.Printf("  [%s]%s\n", p.name, existing)
		fmt.Printf("  Get key: %s\n", p.url)
		fmt.Print("  API key (Enter to skip): ")

		scanner.Scan()
		apiKey := strings.TrimSpace(scanner.Text())

		if apiKey != "" {
			if config.Providers == nil {
				config.Providers = make(map[string]ProviderConfig)
			}
			config.Providers[p.name] = ProviderConfig{
				APIKey: apiKey,
				Model:  config.Providers[p.name].Model, // preserve existing model
			}
			configuredCount++
			if firstProvider == "" {
				firstProvider = p.name
			}
			fmt.Println("  ✓ Saved")
		} else if existing != "" {
			configuredCount++
			if firstProvider == "" {
				firstProvider = p.name
			}
		}
		fmt.Println()
	}

	if configuredCount == 0 {
		fmt.Println("  [!] No API keys configured. Run 'ask --config' when ready.")
		return
	}

	// Set default provider if not set
	if config.DefaultProvider == "" && firstProvider != "" {
		config.DefaultProvider = firstProvider
	}

	// Save config
	saveConfig(config)

	fmt.Println("  You're all set! Try:")
	fmt.Println("    ask What is the meaning of life?")
	fmt.Println("    ask -s  # interactive session")
	fmt.Println("    ask --config  # reconfigure anytime")
	fmt.Println()
}

// runConfigureWizard helps users configure API keys and default models
// If singleProvider is not empty, only configure that specific provider
func runConfigureWizard(singleProvider string) error {
	fmt.Println()
	if singleProvider != "" {
		fmt.Printf("[*] Configure %s\n", singleProvider)
	} else {
		fmt.Println("[*] Configure Ask")
	}
	fmt.Println()
	fmt.Println("This wizard will help you set up API keys and default models.")
	fmt.Println("Press Enter to keep existing values.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Load existing config
	config := &Config{
		Providers: make(map[string]ProviderConfig),
	}
	if existing, err := LoadConfigSafe(); err == nil && existing != nil {
		config = existing
	}

	var firstProvider string

	// Filter providers if single provider specified
	providers := providerInfo
	if singleProvider != "" {
		found := false
		for _, p := range providerInfo {
			if p.name == singleProvider {
				providers = []struct {
					name string
					desc string
					url  string
				}{p}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown provider: %s", singleProvider)
		}
	}

	for _, p := range providers {
		fmt.Printf("[>] %s\n", strings.ToUpper(p.name))
		fmt.Printf("    %s\n", p.url)

		existing := config.Providers[p.name]
		hasKey := existing.APIKey != "" && !isPlaceholderKey(existing.APIKey)

		// Show current status
		if hasKey {
			fmt.Printf("    Current: %s...%s\n", existing.APIKey[:4], existing.APIKey[len(existing.APIKey)-4:])
		}

		fmt.Print("    API key (Enter to skip/keep): ")
		scanner.Scan()
		apiKey := strings.TrimSpace(scanner.Text())

		if apiKey != "" {
			existing.APIKey = apiKey
			config.Providers[p.name] = existing
			fmt.Println("    ✓ Updated")
		} else if hasKey {
			fmt.Println("    ✓ Kept existing")
		}

		// Track first configured provider
		if (apiKey != "" || hasKey) && firstProvider == "" {
			firstProvider = p.name
		}

		// If we have a key, ask about default model
		finalKey := config.Providers[p.name].APIKey
		if finalKey != "" && !isPlaceholderKey(finalKey) {
			prov := createProvider(p.name, finalKey, "")
			if prov != nil {
				models, err := prov.ListModels()
				if err == nil && len(models) > 0 {
					fmt.Println("\n    Available models:")
					for i, model := range models {
						modelID := strings.TrimPrefix(model.ID, "models/")
						current := ""
						if modelID == existing.Model {
							current = " \033[32m(current)\033[0m"
						}
						fmt.Printf("    %2d. %s%s\n", i+1, modelID, current)
					}

					fmt.Printf("    Select default [1-%d] (Enter to skip): ", len(models))
					scanner.Scan()
					choice := strings.TrimSpace(scanner.Text())

					if num, err := strconv.Atoi(choice); err == nil && num > 0 && num <= len(models) {
						modelID := strings.TrimPrefix(models[num-1].ID, "models/")
						pc := config.Providers[p.name]
						pc.Model = modelID
						config.Providers[p.name] = pc
						fmt.Printf("    ✓ Default set to: %s\n", modelID)
					}
				}
			}
		}

		fmt.Println()
	}

	// Set default provider
	if config.DefaultProvider == "" && firstProvider != "" {
		config.DefaultProvider = firstProvider
	}

	// Save
	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Println("[+] Configuration saved!")
	fmt.Println("    Run 'ask --config' anytime to update.")
	fmt.Println()

	return nil
}

// LoadConfigSafe loads config without error on missing file
func LoadConfigSafe() (*Config, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "ask", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Try current directory
		data, err = os.ReadFile("config.yaml")
		if err != nil {
			return nil, err
		}
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// saveConfig saves the configuration file
func saveConfig(config *Config) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "ask")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}

	fmt.Printf("\n[+] Saved to: %s\n", configPath)
	return nil
}

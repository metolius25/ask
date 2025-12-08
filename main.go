// Package main implements a CLI tool for querying AI models from the terminal.
// It supports multiple providers (Gemini, Claude, ChatGPT, DeepSeek) with
// beautiful markdown rendering.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"ask/provider"

	"github.com/charmbracelet/glamour"
)

// Default models for each provider (used as fallback)
var defaultModels = map[string]string{
	"gemini":   "gemini-2.5-flash",
	"claude":   "claude-3-5-sonnet-20241022",
	"chatgpt":  "gpt-4o",
	"deepseek": "deepseek-chat",
	"mistral":  "mistral-large-latest",
	"qwen":     "qwen-plus",
}

func main() {
	// Define flags with short aliases
	providerFlag := flag.String("provider", "", "AI provider to use (gemini, claude, chatgpt, deepseek, mistral)")
	flag.StringVar(providerFlag, "p", "", "AI provider (short for -provider)")

	modelFlag := flag.String("model", "", "Model to use (overrides config)")
	flag.StringVar(modelFlag, "m", "", "Model (short for -model)")

	profileFlag := flag.String("profile", "", "Use a named profile from config")
	flag.StringVar(profileFlag, "P", "", "Profile (short for -profile)")

	listModels := flag.Bool("list-models", false, "List available models for all providers")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.BoolVar(versionFlag, "v", false, "Version (short for -version)")

	sessionFlag := flag.Bool("session", false, "Start interactive session mode")
	flag.BoolVar(sessionFlag, "s", false, "Session (short for -session)")
	// Keep -S for backwards compatibility
	legacySessionFlag := flag.Bool("S", false, "Start interactive session mode (deprecated, use -s)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Printf("%s v%s - AI CLI Client\n\n", AppName, Version)
		fmt.Println("Usage: ask [flags] [your prompt here]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ask What is the meaning of life?")
		fmt.Println("  ask -m gpt-4o Write a haiku about Go")
		fmt.Println("  ask -p claude Explain quantum computing")
		fmt.Println("  ask -P fast Tell me a joke")
		fmt.Println("  ask -s  # Start interactive session mode")
		fmt.Println("  ask --list-models")
		fmt.Println("  ask -v")
		fmt.Println("  ask --config        # Configure all providers")
		fmt.Println("  ask --config qwen   # Configure specific provider")
	}

	// Handle --config BEFORE flag.Parse() to avoid parsing issues
	for i, arg := range os.Args[1:] {
		if arg == "--config" || arg == "-config" {
			providerArg := ""
			// Check if next arg exists and is not a flag
			if i+2 < len(os.Args) && !strings.HasPrefix(os.Args[i+2], "-") {
				providerArg = os.Args[i+2]
			}
			if err := runConfigureWizard(providerArg); err != nil {
				fmt.Fprintf(os.Stderr, "Configuration failed: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		if strings.HasPrefix(arg, "--config=") {
			providerArg := strings.TrimPrefix(arg, "--config=")
			if err := runConfigureWizard(providerArg); err != nil {
				fmt.Fprintf(os.Stderr, "Configuration failed: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("%s v%s\n", AppName, Version)
		os.Exit(0)
	}

	// Handle list-models command
	if *listModels {
		printAvailableModels()
		os.Exit(0)
	}

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		// Check for specific error types and provide helpful messages
		switch e := err.(type) {
		case *ConfigNotFoundError:
			runInteractiveSetup()
			os.Exit(0)
		case *PlaceholderKeyError:
			printPlaceholderKeyHelp(e.Provider)
		default:
			fmt.Fprintf(os.Stderr, "[!] Error loading config: %v\n\n", err)
			printQuickHelp()
		}
		os.Exit(1)
	}

	// Resolve provider and model using the new resolver
	selectedProvider, selectedModel, err := ResolveModelAndProvider(
		*providerFlag, *modelFlag, *profileFlag, config,
	)
	if err != nil {
		switch e := err.(type) {
		case *ProfileError:
			fmt.Fprintf(os.Stderr, "[!] %v\n", e)
		default:
			fmt.Fprintf(os.Stderr, "[!] Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Validate provider exists in config
	providerConfig, exists := config.Providers[selectedProvider]
	if !exists {
		fmt.Fprintf(os.Stderr, "[!] Provider '%s' not found in config\n\n", selectedProvider)
		fmt.Fprintf(os.Stderr, "Available providers in your config: %s\n", getConfiguredProviders(config))
		os.Exit(1)
	}

	// Check for placeholder key
	if isPlaceholderKey(providerConfig.APIKey) {
		printPlaceholderKeyHelp(selectedProvider)
		os.Exit(1)
	}

	// Apply fallback model if still empty
	if selectedModel == "" {
		if providerConfig.Model != "" {
			selectedModel = providerConfig.Model
		} else if dm, ok := defaultModels[selectedProvider]; ok {
			selectedModel = dm
		}
	}

	// Create the provider
	p := createProvider(selectedProvider, providerConfig.APIKey, selectedModel)
	if p == nil {
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", selectedProvider)
		fmt.Fprintf(os.Stderr, "Supported providers: gemini, claude, chatgpt, deepseek, mistral\n")
		os.Exit(1)
	}

	// Handle session mode (support both -s and legacy -S)
	if *sessionFlag || *legacySessionFlag {
		if err := RunSessionREPL(p, selectedProvider, selectedModel); err != nil {
			fmt.Fprintf(os.Stderr, "\n[!] Session error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Get the prompt (everything after flags)
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	prompt := strings.Join(args, " ")

	// Query the provider
	var responseBuffer strings.Builder
	if err := p.QueryStream(prompt, &responseBuffer); err != nil {
		fmt.Fprintf(os.Stderr, "\nError querying %s: %v\n", selectedProvider, err)
		os.Exit(1)
	}

	// Render the markdown response
	response := responseBuffer.String()
	if err := renderMarkdown(response); err != nil {
		fmt.Println(response)
	}
}

// createProvider creates a provider instance
func createProvider(name, apiKey, model string) provider.Provider {
	switch name {
	case "gemini":
		return provider.NewGeminiProvider(apiKey, model)
	case "claude":
		return provider.NewClaudeProvider(apiKey, model)
	case "chatgpt":
		return provider.NewChatGPTProvider(apiKey, model)
	case "deepseek":
		return provider.NewDeepSeekProvider(apiKey, model)
	case "mistral":
		return provider.NewMistralProvider(apiKey, model)
	case "qwen":
		return provider.NewQwenProvider(apiKey, model)
	default:
		return nil
	}
}

func renderMarkdown(content string) error {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return err
	}

	out, err := r.Render(content)
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}

func printPlaceholderKeyHelp(provider string) {
	fmt.Printf("!! API key not configured for '%s'\n\n", provider)
	fmt.Println("It looks like you haven't added your API key yet.")
	fmt.Println()
	fmt.Println(" Get an API key:")
	fmt.Println()

	switch provider {
	case "gemini":
		fmt.Println("   Visit: https://makersuite.google.com/app/apikey")
	case "claude":
		fmt.Println("   Visit: https://console.anthropic.com/")
	case "chatgpt":
		fmt.Println("   Visit: https://platform.openai.com/api-keys")
	case "deepseek":
		fmt.Println("   Visit: https://platform.deepseek.com/")
	case "mistral":
		fmt.Println("   Visit: https://console.mistral.ai/")
	default:
		fmt.Printf("   Check your provider's documentation for '%s'\n", provider)
	}

	fmt.Println()
	fmt.Println("   Then edit your config.yaml and replace the placeholder with your real API key:")
	fmt.Println()
	fmt.Println("   providers:")
	fmt.Printf("     %s:\n", provider)
	fmt.Println("       api_key: your-actual-api-key-here")
	fmt.Println()
}

func printQuickHelp() {
	fmt.Println("  Quick troubleshooting:")
	fmt.Println()
	fmt.Println("   1. Make sure config.yaml exists in current directory or ~/.config/ask/")
	fmt.Println("   2. Check that at least one provider has a valid API key")
	fmt.Println("   3. Run 'ask --configure' to set up interactively")
	fmt.Println()
}

func printAvailableModels() {
	// Try to load config, but don't require it
	config, err := LoadConfig()

	// If no config, show fallback models cleanly
	if err != nil {
		fmt.Println("  Available Models (default list)")
		fmt.Println()
		fmt.Println("  Configure your API keys in config.yaml to see live models from providers")
		fmt.Println()
		printFallbackModels()
		return
	}

	fmt.Println("  Fetching Available Models...")
	fmt.Println()

	providers := []struct {
		name string
		key  string
	}{
		{"gemini", "gemini"},
		{"claude", "claude"},
		{"chatgpt", "chatgpt"},
		{"deepseek", "deepseek"},
		{"mistral", "mistral"},
	}

	for _, p := range providers {
		// Check if provider is configured
		providerConfig, exists := config.Providers[p.name]
		if !exists || providerConfig.APIKey == "" || isPlaceholderKey(providerConfig.APIKey) {
			fmt.Printf("[>] %s (not configured)\n", strings.ToUpper(p.name))

			// Show fallback models from the provider's own implementation
			prov := createProvider(p.name, "", "")
			if prov != nil {
				models, _ := prov.ListModels()
				for _, model := range models {
					fmt.Printf("   • %s\n", model.ID)
				}
			}
			fmt.Println()
			continue
		}

		// Create provider instance
		prov := createProvider(p.name, providerConfig.APIKey, "")
		if prov == nil {
			continue
		}

		// Fetch models
		models, err := prov.ListModels()
		if err != nil || len(models) == 0 {
			fmt.Printf("[>] %s (API error - showing defaults)\n", strings.ToUpper(p.name))
			fmt.Println()
			continue
		}

		fmt.Printf("[>] %s ✓\n", strings.ToUpper(p.name))
		for _, model := range models {
			modelID := model.ID
			// Clean up Gemini model names
			modelID = strings.TrimPrefix(modelID, "models/")

			if model.Description != "" {
				fmt.Printf("   • %s - %s\n", modelID, model.Description)
			} else {
				fmt.Printf("   • %s\n", modelID)
			}
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  ask -m <model-name> Your prompt here")
	fmt.Println("  ask -m gemini/gemini-2.5-pro Explain AI")
}

func printFallbackModels() {
	providers := []string{"gemini", "claude", "chatgpt", "deepseek", "mistral"}

	for _, name := range providers {
		prov := createProvider(name, "", "")
		if prov == nil {
			continue
		}

		models, _ := prov.ListModels()
		fmt.Printf("[>] %s\n", strings.ToUpper(name))
		for _, model := range models {
			fmt.Printf("   • %s\n", model.ID)
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  ask -m <model-name> Your prompt here")
	fmt.Println("  ask -m gemini/gemini-2.5-pro Explain AI")
}

func getConfiguredProviders(config *Config) string {
	var providers []string
	for name := range config.Providers {
		providers = append(providers, name)
	}
	return strings.Join(providers, ", ")
}

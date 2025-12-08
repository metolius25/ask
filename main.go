// Package main implements a CLI tool for querying AI models from the terminal.
// It supports multiple providers (Gemini, Claude, ChatGPT, DeepSeek) with
// real-time streaming and beautiful markdown rendering.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"ask/provider"

	"github.com/charmbracelet/glamour"
)

func main() {
	// Define flags
	providerFlag := flag.String("provider", "", "AI provider to use (gemini, claude, chatgpt, deepseek, mistral)")
	modelFlag := flag.String("model", "", "Model to use (overrides config)")
	listModels := flag.Bool("list-models", false, "List available models for all providers")
	configureFlag := flag.Bool("configure", false, "Configure default models interactively")
	versionFlag := flag.Bool("version", false, "Show version information")
	sessionFlag := flag.Bool("S", false, "Start interactive session mode")

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
		fmt.Println("  ask -provider claude Explain quantum computing")
		fmt.Println("  ask -model gpt-4o-mini Write a haiku about Go")
		fmt.Println("  ask -provider gemini -model gemini-1.5-pro Tell me a joke")
		fmt.Println("  ask -S  # Start interactive session mode")
		fmt.Println("  ask --list-models")
		fmt.Println("  ask --version")
		fmt.Println("  ask --configure")
	}

	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("%s v%s\n", AppName, Version)
		os.Exit(0)
	}

	// Handle configure command
	if *configureFlag {
		if err := runConfigureWizard(); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle list-models command
	if *listModels {
		printAvailableModels()
		os.Exit(0)
	}

	// Handle session mode
	if *sessionFlag {
		// Load configuration
		config, err := LoadConfig()
		if err != nil {
			// Check for specific error types and provide helpful messages
			switch e := err.(type) {
			case *ConfigNotFoundError:
				printFirstRunHelp()
			case *PlaceholderKeyError:
				printPlaceholderKeyHelp(e.Provider)
			default:
				fmt.Fprintf(os.Stderr, "[!] Error loading config: %v\n\n", err)
				printQuickHelp()
			}
			os.Exit(1)
		}

		// Determine which provider to use (flag overrides config)
		selectedProvider := config.DefaultProvider
		if *providerFlag != "" {
			selectedProvider = *providerFlag
		}

		// Validate provider exists
		providerConfig, exists := config.Providers[selectedProvider]
		if !exists {
			fmt.Fprintf(os.Stderr, "[!] Provider '%s' not found in config\n\n", selectedProvider)
			fmt.Fprintf(os.Stderr, "Available providers in your config: %s\n", getConfiguredProviders(config))
			fmt.Fprintf(os.Stderr, "Add configuration for '%s' in your config.yaml\n", selectedProvider)
			os.Exit(1)
		}

		// Check for placeholder key
		if isPlaceholderKey(providerConfig.APIKey) {
			printPlaceholderKeyHelp(selectedProvider)
			os.Exit(1)
		}

		// Determine which model to use (flag overrides config)
		selectedModel := providerConfig.Model
		if *modelFlag != "" {
			selectedModel = *modelFlag
		}

		// If still empty, use default
		if selectedModel == "" {
			selectedModel = GetDefaultModel(selectedProvider)
		}

		// Create the appropriate provider
		var p provider.Provider
		switch selectedProvider {
		case "gemini":
			if selectedModel == "" {
				selectedModel = "gemini-2.5-flash" // Fallback
			}
			p = provider.NewGeminiProvider(providerConfig.APIKey, selectedModel)
		case "claude":
			if selectedModel == "" {
				selectedModel = "claude-3-5-sonnet-20241022" // Fallback
			}
			p = provider.NewClaudeProvider(providerConfig.APIKey, selectedModel)
		case "chatgpt":
			if selectedModel == "" {
				selectedModel = "gpt-4o" // Fallback
			}
			p = provider.NewChatGPTProvider(providerConfig.APIKey, selectedModel)
		case "deepseek":
			if selectedModel == "" {
				selectedModel = "deepseek-chat" // Fallback
			}
			p = provider.NewDeepSeekProvider(providerConfig.APIKey, selectedModel)
		case "mistral":
			if selectedModel == "" {
				selectedModel = "mistral-large-latest" // Fallback
			}
			p = provider.NewMistralProvider(providerConfig.APIKey, selectedModel)
		default:
			fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", selectedProvider)
			fmt.Fprintf(os.Stderr, "Supported providers: gemini, claude, chatgpt, deepseek, mistral\n")
			os.Exit(1)
		}

		// Run interactive session TUI
		if err := RunSessionTUI(p, selectedProvider, selectedModel); err != nil {
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

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		// Check for specific error types and provide helpful messages
		switch e := err.(type) {
		case *ConfigNotFoundError:
			printFirstRunHelp()
		case *PlaceholderKeyError:
			printPlaceholderKeyHelp(e.Provider)
		default:
			fmt.Fprintf(os.Stderr, "[!] Error loading config: %v\n\n", err)
			printQuickHelp()
		}
		os.Exit(1)
	}

	// Determine which provider to use (flag overrides config)
	selectedProvider := config.DefaultProvider
	if *providerFlag != "" {
		selectedProvider = *providerFlag
	}

	// Validate provider exists
	providerConfig, exists := config.Providers[selectedProvider]
	if !exists {
		fmt.Fprintf(os.Stderr, "[!] Provider '%s' not found in config\n\n", selectedProvider)
		fmt.Fprintf(os.Stderr, "Available providers in your config: %s\n", getConfiguredProviders(config))
		fmt.Fprintf(os.Stderr, "Add configuration for '%s' in your config.yaml\n", selectedProvider)
		os.Exit(1)
	}

	// Check for placeholder key
	if isPlaceholderKey(providerConfig.APIKey) {
		printPlaceholderKeyHelp(selectedProvider)
		os.Exit(1)
	}

	// Determine which model to use (flag overrides config)
	selectedModel := providerConfig.Model
	if *modelFlag != "" {
		selectedModel = *modelFlag
	}

	// If still empty, use default
	if selectedModel == "" {
		selectedModel = GetDefaultModel(selectedProvider)
	}

	// Create the appropriate provider
	var p provider.Provider
	switch selectedProvider {
	case "gemini":
		p = provider.NewGeminiProvider(providerConfig.APIKey, selectedModel)
	case "claude":
		p = provider.NewClaudeProvider(providerConfig.APIKey, selectedModel)
	case "chatgpt":
		p = provider.NewChatGPTProvider(providerConfig.APIKey, selectedModel)
	case "deepseek":
		p = provider.NewDeepSeekProvider(providerConfig.APIKey, selectedModel)
	case "mistral":
		p = provider.NewMistralProvider(providerConfig.APIKey, selectedModel)
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", selectedProvider)
		fmt.Fprintf(os.Stderr, "Supported providers: gemini, claude, chatgpt, deepseek, mistral\n")
		os.Exit(1)
	}

	// Query the provider and stream the response to stdout
	// Collect the response in a buffer for markdown rendering
	var responseBuffer strings.Builder
	if err := p.QueryStream(prompt, &responseBuffer); err != nil {
		fmt.Fprintf(os.Stderr, "\nError querying %s: %v\n", selectedProvider, err)
		os.Exit(1)
	}

	// Render the markdown response beautifully
	response := responseBuffer.String()
	if err := renderMarkdown(response); err != nil {
		// Fallback to plain text if rendering fails
		fmt.Println(response)
	}
}

func renderMarkdown(content string) error {
	// Create a glamour renderer for the terminal
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

func printFirstRunHelp() {
	fmt.Println("Welcome to Ask - AI CLI Client!")
	fmt.Println()
	fmt.Println("It looks like this is your first time running the app.")
	fmt.Println("Let's get you set up in 4 easy steps:")
	fmt.Println()
	fmt.Println("   Step 1: Create a config file")
	fmt.Println("   Copy the example config:")
	fmt.Println("   $ cp config.yaml.example config.yaml")
	fmt.Println()
	fmt.Println("   Step 2: Choose your AI provider")
	fmt.Println("   Pick one (or configure multiple):")
	fmt.Println()
	fmt.Println("   • Gemini   - Google's latest models (fast, free tier available)")
	fmt.Println("   • Claude   - Anthropic's models (excellent reasoning)")
	fmt.Println("   • ChatGPT  - OpenAI's models (including o1)")
	fmt.Println("   • DeepSeek - Cost-effective option")
	fmt.Println()
	fmt.Println("   Step 3: Get an API key for your chosen provider")
	fmt.Println()
	fmt.Println("   • Gemini  : https://makersuite.google.com/app/apikey")
	fmt.Println("   • Claude  : https://console.anthropic.com/")
	fmt.Println("   • ChatGPT : https://platform.openai.com/api-keys")
	fmt.Println("   • DeepSeek: https://platform.deepseek.com/")
	fmt.Println()
	fmt.Println("   Step 4: Configure config.yaml")
	fmt.Println("   1. Set default_provider to your chosen provider")
	fmt.Println("   2. Add your API key under that provider's section")
	fmt.Println()
	fmt.Println("Then you're ready to go!")
	fmt.Println("   $ ask What is the meaning of life?")
	fmt.Println("   $ ask -provider claude Explain quantum computing")
	fmt.Println("   $ ask --list-models")
	fmt.Println()
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
	fmt.Println("   2. Check that default_provider is set")
	fmt.Println("   3. Verify your API key is configured correctly")
	fmt.Println()
	fmt.Println("   See config.yaml.example for reference")
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
			var prov provider.Provider
			switch p.name {
			case "gemini":
				prov = provider.NewGeminiProvider("", "")
			case "claude":
				prov = provider.NewClaudeProvider("", "")
			case "chatgpt":
				prov = provider.NewChatGPTProvider("", "")
			case "deepseek":
				prov = provider.NewDeepSeekProvider("", "")
			case "mistral":
				prov = provider.NewMistralProvider("", "")
			}

			models, _ := prov.ListModels()
			defaultModel := GetDefaultModel(p.name)
			for _, model := range models {
				if model.ID == defaultModel {
					fmt.Printf("   • %s (default)\n", model.ID)
				} else {
					fmt.Printf("   • %s\n", model.ID)
				}
			}
			fmt.Println()
			continue
		}

		// Create provider instance
		var prov provider.Provider
		switch p.name {
		case "gemini":
			prov = provider.NewGeminiProvider(providerConfig.APIKey, "")
		case "claude":
			prov = provider.NewClaudeProvider(providerConfig.APIKey, "")
		case "chatgpt":
			prov = provider.NewChatGPTProvider(providerConfig.APIKey, "")
		case "deepseek":
			prov = provider.NewDeepSeekProvider(providerConfig.APIKey, "")
		case "mistral":
			prov = provider.NewMistralProvider(providerConfig.APIKey, "")
		}
		// Fetch models
		models, err := prov.ListModels()
		if err != nil || len(models) == 0 {
			fmt.Printf("[>] %s (API error - showing defaults)\n", strings.ToUpper(p.name))
			fmt.Println()
			continue
		}

		defaultModel := GetDefaultModel(p.name)

		fmt.Printf("[>] %s ✓\n", strings.ToUpper(p.name))
		for _, model := range models {
			modelID := model.ID
			// Clean up Gemini model names (remove "models/" prefix if present)
			modelID = strings.TrimPrefix(modelID, "models/")

			if model.Description != "" {
				if modelID == defaultModel {
					fmt.Printf("   • %s (default) - %s\n", modelID, model.Description)
				} else {
					fmt.Printf("   • %s - %s\n", modelID, model.Description)
				}
			} else {
				if modelID == defaultModel {
					fmt.Printf("   • %s (default)\n", modelID)
				} else {
					fmt.Printf("   • %s\n", modelID)
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  ask -model <model-name> Your prompt here")
	fmt.Println("  ask -provider gemini -model gemini-1.5-pro Explain AI")
}

func printFallbackModels() {
	providers := []struct {
		name     string
		provider provider.Provider
	}{
		{"gemini", provider.NewGeminiProvider("", "")},
		{"claude", provider.NewClaudeProvider("", "")},
		{"chatgpt", provider.NewChatGPTProvider("", "")},
		{"deepseek", provider.NewDeepSeekProvider("", "")},
		{"mistral", provider.NewMistralProvider("", "")},
	}

	for _, p := range providers {
		// Get fallback models from the provider itself
		models, _ := p.provider.ListModels()
		defaultModel := GetDefaultModel(p.name)

		fmt.Printf("[>] %s\n", strings.ToUpper(p.name))
		for _, model := range models {
			if model.ID == defaultModel {
				fmt.Printf("   • %s (default)\n", model.ID)
			} else {
				fmt.Printf("   • %s\n", model.ID)
			}
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  ask -model <model-name> Your prompt here")
	fmt.Println("  ask -provider gemini -model gemini-1.5-pro Explain AI")
}

func getConfiguredProviders(config *Config) string {
	var providers []string
	for name := range config.Providers {
		providers = append(providers, name)
	}
	return strings.Join(providers, ", ")
}

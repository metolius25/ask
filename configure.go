package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ask/provider"
)

// runConfigureWizard helps users set up their preferred default models
func runConfigureWizard() error {
	fmt.Println("[*] Configure Default Models")
	fmt.Println()
	fmt.Println("This wizard will help you set your preferred default models for each provider.")
	fmt.Println("Models will be fetched from the provider APIs if you have API keys configured.")
	fmt.Println()

	// Load existing config to get API keys
	config, err := LoadConfig()
	if err != nil {
		fmt.Println("[!]  No config.yaml found. Please set up your API keys first:")
		fmt.Println("   cp config.yaml.example config.yaml")
		fmt.Println("   # Then edit config.yaml with your API keys")
		return fmt.Errorf("config file not found")
	}

	defaults := &DefaultsConfig{
		Defaults: make(map[string]string),
	}

	// Load existing defaults if any
	existing := loadDefaults()
	if existing != nil && existing.Defaults != nil {
		defaults = existing
	}

	providers := []string{"gemini", "claude", "chatgpt", "deepseek", "mistral"}
	scanner := bufio.NewScanner(os.Stdin)

	for _, providerName := range providers {
		fmt.Printf("\n[>] Configuring %s\n", strings.ToUpper(providerName))

		// Check if provider is configured
		providerConfig, exists := config.Providers[providerName]
		if !exists || providerConfig.APIKey == "" || isPlaceholderKey(providerConfig.APIKey) {
			fmt.Printf("   [!]  %s not configured (no valid API key)\n", providerName)
			fmt.Print("   Skip this provider? [Y/n]: ")
			scanner.Scan()
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response == "" || response == "y" || response == "yes" {
				continue
			}
		}

		// Fetch available models
		var prov provider.Provider
		switch providerName {
		case "gemini":
			if exists && providerConfig.APIKey != "" {
				prov = provider.NewGeminiProvider(providerConfig.APIKey, "")
			} else {
				prov = provider.NewGeminiProvider("", "")
			}
		case "claude":
			if exists && providerConfig.APIKey != "" {
				prov = provider.NewClaudeProvider(providerConfig.APIKey, "")
			} else {
				prov = provider.NewClaudeProvider("", "")
			}
		case "chatgpt":
			if exists && providerConfig.APIKey != "" {
				prov = provider.NewChatGPTProvider(providerConfig.APIKey, "")
			} else {
				prov = provider.NewChatGPTProvider("", "")
			}
		case "deepseek":
			if exists && providerConfig.APIKey != "" {
				prov = provider.NewDeepSeekProvider(providerConfig.APIKey, "")
			} else {
				prov = provider.NewDeepSeekProvider("", "")
			}
		case "mistral":
			if exists && providerConfig.APIKey != "" {
				prov = provider.NewMistralProvider(providerConfig.APIKey, "")
			} else {
				prov = provider.NewMistralProvider("", "")
			}
		}

		fmt.Print("   Fetching models... ")
		models, err := prov.ListModels()
		if err != nil || len(models) == 0 {
			fmt.Println("[!] Failed to fetch models")
			fmt.Println("   Using fallback list")
		} else {
			fmt.Println("[+]")
		}

		if len(models) == 0 {
			fmt.Println("   No models available")
			continue
		}

		// Display models
		fmt.Println("\n   Available models:")
		for i, model := range models {
			modelID := model.ID
			// Clean up Gemini model names
			modelID = strings.TrimPrefix(modelID, "models/")

			current := ""
			if modelID == defaults.Defaults[providerName] {
				current = " (current default)"
			}

			description := ""
			if model.Description != "" && len(model.Description) < 60 {
				description = " - " + model.Description
			}

			fmt.Printf("   %2d. %s%s%s\n", i+1, modelID, current, description)
		}

		fmt.Printf("\n   Select default model for %s [1-%d or model name]: ", providerName, len(models))
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		if choice == "" {
			// Keep existing default or use first model
			if defaults.Defaults[providerName] == "" {
				defaults.Defaults[providerName] = models[0].ID
			}
			continue
		}

		// Try to parse as number
		if num, err := strconv.Atoi(choice); err == nil && num > 0 && num <= len(models) {
			modelID := models[num-1].ID
			modelID = strings.TrimPrefix(modelID, "models/")
			defaults.Defaults[providerName] = modelID
			fmt.Printf("   [+] Set default to: %s\n", modelID)
		} else {
			// Treat as model name
			defaults.Defaults[providerName] = choice
			fmt.Printf("   [+] Set default to: %s\n", choice)
		}
	}

	// Save configuration
	fmt.Println("\n[*] Saving configuration...")
	if err := SaveDefaults(defaults); err != nil {
		return fmt.Errorf("failed to save defaults: %w", err)
	}

	configPath := fmt.Sprintf("%s/.config/ask/defaults.yaml", os.Getenv("HOME"))
	fmt.Printf("[+] Configuration saved to: %s\n", configPath)
	fmt.Println("\nYou can now use 'ask' without specifying models!")
	fmt.Println("To reconfigure, run: ask --configure")

	return nil
}

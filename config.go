// Package main provides configuration management for the Ask CLI tool.
// It handles loading and validating YAML configuration files with API keys
// and provider settings.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model,omitempty"`
}

func LoadConfig() (*Config, error) {
	// Try to load from current directory first
	configPath := "config.yaml"
	data, err := os.ReadFile(configPath)

	if err != nil {
		// Try ~/.config/ask/config.yaml
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		configPath = filepath.Join(homeDir, ".config", "ask", "config.yaml")
		data, err = os.ReadFile(configPath)
		if err != nil {
			return nil, &ConfigNotFoundError{}
		}
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if config.DefaultProvider == "" {
		return nil, fmt.Errorf("default_provider not set in config")
	}

	providerConfig, exists := config.Providers[config.DefaultProvider]
	if !exists {
		return nil, fmt.Errorf("default provider '%s' not found in providers config", config.DefaultProvider)
	}

	if providerConfig.APIKey == "" {
		return nil, fmt.Errorf("api_key not set for provider '%s'", config.DefaultProvider)
	}

	// Check for placeholder API keys
	if isPlaceholderKey(providerConfig.APIKey) {
		return nil, &PlaceholderKeyError{Provider: config.DefaultProvider}
	}

	return &config, nil
}

// ConfigNotFoundError indicates config file doesn't exist
type ConfigNotFoundError struct{}

func (e *ConfigNotFoundError) Error() string {
	return "config file not found"
}

// PlaceholderKeyError indicates API key hasn't been configured
type PlaceholderKeyError struct {
	Provider string
}

func (e *PlaceholderKeyError) Error() string {
	return fmt.Sprintf("placeholder API key detected for provider '%s'", e.Provider)
}

// isPlaceholderKey checks if the API key is a placeholder value
func isPlaceholderKey(key string) bool {
	placeholders := []string{
		"YOUR_",
		"REPLACE_",
		"INSERT_",
		"ADD_YOUR_",
		"PASTE_",
	}

	for _, placeholder := range placeholders {
		if len(key) > 0 && (key[:min(len(key), len(placeholder))] == placeholder ||
			key == "your-api-key-here" ||
			key == "sk-..." ||
			key == "***") {
			return true
		}
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

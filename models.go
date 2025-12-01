package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultsConfig holds user's preferred default models
type DefaultsConfig struct {
	Defaults map[string]string `yaml:"defaults"`
}

// GetDefaultModel returns the user's preferred default model for a provider.
// It reads from defaults.yaml if available, otherwise returns the first available
// model from the provider.
func GetDefaultModel(providerName string) string {
	// Try to load user's preferences
	defaults := loadDefaults()
	if model, ok := defaults.Defaults[providerName]; ok && model != "" {
		return model
	}

	// Fallback: provider will use first available model
	return ""
}

// loadDefaults loads the defaults configuration file
func loadDefaults() *DefaultsConfig {
	// Check possible locations
	locations := []string{
		"./defaults.yaml",
		filepath.Join(os.Getenv("HOME"), ".config", "ask", "defaults.yaml"),
	}

	for _, loc := range locations {
		if data, err := os.ReadFile(loc); err == nil {
			var config DefaultsConfig
			if err := yaml.Unmarshal(data, &config); err == nil {
				return &config
			}
		}
	}

	// Return empty config if file doesn't exist
	return &DefaultsConfig{Defaults: make(map[string]string)}
}

// SaveDefaults saves the defaults configuration
func SaveDefaults(defaults *DefaultsConfig) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "ask")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(defaults)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "defaults.yaml"), data, 0644)
}

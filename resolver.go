// Package main provides model-to-provider resolution and profile handling.
package main

import (
	"strings"
)

// Model prefix to provider mapping
var modelPrefixes = map[string]string{
	"gemini":    "gemini",
	"gpt":       "chatgpt",
	"o1":        "chatgpt",
	"o3":        "chatgpt",
	"claude":    "claude",
	"deepseek":  "deepseek",
	"mistral":   "mistral",
	"codestral": "mistral",
	"qwen":      "qwen",
	"pixtral":   "mistral",
	"ministral": "mistral",
}

// ResolveProviderFromModel attempts to detect which provider a model belongs to
// based on its name prefix. Returns empty string if unknown.
func ResolveProviderFromModel(model string) string {
	model = strings.ToLower(model)

	for prefix, provider := range modelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return provider
		}
	}

	return ""
}

// ParseModelSpec parses a model specification which can be:
// - "modelname" -> returns ("", "modelname")
// - "provider/modelname" -> returns ("provider", "modelname")
func ParseModelSpec(spec string) (provider, model string) {
	if spec == "" {
		return "", ""
	}

	parts := strings.SplitN(spec, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", spec
}

// ResolveModelAndProvider determines the final provider and model to use
// given user inputs and config. Priority:
// 1. Explicit provider flag
// 2. Provider from model spec (e.g., "gemini/gemini-2.5-flash")
// 3. Auto-detected provider from model name
// 4. Config default provider
func ResolveModelAndProvider(
	providerFlag, modelFlag, profileFlag string,
	config *Config,
) (provider, model string, err error) {

	// Handle profile first
	if profileFlag != "" {
		if config.Profiles == nil {
			return "", "", &ProfileError{Name: profileFlag, Reason: "no profiles defined in config"}
		}
		profileSpec, exists := config.Profiles[profileFlag]
		if !exists {
			return "", "", &ProfileError{Name: profileFlag, Reason: "profile not found"}
		}
		// Parse profile spec (e.g., "gemini/gemini-2.5-flash")
		provider, model = ParseModelSpec(profileSpec)
		if provider == "" {
			provider = ResolveProviderFromModel(model)
		}
		if provider == "" {
			return "", "", &ProfileError{Name: profileFlag, Reason: "cannot determine provider from profile"}
		}
		return provider, model, nil
	}

	// Parse model spec if provided
	var specProvider, specModel string
	if modelFlag != "" {
		specProvider, specModel = ParseModelSpec(modelFlag)
	}

	// Determine provider
	if providerFlag != "" {
		provider = providerFlag
	} else if specProvider != "" {
		provider = specProvider
	} else if specModel != "" {
		provider = ResolveProviderFromModel(specModel)
	}

	// Fall back to config default
	if provider == "" {
		provider = config.DefaultProvider
	}

	// Determine model
	if specModel != "" {
		model = specModel
	} else if provider != "" {
		// Use provider's default model from config
		if pc, exists := config.Providers[provider]; exists && pc.Model != "" {
			model = pc.Model
		}
	}

	return provider, model, nil
}

// ProfileError indicates an issue with profile resolution
type ProfileError struct {
	Name   string
	Reason string
}

func (e *ProfileError) Error() string {
	return "profile '" + e.Name + "': " + e.Reason
}

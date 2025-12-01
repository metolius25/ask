// Package provider defines the interface and implementations for AI model providers.
// Each provider (Gemini, Claude, ChatGPT, DeepSeek) implements streaming queries
// and dynamic model discovery.
package provider

import "io"

// ModelInfo contains information about an available model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
}

// Provider defines the interface for AI model providers
type Provider interface {
	// QueryStream sends a prompt and streams the response to the writer in real-time
	QueryStream(prompt string, writer io.Writer) error

	// ListModels returns available models for this provider
	ListModels() ([]ModelInfo, error)
}

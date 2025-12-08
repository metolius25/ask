// Package provider defines the interface and implementations for AI model providers.
// Each provider (Gemini, Claude, ChatGPT, DeepSeek) implements streaming queries
// and dynamic model discovery.
package provider

import (
	"fmt"
	"io"
)

// HandleAPIError returns a user-friendly error message for common API errors
func HandleAPIError(statusCode int, body []byte, providerName string) error {
	switch statusCode {
	case 401:
		return fmt.Errorf("[!] Invalid API key for %s. Check your config.yaml", providerName)
	case 402:
		return fmt.Errorf("[!] Insufficient balance/credits for %s. Please add funds to your account", providerName)
	case 429:
		return fmt.Errorf("[!] Rate limit exceeded for %s. Please wait and try again", providerName)
	default:
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
}

// Message represents a single message in a conversation
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

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

	// QueryStreamWithHistory sends a prompt with conversation history and streams the response
	QueryStreamWithHistory(messages []Message, writer io.Writer) error

	// ListModels returns available models for this provider
	ListModels() ([]ModelInfo, error)
}

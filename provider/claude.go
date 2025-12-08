package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ClaudeProvider struct {
	apiKey string
	model  string
}

func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	// If no model specified, use first available from fallback list
	if model == "" {
		fallbackModels := getFallbackClaudeModels()
		if len(fallbackModels) > 0 {
			model = fallbackModels[0].ID
		}
	}
	return &ClaudeProvider{
		apiKey: apiKey,
		model:  model,
	}
}

type claudeRequest struct {
	Model     string          `json:"model"`
	Messages  []claudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
}

func (c *ClaudeProvider) QueryStream(prompt string, writer io.Writer) error {
	reqBody := claudeRequest{
		Model: c.model,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 4096,
		Stream:    true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := secureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Claude")
	}

	// Parse SSE stream
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading stream: %w", err)
		}
		if n == 0 {
			break
		}

		// Parse SSE events
		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					continue
				}

				var event claudeStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					if event.Type == "content_block_delta" && event.Delta.Text != "" {
						fmt.Fprint(writer, event.Delta.Text)
					}
				}
			}
		}
	}

	return nil
}

func (c *ClaudeProvider) QueryStreamWithHistory(messages []Message, writer io.Writer) error {
	// Convert our Message type to Claude's message format
	var claudeMessages []claudeMessage
	for _, msg := range messages {
		claudeMessages = append(claudeMessages, claudeMessage(msg))
	}

	reqBody := claudeRequest{
		Model:     c.model,
		Messages:  claudeMessages,
		MaxTokens: 4096,
		Stream:    true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := secureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Claude")
	}

	// Parse SSE stream
	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading stream: %w", err)
		}
		if n == 0 {
			break
		}

		// Parse SSE events
		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					continue
				}

				var event claudeStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					if event.Type == "content_block_delta" && event.Delta.Text != "" {
						fmt.Fprint(writer, event.Delta.Text)
					}
				}
			}
		}
	}

	return nil
}

func (c *ClaudeProvider) ListModels() ([]ModelInfo, error) {
	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return getFallbackClaudeModels(), nil
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := secureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return getFallbackClaudeModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getFallbackClaudeModels(), nil
	}

	var result struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getFallbackClaudeModels(), nil
	}

	if len(result.Data) == 0 {
		return getFallbackClaudeModels(), nil
	}

	var models []ModelInfo
	for _, m := range result.Data {
		models = append(models, ModelInfo{
			ID:          m.ID,
			Name:        m.DisplayName,
			Description: "",
		})
	}

	return models, nil
}

func getFallbackClaudeModels() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Description: "Balanced intelligence and speed"},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Description: "Fast and efficient"},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Description: "Most capable"},
		{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", Description: "Balanced performance"},
	}
}

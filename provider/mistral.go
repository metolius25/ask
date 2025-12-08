package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type MistralProvider struct {
	apiKey string
	model  string
}

func NewMistralProvider(apiKey, model string) *MistralProvider {
	// If no model specified, use first available from fallback list
	if model == "" {
		fallbackModels := getFallbackMistralModels()
		if len(fallbackModels) > 0 {
			model = fallbackModels[0].ID
		}
	}
	return &MistralProvider{
		apiKey: apiKey,
		model:  model,
	}
}

type mistralRequest struct {
	Model    string           `json:"model"`
	Messages []mistralMessage `json:"messages"`
	Stream   bool             `json:"stream"`
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (m *MistralProvider) QueryStream(prompt string, writer io.Writer) error {
	reqBody := mistralRequest{
		Model: m.model,
		Messages: []mistralMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Mistral")
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp mistralStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
				if len(streamResp.Choices) > 0 {
					content := streamResp.Choices[0].Delta.Content
					if content != "" {
						fmt.Fprint(writer, content)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

func (m *MistralProvider) QueryStreamWithHistory(messages []Message, writer io.Writer) error {
	// Convert our Message type to Mistral's message format
	var mistralMessages []mistralMessage
	for _, msg := range messages {
		mistralMessages = append(mistralMessages, mistralMessage(msg))
	}

	reqBody := mistralRequest{
		Model:    m.model,
		Messages: mistralMessages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Mistral")
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp mistralStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err == nil {
				if len(streamResp.Choices) > 0 {
					content := streamResp.Choices[0].Delta.Content
					if content != "" {
						fmt.Fprint(writer, content)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

func (m *MistralProvider) ListModels() ([]ModelInfo, error) {
	req, err := http.NewRequest("GET", "https://api.mistral.ai/v1/models", nil)
	if err != nil {
		return getFallbackMistralModels(), nil
	}

	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return getFallbackMistralModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getFallbackMistralModels(), nil
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getFallbackMistralModels(), nil
	}

	if len(result.Data) == 0 {
		return getFallbackMistralModels(), nil
	}

	var models []ModelInfo
	for _, m := range result.Data {
		models = append(models, ModelInfo{
			ID:          m.ID,
			Name:        m.ID,
			Description: "",
		})
	}

	return models, nil
}

func getFallbackMistralModels() []ModelInfo {
	return []ModelInfo{
		{ID: "mistral-large-latest", Name: "Mistral Large", Description: "Most capable model"},
		{ID: "mistral-small-latest", Name: "Mistral Small", Description: "Fast and efficient"},
		{ID: "codestral-latest", Name: "Codestral", Description: "Code generation"},
		{ID: "ministral-8b-latest", Name: "Ministral 8B", Description: "Lightweight model"},
	}
}

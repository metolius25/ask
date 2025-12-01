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

type ChatGPTProvider struct {
	apiKey string
	model  string
}

func NewChatGPTProvider(apiKey, model string) *ChatGPTProvider {
	// If no model specified, use first available from fallback list
	if model == "" {
		fallbackModels := getFallbackChatGPTModels()
		if len(fallbackModels) > 0 {
			model = fallbackModels[0].ID
		}
	}
	return &ChatGPTProvider{
		apiKey: apiKey,
		model:  model,
	}
}

type chatGPTRequest struct {
	Model    string           `json:"model"`
	Messages []chatGPTMessage `json:"messages"`
	Stream   bool             `json:"stream"`
}

type chatGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatGPTStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (c *ChatGPTProvider) QueryStream(prompt string, writer io.Writer) error {
	reqBody := chatGPTRequest{
		Model: c.model,
		Messages: []chatGPTMessage{
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

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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

			var streamResp chatGPTStreamResponse
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

func (c *ChatGPTProvider) ListModels() ([]ModelInfo, error) {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return getFallbackChatGPTModels(), nil
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return getFallbackChatGPTModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getFallbackChatGPTModels(), nil
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getFallbackChatGPTModels(), nil
	}

	if len(result.Data) == 0 {
		return getFallbackChatGPTModels(), nil
	}

	var models []ModelInfo
	// Filter for chat models only
	for _, m := range result.Data {
		if strings.Contains(m.ID, "gpt") || strings.Contains(m.ID, "o1") {
			models = append(models, ModelInfo{
				ID:          m.ID,
				Name:        m.ID,
				Description: "",
			})
		}
	}

	if len(models) == 0 {
		return getFallbackChatGPTModels(), nil
	}

	return models, nil
}

func getFallbackChatGPTModels() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", Description: "Most capable multimodal model"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast and affordable"},
		{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Description: "Advanced reasoning"},
		{ID: "o1-preview", Name: "o1 Preview", Description: "Reasoning model"},
		{ID: "o1-mini", Name: "o1 Mini", Description: "Lightweight reasoning"},
	}
}

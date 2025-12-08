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

const qwenAPIURL = "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions"

type QwenProvider struct {
	apiKey string
	model  string
}

func NewQwenProvider(apiKey, model string) *QwenProvider {
	if model == "" {
		fallbackModels := getFallbackQwenModels()
		if len(fallbackModels) > 0 {
			model = fallbackModels[0].ID
		}
	}
	return &QwenProvider{
		apiKey: apiKey,
		model:  model,
	}
}

type qwenRequest struct {
	Model    string        `json:"model"`
	Messages []qwenMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type qwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type qwenStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (q *QwenProvider) QueryStream(prompt string, writer io.Writer) error {
	reqBody := qwenRequest{
		Model: q.model,
		Messages: []qwenMessage{
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

	req, err := http.NewRequest("POST", qwenAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+q.apiKey)

	client := secureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Qwen")
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

			var streamResp qwenStreamResponse
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

func (q *QwenProvider) QueryStreamWithHistory(messages []Message, writer io.Writer) error {
	var qwenMessages []qwenMessage
	for _, msg := range messages {
		qwenMessages = append(qwenMessages, qwenMessage(msg))
	}

	reqBody := qwenRequest{
		Model:    q.model,
		Messages: qwenMessages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", qwenAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+q.apiKey)

	client := secureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleAPIError(resp.StatusCode, body, "Qwen")
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

			var streamResp qwenStreamResponse
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

func (q *QwenProvider) ListModels() ([]ModelInfo, error) {
	// Qwen doesn't have a public models list API, return fallback
	return getFallbackQwenModels(), nil
}

func getFallbackQwenModels() []ModelInfo {
	return []ModelInfo{
		{ID: "qwen-max", Name: "Qwen Max", Description: "Most capable Qwen model"},
		{ID: "qwen-plus", Name: "Qwen Plus", Description: "Balanced performance"},
		{ID: "qwen-turbo", Name: "Qwen Turbo", Description: "Fast and efficient"},
		{ID: "qwen2.5-72b-instruct", Name: "Qwen 2.5 72B", Description: "Large instruction model"},
		{ID: "qwen2.5-32b-instruct", Name: "Qwen 2.5 32B", Description: "Medium instruction model"},
	}
}

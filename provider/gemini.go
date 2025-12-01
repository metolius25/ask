package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	apiKey string
	model  string
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	// If no model specified, it will be set to first available from fallback list
	// when the provider is actually used
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
	}
}

func (g *GeminiProvider) QueryStream(prompt string, writer io.Writer) error {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	// Normalize model name (remove "models/" prefix if present)
	modelName := g.model

	// If no model specified, use first available from fallback
	if modelName == "" {
		fallbackModels := getFallbackGeminiModels()
		if len(fallbackModels) > 0 {
			modelName = fallbackModels[0].ID
		}
	}

	if len(modelName) > 7 && modelName[:7] == "models/" {
		modelName = modelName[7:]
	}

	model := client.GenerativeModel(modelName)

	// Configure safety settings to be less restrictive
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockOnlyHigh,
		},
	}

	// Stream the response
	iter := model.GenerateContentStream(ctx, genai.Text(prompt))
	hasContent := false

	for {
		resp, err := iter.Next()
		if err != nil {
			// Check if we've reached the end of the stream
			if err.Error() == "no more items in iterator" {
				break
			}
			return fmt.Errorf("error during streaming: %w", err)
		}

		for _, cand := range resp.Candidates {
			// Check if response was blocked
			if cand.FinishReason != 0 && cand.FinishReason != 1 { // 0=UNSPECIFIED, 1=STOP (normal)
				return fmt.Errorf("response blocked (reason: %v). This may be due to safety filters", cand.FinishReason)
			}

			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					fmt.Fprint(writer, part)
					hasContent = true
				}
			}
		}
	}

	if !hasContent {
		return fmt.Errorf("no content received from model - response may have been filtered")
	}

	return nil
}

func (g *GeminiProvider) ListModels() ([]ModelInfo, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
	if err != nil {
		// Fallback to hardcoded list if API call fails
		return getFallbackGeminiModels(), nil
	}
	defer client.Close()

	var models []ModelInfo
	iter := client.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err != nil {
			break
		}

		// Only include generative models that support generateContent
		if model.SupportedGenerationMethods != nil {
			for _, method := range model.SupportedGenerationMethods {
				if method == "generateContent" {
					models = append(models, ModelInfo{
						ID:          model.Name,
						Name:        model.DisplayName,
						Description: model.Description,
					})
					break
				}
			}
		}
	}

	// If no models found, return fallback
	if len(models) == 0 {
		return getFallbackGeminiModels(), nil
	}

	return models, nil
}

func getFallbackGeminiModels() []ModelInfo {
	return []ModelInfo{
		{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Description: "Fast and versatile"},
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Description: "Advanced reasoning"},
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Description: "Previous generation fast model"},
		{ID: "gemini-flash-latest", Name: "Gemini Flash Latest", Description: "Latest Flash release"},
	}
}

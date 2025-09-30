package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-cv-summarize/internal/config"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	config *config.OpenAIConfig
}

func NewOpenAIClient(cfg *config.OpenAIConfig) *OpenAIClient {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.BaseURL

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIClient{
		client: client,
		config: cfg,
	}
}

func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("input text cannot be empty")
	}

	// Truncate if exceeds token limit
	if len(text) > 8000 {
		text = text[:8000]
	}

	text = strings.TrimSpace(text)
	if text == "" || len(text) < 3 {
		return nil, fmt.Errorf("input text is invalid")
	}

	if strings.Contains(text, "\x00") {
		return nil, fmt.Errorf("input text contains null bytes")
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	embedding := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

func (c *OpenAIClient) GenerateCompletion(ctx context.Context, prompt string, temperature float32) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: c.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: temperature,
		MaxTokens:   2000,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) GenerateStructuredCompletion(ctx context.Context, prompt string, temperature float32) (string, error) {
	structuredPrompt := fmt.Sprintf(`%s

IMPORTANT: Respond with ONLY valid JSON. Do not include any additional text, explanations, or formatting outside the JSON object.`, prompt)

	req := openai.ChatCompletionRequest{
		Model: c.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: structuredPrompt,
			},
		},
		Temperature: temperature,
		MaxTokens:   2000,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create structured completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) GenerateCompletionWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoffDuration := time.Duration(i*i) * time.Second
			time.Sleep(backoffDuration)
		}

		result, err := c.GenerateCompletion(ctx, prompt, temperature)
		if err == nil {
			return result, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *OpenAIClient) GenerateStructuredCompletionWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoffDuration := time.Duration(i*i) * time.Second
			time.Sleep(backoffDuration)
		}

		result, err := c.GenerateStructuredCompletion(ctx, prompt, temperature)
		if err == nil {
			return result, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

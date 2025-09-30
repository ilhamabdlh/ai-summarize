package llm

import (
	"ai-cv-summarize/internal/config"
	"context"
)

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	GenerateCompletion(ctx context.Context, prompt string, temperature float32) (string, error)
	GenerateStructuredCompletion(ctx context.Context, prompt string, temperature float32) (string, error)
	GenerateCompletionWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error)
	GenerateStructuredCompletionWithRetry(ctx context.Context, prompt string, temperature float32, maxRetries int) (string, error)
}

// LLMFactory creates LLM clients based on configuration
type LLMFactory struct{}

func NewLLMFactory() *LLMFactory {
	return &LLMFactory{}
}

// CreateClient creates an LLM client based on the provided configuration
func (f *LLMFactory) CreateClient(openAIConfig *config.OpenAIConfig, openRouterConfig *config.OpenRouterConfig) LLMClient {
	// Prioritize OpenAI if API key is available
	if openAIConfig.APIKey != "" {
		return NewOpenAIClient(openAIConfig)
	}

	// Fallback to OpenRouter if OpenAI is not available
	if openRouterConfig.APIKey != "" {
		return NewOpenRouterClient(openRouterConfig)
	}

	// If neither is available, return OpenAI client with empty config (will fail gracefully)
	return NewOpenAIClient(openAIConfig)
}

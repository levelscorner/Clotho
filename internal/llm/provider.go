package llm

import "context"

// CompletionRequest contains the parameters for an LLM completion call.
//
// Universal fields (Model/SystemPrompt/UserPrompt/Temperature/MaxTokens) keep
// their simple types for back-compat. Newer sampling knobs are nullable
// pointers so that adapters can tell "unset, use the provider default" from
// "explicitly set to zero". Adapters drop any field their provider doesn't
// honor (e.g. OpenAI ignores TopK).
type CompletionRequest struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int

	// Near-universal sampling knobs. Nil means "don't send".
	TopP             *float64
	TopK             *int
	StopSequences    []string
	Seed             *int
	FrequencyPenalty *float64
	PresencePenalty  *float64
}

// CompletionResponse contains the result of an LLM completion call.
type CompletionResponse struct {
	Content string
	Usage   TokenUsage
	CostUSD float64
}

// TokenUsage tracks prompt and completion token counts.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamChunk represents a single chunk from a streaming completion.
//
// Reasoning carries chain-of-thought tokens that some reasoning models emit
// as a dedicated delta stream (Ollama gemma/qwen reasoning variants, Gemini
// 2.5 thinking). It must stay separate from Content so the UI can render a
// "Thinking…" panel distinct from the visible output.
type StreamChunk struct {
	Content   string
	Reasoning string
	Usage     *TokenUsage
	Done      bool
}

// Provider is the interface for LLM completion backends.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

package llm

import "context"

// CompletionRequest contains the parameters for an LLM completion call.
type CompletionRequest struct {
	Model        string
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
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
type StreamChunk struct {
	Content string
	Done    bool
}

// Provider is the interface for LLM completion backends.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

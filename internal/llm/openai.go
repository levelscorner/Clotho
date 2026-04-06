package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements Provider using the OpenAI-compatible API.
type OpenAIProvider struct {
	client *openai.Client
}

// NewOpenAI creates a new OpenAIProvider with the given API key.
func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
	}
}

// newOpenAICompatible creates an OpenAIProvider pointing at a custom base URL.
// extraHeaders are injected into every outgoing request.
func newOpenAICompatible(apiKey, baseURL string, extraHeaders map[string]string) *OpenAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL

	if len(extraHeaders) > 0 {
		cfg.HTTPClient = &headerTransport{
			base:    http.DefaultClient,
			headers: extraHeaders,
		}
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(cfg),
	}
}

// headerTransport wraps an http.Client and injects extra headers.
type headerTransport struct {
	base    *http.Client
	headers map[string]string
}

func (h *headerTransport) Do(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return h.base.Do(req)
}

// Complete performs a non-streaming chat completion request.
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	messages := buildMessages(req.SystemPrompt, req.UserPrompt)

	chatReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openai complete: %w", err)
	}

	if len(resp.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("openai complete: no choices returned")
	}

	usage := TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return CompletionResponse{
		Content: resp.Choices[0].Message.Content,
		Usage:   usage,
		CostUSD: CalculateCost(req.Model, usage),
	}, nil
}

// Stream performs a streaming chat completion request and returns a channel of chunks.
func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	messages := buildMessages(req.SystemPrompt, req.UserPrompt)

	chatReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai stream: %w", err)
	}

	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				ch <- StreamChunk{Done: true}
				return
			}
			if err != nil {
				ch <- StreamChunk{Done: true}
				return
			}

			if len(resp.Choices) > 0 {
				ch <- StreamChunk{
					Content: resp.Choices[0].Delta.Content,
					Done:    false,
				}
			}
		}
	}()

	return ch, nil
}

func buildMessages(systemPrompt, userPrompt string) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage
	if systemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userPrompt,
	})
	return messages
}

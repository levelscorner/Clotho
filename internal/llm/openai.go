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
//
// providerName identifies which capability profile to honor at translate
// time — "openai" for the real API, "ollama" / "openrouter" for the
// OpenAI-compatible endpoints those vendors expose. Fields the active
// profile doesn't honor (e.g. top_k on OpenAI proper) are dropped silently.
type OpenAIProvider struct {
	client       *openai.Client
	providerName string
}

// NewOpenAI creates a new OpenAIProvider with the given API key.
func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client:       openai.NewClient(apiKey),
		providerName: "openai",
	}
}

// newOpenAICompatible creates an OpenAIProvider pointing at a custom base URL.
// extraHeaders are injected into every outgoing request. providerName must
// match a key in capabilityTable so translate() drops the right fields.
func newOpenAICompatible(apiKey, baseURL, providerName string, extraHeaders map[string]string) *OpenAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL

	if len(extraHeaders) > 0 {
		cfg.HTTPClient = &headerTransport{
			base:    http.DefaultClient,
			headers: extraHeaders,
		}
	}

	return &OpenAIProvider{
		client:       openai.NewClientWithConfig(cfg),
		providerName: providerName,
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
	chatReq := p.translate(req, false)

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
	chatReq := p.translate(req, true)

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

// translate maps the universal CompletionRequest onto the go-openai
// ChatCompletionRequest. Fields not honored by the active provider profile
// (capabilityTable) are dropped silently. Nil pointers mean "unset" and map
// to zero-value in the outbound struct, which go-openai's `omitempty` tags
// then strip from the wire payload.
func (p *OpenAIProvider) translate(req CompletionRequest, stream bool) openai.ChatCompletionRequest {
	caps := CapabilitiesFor(p.providerName)

	chatReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    buildMessages(req.SystemPrompt, req.UserPrompt),
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Stream:      stream,
	}

	if req.TopP != nil {
		chatReq.TopP = float32(*req.TopP)
	}
	// go-openai has no TopK field. If the active profile honors it we'd
	// need a raw-body path; the OpenAI SDK itself doesn't wire top_k, so
	// top_k reaches Ollama only once we add a raw extension. Phase 1
	// drops TopK silently across all openai-compat adapters.
	if caps.StopSequences && len(req.StopSequences) > 0 {
		chatReq.Stop = req.StopSequences
	}
	if caps.Seed && req.Seed != nil {
		seed := *req.Seed
		chatReq.Seed = &seed
	}
	if caps.FrequencyPenalty && req.FrequencyPenalty != nil {
		chatReq.FrequencyPenalty = float32(*req.FrequencyPenalty)
	}
	if caps.PresencePenalty && req.PresencePenalty != nil {
		chatReq.PresencePenalty = float32(*req.PresencePenalty)
	}

	return chatReq
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

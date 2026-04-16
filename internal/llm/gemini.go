package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/user/clotho/internal/util/redact"
)

// geminiBaseURL is a var (not const) so integration tests can redirect
// traffic at an httptest.Server without a build-tag dance.
var geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiProvider implements Provider using the Google AI Studio REST API.
type GeminiProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGemini creates a new GeminiProvider with the given API key.
func NewGemini(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// geminiRequest is the request body for the Gemini generateContent API.
type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature      float64  `json:"temperature"`
	MaxOutputTokens  int      `json:"maxOutputTokens"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	PresencePenalty  *float64 `json:"presencePenalty,omitempty"`
}

// geminiResponse is the response from the Gemini generateContent API.
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

// Complete performs a non-streaming completion request to Gemini.
func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, req.Model, p.apiKey)

	body := p.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("gemini complete: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("gemini complete: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("gemini complete: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return CompletionResponse{}, fmt.Errorf("gemini complete: status %d: %s", httpResp.StatusCode, redact.Secrets(string(respBody)))
	}

	var gemResp geminiResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&gemResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("gemini complete: decode response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return CompletionResponse{}, fmt.Errorf("gemini complete: no candidates returned")
	}

	usage := TokenUsage{
		PromptTokens:     gemResp.UsageMetadata.PromptTokenCount,
		CompletionTokens: gemResp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      gemResp.UsageMetadata.PromptTokenCount + gemResp.UsageMetadata.CandidatesTokenCount,
	}

	return CompletionResponse{
		Content: gemResp.Candidates[0].Content.Parts[0].Text,
		Usage:   usage,
		CostUSD: CalculateCost(req.Model, usage),
	}, nil
}

// Stream performs a streaming completion request to Gemini using SSE.
func (p *GeminiProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", geminiBaseURL, req.Model, p.apiKey)

	body := p.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini stream: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("gemini stream: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini stream: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("gemini stream: status %d: %s", httpResp.StatusCode, redact.Secrets(string(respBody)))
	}

	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var gemResp geminiResponse
			if err := json.Unmarshal([]byte(data), &gemResp); err != nil {
				continue
			}

			if len(gemResp.Candidates) > 0 && len(gemResp.Candidates[0].Content.Parts) > 0 {
				ch <- StreamChunk{
					Content: gemResp.Candidates[0].Content.Parts[0].Text,
					Done:    false,
				}
			}
		}
		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

// ListModels returns the available Gemini models.
func (p *GeminiProvider) ListModels() []string {
	return []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"}
}

func (p *GeminiProvider) buildRequestBody(req CompletionRequest) geminiRequest {
	caps := CapabilitiesFor("gemini")

	gc := geminiGenConfig{
		Temperature:     req.Temperature,
		MaxOutputTokens: req.MaxTokens,
	}
	if req.TopP != nil {
		v := *req.TopP
		gc.TopP = &v
	}
	if caps.TopK && req.TopK != nil {
		v := *req.TopK
		gc.TopK = &v
	}
	if caps.StopSequences && len(req.StopSequences) > 0 {
		gc.StopSequences = req.StopSequences
	}
	if caps.Seed && req.Seed != nil {
		v := *req.Seed
		gc.Seed = &v
	}
	if caps.FrequencyPenalty && req.FrequencyPenalty != nil {
		v := *req.FrequencyPenalty
		gc.FrequencyPenalty = &v
	}
	if caps.PresencePenalty && req.PresencePenalty != nil {
		v := *req.PresencePenalty
		gc.PresencePenalty = &v
	}

	gr := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: req.UserPrompt}}},
		},
		GenerationConfig: gc,
	}

	if req.SystemPrompt != "" {
		gr.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	return gr
}

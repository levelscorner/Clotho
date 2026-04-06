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
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

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
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
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
		return CompletionResponse{}, fmt.Errorf("gemini complete: status %d: %s", httpResp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("gemini stream: status %d: %s", httpResp.StatusCode, string(respBody))
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

func (p *GeminiProvider) buildRequestBody(req CompletionRequest) geminiRequest {
	gr := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: req.UserPrompt}}},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		},
	}

	if req.SystemPrompt != "" {
		gr.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	return gr
}

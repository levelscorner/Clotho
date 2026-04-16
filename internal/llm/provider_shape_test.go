package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOpenAIProviderShape spins a fake OpenAI endpoint and asserts the
// knobs on CompletionRequest reach the outbound JSON body. Capability-
// gated fields (top_p, seed, penalty pair) should be present when set;
// top_k is not in the go-openai SDK, so it's dropped silently.
func TestOpenAIProviderShape(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "cmpl-1",
			"object": "chat.completion",
			"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	p := newOpenAICompatible("test-key", server.URL, "openai", nil)

	topP := 0.8
	seed := 42
	freq := 0.25
	pres := 0.1
	_, err := p.Complete(context.Background(), CompletionRequest{
		Model:            "gpt-4o-mini",
		SystemPrompt:     "sys",
		UserPrompt:       "hi",
		Temperature:      0.7,
		MaxTokens:        128,
		TopP:             &topP,
		StopSequences:    []string{"###"},
		Seed:             &seed,
		FrequencyPenalty: &freq,
		PresencePenalty:  &pres,
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if gotBody["model"] != "gpt-4o-mini" {
		t.Errorf("model = %v, want gpt-4o-mini", gotBody["model"])
	}
	if gotBody["top_p"] == nil {
		t.Errorf("top_p missing from request body: %v", gotBody)
	}
	if gotBody["seed"] == nil {
		t.Errorf("seed missing from request body: %v", gotBody)
	}
	if gotBody["frequency_penalty"] == nil {
		t.Errorf("frequency_penalty missing from request body")
	}
	if gotBody["presence_penalty"] == nil {
		t.Errorf("presence_penalty missing from request body")
	}
	stops, _ := gotBody["stop"].([]any)
	if len(stops) != 1 || stops[0] != "###" {
		t.Errorf("stop = %v, want [\"###\"]", gotBody["stop"])
	}
}

// TestOllamaProviderDropsPenalty confirms that the "ollama" profile drops
// frequency/presence penalty (Ollama's OpenAI-compat endpoint treats these
// silently but we honor the capability table for consistency).
func TestOllamaProviderDropsPenalty(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "cmpl-1",
			"object": "chat.completion",
			"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	p := newOpenAICompatible("ollama", server.URL, "ollama", nil)

	freq := 0.3
	pres := 0.2
	seed := 7
	_, err := p.Complete(context.Background(), CompletionRequest{
		Model:            "llama3.1",
		UserPrompt:       "hi",
		Temperature:      0.5,
		MaxTokens:        32,
		Seed:             &seed,
		FrequencyPenalty: &freq,
		PresencePenalty:  &pres,
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Seed is honored by ollama profile.
	if gotBody["seed"] == nil {
		t.Errorf("seed should be present for ollama, got body %v", gotBody)
	}
	// Penalties should be dropped (go-openai omits zero-value with omitempty).
	if _, ok := gotBody["frequency_penalty"]; ok {
		t.Errorf("frequency_penalty should be dropped for ollama profile")
	}
	if _, ok := gotBody["presence_penalty"]; ok {
		t.Errorf("presence_penalty should be dropped for ollama profile")
	}
}

// TestGeminiProviderShape asserts all near-universal knobs land in the
// generationConfig block of the outbound Gemini request.
func TestGeminiProviderShape(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{"content": {"parts": [{"text": "ok"}]}}],
			"usageMetadata": {"promptTokenCount": 1, "candidatesTokenCount": 1}
		}`))
	}))
	defer server.Close()

	// Swap the hardcoded base URL with the test server. We need a fresh
	// GeminiProvider whose http calls hit the test server — inject via
	// overriding geminiBaseURL for the duration of this test.
	origBase := geminiBaseURL
	defer func() { geminiBaseURL = origBase }()
	geminiBaseURL = server.URL

	p := NewGemini("test-key")

	topP := 0.95
	topK := 40
	seed := 7
	freq := 0.1
	pres := 0.2

	_, err := p.Complete(context.Background(), CompletionRequest{
		Model:            "gemini-2.5-flash",
		SystemPrompt:     "sys",
		UserPrompt:       "hi",
		Temperature:      0.6,
		MaxTokens:        64,
		TopP:             &topP,
		TopK:             &topK,
		StopSequences:    []string{"STOP"},
		Seed:             &seed,
		FrequencyPenalty: &freq,
		PresencePenalty:  &pres,
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	gen, ok := gotBody["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig missing: %v", gotBody)
	}
	if gen["topP"] == nil {
		t.Errorf("topP missing: %v", gen)
	}
	if gen["topK"] == nil {
		t.Errorf("topK missing: %v", gen)
	}
	if gen["seed"] == nil {
		t.Errorf("seed missing: %v", gen)
	}
	if gen["frequencyPenalty"] == nil {
		t.Errorf("frequencyPenalty missing: %v", gen)
	}
	if gen["presencePenalty"] == nil {
		t.Errorf("presencePenalty missing: %v", gen)
	}
	stops, _ := gen["stopSequences"].([]any)
	if len(stops) != 1 || stops[0] != "STOP" {
		t.Errorf("stopSequences = %v, want [\"STOP\"]", gen["stopSequences"])
	}
}

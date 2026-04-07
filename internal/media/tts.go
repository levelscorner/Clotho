package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const openaiSpeechURL = "https://api.openai.com/v1/audio/speech"

// TTS implements Provider using the OpenAI TTS API (synchronous).
type TTS struct {
	apiKey string
	client *http.Client

	mu      sync.RWMutex
	results map[string]MediaStatus // synthetic job store
}

// NewTTS creates a TTS provider with the given API key.
func NewTTS(apiKey string) *TTS {
	return &TTS{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		results: make(map[string]MediaStatus),
	}
}

// ttsRequest is the POST body for the OpenAI audio/speech API.
type ttsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

// Submit generates speech synchronously via OpenAI TTS and stores the result as a base64 data URI.
func (t *TTS) Submit(ctx context.Context, req MediaRequest) (string, error) {
	model := req.Model
	if model == "" {
		model = "tts-1"
	}

	voice := req.Voice
	if voice == "" {
		voice = "alloy"
	}

	body := ttsRequest{
		Model: model,
		Input: req.Prompt,
		Voice: voice,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("tts: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiSpeechURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("tts: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+t.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("tts: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("tts: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error JSON
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("tts: API error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("tts: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	// Response is raw MP3 bytes; encode as base64 data URI
	encoded := base64.StdEncoding.EncodeToString(respBody)
	dataURI := "data:audio/mp3;base64," + encoded

	jobID := uuid.New().String()
	t.storeResult(jobID, MediaStatus{
		State:  "succeeded",
		Output: dataURI,
	})

	return jobID, nil
}

// Poll returns the stored result for a TTS job (always immediately available).
func (t *TTS) Poll(_ context.Context, jobID string) (MediaStatus, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status, ok := t.results[jobID]
	if !ok {
		return MediaStatus{}, fmt.Errorf("tts: unknown job %s", jobID)
	}
	return status, nil
}

// storeResult saves a completed result for later retrieval by Poll.
func (t *TTS) storeResult(jobID string, status MediaStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results[jobID] = status
}

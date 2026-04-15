package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/util/redact"
)

// Kokoro implements Provider using a local Kokoro-FastAPI server that exposes
// an OpenAI-compatible /v1/audio/speech endpoint.
//
// The upstream project (https://github.com/remsky/Kokoro-FastAPI) wraps the
// 82M-parameter Kokoro TTS model behind a FastAPI server on port 8880 by
// default. Because the endpoint shape mirrors OpenAI TTS, the request/response
// plumbing is nearly identical to tts.go, but we keep it a separate type to
// make the provider name explicit at the registry level.
//
// All inference is local: cost is always zero and no API key is required.
type Kokoro struct {
	baseURL string
	client  *http.Client

	mu      sync.RWMutex
	results map[string]MediaStatus // synthetic job store
}

// NewKokoro creates a Kokoro provider pointed at the given base URL.
// Example: "http://localhost:8880".
func NewKokoro(baseURL string) *Kokoro {
	return &Kokoro{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		results: make(map[string]MediaStatus),
	}
}

// kokoroSpeechRequest is the OpenAI-compatible POST body understood by
// Kokoro-FastAPI's /v1/audio/speech endpoint.
type kokoroSpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// Submit synthesises speech synchronously and stores the result as a base64
// data URI. If Kokoro is unreachable, returns an error with enough context for
// the caller to surface the "Kokoro not running" hint in the UI.
func (k *Kokoro) Submit(ctx context.Context, req MediaRequest) (string, error) {
	if k.baseURL == "" {
		return "", fmt.Errorf("kokoro: base URL not configured")
	}

	model := req.Model
	if model == "" {
		model = "kokoro"
	}

	// Kokoro ships with a handful of voices under the v1_0 bundle. "af_bella"
	// is a warm female default. Callers can override via MediaRequest.Voice.
	voice := req.Voice
	if voice == "" {
		voice = "af_bella"
	}

	body := kokoroSpeechRequest{
		Model:          model,
		Input:          req.Prompt,
		Voice:          voice,
		ResponseFormat: "mp3",
		Speed:          1.0,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("kokoro: marshal request: %w", err)
	}

	url := k.baseURL + "/v1/audio/speech"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("kokoro: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Kokoro-FastAPI accepts (but does not require) an Authorization header —
	// we omit it entirely since there is no secret to protect.

	resp, err := k.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("kokoro: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("kokoro: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error JSON — Kokoro returns {"detail": "..."} on 4xx.
		var errResp struct {
			Detail string `json:"detail"`
			Error  struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil {
			if errResp.Detail != "" {
				return "", fmt.Errorf("kokoro: API error: %s", errResp.Detail)
			}
			if errResp.Error.Message != "" {
				return "", fmt.Errorf("kokoro: API error: %s", errResp.Error.Message)
			}
		}
		return "", fmt.Errorf("kokoro: unexpected status %d: %s", resp.StatusCode, redact.Secrets(string(respBody)))
	}

	// Response body is raw MP3 audio. Encode as base64 data URI so the
	// existing media-rendering path (which handles DALL-E base64 output)
	// can surface it without disk I/O.
	encoded := base64.StdEncoding.EncodeToString(respBody)
	dataURI := "data:audio/mp3;base64," + encoded

	jobID := uuid.New().String()
	k.storeResult(jobID, MediaStatus{
		State:  "succeeded",
		Output: dataURI,
	})

	return jobID, nil
}

// Poll returns the stored result for a Kokoro job. Kokoro inference is
// synchronous, so every Submit-returned jobID is already terminal.
func (k *Kokoro) Poll(_ context.Context, jobID string) (MediaStatus, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	status, ok := k.results[jobID]
	if !ok {
		return MediaStatus{}, fmt.Errorf("kokoro: unknown job %s", jobID)
	}
	return status, nil
}

// storeResult persists a completed result for later retrieval by Poll.
func (k *Kokoro) storeResult(jobID string, status MediaStatus) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.results[jobID] = status
}

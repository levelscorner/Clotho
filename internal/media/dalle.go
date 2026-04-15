package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/util/redact"
)

const openaiImagesURL = "https://api.openai.com/v1/images/generations"

// DALLE implements Provider using the OpenAI DALL-E API (synchronous).
type DALLE struct {
	apiKey string
	client *http.Client

	mu      sync.RWMutex
	results map[string]MediaStatus // synthetic job store
}

// NewDALLE creates a DALL-E provider with the given API key.
func NewDALLE(apiKey string) *DALLE {
	return &DALLE{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second, // image gen can be slow
		},
		results: make(map[string]MediaStatus),
	}
}

// dalleRequest is the POST body for the OpenAI images/generations API.
type dalleRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
	N       int    `json:"n"`
}

// dalleResponse is the response from the OpenAI images/generations API.
type dalleResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Submit generates an image synchronously via DALL-E and stores the result.
func (d *DALLE) Submit(ctx context.Context, req MediaRequest) (string, error) {
	model := req.Model
	if model == "" {
		model = "dall-e-3"
	}

	size := aspectRatioToSize(req.AspectRatio)
	n := req.NumOutputs
	if n <= 0 {
		n = 1
	}

	body := dalleRequest{
		Model:   model,
		Prompt:  req.Prompt,
		Size:    size,
		Quality: "standard",
		N:       n,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("dalle: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiImagesURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("dalle: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("dalle: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("dalle: read response: %w", err)
	}

	var dalleResp dalleResponse
	if err := json.Unmarshal(respBody, &dalleResp); err != nil {
		return "", fmt.Errorf("dalle: unmarshal response: %w", err)
	}

	if dalleResp.Error != nil {
		jobID := uuid.New().String()
		d.storeResult(jobID, MediaStatus{
			State: "failed",
			Error: dalleResp.Error.Message,
		})
		return jobID, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("dalle: unexpected status %d: %s", resp.StatusCode, redact.Secrets(string(respBody)))
	}

	if len(dalleResp.Data) == 0 {
		return "", fmt.Errorf("dalle: no images returned")
	}

	jobID := uuid.New().String()
	d.storeResult(jobID, MediaStatus{
		State:  "succeeded",
		Output: dalleResp.Data[0].URL,
	})

	return jobID, nil
}

// Poll returns the stored result for a DALL-E job (always immediately available).
func (d *DALLE) Poll(_ context.Context, jobID string) (MediaStatus, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status, ok := d.results[jobID]
	if !ok {
		return MediaStatus{}, fmt.Errorf("dalle: unknown job %s", jobID)
	}
	return status, nil
}

// storeResult saves a completed result for later retrieval by Poll.
func (d *DALLE) storeResult(jobID string, status MediaStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.results[jobID] = status
}

// aspectRatioToSize maps common aspect ratios to DALL-E size parameters.
func aspectRatioToSize(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "1792x1024"
	case "9:16":
		return "1024x1792"
	case "1:1", "":
		return "1024x1024"
	default:
		return "1024x1024"
	}
}

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

	"github.com/user/clotho/internal/util/redact"
)

const replicateBaseURL = "https://api.replicate.com/v1"

// Replicate implements Provider using the Replicate API.
type Replicate struct {
	apiToken string
	client   *http.Client
}

// NewReplicate creates a Replicate provider with the given API token.
func NewReplicate(apiToken string) *Replicate {
	return &Replicate{
		apiToken: apiToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// replicateCreateRequest is the POST body for creating a prediction.
type replicateCreateRequest struct {
	Model string                 `json:"model"`
	Input map[string]interface{} `json:"input"`
}

// replicatePrediction is the response from the Replicate predictions API.
type replicatePrediction struct {
	ID     string      `json:"id"`
	Status string      `json:"status"` // starting, processing, succeeded, failed, canceled
	Output interface{} `json:"output"` // array of URLs on success
	Error  interface{} `json:"error"`  // error details on failure
}

// modelMap maps short model names to full Replicate model identifiers.
var modelMap = map[string]string{
	"flux-1.1-pro":             "black-forest-labs/flux-1.1-pro",
	"stable-video-diffusion":   "stability-ai/stable-video-diffusion",
}

var modelMapMu sync.RWMutex

// Submit creates a prediction on Replicate and returns the prediction ID.
func (r *Replicate) Submit(ctx context.Context, req MediaRequest) (string, error) {
	model := resolveModel(req.Model)

	input := map[string]interface{}{
		"prompt": req.Prompt,
	}
	if req.AspectRatio != "" {
		input["aspect_ratio"] = req.AspectRatio
	}
	if req.NumOutputs > 0 {
		input["num_outputs"] = req.NumOutputs
	}
	if req.Duration > 0 {
		input["duration"] = req.Duration
	}
	if req.ImageURL != "" {
		input["image"] = req.ImageURL
	}
	for k, v := range req.Extra {
		input[k] = v
	}

	body := replicateCreateRequest{
		Model: model,
		Input: input,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("replicate: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, replicateBaseURL+"/predictions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("replicate: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Prefer", "respond-async")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("replicate: submit request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("replicate: read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("replicate: unexpected status %d: %s", resp.StatusCode, redact.Secrets(string(respBody)))
	}

	var prediction replicatePrediction
	if err := json.Unmarshal(respBody, &prediction); err != nil {
		return "", fmt.Errorf("replicate: unmarshal response: %w", err)
	}

	return prediction.ID, nil
}

// Poll checks the status of a Replicate prediction.
func (r *Replicate) Poll(ctx context.Context, jobID string) (MediaStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, replicateBaseURL+"/predictions/"+jobID, nil)
	if err != nil {
		return MediaStatus{}, fmt.Errorf("replicate: create poll request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.apiToken)

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return MediaStatus{}, fmt.Errorf("replicate: poll request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return MediaStatus{}, fmt.Errorf("replicate: read poll response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return MediaStatus{}, fmt.Errorf("replicate: poll status %d: %s", resp.StatusCode, redact.Secrets(string(respBody)))
	}

	var prediction replicatePrediction
	if err := json.Unmarshal(respBody, &prediction); err != nil {
		return MediaStatus{}, fmt.Errorf("replicate: unmarshal poll response: %w", err)
	}

	status := mapReplicateStatus(prediction.Status)

	var output string
	if prediction.Output != nil {
		output = extractReplicateOutput(prediction.Output)
	}

	var errMsg string
	if prediction.Error != nil {
		if e, ok := prediction.Error.(string); ok {
			errMsg = e
		} else {
			errBytes, _ := json.Marshal(prediction.Error)
			errMsg = string(errBytes)
		}
	}

	return MediaStatus{
		State:  status,
		Output: output,
		Error:  errMsg,
	}, nil
}

// mapReplicateStatus converts Replicate status strings to MediaStatus state values.
func mapReplicateStatus(status string) string {
	switch status {
	case "starting", "processing":
		return "processing"
	case "succeeded":
		return "succeeded"
	case "failed":
		return "failed"
	case "canceled":
		return "cancelled"
	default:
		return "pending"
	}
}

// extractReplicateOutput extracts the first URL from the Replicate output field.
func extractReplicateOutput(output interface{}) string {
	switch v := output.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// resolveModel maps short model names to full Replicate identifiers.
func resolveModel(model string) string {
	modelMapMu.RLock()
	defer modelMapMu.RUnlock()

	if full, ok := modelMap[model]; ok {
		return full
	}
	return model
}

package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/user/clotho/internal/util/redact"
)

// ComfyUI implements Provider against a local ComfyUI server
// (https://github.com/comfyanonymous/ComfyUI).
//
// The integration uses ComfyUI's workflow API:
//
//	POST /prompt                      queue a workflow, returns prompt_id
//	GET  /history/{prompt_id}         poll execution state + output paths
//	GET  /view?filename=...&type=...  fetch generated images
//
// We embed a FLUX.1-schnell text-to-image workflow and substitute the
// caller's prompt + a random seed. Inference cost is always zero.
//
// Submit() returns the ComfyUI prompt_id as the jobID. Poll() converts
// ComfyUI's history entry into a MediaStatus. Completed images are
// fetched over HTTP and returned as base64 data URIs so they flow
// through the existing rendering path without touching disk.
type ComfyUI struct {
	baseURL string
	client  *http.Client
}

// NewComfyUI creates a ComfyUI provider pointed at the given base URL.
// Example: "http://localhost:8188".
func NewComfyUI(baseURL string) *ComfyUI {
	return &ComfyUI{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			// Generous timeout — FLUX.1-schnell on M-series MPS typically
			// completes in 5-20s but first-request weight loading can
			// stretch this. The engine has its own deadline above this.
			Timeout: 5 * time.Minute,
		},
	}
}

// fluxSchnellWorkflow is the baseline FLUX.1-schnell text-to-image workflow.
// It uses the all-in-one checkpoint from Comfy-Org/flux1-schnell which
// includes UNet, VAE, and text encoders in a single safetensors file.
//
// Template placeholders:
//
//	{{PROMPT}}  — user's text prompt (JSON-escaped)
//	{{SEED}}    — integer seed
//	{{WIDTH}}   — output width
//	{{HEIGHT}}  — output height
//	{{STEPS}}   — sampler steps (FLUX schnell needs 4)
var fluxSchnellWorkflow = `{
  "1": {
    "class_type": "CheckpointLoaderSimple",
    "inputs": { "ckpt_name": "flux1-schnell-fp8.safetensors" }
  },
  "2": {
    "class_type": "CLIPTextEncode",
    "inputs": { "text": "{{PROMPT}}", "clip": ["1", 1] }
  },
  "3": {
    "class_type": "CLIPTextEncode",
    "inputs": { "text": "", "clip": ["1", 1] }
  },
  "4": {
    "class_type": "EmptyLatentImage",
    "inputs": { "width": {{WIDTH}}, "height": {{HEIGHT}}, "batch_size": 1 }
  },
  "5": {
    "class_type": "KSampler",
    "inputs": {
      "seed": {{SEED}},
      "steps": {{STEPS}},
      "cfg": 1.0,
      "sampler_name": "euler",
      "scheduler": "simple",
      "denoise": 1.0,
      "model": ["1", 0],
      "positive": ["2", 0],
      "negative": ["3", 0],
      "latent_image": ["4", 0]
    }
  },
  "6": {
    "class_type": "VAEDecode",
    "inputs": { "samples": ["5", 0], "vae": ["1", 2] }
  },
  "7": {
    "class_type": "SaveImage",
    "inputs": { "filename_prefix": "clotho_", "images": ["6", 0] }
  }
}`

type comfyPromptRequest struct {
	Prompt   map[string]any `json:"prompt"`
	ClientID string         `json:"client_id,omitempty"`
}

type comfyPromptResponse struct {
	PromptID string `json:"prompt_id"`
	// node_errors and error fields can also be present on validation failure.
	Error      any            `json:"error,omitempty"`
	NodeErrors map[string]any `json:"node_errors,omitempty"`
}

type comfyHistoryEntry struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
	} `json:"outputs"`
	Status struct {
		StatusStr string `json:"status_str"`
		Completed bool   `json:"completed"`
		Messages  []any  `json:"messages"`
	} `json:"status"`
}

// Submit queues a text-to-image workflow with the caller's prompt and returns
// the ComfyUI prompt_id for later polling.
func (c *ComfyUI) Submit(ctx context.Context, req MediaRequest) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("comfyui: base URL not configured")
	}

	width, height := parseAspectRatio(req.AspectRatio)
	steps := 4 // FLUX.1-schnell converges in 4 steps
	seed := int(rand.Int64N(1_000_000_000))

	filled := fluxSchnellWorkflow
	filled = strings.ReplaceAll(filled, "{{PROMPT}}", jsonEscape(req.Prompt))
	filled = strings.ReplaceAll(filled, "{{SEED}}", fmt.Sprintf("%d", seed))
	filled = strings.ReplaceAll(filled, "{{WIDTH}}", fmt.Sprintf("%d", width))
	filled = strings.ReplaceAll(filled, "{{HEIGHT}}", fmt.Sprintf("%d", height))
	filled = strings.ReplaceAll(filled, "{{STEPS}}", fmt.Sprintf("%d", steps))

	var workflow map[string]any
	if err := json.Unmarshal([]byte(filled), &workflow); err != nil {
		return "", fmt.Errorf("comfyui: invalid embedded workflow: %w", err)
	}

	body := comfyPromptRequest{
		Prompt:   workflow,
		ClientID: "clotho",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("comfyui: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/prompt", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("comfyui: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("comfyui: queue prompt: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("comfyui: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("comfyui: unexpected status %d: %s", resp.StatusCode, redact.Secrets(truncate(string(respBody), 300)))
	}

	var parsed comfyPromptResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("comfyui: decode prompt response: %w", err)
	}
	if parsed.PromptID == "" {
		return "", fmt.Errorf("comfyui: server accepted workflow but returned no prompt_id; body=%s", redact.Secrets(truncate(string(respBody), 300)))
	}
	return parsed.PromptID, nil
}

// Poll checks ComfyUI history for the given prompt_id and returns the current
// MediaStatus. Completed images are fetched and returned as base64 data URIs.
func (c *ComfyUI) Poll(ctx context.Context, jobID string) (MediaStatus, error) {
	histURL := fmt.Sprintf("%s/history/%s", c.baseURL, url.PathEscape(jobID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, histURL, nil)
	if err != nil {
		return MediaStatus{}, fmt.Errorf("comfyui: poll request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return MediaStatus{}, fmt.Errorf("comfyui: poll: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MediaStatus{}, fmt.Errorf("comfyui: poll status %d", resp.StatusCode)
	}

	var hist map[string]comfyHistoryEntry
	if err := json.NewDecoder(resp.Body).Decode(&hist); err != nil {
		return MediaStatus{}, fmt.Errorf("comfyui: decode history: %w", err)
	}

	entry, ok := hist[jobID]
	if !ok {
		// Not yet in history → still queued or running.
		return MediaStatus{State: "processing"}, nil
	}

	switch strings.ToLower(entry.Status.StatusStr) {
	case "error":
		return MediaStatus{
			State: "failed",
			Error: "comfyui workflow error; inspect ComfyUI logs",
		}, nil
	case "success":
		// Find first produced image across all SaveImage nodes.
		var img *struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		}
		for _, out := range entry.Outputs {
			if len(out.Images) == 0 {
				continue
			}
			img = &out.Images[0]
			break
		}
		if img == nil {
			return MediaStatus{State: "failed", Error: "comfyui completed but produced no image"}, nil
		}
		dataURI, err := c.fetchImage(ctx, img.Filename, img.Subfolder, img.Type)
		if err != nil {
			return MediaStatus{State: "failed", Error: err.Error()}, nil
		}
		return MediaStatus{State: "succeeded", Output: dataURI}, nil
	default:
		// "executing", "queued", empty string → still in flight.
		return MediaStatus{State: "processing"}, nil
	}
}

// fetchImage downloads a generated image from ComfyUI's /view endpoint and
// returns a base64 data URI suitable for inline rendering.
func (c *ComfyUI) fetchImage(ctx context.Context, filename, subfolder, imgType string) (string, error) {
	if filename == "" {
		return "", errors.New("comfyui: empty image filename")
	}
	q := url.Values{}
	q.Set("filename", filename)
	if subfolder != "" {
		q.Set("subfolder", subfolder)
	}
	if imgType != "" {
		q.Set("type", imgType)
	}
	imgURL := fmt.Sprintf("%s/view?%s", c.baseURL, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return "", fmt.Errorf("comfyui: fetch request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("comfyui: fetch image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("comfyui: fetch image status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("comfyui: read image: %w", err)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	return fmt.Sprintf("data:%s;base64,%s", ct, base64Encode(b)), nil
}

// parseAspectRatio maps a ratio like "16:9" or "1:1" to (width, height) in
// pixels, defaulting to 1024x1024 when empty/invalid. FLUX prefers dimensions
// that are multiples of 64.
func parseAspectRatio(aspect string) (int, int) {
	switch strings.TrimSpace(aspect) {
	case "", "1:1":
		return 1024, 1024
	case "16:9":
		return 1344, 768
	case "9:16":
		return 768, 1344
	case "4:3":
		return 1152, 896
	case "3:4":
		return 896, 1152
	case "21:9":
		return 1536, 640
	default:
		return 1024, 1024
	}
}

// jsonEscape returns s escaped so it can be embedded inside a JSON string
// literal. We can't use json.Marshal directly because the template already
// includes the surrounding quotes.
func jsonEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}
	// Strip the surrounding quotes that Marshal adds.
	return string(b[1 : len(b)-1])
}

// truncate limits a string to n runes for logging without dragging in a new
// dependency on unicode/utf8 just for an error path.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// base64Encode is a thin wrapper for testability.
var base64Encode = func(b []byte) string {
	return base64StdEncode(b)
}

func base64StdEncode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

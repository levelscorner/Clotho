package media

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestComfyUI_Submit_HappyPath(t *testing.T) {
	t.Parallel()

	var receivedWorkflow map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prompt" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		var body comfyPromptRequest
		_ = decodeJSON(r, &body)
		receivedWorkflow = body.Prompt
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prompt_id":"test-prompt-123","number":0,"node_errors":{}}`))
	}))
	defer srv.Close()

	c := NewComfyUI(srv.URL)
	jobID, err := c.Submit(context.Background(), MediaRequest{
		Prompt:      "a red cube",
		AspectRatio: "1:1",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if jobID != "test-prompt-123" {
		t.Errorf("expected prompt_id passthrough, got %q", jobID)
	}
	// Verify the prompt got substituted into the CLIPTextEncode node (node "2").
	node2, ok := receivedWorkflow["2"].(map[string]any)
	if !ok {
		t.Fatalf("expected node 2 in workflow, got %v", receivedWorkflow)
	}
	inputs := node2["inputs"].(map[string]any)
	if inputs["text"] != "a red cube" {
		t.Errorf("prompt substitution failed: got %q", inputs["text"])
	}
}

func TestComfyUI_Submit_AspectRatios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		aspect string
		w, h   int
	}{
		{"", 1024, 1024},
		{"1:1", 1024, 1024},
		{"16:9", 1344, 768},
		{"9:16", 768, 1344},
		{"4:3", 1152, 896},
		{"weird", 1024, 1024}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.aspect, func(t *testing.T) {
			w, h := parseAspectRatio(tt.aspect)
			if w != tt.w || h != tt.h {
				t.Errorf("aspect %q: got %dx%d, want %dx%d", tt.aspect, w, h, tt.w, tt.h)
			}
		})
	}
}

func TestComfyUI_Submit_PromptEscaping(t *testing.T) {
	t.Parallel()

	var receivedWorkflow map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body comfyPromptRequest
		_ = decodeJSON(r, &body)
		receivedWorkflow = body.Prompt
		_, _ = w.Write([]byte(`{"prompt_id":"x"}`))
	}))
	defer srv.Close()

	// Prompt contains characters that would break naive string interpolation
	// into JSON: double quotes, backslash, newlines.
	tricky := `line 1 "quoted" \ line 2
line 3`
	c := NewComfyUI(srv.URL)
	_, err := c.Submit(context.Background(), MediaRequest{Prompt: tricky})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	node2 := receivedWorkflow["2"].(map[string]any)
	inputs := node2["inputs"].(map[string]any)
	if inputs["text"] != tricky {
		t.Errorf("escaping lost content: got %q want %q", inputs["text"], tricky)
	}
}

func TestComfyUI_Submit_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"workflow validation failed"}`))
	}))
	defer srv.Close()

	c := NewComfyUI(srv.URL)
	_, err := c.Submit(context.Background(), MediaRequest{Prompt: "x"})
	if err == nil {
		t.Fatal("expected error on 500 response")
	}
	if !strings.Contains(err.Error(), "comfyui:") {
		t.Errorf("expected namespaced error, got: %v", err)
	}
}

func TestComfyUI_Poll_Processing(t *testing.T) {
	t.Parallel()

	// History endpoint returns an empty object = still queued/running.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewComfyUI(srv.URL)
	status, err := c.Poll(context.Background(), "some-id")
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if status.State != "processing" {
		t.Errorf("expected processing, got %q", status.State)
	}
}

func TestComfyUI_Poll_Success(t *testing.T) {
	t.Parallel()

	// Serve a history entry with one image, then a fake PNG byte sequence on /view.
	var mu sync.Mutex
	viewCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/history/"):
			_, _ = w.Write([]byte(`{
				"promptid-42": {
					"outputs": {
						"7": { "images": [ { "filename": "clotho_00001_.png", "subfolder": "", "type": "output" } ] }
					},
					"status": { "status_str": "success", "completed": true, "messages": [] }
				}
			}`))
		case r.URL.Path == "/view":
			mu.Lock()
			viewCalled = true
			mu.Unlock()
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) // PNG magic
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewComfyUI(srv.URL)
	status, err := c.Poll(context.Background(), "promptid-42")
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if status.State != "succeeded" {
		t.Errorf("expected succeeded, got %q (error=%s)", status.State, status.Error)
	}
	if !strings.HasPrefix(status.Output, "data:image/png;base64,") {
		t.Errorf("expected PNG data URI, got %q", status.Output)
	}
	if !viewCalled {
		t.Error("expected /view to be called to fetch the image")
	}
}

func TestComfyUI_Poll_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"pid": {
				"outputs": {},
				"status": { "status_str": "error", "completed": false, "messages": [] }
			}
		}`))
	}))
	defer srv.Close()

	c := NewComfyUI(srv.URL)
	status, err := c.Poll(context.Background(), "pid")
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if status.State != "failed" {
		t.Errorf("expected failed, got %q", status.State)
	}
}

// decodeJSON is a tiny helper reused across kokoro + comfyui tests.
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

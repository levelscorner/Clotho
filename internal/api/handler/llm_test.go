package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLLMHandler_Models(t *testing.T) {
	t.Parallel()

	const happyBody = `{"models":[{"name":"llama3:latest","size":4700000000,"modified_at":"2026-04-01T12:00:00Z"},{"name":"mistral:7b","size":4100000000,"modified_at":"2026-03-15T09:00:00Z"}]}`

	tests := []struct {
		name           string
		provider       string
		upstream       http.HandlerFunc
		clientTimeout  time.Duration
		wantStatus     int
		wantRespStatus string
		wantCount      int
		wantFirstName  string
	}{
		{
			name:     "happy path",
			provider: "ollama",
			upstream: func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/api/tags") {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(happyBody))
			},
			wantStatus:     http.StatusOK,
			wantRespStatus: "ok",
			wantCount:      2,
			wantFirstName:  "llama3:latest",
		},
		{
			name:     "upstream 500",
			provider: "ollama",
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantStatus:     http.StatusOK,
			wantRespStatus: "ollama_not_running",
			wantCount:      0,
		},
		{
			name:     "malformed JSON",
			provider: "ollama",
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"models": [not-json`))
			},
			wantStatus:     http.StatusOK,
			wantRespStatus: "ollama_not_running",
			wantCount:      0,
		},
		{
			name:     "timeout",
			provider: "ollama",
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(200 * time.Millisecond)
				_, _ = w.Write([]byte(happyBody))
			},
			clientTimeout:  20 * time.Millisecond,
			wantStatus:     http.StatusOK,
			wantRespStatus: "ollama_not_running",
			wantCount:      0,
		},
		{
			name:       "invalid provider",
			provider:   "openai",
			upstream:   func(w http.ResponseWriter, _ *http.Request) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tc.upstream)
			defer srv.Close()

			h := NewLLMHandler(srv.URL)
			if tc.clientTimeout > 0 {
				h.HTTPClient = &http.Client{Timeout: tc.clientTimeout}
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/llm/models?provider="+tc.provider, nil)
			rr := httptest.NewRecorder()

			h.Models(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body=%s)", rr.Code, tc.wantStatus, rr.Body.String())
			}

			if tc.wantStatus != http.StatusOK {
				return
			}

			var body ModelsResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v (body=%s)", err, rr.Body.String())
			}

			if body.Status != tc.wantRespStatus {
				t.Errorf("response status: got %q, want %q", body.Status, tc.wantRespStatus)
			}
			if len(body.Models) != tc.wantCount {
				t.Errorf("model count: got %d, want %d", len(body.Models), tc.wantCount)
			}
			if tc.wantFirstName != "" && len(body.Models) > 0 && body.Models[0].Name != tc.wantFirstName {
				t.Errorf("first model: got %q, want %q", body.Models[0].Name, tc.wantFirstName)
			}
		})
	}
}

func TestLLMHandler_Models_UnreachableUpstream(t *testing.T) {
	t.Parallel()

	// Start a server, capture URL, then close it — now the URL is unreachable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	h := NewLLMHandler(url)
	h.HTTPClient = &http.Client{Timeout: 200 * time.Millisecond}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/llm/models?provider=ollama", nil)
	rr := httptest.NewRecorder()
	h.Models(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	var body ModelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ollama_not_running" {
		t.Errorf("status: got %q, want ollama_not_running", body.Status)
	}
	if len(body.Models) != 0 {
		t.Errorf("models: got %d, want 0", len(body.Models))
	}
}

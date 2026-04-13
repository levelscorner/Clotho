package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// OllamaModel is the shape returned for a single Ollama model in /api/v1/llm/models.
type OllamaModel struct {
	Name     string `json:"name"`
	Size     int64  `json:"size,omitempty"`
	Modified string `json:"modified,omitempty"`
}

// ModelsResponse is returned from GET /api/v1/llm/models.
// Status is "ok" on success or "ollama_not_running" when the upstream
// Ollama daemon is unreachable / returning errors / returning malformed JSON.
type ModelsResponse struct {
	Models []OllamaModel `json:"models"`
	Status string        `json:"status"`
}

// LLMHandler serves LLM-provider discovery endpoints.
type LLMHandler struct {
	OllamaBaseURL string
	HTTPClient    *http.Client
}

// NewLLMHandler constructs an LLMHandler that will query the given Ollama base URL.
// A short timeout is used so a down daemon never stalls the UI.
func NewLLMHandler(baseURL string) *LLMHandler {
	return &LLMHandler{
		OllamaBaseURL: baseURL,
		HTTPClient:    &http.Client{Timeout: 3 * time.Second},
	}
}

// Routes registers LLM discovery routes.
func (h *LLMHandler) Routes(r chi.Router) {
	r.Get("/api/v1/llm/models", h.Models)
}

// ollamaTagsResponse mirrors the shape returned by `GET {ollama}/api/tags`.
type ollamaTagsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		Size       int64  `json:"size"`
		ModifiedAt string `json:"modified_at"`
	} `json:"models"`
}

// Models handles GET /api/v1/llm/models?provider=ollama.
// Only "ollama" is supported today; other providers are served by ProviderHandler.
// On any transport / parse error the response is 200 with an empty model list
// and status "ollama_not_running" — never 5xx — so the frontend can render a hint.
func (h *LLMHandler) Models(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider != "ollama" {
		writeError(w, http.StatusBadRequest, "unsupported provider")
		return
	}

	notRunning := ModelsResponse{Models: []OllamaModel{}, Status: "ollama_not_running"}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.OllamaBaseURL+"/api/tags", nil)
	if err != nil {
		writeJSON(w, http.StatusOK, notRunning)
		return
	}

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusOK, notRunning)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusOK, notRunning)
		return
	}

	var parsed ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		writeJSON(w, http.StatusOK, notRunning)
		return
	}

	models := make([]OllamaModel, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		models = append(models, OllamaModel{
			Name:     m.Name,
			Size:     m.Size,
			Modified: m.ModifiedAt,
		})
	}

	writeJSON(w, http.StatusOK, ModelsResponse{Models: models, Status: "ok"})
}

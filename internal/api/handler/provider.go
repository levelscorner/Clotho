package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/user/clotho/internal/llm"
)

// ProviderInfo describes an LLM provider and its suggested models.
type ProviderInfo struct {
	Name      string   `json:"name"`
	Available bool     `json:"available"`
	Models    []string `json:"models"`
}

// providerModels maps provider names to their suggested models.
var providerModels = map[string][]string{
	"openai":     {"gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"},
	"gemini":     {"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"},
	"openrouter": {"anthropic/claude-sonnet-4", "google/gemini-2.0-flash-exp", "meta-llama/llama-3-70b", "mistralai/mistral-large"},
	"ollama":     {"llama3", "mistral", "phi3", "gemma2"},
}

// ProviderHandler handles the provider listing endpoint.
type ProviderHandler struct {
	registry *llm.ProviderRegistry
}

// NewProviderHandler creates a ProviderHandler.
func NewProviderHandler(registry *llm.ProviderRegistry) *ProviderHandler {
	return &ProviderHandler{registry: registry}
}

// Routes registers provider routes on the given router.
func (h *ProviderHandler) Routes(r chi.Router) {
	r.Get("/api/providers", h.List)
}

// List handles GET /api/providers.
func (h *ProviderHandler) List(w http.ResponseWriter, _ *http.Request) {
	registered := h.registry.List()
	registeredSet := make(map[string]bool, len(registered))
	for _, name := range registered {
		registeredSet[name] = true
	}

	// Return all known providers, marking which are available.
	allProviders := []string{"openai", "gemini", "openrouter", "ollama"}
	result := make([]ProviderInfo, 0, len(allProviders))

	for _, name := range allProviders {
		models := providerModels[name]
		if models == nil {
			models = []string{}
		}
		result = append(result, ProviderInfo{
			Name:      name,
			Available: registeredSet[name],
			Models:    models,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

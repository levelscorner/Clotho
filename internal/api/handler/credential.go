package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/llm"
	"github.com/user/clotho/internal/store"
)

// CredentialHandler handles credential CRUD endpoints.
type CredentialHandler struct {
	credentials store.CredentialStore
}

// NewCredentialHandler creates a CredentialHandler.
func NewCredentialHandler(credentials store.CredentialStore) *CredentialHandler {
	return &CredentialHandler{credentials: credentials}
}

// Routes registers credential routes on the given router.
func (h *CredentialHandler) Routes(r chi.Router) {
	r.Post("/api/credentials", h.Create)
	r.Get("/api/credentials", h.List)
	r.Delete("/api/credentials/{id}", h.Delete)
	r.Post("/api/credentials/{id}/test", h.Test)
}

// Create handles POST /api/credentials.
func (h *CredentialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	cred, err := h.credentials.Create(r.Context(), domain.Credential{
		TenantID:     tenantID,
		Provider:     req.Provider,
		PlaintextKey: req.APIKey,
		Label:        req.Label,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create credential")
		return
	}

	writeJSON(w, http.StatusCreated, dto.CredentialFromDomain(cred))
}

// List handles GET /api/credentials.
func (h *CredentialHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	creds, err := h.credentials.ListByTenant(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	writeJSON(w, http.StatusOK, dto.CredentialsFromDomain(creds))
}

// Delete handles DELETE /api/credentials/{id}.
func (h *CredentialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credential ID")
		return
	}

	if err := h.credentials.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "credential not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// credentialTestModels picks the cheapest 1-token-OK model for each
// provider. Anything that exists, accepts MaxTokens=4, and reliably
// rejects bad keys with 401 works. We deliberately avoid o-series /
// gemini-2.5 reasoning models — those burn budget on chain-of-thought
// before any visible token, making "test latency" misleading.
var credentialTestModels = map[string]string{
	"openai":     "gpt-4o-mini",
	"gemini":     "gemini-1.5-flash",
	"openrouter": "openai/gpt-4o-mini",
}

// Test handles POST /api/credentials/{id}/test. Sends a 1-token "ping"
// completion through the credential's provider so the user finds out a
// key is invalid at *configuration* time, not on their first pipeline
// run an hour later. Always returns 200 OK at the HTTP layer; clients
// inspect the `ok` field to render success/failure inline.
func (h *CredentialHandler) Test(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credential ID")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	cred, err := h.credentials.Get(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "credential not found")
		return
	}

	apiKey, err := h.credentials.GetDecrypted(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decrypt credential")
		return
	}

	provider, err := engine.CreateProviderFromCredential(cred.Provider, apiKey)
	if err != nil {
		writeJSON(w, http.StatusOK, dto.CredentialTestResponse{
			OK:       false,
			Provider: cred.Provider,
			Message:  "Provider not supported for credential testing.",
		})
		return
	}

	model, ok := credentialTestModels[cred.Provider]
	if !ok {
		// Fall back to a generic name; the provider call may 404 it,
		// which still surfaces as a clear FailureProvider4xx.
		model = "default"
	}

	// Tight timeout — credential test should never hang the UI.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	_, callErr := provider.Complete(ctx, llm.CompletionRequest{
		Model:      model,
		UserPrompt: "ping",
		MaxTokens:  4,
	})
	latencyMs := time.Since(start).Milliseconds()

	if callErr != nil {
		failure := engine.ClassifyProviderError(callErr, cred.Provider, model)
		writeJSON(w, http.StatusOK, dto.CredentialTestResponse{
			OK:        false,
			LatencyMs: latencyMs,
			Provider:  cred.Provider,
			Model:     model,
			Failure:   failure,
			Message:   failure.Message,
		})
		return
	}

	writeJSON(w, http.StatusOK, dto.CredentialTestResponse{
		OK:        true,
		LatencyMs: latencyMs,
		Provider:  cred.Provider,
		Model:     model,
		Message:   "Connected.",
	})
}

// Compile-time assertion that domain is reachable from this file even
// after the engine refactor — keeps the import block honest if someone
// later removes domain.Credential references.
var _ = domain.Credential{}

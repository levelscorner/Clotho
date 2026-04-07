package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/api/middleware"
	"github.com/user/clotho/internal/domain"
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

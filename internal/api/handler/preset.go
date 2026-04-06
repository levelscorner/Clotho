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

// PresetHandler handles preset CRUD endpoints.
type PresetHandler struct {
	presets store.PresetStore
}

// NewPresetHandler creates a PresetHandler.
func NewPresetHandler(presets store.PresetStore) *PresetHandler {
	return &PresetHandler{presets: presets}
}

// Routes registers preset routes on the given router.
func (h *PresetHandler) Routes(r chi.Router) {
	r.Get("/api/presets", h.List)
	r.Post("/api/presets", h.Create)
	r.Get("/api/presets/{id}", h.Get)
	r.Put("/api/presets/{id}", h.Update)
	r.Delete("/api/presets/{id}", h.Delete)
}

// List handles GET /api/presets.
func (h *PresetHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	presets, err := h.presets.List(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list presets")
		return
	}

	writeJSON(w, http.StatusOK, dto.PresetsFromDomain(presets))
}

// Create handles POST /api/presets.
func (h *PresetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	preset, err := h.presets.Create(r.Context(), domain.AgentPreset{
		TenantID:    &tenantID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Config:      req.Config,
		Icon:        req.Icon,
		IsBuiltIn:   false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create preset")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PresetFromDomain(preset))
}

// Get handles GET /api/presets/{id}.
func (h *PresetHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid preset ID")
		return
	}

	preset, err := h.presets.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "preset not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PresetFromDomain(preset))
}

// Update handles PUT /api/presets/{id}.
func (h *PresetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid preset ID")
		return
	}

	var req dto.UpdatePresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	if err := h.presets.Update(r.Context(), domain.AgentPreset{
		ID:          id,
		TenantID:    &tenantID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Config:      req.Config,
		Icon:        req.Icon,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update preset")
		return
	}

	preset, err := h.presets.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated preset")
		return
	}

	writeJSON(w, http.StatusOK, dto.PresetFromDomain(preset))
}

// Delete handles DELETE /api/presets/{id}.
func (h *PresetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid preset ID")
		return
	}

	if err := h.presets.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "preset not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/clotho/internal/api/dto"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/store"
)

// PipelineHandler handles pipeline and pipeline version endpoints.
type PipelineHandler struct {
	pipelines store.PipelineStore
	versions  store.PipelineVersionStore
}

// NewPipelineHandler creates a PipelineHandler.
func NewPipelineHandler(pipelines store.PipelineStore, versions store.PipelineVersionStore) *PipelineHandler {
	return &PipelineHandler{pipelines: pipelines, versions: versions}
}

// Routes registers pipeline routes on the given router.
func (h *PipelineHandler) Routes(r chi.Router) {
	r.Get("/api/projects/{projectID}/pipelines", h.ListByProject)
	r.Post("/api/projects/{projectID}/pipelines", h.Create)
	r.Get("/api/pipelines/{id}", h.Get)
	r.Put("/api/pipelines/{id}", h.Update)
	r.Delete("/api/pipelines/{id}", h.Delete)
	r.Post("/api/pipelines/{id}/versions", h.SaveVersion)
	r.Get("/api/pipelines/{id}/versions", h.ListVersions)
	r.Get("/api/pipelines/{id}/versions/latest", h.GetLatestVersion)
	r.Get("/api/pipelines/{id}/versions/{version}", h.GetVersion)
}

// Create handles POST /api/projects/{projectID}/pipelines.
func (h *PipelineHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var req dto.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	pipeline, err := h.pipelines.Create(r.Context(), domain.Pipeline{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pipeline")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PipelineFromDomain(pipeline))
}

// ListByProject handles GET /api/projects/{projectID}/pipelines.
func (h *PipelineHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	pipelines, err := h.pipelines.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipelines")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelinesFromDomain(pipelines))
}

// Get handles GET /api/pipelines/{id}.
func (h *PipelineHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	pipeline, err := h.pipelines.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineFromDomain(pipeline))
}

// Update handles PUT /api/pipelines/{id}.
func (h *PipelineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	var req dto.UpdatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.pipelines.Update(r.Context(), domain.Pipeline{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update pipeline")
		return
	}

	pipeline, err := h.pipelines.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated pipeline")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineFromDomain(pipeline))
}

// Delete handles DELETE /api/pipelines/{id}.
func (h *PipelineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	if err := h.pipelines.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SaveVersion handles POST /api/pipelines/{id}/versions.
func (h *PipelineHandler) SaveVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	var req dto.SaveVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Graph.Nodes == nil {
		writeError(w, http.StatusBadRequest, "graph with at least a nodes array is required")
		return
	}

	// Determine next version number
	nextVersion := 1
	latest, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err == nil {
		nextVersion = latest.Version + 1
	}

	pv, err := h.versions.Create(r.Context(), domain.PipelineVersion{
		PipelineID: pipelineID,
		Version:    nextVersion,
		Graph:      req.Graph,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save version")
		return
	}

	writeJSON(w, http.StatusCreated, dto.PipelineVersionFromDomain(pv))
}

// ListVersions handles GET /api/pipelines/{id}/versions.
func (h *PipelineHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	versions, err := h.versions.ListByPipeline(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionsFromDomain(versions))
}

// GetVersion handles GET /api/pipelines/{id}/versions/{version}.
func (h *PipelineHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version number")
		return
	}

	pv, err := h.versions.GetByVersion(r.Context(), pipelineID, version)
	if err != nil {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionFromDomain(pv))
}

// GetLatestVersion handles GET /api/pipelines/{id}/versions/latest.
func (h *PipelineHandler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	pipelineID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline ID")
		return
	}

	pv, err := h.versions.GetLatest(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no versions found")
		return
	}

	writeJSON(w, http.StatusOK, dto.PipelineVersionFromDomain(pv))
}

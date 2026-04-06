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

// ProjectHandler handles project CRUD endpoints.
type ProjectHandler struct {
	projects store.ProjectStore
}

// NewProjectHandler creates a ProjectHandler.
func NewProjectHandler(projects store.ProjectStore) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// Routes registers project routes on the given router.
func (h *ProjectHandler) Routes(r chi.Router) {
	r.Post("/api/projects", h.Create)
	r.Get("/api/projects", h.List)
	r.Get("/api/projects/{id}", h.Get)
	r.Put("/api/projects/{id}", h.Update)
	r.Delete("/api/projects/{id}", h.Delete)
}

// Create handles POST /api/projects.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := middleware.TenantIDFromContext(r.Context())

	project, err := h.projects.Create(r.Context(), domain.Project{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, dto.ProjectFromDomain(project))
}

// List handles GET /api/projects.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	projects, err := h.projects.List(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	writeJSON(w, http.StatusOK, dto.ProjectsFromDomain(projects))
}

// Get handles GET /api/projects/{id}.
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	project, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.ProjectFromDomain(project))
}

// Update handles PUT /api/projects/{id}.
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	var req dto.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.projects.Update(r.Context(), domain.Project{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}

	project, err := h.projects.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated project")
		return
	}

	writeJSON(w, http.StatusOK, dto.ProjectFromDomain(project))
}

// Delete handles DELETE /api/projects/{id}.
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project ID")
		return
	}

	if err := h.projects.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
